package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/profile-service/internal/domain"
	"github.com/omamx/profile-service/internal/infrastructure/cache"
	"github.com/omamx/profile-service/internal/repository"
)

type AddressUsecase struct {
	addrRepo     *repository.AddressRepository
	fuzzyMatcher *AddressFuzzyMatcher
	cache        *cache.DualLayerCache
	pool         *pgxpool.Pool
}

func NewAddressUsecase(
	addrRepo *repository.AddressRepository,
	fuzzyMatcher *AddressFuzzyMatcher,
	cache *cache.DualLayerCache,
	pool *pgxpool.Pool,
) *AddressUsecase {
	return &AddressUsecase{
		addrRepo:     addrRepo,
		fuzzyMatcher: fuzzyMatcher,
		cache:        cache,
		pool:         pool,
	}
}

// ListAddresses returns non-deleted addresses for customer.
func (u *AddressUsecase) ListAddresses(ctx context.Context, customerID uuid.UUID) ([]*domain.ShippingAddress, error) {
	return u.addrRepo.ListAddressesByCustomerID(ctx, customerID)
}

// CreateAddress saves a new shipping address.
func (u *AddressUsecase) CreateAddress(ctx context.Context, addr *domain.ShippingAddress) error {
	now := time.Now().UTC()
	if addr.ID == uuid.Nil {
		addr.ID = uuid.New()
	}
	addr.CreatedAt = now
	addr.UpdatedAt = now

	// If this is the customer's first address, make it default automatically
	count, err := u.addrRepo.CountDefaultAddresses(ctx, addr.CustomerID)
	if err == nil && count == 0 {
		addr.IsDefault = true
	}

	if err := u.addrRepo.CreateAddress(ctx, addr); err != nil {
		return err
	}

	// Invalidate customer cache
	_ = u.cache.Invalidate(ctx, addr.CustomerID.String())
	return nil
}

// SwitchDefaultAddress atomically sets an address as default and purges dual-layer cache.
func (u *AddressUsecase) SwitchDefaultAddress(ctx context.Context, customerID, addressID uuid.UUID) error {
	if err := u.addrRepo.SwitchDefaultAddress(ctx, customerID, addressID); err != nil {
		return err
	}

	// Purge local L1 + remote Redis and broadcast invalidation to prevent stale address read
	_ = u.cache.Invalidate(ctx, customerID.String())
	return nil
}

// DeleteAddress soft-deletes address and purges cache.
func (u *AddressUsecase) DeleteAddress(ctx context.Context, customerID, addressID uuid.UUID) error {
	if err := u.addrRepo.DeleteAddress(ctx, customerID, addressID); err != nil {
		return err
	}
	_ = u.cache.Invalidate(ctx, customerID.String())
	return nil
}

// ValidateAddress runs the 2-Stage Fuzzy Matching pipeline against administrative units.
// Note: This is an Async/UI path and is STRICTLY excluded from the Checkout Critical Path.
func (u *AddressUsecase) ValidateAddress(ctx context.Context, userInput string) (*MatchResult, error) {
	// Stage 1: Pre-filter top candidates from DB using pg_trgm similarity
	query := `
		SELECT code, name, level
		FROM administrative_units
		WHERE level = 'WARD'
		  AND name % $1
		ORDER BY similarity(name, $1) DESC
		LIMIT 20
	`
	rows, err := u.pool.Query(ctx, query, userInput)
	if err != nil {
		return nil, fmt.Errorf("failed querying administrative candidates: %w", err)
	}
	defer rows.Close()

	var candidates []AdministrativeCandidate
	for rows.Next() {
		var c AdministrativeCandidate
		if err := rows.Scan(&c.Code, &c.Name, &c.Level); err == nil {
			candidates = append(candidates, c)
		}
	}

	if len(candidates) == 0 {
		// Fallback: If no direct trigram match, select active wards of Thừa Thiên Huế for in-memory Levenshtein
		fallbackQuery := `SELECT code, name, level FROM administrative_units WHERE level = 'WARD' LIMIT 50`
		fbRows, fbErr := u.pool.Query(ctx, fallbackQuery)
		if fbErr == nil {
			defer fbRows.Close()
			for fbRows.Next() {
				var c AdministrativeCandidate
				if err := fbRows.Scan(&c.Code, &c.Name, &c.Level); err == nil {
					candidates = append(candidates, c)
				}
			}
		}
	}

	// Stage 2: Refine similarity with Vietnamese diacritics stripping & Levenshtein
	result := u.fuzzyMatcher.RefineCandidates(userInput, candidates)
	if result == nil {
		return nil, fmt.Errorf("no matching administrative ward found for input: %s", userInput)
	}

	return result, nil
}

// GetDeliveryAddressCheckout fetches delivery address on Checkout Critical Path ($P99 <= 5ms).
// Bypasses fuzzy matching; uses direct primary key / default B-Tree lookup.
func (u *AddressUsecase) GetDeliveryAddressCheckout(ctx context.Context, addressID, customerID uuid.UUID) (*domain.ShippingAddress, error) {
	if addressID != uuid.Nil {
		return u.addrRepo.GetAddressByID(ctx, addressID)
	}
	return u.addrRepo.GetDefaultAddress(ctx, customerID)
}
