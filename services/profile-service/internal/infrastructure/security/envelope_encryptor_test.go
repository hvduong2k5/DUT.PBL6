package security

import (
	"context"
	"encoding/hex"
	"testing"
	"time"

	"github.com/omamx/profile-service/internal/domain"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// 1. NHÓM TEST TAMPER DETECTION (GCM Authentication Tag)
// =============================================================================
func TestDecryptPII_TamperedCiphertext_ReturnsErrorNotPanic(t *testing.T) {
	ctx := context.Background()
	vault := NewMockVaultClient()
	enc := NewEnvelopeEncryptor(vault, "profile-pii-kek", 5*time.Minute)

	rawCCCD := "046098001234"
	payload, err := enc.Encrypt(ctx, rawCCCD)
	require.NoError(t, err)

	// Giả lập hacker can thiệp vào cơ sở dữ liệu làm biến đổi 1 byte cuối cùng của Ciphertext
	payload.Ciphertext[len(payload.Ciphertext)-1] ^= 0xFF

	// Thao tác giải mã bắt buộc phải phát hiện can thiệp và không được panic
	require.NotPanics(t, func() {
		decrypted, err := enc.Decrypt(ctx, payload)
		require.ErrorIs(t, err, domain.ErrTamperDetected, "bắt buộc phải trả về ErrTamperDetected khi auth tag không khớp")
		require.Empty(t, decrypted)
	})
}

// =============================================================================
// 2. NHÓM TEST NONCE UNIQUENESS (Chống Tái Sử Dụng Nonce)
// =============================================================================
func TestEncryptPII_1000Calls_NeverReusesNonce(t *testing.T) {
	ctx := context.Background()
	vault := NewMockVaultClient()
	enc := NewEnvelopeEncryptor(vault, "profile-pii-kek", 5*time.Minute)

	seen := make(map[string]bool, 1000)
	rawCCCD := "046098001234"

	for i := 0; i < 1000; i++ {
		payload, err := enc.Encrypt(ctx, rawCCCD)
		require.NoError(t, err)
		require.Len(t, payload.Nonce, 12)

		nonceHex := hex.EncodeToString(payload.Nonce)
		require.False(t, seen[nonceHex], "Phát hiện Nonce bị trùng lặp ở lần gọi thứ %d: %s", i, nonceHex)
		seen[nonceHex] = true
	}
}

// =============================================================================
// 3. NHÓM TEST KEY ROTATION ROUND-TRIP (Zero-Downtime Re-wrapping)
// =============================================================================
func TestKeyRotation_RewrapDEK_OldCiphertextStillDecryptable(t *testing.T) {
	ctx := context.Background()
	vault := NewMockVaultClient()
	enc := NewEnvelopeEncryptor(vault, "profile-pii-kek", 0) // Disable cache to test Vault transit interaction

	rawCCCD := "046098001234"

	// 1. Mã hóa bản ghi ban đầu dưới KEK Version 1
	payload, err := enc.Encrypt(ctx, rawCCCD)
	require.NoError(t, err)
	require.Equal(t, 1, payload.KEKVersion)

	// 2. Xoay KEK trên Vault lên Version 2
	newKEKVer := vault.RotateKEK()
	require.Equal(t, 2, newKEKVer)

	// 3. Bản ghi cũ VẪN GIẢI MÃ ĐƯỢC bình thường (nhờ KEK versioning)
	decryptedBeforeRewrap, err := enc.Decrypt(ctx, payload)
	require.NoError(t, err)
	require.Equal(t, rawCCCD, decryptedBeforeRewrap)

	// 4. Background Job Re-wrap DEK từ v1 sang v2 (không giải mã lại ciphertext CCCD)
	rewrappedDEK, newVersion, err := vault.RewrapDEK(ctx, "profile-pii-kek", payload.EncryptedDEK, payload.KEKVersion)
	require.NoError(t, err)
	require.Equal(t, 2, newVersion)

	// Cập nhật payload mới trong DB
	payload.EncryptedDEK = rewrappedDEK
	payload.KEKVersion = newVersion

	// 5. Giải mã với bản ghi đã re-wrap: Kết quả vẫn chính xác 100%
	decryptedAfterRewrap, err := enc.Decrypt(ctx, payload)
	require.NoError(t, err)
	require.Equal(t, rawCCCD, decryptedAfterRewrap)
}

// =============================================================================
// 4. NHÓM TEST IN-MEMORY SHORT-LIVED DEK CACHE & ZEROIZE ON EVICTION
// =============================================================================
func TestEnvelopeEncryption_InMemoryDEKCache(t *testing.T) {
	ctx := context.Background()
	vault := NewMockVaultClient()
	// TTL ngắn 50ms phục vụ test
	enc := NewEnvelopeEncryptor(vault, "profile-pii-kek", 50*time.Millisecond)

	rawCCCD := "046098001234"
	payload, err := enc.Encrypt(ctx, rawCCCD)
	require.NoError(t, err)

	vault.decryptCallCount.Store(0)

	// Lần 1: Cache Miss -> Gọi Vault Unwrap DEK
	res1, err := enc.Decrypt(ctx, payload)
	require.NoError(t, err)
	require.Equal(t, rawCCCD, res1)
	require.Equal(t, int64(1), vault.decryptCallCount.Load(), "Lần 1 phải gọi Vault unwrap")

	// Lần 2: Cache Hit -> Đọc từ RAM, không gọi Vault
	res2, err := enc.Decrypt(ctx, payload)
	require.NoError(t, err)
	require.Equal(t, rawCCCD, res2)
	require.Equal(t, int64(1), vault.decryptCallCount.Load(), "Lần 2 phải đọc từ cache RAM, không gọi Vault")

	// Đợi TTL 50ms hết hạn
	time.Sleep(70 * time.Millisecond)

	// Lần 3: Cache Expired -> Lại gọi Vault
	res3, err := enc.Decrypt(ctx, payload)
	require.NoError(t, err)
	require.Equal(t, rawCCCD, res3)
	require.Equal(t, int64(2), vault.decryptCallCount.Load(), "Lần 3 sau khi cache hết hạn phải gọi lại Vault")
}

// =============================================================================
// 5. TEST ZERO-DOWNTIME KEY ROTATION: MIXED VERSIONS COEXISTENCE DURING REWRAP
// =============================================================================
func TestKeyRotation_MixedVersionsCoexistDuringRewrap(t *testing.T) {
	ctx := context.Background()
	vault := NewMockVaultClient()
	enc := NewEnvelopeEncryptor(vault, "profile-pii-kek", 0)

	cccd1 := "046098001111" // Record 1 (sẽ được rewrap trước)
	cccd2 := "046098002222" // Record 2 (chưa kịp rewrap, vẫn KEK v1)

	// Step 1: Cả 2 bản ghi đều được mã hóa ban đầu dưới KEK Version 1
	rec1, err := enc.Encrypt(ctx, cccd1)
	require.NoError(t, err)
	require.Equal(t, 1, rec1.KEKVersion)

	rec2, err := enc.Encrypt(ctx, cccd2)
	require.NoError(t, err)
	require.Equal(t, 1, rec2.KEKVersion)

	// Step 2: Xoay KEK trên Vault lên Version 2
	newKEKVer := vault.RotateKEK()
	require.Equal(t, 2, newKEKVer)

	// Step 3: Giả lập Background Job mới chạy được 50% tiến độ:
	// Record 1 đã được rewrap thành công sang KEK Version 2
	rewrappedDEK1, newVer1, err := vault.RewrapDEK(ctx, "profile-pii-kek", rec1.EncryptedDEK, rec1.KEKVersion)
	require.NoError(t, err)
	rec1.EncryptedDEK = rewrappedDEK1
	rec1.KEKVersion = newVer1

	// Record 2 VẪN CHƯA KỊP REWRAP (vẫn giữ KEK Version 1 trong DB)
	require.Equal(t, 2, rec1.KEKVersion)
	require.Equal(t, 1, rec2.KEKVersion)

	// Step 4: Kiểm chứng Zero-Downtime:
	// Màn hình HR xem đồng thời danh sách thợ xưởng gồm cả Record 1 và Record 2
	// Cả 2 BẮT BUỘC PHẢI GIẢI MÃ ĐÚNG ĐỒNG THỜI
	decrypted1, err := enc.Decrypt(ctx, rec1)
	require.NoError(t, err, "Record 1 (đã rewrap v2) phải giải mã thành công")
	require.Equal(t, cccd1, decrypted1)

	decrypted2, err := enc.Decrypt(ctx, rec2)
	require.NoError(t, err, "Record 2 (chưa rewrap, KEK v1) vẫn phải giải mã thành công mà không gây downtime hay crash")
	require.Equal(t, cccd2, decrypted2)
}

func TestEnvelopeEncryption_DEKZeroizeOnEviction(t *testing.T) {
	ctx := context.Background()
	vault := NewMockVaultClient()

	// TTL siêu ngắn 20ms để test OnEvict zeroize
	enc := NewEnvelopeEncryptor(vault, "profile-pii-kek", 20*time.Millisecond)

	rawCCCD := "046098005555"
	payload, err := enc.Encrypt(ctx, rawCCCD)
	require.NoError(t, err)

	// Lần đầu đọc để đưa DEK vào cache
	_, err = enc.Decrypt(ctx, payload)
	require.NoError(t, err)

	// Lấy trực tiếp tham chiếu slice DEK trong cache trước khi evict
	keyHash := hashDEKKey(payload.EncryptedDEK)
	cachedDEK, found := enc.dekCache.Get(keyHash)
	require.True(t, found)
	require.Len(t, cachedDEK, 32)

	// Xác nhận trước khi evict, DEK không phải là mảng toàn số 0
	isAllZeroBefore := true
	for _, b := range cachedDEK {
		if b != 0 {
			isAllZeroBefore = false
			break
		}
	}
	require.False(t, isAllZeroBefore, "DEK trong RAM trước khi evict phải có dữ liệu khóa thực tế")

	// Đợi quá 20ms và trigger eviction
	time.Sleep(30 * time.Millisecond)
	// Gọi Get để trigger dọn dẹp phần tử quá hạn trong expirable LRU
	enc.dekCache.Remove(keyHash)

	// Xác nhận OnEvict hook đã zeroize toàn bộ byte của DEK trong RAM
	for idx, b := range cachedDEK {
		require.Equal(t, byte(0), b, "Byte thứ %d của DEK phải được zeroize về 0 sau khi evict", idx)
	}
}
