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

func TestCustomerRepository_OptimisticConcurrencyControl(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping PostgreSQL integration test in short mode")
	}

	ctx := context.Background()
	pool, cleanup := testhelper.SetupPostgresContainer(t)
	defer cleanup()

	repo := repository.NewCustomerRepository(pool)

	// Seed baseline customer
	custID := uuid.New()
	userID := uuid.New()
	now := time.Now().UTC()

	baseCustomer := &domain.CustomerProfile{
		ID:          custID,
		UserID:      userID,
		FullName:    "Lê Thị Thảo",
		PhoneNumber: "0905123456",
		Email:       "thao.le@hue-ocop.vn",
		Gender:      "FEMALE",
		Status:      "ACTIVE",
		Version:     1,
		CreatedAt:   now,
		UpdatedAt:   now,
		Preferences: map[string]any{"dietary_preference": "VEGAN"},
	}
	err := repo.Create(ctx, baseCustomer)
	require.NoError(t, err)

	type testCase struct {
		name            string
		targetID        uuid.UUID
		expectedVersion int
		newName         string
		newPhone        string
		newEmail        string
		setup           func()
		expectedErr     error
		verifyResult    func(t *testing.T, profile *domain.CustomerProfile)
	}

	tests := []testCase{
		{
			name:            "Case 3: Success - Matching version increments to version 2",
			targetID:        custID,
			expectedVersion: 1,
			newName:         "Lê Thị Thảo (Đã cập nhật)",
			newPhone:        "0905123999",
			newEmail:        "thao.le.updated@hue-ocop.vn",
			setup:           func() {},
			expectedErr:     nil,
			verifyResult: func(t *testing.T, p *domain.CustomerProfile) {
				require.NotNil(t, p)
				require.Equal(t, 2, p.Version, "Version must increment from 1 to 2")
				require.Equal(t, "Lê Thị Thảo (Đã cập nhật)", p.FullName)
				require.Equal(t, "0905123999", p.PhoneNumber)
			},
		},
		{
			name:            "Case 1: Conflict (HTTP 409) - Stale expected version produces ErrOptimisticLockConflict",
			targetID:        custID,
			expectedVersion: 1, // Stale! Current version in DB is now 2
			newName:         "Lê Thị Thảo (Concurrent Attempt)",
			newPhone:        "0905000000",
			newEmail:        "thao.le.conflict@hue-ocop.vn",
			setup:           func() {},
			expectedErr:     domain.ErrOptimisticLockConflict,
			verifyResult: func(t *testing.T, p *domain.CustomerProfile) {
				require.Nil(t, p)
			},
		},
		{
			name:            "Case 2: Not Found (HTTP 404) - Non-existent ID produces ErrCustomerNotFound",
			targetID:        uuid.New(), // Random non-existent UUID
			expectedVersion: 1,
			newName:         "Khách Vãng Lai Ảo",
			newPhone:        "0905999999",
			newEmail:        "ghost@hue-ocop.vn",
			setup:           func() {},
			expectedErr:     domain.ErrCustomerNotFound,
			verifyResult: func(t *testing.T, p *domain.CustomerProfile) {
				require.Nil(t, p)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setup != nil {
				tc.setup()
			}

			updated, err := repo.UpdateProfileWithOptimisticLock(
				ctx,
				tc.targetID,
				tc.newName,
				tc.newPhone,
				tc.newEmail,
				tc.expectedVersion,
			)

			if tc.expectedErr != nil {
				require.ErrorIs(t, err, tc.expectedErr, "Error must match expected error exactly")
			} else {
				require.NoError(t, err)
			}

			if tc.verifyResult != nil {
				tc.verifyResult(t, updated)
			}
		})
	}
}
