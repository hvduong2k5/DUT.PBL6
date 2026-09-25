package domain

import "errors"

var (
	ErrCustomerNotFound       = errors.New("customer profile not found")
	ErrAddressNotFound        = errors.New("shipping address not found")
	ErrOptimisticLockConflict = errors.New("optimistic lock conflict: profile version mismatch")
	ErrOrderAlreadyClaimed    = errors.New("order has already been claimed by another customer")
	ErrClaimNotFound          = errors.New("guest order claim not found")
	ErrTamperDetected         = errors.New("pii decryption failed: data corrupted or tampered")
	ErrInvalidNonce           = errors.New("invalid nonce size for AES-GCM")
	ErrVaultUnavailable       = errors.New("hashicorp vault service is unavailable")
)
