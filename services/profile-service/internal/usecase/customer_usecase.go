package usecase

import (
	"context"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/profile-service/internal/domain"
	"github.com/omamx/profile-service/internal/infrastructure/cache"
	"github.com/omamx/profile-service/internal/repository"
	"github.com/rs/zerolog/log"
	"strings"
)

type CustomerUsecase struct {
	repo  *repository.CustomerRepository
	cache *cache.DualLayerCache
}

func NewCustomerUsecase(repo *repository.CustomerRepository, c *cache.DualLayerCache, _ *pgxpool.Pool) *CustomerUsecase {
	return &CustomerUsecase{repo, c}
}

// Authoritative PII reads bypass cache until durable cache invalidation is available.
func (u *CustomerUsecase) GetProfile(ctx context.Context, id uuid.UUID) (*domain.CustomerProfile, error) {
	return u.repo.GetByID(ctx, id)
}
func (u *CustomerUsecase) GetProfileByUserID(ctx context.Context, id uuid.UUID) (*domain.CustomerProfile, error) {
	return u.repo.GetByUserID(ctx, id)
}
func (u *CustomerUsecase) UpdateProfileWithOCC(ctx context.Context, id uuid.UUID, name, phone, email string, version int) (*domain.CustomerProfile, error) {
	name = strings.TrimSpace(name)
	if err := domain.ValidateText("full_name", name, 150); err != nil {
		return nil, err
	}
	normalized, err := domain.NormalizePhone(phone)
	if err != nil {
		return nil, err
	}
	if version < 1 {
		return nil, &domain.ValidationError{Field: "version", Message: "must be positive"}
	}
	current, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if email != "" && email != current.Email {
		return nil, &domain.ValidationError{Field: "email", Message: "email changes require the identity verification flow"}
	}
	updated, err := u.repo.UpdateProfileWithOptimisticLock(ctx, id, name, normalized, current.Email, version)
	if err != nil {
		return nil, err
	}
	if u.cache != nil {
		if err := u.cache.Invalidate(ctx, id.String()); err != nil {
			log.Warn().Err(err).Msg("profile cache invalidation failed; authoritative reads remain on PostgreSQL")
		}
	}
	return updated, nil
}
