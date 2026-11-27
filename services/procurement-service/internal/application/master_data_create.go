package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math/big"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/omamx/procurement-service/internal/domain"
)

func (s *Service) CreateMasterDataWithIdempotency(ctx context.Context, cmd domain.MasterDataCreateCommand) (json.RawMessage, error) {
	cmd.Actor, cmd.Key = strings.TrimSpace(cmd.Actor), strings.TrimSpace(cmd.Key)
	if cmd.Actor == "" || !validIdempotencyKey(cmd.Key) {
		return nil, domain.ErrInvalidArgument
	}
	canonical := cmd
	canonical.Actor, canonical.Key, canonical.RequestHash = "", "", ""
	switch cmd.Kind {
	case "SUPPLIER":
		if cmd.Supplier == nil || cmd.Material != nil || cmd.Offer != nil {
			return nil, domain.ErrInvalidArgument
		}
		input := cmd.Supplier
		normalized, err := domain.NewSupplierWithLegalIdentifier(input.Code, input.Name, input.SupplierType, input.ContactName, input.Phone, input.Email, input.Address, input.LegalIdentifier)
		if err != nil {
			return nil, err
		}
		if exceeds(normalized.Code, 80) || exceeds(normalized.Name, 240) || exceeds(normalized.ContactName, 160) || exceeds(normalized.Phone, 40) || exceeds(normalized.Email, 254) || exceeds(normalized.LegalIdentifier, 120) {
			return nil, domain.ErrInvalidArgument
		}
		cmd.Supplier = &normalized
		v := normalized
		v.ID = uuid.Nil
		v.CreatedAt, v.UpdatedAt = time.Time{}, time.Time{}
		canonical.Supplier = &v
	case "MATERIAL":
		if cmd.Material == nil || cmd.Supplier != nil || cmd.Offer != nil {
			return nil, domain.ErrInvalidArgument
		}
		input := cmd.Material
		normalized, err := domain.NewRawMaterial(input.Code, input.Name, input.BaseUnit, input.Description)
		if err != nil {
			return nil, err
		}
		if exceeds(normalized.Code, 80) || exceeds(normalized.Name, 240) || exceeds(normalized.BaseUnit, 24) {
			return nil, domain.ErrInvalidArgument
		}
		cmd.Material = &normalized
		v := normalized
		v.ID = uuid.Nil
		v.CreatedAt, v.UpdatedAt = time.Time{}, time.Time{}
		canonical.Material = &v
	case "OFFER":
		if cmd.Offer == nil || cmd.Supplier != nil || cmd.Material != nil {
			return nil, domain.ErrInvalidArgument
		}
		input := cmd.Offer
		if input.EffectiveFrom.IsZero() {
			return nil, domain.ErrInvalidArgument
		}
		normalized, err := domain.NewSupplierMaterialOfferVersion(input.SupplierID, input.MaterialID, input.SupplierSKU, input.Unit, input.UnitPriceMinor, input.Currency, input.MinimumQty, input.LeadTimeDays, input.EffectiveFrom, input.EffectiveTo)
		if err != nil {
			return nil, err
		}
		minimum, _ := new(big.Rat).SetString(normalized.MinimumQty)
		if exceeds(normalized.SupplierSKU, 100) || exceeds(normalized.Unit, 24) || normalized.LeadTimeDays > 2147483647 || minimum.Cmp(big.NewRat(1000000000000, 1)) >= 0 {
			return nil, domain.ErrInvalidArgument
		}
		cmd.Offer = &normalized
		v := normalized
		v.ID = uuid.Nil
		v.Version = 0
		v.CreatedAt, v.UpdatedAt = time.Time{}, time.Time{}
		canonical.Offer = &v
	default:
		return nil, domain.ErrInvalidArgument
	}
	encoded, err := json.Marshal(canonical)
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256(encoded)
	cmd.RequestHash = hex.EncodeToString(digest[:])
	prior, err := s.repo.GetMasterDataCreateResult(ctx, cmd.Actor, cmd.Key)
	if err == nil {
		if prior.Kind != cmd.Kind || prior.RequestHash != cmd.RequestHash {
			return nil, domain.ErrConflict
		}
		return prior.Snapshot, nil
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}
	result, err := s.repo.CreateMasterDataCommand(ctx, cmd)
	if err != nil {
		return nil, err
	}
	return result.Snapshot, nil
}

func exceeds(value string, maximum int) bool { return utf8.RuneCountInString(value) > maximum }
