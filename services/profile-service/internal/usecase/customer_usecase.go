package usecase

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/profile-service/internal/domain"
	"github.com/omamx/profile-service/internal/infrastructure/cache"
	"github.com/omamx/profile-service/internal/repository"
)

type CustomerUsecase struct {
	repo  *repository.CustomerRepository
	cache *cache.DualLayerCache
	pool  *pgxpool.Pool
}

func NewCustomerUsecase(repo *repository.CustomerRepository, cache *cache.DualLayerCache, pool *pgxpool.Pool) *CustomerUsecase {
	return &CustomerUsecase{
		repo:  repo,
		cache: cache,
		pool:  pool,
	}
}

// GetProfile retrieves customer profile checking L1 RAM -> L2 Redis -> DB.
func (u *CustomerUsecase) GetProfile(ctx context.Context, customerID uuid.UUID) (*domain.CustomerProfile, error) {
	custIDStr := customerID.String()

	// Check Dual-Layer Cache
	if profile, hit, err := u.cache.Get(ctx, custIDStr); err == nil && hit {
		return profile, nil
	}

	// Cache Miss -> Fetch from DB
	profile, err := u.repo.GetByID(ctx, customerID)
	if err != nil {
		return nil, err
	}

	// Backfill cache with 15-minute TTL
	_ = u.cache.Set(ctx, custIDStr, profile, 15*time.Minute)
	return profile, nil
}

// UpdateProfileWithOCC updates profile with optimistic locking CAS and performs dual-layer cache invalidation.
func (u *CustomerUsecase) UpdateProfileWithOCC(
	ctx context.Context,
	customerID uuid.UUID,
	fullName, phone, email string,
	expectedVersion int,
) (*domain.CustomerProfile, error) {
	// Execute atomic CAS update in DB
	updated, err := u.repo.UpdateProfileWithOptimisticLock(ctx, customerID, fullName, phone, email, expectedVersion)
	if err != nil {
		return nil, err
	}

	// Synchronously invalidate L1 + L2 and broadcast Redis Pub/Sub to all other pods
	_ = u.cache.Invalidate(ctx, customerID.String())

	// Insert ProfileUpdatedEvent into outbox table
	payload, _ := json.Marshal(map[string]any{
		"customer_id":  customerID.String(),
		"full_name":    updated.FullName,
		"phone_number": updated.PhoneNumber,
		"email":        updated.Email,
		"new_version":  updated.Version,
		"updated_at":   updated.UpdatedAt,
	})

	outboxQuery := `
		INSERT INTO outbox_events (id, aggregate_type, aggregate_id, event_type, payload, topic)
		VALUES ($1, 'CustomerProfile', $2, 'ProfileUpdatedEvent', $3, 'profile.events.v1')
	`
	_, _ = u.pool.Exec(ctx, outboxQuery, uuid.New(), customerID.String(), payload)

	return updated, nil
}
