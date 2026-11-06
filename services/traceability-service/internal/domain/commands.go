package domain

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
)

var (
	ErrInvalidCommand      = errors.New("invalid command identity or idempotency metadata")
	ErrIdempotencyConflict = errors.New("idempotency key already used for another request")
	ErrNotFound            = errors.New("profile or revision not found")
	ErrConflict            = errors.New("profile version, binding or state conflict")
	ErrForbidden           = errors.New("permission denied")
)

type CommandMetadata struct{ PrincipalID, Key, RequestID, TraceParent string }
type CommandResult struct {
	Status   int
	Body     json.RawMessage
	Replayed bool
}
type CreateDraftRequest struct {
	Binding
	Content Content `json:"content"`
}
type EditDraftRequest struct {
	ExpectedVersion int64   `json:"expected_version"`
	Content         Content `json:"content"`
}

func NewUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	h := hex.EncodeToString(b[:])
	return h[:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:], nil
}
func NewQRCode() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), nil
}
