package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omamx/profile-service/internal/domain"
	"github.com/omamx/profile-service/internal/repository"
	"github.com/omamx/profile-service/tests/testhelper"
	"github.com/stretchr/testify/require"
)

// Provisioning for an Identity-registered customer: the first UserRegistered
// event creates the profile (phone still empty, status PENDING_VERIFICATION),
// redeliveries are no-ops, and Order's resolution path (GetByUserID) then
// finds the customer.
func TestCustomerRepository_CreateIfAbsent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping PostgreSQL integration test in short mode")
	}

	ctx := context.Background()
	pool, cleanup := testhelper.SetupPostgresContainer(t)
	defer cleanup()

	repo := repository.NewCustomerRepository(pool)
	userID := uuid.New()
	now := time.Now().UTC()

	newProfile := func() *domain.CustomerProfile {
		return &domain.CustomerProfile{
			ID:          uuid.New(),
			UserID:      userID,
			FullName:    "Nguyen Van Khach",
			PhoneNumber: "",
			Email:       "khach@example.vn",
			Gender:      "UNSPECIFIED",
			Status:      string(domain.CustomerStatusPendingVerification),
			Version:     1,
			CreatedAt:   now,
			UpdatedAt:   now,
			Preferences: map[string]any{"dietary_preference": "NORMAL"},
		}
	}

	created, err := repo.CreateIfAbsent(ctx, newProfile())
	require.NoError(t, err)
	require.True(t, created, "first delivery must create the profile")

	created, err = repo.CreateIfAbsent(ctx, newProfile())
	require.NoError(t, err)
	require.False(t, created, "redelivered event must not create a second profile")

	got, err := repo.GetByUserID(ctx, userID)
	require.NoError(t, err, "Order resolve path must find the provisioned customer")
	require.Equal(t, userID, got.UserID)
	require.Equal(t, string(domain.CustomerStatusPendingVerification), got.Status)
	require.Empty(t, got.PhoneNumber)
}

func TestCustomerRepository_SetStatusByUserID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping PostgreSQL integration test in short mode")
	}

	ctx := context.Background()
	pool, cleanup := testhelper.SetupPostgresContainer(t)
	defer cleanup()

	repo := repository.NewCustomerRepository(pool)
	userID := uuid.New()
	now := time.Now().UTC()

	_, err := repo.CreateIfAbsent(ctx, &domain.CustomerProfile{
		ID:          uuid.New(),
		UserID:      userID,
		FullName:    "Nguyen Van Khach",
		PhoneNumber: "",
		Email:       "khach@example.vn",
		Gender:      "UNSPECIFIED",
		Status:      string(domain.CustomerStatusPendingVerification),
		Version:     1,
		CreatedAt:   now,
		UpdatedAt:   now,
		Preferences: map[string]any{"dietary_preference": "NORMAL"},
	})
	require.NoError(t, err)

	// Reactivation only applies from SUSPENDED: a pending-verification
	// profile must not be flipped to ACTIVE by a lifecycle event.
	changed, err := repo.SetStatusByUserID(ctx, userID, domain.CustomerStatusActive, domain.CustomerStatusSuspended)
	require.NoError(t, err)
	require.False(t, changed)

	changed, err = repo.SetStatusByUserID(ctx, userID, domain.CustomerStatusSuspended, "")
	require.NoError(t, err)
	require.True(t, changed)

	got, err := repo.GetByUserID(ctx, userID)
	require.NoError(t, err)
	require.Equal(t, string(domain.CustomerStatusSuspended), got.Status)

	changed, err = repo.SetStatusByUserID(ctx, userID, domain.CustomerStatusActive, domain.CustomerStatusSuspended)
	require.NoError(t, err)
	require.True(t, changed)

	// Repeating the same transition is a no-op, not an error.
	changed, err = repo.SetStatusByUserID(ctx, userID, domain.CustomerStatusSuspended, "")
	require.NoError(t, err)
	require.True(t, changed)
	changed, err = repo.SetStatusByUserID(ctx, userID, domain.CustomerStatusSuspended, "")
	require.NoError(t, err)
	require.False(t, changed, "already in target status must report no change")

	// Unknown user: no row, no error (the profile may simply not exist yet).
	changed, err = repo.SetStatusByUserID(ctx, uuid.New(), domain.CustomerStatusSuspended, "")
	require.NoError(t, err)
	require.False(t, changed)
}
