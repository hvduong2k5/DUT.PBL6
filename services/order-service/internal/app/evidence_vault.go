package app

import (
	"context"
	"io"

	"github.com/omamx/order-service/internal/persistence"
)

// evidenceContentVault is the content half of the local evidence vault
// (F-42 slice 3): the EvidenceVault contract only tickets and describes
// objects; moving the bytes happens through the capability-signed URLs
// this service serves itself when the configured vault is the local one.
// An external vault's URLs point at the finance service instead, and
// these helpers then report the capability as unavailable.
type evidenceContentVault interface {
	PutContent(context.Context, string, int64, string, io.Reader) error
	GetContent(context.Context, string, int64, string) (string, []byte, error)
}

func (s *Service) EvidenceVaultPut(ctx context.Context, id string, exp int64, sig string, body io.Reader) error {
	v, ok := s.PaymentVault.(evidenceContentVault)
	if !ok {
		return persistence.ErrCapability
	}
	return v.PutContent(ctx, id, exp, sig, body)
}

func (s *Service) EvidenceVaultGet(ctx context.Context, id string, exp int64, sig string) (string, []byte, error) {
	v, ok := s.PaymentVault.(evidenceContentVault)
	if !ok {
		return "", nil, persistence.ErrCapability
	}
	return v.GetContent(ctx, id, exp, sig)
}
