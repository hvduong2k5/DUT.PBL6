package domain

import "errors"

var (
	ErrNotFound              = errors.New("resource not found")
	ErrForbidden             = errors.New("operation is forbidden")
	ErrConflict              = errors.New("resource conflict")
	ErrInvalidArgument       = errors.New("invalid argument")
	ErrIdempotencyConflict   = errors.New("idempotency key was reused with a different request")
	ErrMediaNotReady         = errors.New("media is not ready")
	ErrMediaPolicy           = errors.New("media violates purpose policy")
	ErrDependencyUnavailable = errors.New("required dependency is unavailable")
)
