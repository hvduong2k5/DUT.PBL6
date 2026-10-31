package integration_test

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

func TestGuestOrderClaim_RevokedClaim_CanBeReclaimedByCorrectOwner(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping PostgreSQL integration test in short mode")
	}

	ctx := context.Background()
	pool, cleanup := testhelper.SetupPostgresContainer(t)
	defer cleanup()

	custRepo := repository.NewCustomerRepository(pool)
	claimRepo := repository.NewClaimRepository(pool)

	now := time.Now().UTC()
	custA := uuid.New()
	custB := uuid.New()

	// Seed customer A and customer B
	_ = custRepo.Create(ctx, &domain.CustomerProfile{
		ID: custA, UserID: uuid.New(), FullName: "Nguyễn Văn A", PhoneNumber: "0905111111", Status: "ACTIVE", Version: 1, CreatedAt: now, UpdatedAt: now,
	})
	_ = custRepo.Create(ctx, &domain.CustomerProfile{
		ID: custB, UserID: uuid.New(), FullName: "Trần Thị B", PhoneNumber: "0905222222", Status: "ACTIVE", Version: 1, CreatedAt: now, UpdatedAt: now,
	})

	orderID := "ORD-20261015-0042"

	// Step 1: Customer A erroneously claims the order
	err := claimRepo.CreateClaim(ctx, custA, orderID, "0905111111")
	require.NoError(t, err)

	// Step 2: Customer A's claim is investigated and REVOKED by customer care
	err = claimRepo.RevokeClaim(ctx, orderID)
	require.NoError(t, err)

	// Step 3: Customer B (the legitimate owner) claims the same order
	// With Partial Unique Index (WHERE claim_status = 'VERIFIED'), this MUST SUCCEED without duplicate key violation!
	err = claimRepo.CreateClaim(ctx, custB, orderID, "0905222222")
	require.NoError(t, err, "Legitimate owner must be able to reclaim order after wrongful claim is REVOKED")

	// Step 4: Verify the active claim belongs to Customer B
	activeClaim, err := claimRepo.GetActiveClaim(ctx, orderID)
	require.NoError(t, err)
	require.Equal(t, custB, activeClaim.CustomerID)
	require.Equal(t, "VERIFIED", activeClaim.ClaimStatus)
}

func TestGuestOrderClaim_AlreadyVerifiedClaim_RejectsDoubleClaim(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping PostgreSQL integration test in short mode")
	}

	ctx := context.Background()
	pool, cleanup := testhelper.SetupPostgresContainer(t)
	defer cleanup()

	custRepo := repository.NewCustomerRepository(pool)
	claimRepo := repository.NewClaimRepository(pool)

	now := time.Now().UTC()
	custA := uuid.New()
	custB := uuid.New()

	_ = custRepo.Create(ctx, &domain.CustomerProfile{
		ID: custA, UserID: uuid.New(), FullName: "Hoàng Văn C", PhoneNumber: "0905333333", Status: "ACTIVE", Version: 1, CreatedAt: now, UpdatedAt: now,
	})
	_ = custRepo.Create(ctx, &domain.CustomerProfile{
		ID: custB, UserID: uuid.New(), FullName: "Phạm Thị D", PhoneNumber: "0905444444", Status: "ACTIVE", Version: 1, CreatedAt: now, UpdatedAt: now,
	})

	orderID := "ORD-20261015-0099"

	// Customer A claims order
	err := claimRepo.CreateClaim(ctx, custA, orderID, "0905333333")
	require.NoError(t, err)

	// Customer B attempts to claim the same order while Customer A's claim is active VERIFIED
	err = claimRepo.CreateClaim(ctx, custB, orderID, "0905444444")
	require.ErrorIs(t, err, domain.ErrOrderAlreadyClaimed, "Attempting to claim an already verified order must yield ErrOrderAlreadyClaimed")
}
