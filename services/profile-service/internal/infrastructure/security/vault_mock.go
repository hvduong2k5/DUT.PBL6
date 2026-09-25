package security

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"errors"
	"sync/atomic"
)

// MockVaultClient simulates HashiCorp Vault Transit Engine in memory for dev and testing
type MockVaultClient struct {
	currentKEKVersion int32
	vaultKeys         map[int][]byte
	decryptCallCount  atomic.Int64
	encryptCallCount  atomic.Int64
	rewrapCallCount   atomic.Int64
}

func NewMockVaultClient() *MockVaultClient {
	kek1 := sha256.Sum256([]byte("mock-master-kek-version-1-key-secret-seed"))
	kek2 := sha256.Sum256([]byte("mock-master-kek-version-2-key-secret-seed"))

	return &MockVaultClient{
		currentKEKVersion: 1,
		vaultKeys: map[int][]byte{
			1: kek1[:],
			2: kek2[:],
		},
	}
}

func (m *MockVaultClient) EncryptDEK(ctx context.Context, keyName string, dek []byte) ([]byte, int, error) {
	m.encryptCallCount.Add(1)
	ver := int(atomic.LoadInt32(&m.currentKEKVersion))
	kek := m.vaultKeys[ver]

	block, err := aes.NewCipher(kek)
	if err != nil {
		return nil, 0, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, 0, err
	}
	nonce := make([]byte, 12)
	copy(nonce, "fixed-nonce1")
	wrapped := gcm.Seal(nil, nonce, dek, nil)

	return wrapped, ver, nil
}

func (m *MockVaultClient) DecryptDEK(ctx context.Context, keyName string, encryptedDEK []byte, keyVersion int) ([]byte, error) {
	m.decryptCallCount.Add(1)
	kek, exists := m.vaultKeys[keyVersion]
	if !exists {
		return nil, errors.New("kek version not found in vault")
	}

	block, err := aes.NewCipher(kek)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, 12)
	copy(nonce, "fixed-nonce1")
	dek, err := gcm.Open(nil, nonce, encryptedDEK, nil)
	if err != nil {
		return nil, err
	}

	return dek, nil
}

func (m *MockVaultClient) RewrapDEK(ctx context.Context, keyName string, encryptedDEK []byte, keyVersion int) ([]byte, int, error) {
	m.rewrapCallCount.Add(1)
	dek, err := m.DecryptDEK(ctx, keyName, encryptedDEK, keyVersion)
	if err != nil {
		return nil, 0, err
	}
	return m.EncryptDEK(ctx, keyName, dek)
}

func (m *MockVaultClient) RotateKEK() int {
	return int(atomic.AddInt32(&m.currentKEKVersion, 1))
}
