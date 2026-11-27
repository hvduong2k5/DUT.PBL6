package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/omamx/procurement-service/internal/domain"
)

func (s *Service) ExecuteSupplierContractCommand(ctx context.Context, c domain.SupplierContractCommand) (json.RawMessage, error) {
	c.Actor, c.Key, c.Reason = strings.TrimSpace(c.Actor), strings.TrimSpace(c.Key), strings.TrimSpace(c.Reason)
	c.ExpectedUpdatedAt = c.ExpectedUpdatedAt.UTC()
	if c.Actor == "" || exceeds(c.Actor, 160) || !validIdempotencyKey(c.Key) {
		return nil, domain.ErrInvalidArgument
	}
	switch c.Action {
	case "CREATE":
		if c.Contract == nil || c.ContractID != uuid.Nil || !c.ExpectedUpdatedAt.IsZero() || c.Reason != "" {
			return nil, domain.ErrInvalidArgument
		}
		p := c.Contract
		// Attachment verification/download is a separate capability, not an opaque URL shortcut.
		if strings.TrimSpace(p.DocumentFileID) != "" {
			return nil, domain.ErrInvalidArgument
		}
		v, err := domain.NewSupplierContract(p.SupplierID, p.ContractNumber, strings.ToUpper(p.Scope), p.PaymentTerms, "", p.EffectiveFrom, p.EffectiveTo, c.Actor)
		if err != nil {
			return nil, err
		}
		if exceeds(v.ContractNumber, 100) || exceeds(v.Scope, 240) || exceeds(v.PaymentTerms, 4000) {
			return nil, domain.ErrInvalidArgument
		}
		c.Contract = &v
	case "ACTIVATE", "TERMINATE":
		if c.ContractID == uuid.Nil || c.ExpectedUpdatedAt.IsZero() || c.Contract != nil {
			return nil, domain.ErrInvalidArgument
		}
		if c.Action == "TERMINATE" && (c.Reason == "" || exceeds(c.Reason, 2000)) {
			return nil, domain.ErrInvalidArgument
		}
		if c.Action == "ACTIVATE" && c.Reason != "" {
			return nil, domain.ErrInvalidArgument
		}
	default:
		return nil, domain.ErrInvalidArgument
	}
	canonical := c
	canonical.Actor, canonical.Key, canonical.RequestHash = "", "", ""
	if c.Contract != nil {
		v := *c.Contract
		v.ID = uuid.Nil
		v.Revision = 0
		v.CreatedBy = ""
		v.CreatedAt, v.UpdatedAt = time.Time{}, time.Time{}
		canonical.Contract = &v
	}
	encoded, err := json.Marshal(canonical)
	if err != nil {
		return nil, err
	}
	hash := sha256.Sum256(encoded)
	c.RequestHash = hex.EncodeToString(hash[:])
	prior, err := s.repo.GetSupplierContractCommandResult(ctx, c.Actor, c.Key)
	if err == nil {
		if prior.RequestHash != c.RequestHash {
			return nil, domain.ErrConflict
		}
		return prior.Snapshot, nil
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}
	result, err := s.repo.ExecuteSupplierContractCommand(ctx, c)
	if err != nil {
		return nil, err
	}
	return result.Snapshot, nil
}
