package domain

import "errors"

type ValidationError struct{ Field, Message string }

func (e *ValidationError) Error() string { return e.Field + ": " + e.Message }

var (
	ErrClaimVerificationUnavailable = errors.New("guest claim verification is unavailable")
	ErrEmployeeNotFound             = errors.New("employee profile not found")
	ErrCustomerNotFound             = errors.New("customer profile not found")
	ErrAddressNotFound              = errors.New("shipping address not found")
	ErrAddressMatchNotFound         = errors.New("no matching administrative ward")
	ErrOptimisticLockConflict       = errors.New("optimistic lock conflict: profile version mismatch")
	ErrOrderAlreadyClaimed          = errors.New("order has already been claimed by another customer")
	ErrClaimNotFound                = errors.New("guest order claim not found")
	ErrTamperDetected               = errors.New("pii decryption failed: data corrupted or tampered")
	ErrInvalidNonce                 = errors.New("invalid nonce size for AES-GCM")
	ErrVaultUnavailable             = errors.New("hashicorp vault service is unavailable")
)
