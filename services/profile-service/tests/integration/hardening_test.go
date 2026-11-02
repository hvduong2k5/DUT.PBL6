package integration_test

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/omamx/profile-service/internal/domain"
	"github.com/omamx/profile-service/internal/repository"
	"github.com/omamx/profile-service/internal/usecase"
	"github.com/omamx/profile-service/tests/testhelper"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"
)

func TestProfileIntegrity(t *testing.T) {
	if testing.Short() {
		t.Skip("real PostgreSQL integration")
	}
	ctx := context.Background()
	pool, cleanup := testhelper.SetupPostgresContainer(t)
	defer cleanup()
	customers := repository.NewCustomerRepository(pool)
	addresses := repository.NewAddressRepository(pool)
	claims := repository.NewClaimRepository(pool)
	id, userID := uuid.New(), uuid.New()
	now := time.Now().UTC()
	require.NoError(t, customers.Create(ctx, &domain.CustomerProfile{ID: id, UserID: userID, FullName: "Customer", PhoneNumber: "0905111111", Status: "ACTIVE", Version: 1, CreatedAt: now, UpdatedAt: now}))
	profile, err := customers.GetByUserID(ctx, userID)
	require.NoError(t, err)
	require.Equal(t, id, profile.ID)
	require.NotEqual(t, id, userID)
	makeAddress := func(i int) *domain.ShippingAddress {
		return &domain.ShippingAddress{ID: uuid.New(), CustomerID: id, RecipientName: "Recipient", PhoneNumber: "0905111111", StreetAddress: fmt.Sprintf("%d Lê Lợi", i+1), WardCode: "WARD-TH-001", ProvinceCode: "75", Label: "HOME", CreatedAt: now, UpdatedAt: now}
	}
	t.Run("concurrent first creation and switches", func(t *testing.T) {
		const n = 20
		results := make(chan error, n)
		var wg sync.WaitGroup
		for i := 0; i < n; i++ {
			wg.Add(1)
			go func(i int) { defer wg.Done(); results <- addresses.CreateAddress(ctx, makeAddress(i)) }(i)
		}
		wg.Wait()
		close(results)
		for err := range results {
			require.NoError(t, err)
		}
		count, err := addresses.CountDefaultAddresses(ctx, id)
		require.NoError(t, err)
		require.Equal(t, 1, count)
		list, err := addresses.ListAddressesByCustomerID(ctx, id)
		require.NoError(t, err)
		require.Len(t, list, n)
		results = make(chan error, n)
		for _, a := range list {
			wg.Add(1)
			go func(a *domain.ShippingAddress) {
				defer wg.Done()
				results <- addresses.SwitchDefaultAddress(ctx, id, a.ID)
			}(a)
		}
		wg.Wait()
		close(results)
		for err := range results {
			require.NoError(t, err)
		}
		count, err = addresses.CountDefaultAddresses(ctx, id)
		require.NoError(t, err)
		require.Equal(t, 1, count)
	})
	t.Run("ownership version and replacement default", func(t *testing.T) {
		a, err := addresses.GetDefaultAddress(ctx, id)
		require.NoError(t, err)
		_, err = addresses.GetOwnedAddress(ctx, uuid.New(), a.ID)
		require.ErrorIs(t, err, domain.ErrAddressNotFound)
		stale := *a
		a.StreetAddress = "Updated street"
		require.NoError(t, addresses.UpdateAddress(ctx, a))
		require.ErrorIs(t, addresses.UpdateAddress(ctx, &stale), domain.ErrOptimisticLockConflict)
		require.ErrorIs(t, addresses.DeleteAddressWithVersion(ctx, id, a.ID, stale.Version), domain.ErrOptimisticLockConflict)
		current, err := addresses.GetOwnedAddress(ctx, id, a.ID)
		require.NoError(t, err)
		require.NoError(t, addresses.DeleteAddressWithVersion(ctx, id, a.ID, current.Version))
		count, err := addresses.CountDefaultAddresses(ctx, id)
		require.NoError(t, err)
		require.Equal(t, 1, count)
		replacement, err := addresses.GetDefaultAddress(ctx, id)
		require.NoError(t, err)
		require.NotEqual(t, a.ID, replacement.ID)
		before := replacement.Version
		require.NoError(t, addresses.SwitchDefaultAddressWithVersion(ctx, id, replacement.ID, before))
		replacement, err = addresses.GetDefaultAddress(ctx, id)
		require.NoError(t, err)
		require.Equal(t, before, replacement.Version)
	})
	t.Run("invalid area and canonical names", func(t *testing.T) {
		a := makeAddress(99)
		a.ProvinceCode = "48"
		require.Error(t, addresses.CreateAddress(ctx, a))
		a.ProvinceCode = "75"
		a.WardName = "forged"
		require.NoError(t, addresses.CreateAddress(ctx, a))
		require.NotEqual(t, "forged", a.WardName)
		require.Equal(t, "+84905111111", a.PhoneNumber)
	})
	t.Run("outbox failure rolls back profile address and claim", func(t *testing.T) {
		_, err := pool.Exec(ctx, `CREATE FUNCTION test_reject_outbox() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'test outbox failure'; END $$;
  CREATE TRIGGER test_outbox_failure BEFORE INSERT ON outbox_events FOR EACH ROW EXECUTE FUNCTION test_reject_outbox()`)
		require.NoError(t, err)
		defer func() {
			_, err := pool.Exec(ctx, `DROP TRIGGER test_outbox_failure ON outbox_events; DROP FUNCTION test_reject_outbox()`)
			require.NoError(t, err)
		}()
		_, err = customers.UpdateProfileWithOptimisticLock(ctx, id, "Should rollback", "0905111111", "", 1)
		require.Error(t, err)
		unchanged, err := customers.GetByID(ctx, id)
		require.NoError(t, err)
		require.Equal(t, 1, unchanged.Version)
		require.Equal(t, "Customer", unchanged.FullName)
		a := makeAddress(100)
		require.Error(t, addresses.CreateAddress(ctx, a))
		_, err = addresses.GetOwnedAddress(ctx, id, a.ID)
		require.ErrorIs(t, err, domain.ErrAddressNotFound)
		require.Error(t, claims.CreateClaim(ctx, id, "TEST-ROLLBACK", "0905111111"))
		_, err = claims.GetActiveClaim(ctx, "TEST-ROLLBACK")
		require.ErrorIs(t, err, domain.ErrClaimNotFound)
	})
	t.Run("claim fails closed without verifier", func(t *testing.T) {
		u := usecase.NewClaimUsecase(claims, pool)
		_, err := u.ClaimGuestOrder(ctx, id, usecase.ClaimOrderRequest{OrderID: "TEST-NO-PROOF", PhoneNumber: "0905111111"})
		require.ErrorIs(t, err, domain.ErrClaimVerificationUnavailable)
		_, err = claims.GetActiveClaim(ctx, "TEST-NO-PROOF")
		require.ErrorIs(t, err, domain.ErrClaimNotFound)
	})
}
