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

func (s *Service) MutateMasterData(ctx context.Context, cmd domain.MasterDataMutation) (json.RawMessage, error) {
	cmd.Actor = strings.TrimSpace(cmd.Actor)
	cmd.Key = strings.TrimSpace(cmd.Key)
	cmd.ExpectedUpdatedAt = cmd.ExpectedUpdatedAt.UTC()
	if cmd.Actor == "" || !validIdempotencyKey(cmd.Key) || cmd.ResourceID == uuid.Nil || cmd.ExpectedUpdatedAt.IsZero() {
		return nil, domain.ErrInvalidArgument
	}
	switch cmd.Action {
	case "EDIT_SUPPLIER":
		if cmd.Supplier == nil || cmd.Material != nil {
			return nil, domain.ErrInvalidArgument
		}
		p := cmd.Supplier
		v, err := domain.NewSupplierWithLegalIdentifier(p.Code, p.Name, p.SupplierType, p.ContactName, p.Phone, p.Email, p.Address, p.LegalIdentifier)
		if err != nil {
			return nil, err
		}
		if exceeds(v.Code, 80) || exceeds(v.Name, 240) || exceeds(v.ContactName, 160) || exceeds(v.Phone, 40) || exceeds(v.Email, 254) || exceeds(v.LegalIdentifier, 120) {
			return nil, domain.ErrInvalidArgument
		}
		v.ID = uuid.Nil
		v.CreatedAt, v.UpdatedAt = time.Time{}, time.Time{}
		cmd.Supplier = &v
	case "EDIT_MATERIAL":
		if cmd.Material == nil || cmd.Supplier != nil {
			return nil, domain.ErrInvalidArgument
		}
		p := cmd.Material
		v, err := domain.NewRawMaterial(p.Code, p.Name, p.BaseUnit, p.Description)
		if err != nil {
			return nil, err
		}
		if exceeds(v.Code, 80) || exceeds(v.Name, 240) || exceeds(v.BaseUnit, 24) {
			return nil, domain.ErrInvalidArgument
		}
		v.ID = uuid.Nil
		v.CreatedAt, v.UpdatedAt = time.Time{}, time.Time{}
		cmd.Material = &v
	case "SUPPLIER_STATUS":
		if cmd.Status != "ACTIVE" && cmd.Status != "SUSPENDED" {
			return nil, domain.ErrInvalidArgument
		}
	case "MATERIAL_STATUS":
		if cmd.Status != "ACTIVE" && cmd.Status != "INACTIVE" {
			return nil, domain.ErrInvalidArgument
		}
	case "OFFER_STATUS":
		if cmd.Active == nil || cmd.SupplierID == uuid.Nil {
			return nil, domain.ErrInvalidArgument
		}
	case "PREFERRED_SUPPLIER": // uuid.Nil deliberately clears the preference.
	default:
		return nil, domain.ErrInvalidArgument
	}
	canonical := cmd
	canonical.Actor, canonical.Key, canonical.RequestHash = "", "", ""
	encoded, err := json.Marshal(canonical)
	if err != nil {
		return nil, err
	}
	hash := sha256.Sum256(encoded)
	cmd.RequestHash = hex.EncodeToString(hash[:])
	result, err := s.repo.GetMasterDataMutationResult(ctx, cmd.Actor, cmd.Key)
	if err == nil {
		if result.RequestHash != cmd.RequestHash {
			return nil, domain.ErrConflict
		}
		return result.Snapshot, nil
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}
	result, err = s.repo.MutateMasterData(ctx, cmd)
	if err != nil {
		return nil, err
	}
	return result.Snapshot, nil
}
