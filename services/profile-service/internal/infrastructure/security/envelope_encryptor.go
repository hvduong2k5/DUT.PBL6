package security

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"sync"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/omamx/profile-service/internal/domain"
)

// VaultKMSClient định nghĩa interface giao tiếp với KMS / HashiCorp Vault Transit Engine
type VaultKMSClient interface {
	EncryptDEK(ctx context.Context, keyName string, dek []byte) (encryptedDEK []byte, keyVersion int, err error)
	DecryptDEK(ctx context.Context, keyName string, encryptedDEK []byte, keyVersion int) (dek []byte, err error)
	RewrapDEK(ctx context.Context, keyName string, encryptedDEK []byte, keyVersion int) (rewrappedDEK []byte, newVersion int, err error)
}

type EncryptedPayload struct {
	Ciphertext   []byte
	Nonce        []byte
	EncryptedDEK []byte
	KEKVersion   int
}

type EnvelopeEncryptor struct {
	kmsClient VaultKMSClient
	keyName   string
	// Short-lived in-memory DEK cache with automatic zeroize on eviction
	dekCache *expirable.LRU[string, []byte]
	mu       sync.RWMutex
}

func NewEnvelopeEncryptor(kms VaultKMSClient, keyName string, cacheTTL time.Duration) *EnvelopeEncryptor {
	// Khởi tạo expirable LRU cache (TTL 3-5 phút, max 1000 keys)
	cache := expirable.NewLRU[string, []byte](1000, func(key string, val []byte) {
		// Hook OnEvict: Tự động zeroize DEK trong RAM khi hết hạn
		for i := range val {
			val[i] = 0
		}
	}, cacheTTL)

	return &EnvelopeEncryptor{
		kmsClient: kms,
		keyName:   keyName,
		dekCache:  cache,
	}
}

// Encrypt mã hóa chuỗi plaintext (CCCD) bằng DEK cục bộ và bọc DEK bằng Vault KEK
func (e *EnvelopeEncryptor) Encrypt(ctx context.Context, plaintext string) (*EncryptedPayload, error) {
	// 1. Sinh Data Encryption Key (DEK) 256-bit ngẫu nhiên
	dek := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, dek); err != nil {
		return nil, err
	}
	defer func() {
		// Zeroize DEK plaintext trong RAM sau khi hàm kết thúc
		for i := range dek {
			dek[i] = 0
		}
	}()

	// 2. Dùng DEK mã hóa dữ liệu với AES-256-GCM
	block, err := aes.NewCipher(dek)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize()) // 12 bytes
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), nil)

	// 3. Gửi DEK lên Vault để KEK mã hóa
	encryptedDEK, kekVersion, err := e.kmsClient.EncryptDEK(ctx, e.keyName, dek)
	if err != nil {
		return nil, err
	}

	return &EncryptedPayload{
		Ciphertext:   ciphertext,
		Nonce:        nonce,
		EncryptedDEK: encryptedDEK,
		KEKVersion:   kekVersion,
	}, nil
}

// Decrypt giải mã dữ liệu sử dụng In-Memory DEK Cache và fallback sang Vault
func (e *EnvelopeEncryptor) Decrypt(ctx context.Context, payload *EncryptedPayload) (string, error) {
	if len(payload.Nonce) != 12 {
		return "", domain.ErrInvalidNonce
	}

	// 1. Kiểm tra cache DEK ngắn hạn trong RAM
	cacheKey := hashDEKKey(payload.EncryptedDEK)
	var dek []byte

	e.mu.RLock()
	cachedDEK, found := e.dekCache.Get(cacheKey)
	e.mu.RUnlock()

	if found {
		// Sao chép an toàn để sử dụng
		dek = make([]byte, len(cachedDEK))
		copy(dek, cachedDEK)
	} else {
		// Cache Miss: Gọi Vault Transit Engine unwrap DEK
		unwrappedDEK, err := e.kmsClient.DecryptDEK(ctx, e.keyName, payload.EncryptedDEK, payload.KEKVersion)
		if err != nil {
			return "", domain.ErrVaultUnavailable
		}
		dek = unwrappedDEK

		// Lưu bản sao vào cache RAM
		dekCopy := make([]byte, len(dek))
		copy(dekCopy, dek)
		e.mu.Lock()
		e.dekCache.Add(cacheKey, dekCopy)
		e.mu.Unlock()
	}

	defer func() {
		for i := range dek {
			dek[i] = 0
		}
	}()

	// 2. Dùng DEK giải mã ciphertext
	block, err := aes.NewCipher(dek)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	plaintextBytes, err := gcm.Open(nil, payload.Nonce, payload.Ciphertext, nil)
	if err != nil {
		return "", domain.ErrTamperDetected
	}

	return string(plaintextBytes), nil
}

func hashDEKKey(encryptedDEK []byte) string {
	h := sha256.Sum256(encryptedDEK)
	return hex.EncodeToString(h[:])
}
