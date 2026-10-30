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
	"time"
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

// ProvisionFromRegistration creates the customer profile for a principal that
// Identity has just registered, so Order can resolve principal -> customer.
//
// Identity's UserRegistered event carries no phone number, while the schema
// requires one; the profile is therefore created with an empty phone and
// status PENDING_VERIFICATION, and the customer completes the number through
// the normal profile update flow. Provisioning is idempotent: a redelivered
// event returns created=false and leaves the existing profile untouched.
func (u *CustomerUsecase) ProvisionFromRegistration(ctx context.Context, principalID uuid.UUID, email, firstName, lastName string) (bool, error) {
	if principalID == uuid.Nil {
		return false, &domain.ValidationError{Field: "user_id", Message: "must be a non-zero UUID"}
	}
	fullName := strings.TrimSpace(strings.TrimSpace(firstName) + " " + strings.TrimSpace(lastName))
	if fullName == "" {
		// Registration always carries an email; use it as a display-name
		// placeholder until the customer sets a real name.
		fullName = strings.TrimSpace(email)
	}
	now := time.Now().UTC()
	profile := &domain.CustomerProfile{
		ID:          uuid.New(),
		UserID:      principalID,
		FullName:    fullName,
		PhoneNumber: "",
		Email:       strings.TrimSpace(email),
		Gender:      "UNSPECIFIED",
		Preferences: map[string]any{"favorite_products": []string{}, "dietary_preference": "NORMAL", "allergy_alert": []string{}},
		Status:      string(domain.CustomerStatusPendingVerification),
		Version:     1,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	return u.repo.CreateIfAbsent(ctx, profile)
}

// SuspendByUserID mirrors an Identity account suspension onto the profile.
func (u *CustomerUsecase) SuspendByUserID(ctx context.Context, userID uuid.UUID) error {
	_, err := u.repo.SetStatusByUserID(ctx, userID, domain.CustomerStatusSuspended, "")
	return err
}

// ReactivateByUserID mirrors an Identity account reactivation. Only a
// suspended profile is reactivated; a profile still pending verification
// keeps its status.
func (u *CustomerUsecase) ReactivateByUserID(ctx context.Context, userID uuid.UUID) error {
	_, err := u.repo.SetStatusByUserID(ctx, userID, domain.CustomerStatusActive, domain.CustomerStatusSuspended)
	return err
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
