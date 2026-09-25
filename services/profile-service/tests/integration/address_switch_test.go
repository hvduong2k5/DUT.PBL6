package integration_test

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omamx/profile-service/internal/domain"
	"github.com/omamx/profile-service/internal/repository"
	"github.com/omamx/profile-service/tests/testhelper"
	"github.com/stretchr/testify/require"
)

func TestAtomicDefaultAddressSwitcher_100ConcurrentGoroutines_AlwaysExactlyOneDefault(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping PostgreSQL integration test in short mode")
	}

	ctx := context.Background()
	pool, cleanup := testhelper.SetupPostgresContainer(t)
	defer cleanup()

	// Seed administrative units for foreign keys
	err := testhelper.SeedSampleAdministrativeUnits(ctx, pool)
	require.NoError(t, err)

	custRepo := repository.NewCustomerRepository(pool)
	addrRepo := repository.NewAddressRepository(pool)

	// Step 1: Create a customer
	customerID := uuid.New()
	userID := uuid.New()
	now := time.Now().UTC()

	err = custRepo.Create(ctx, &domain.CustomerProfile{
		ID:          customerID,
		UserID:      userID,
		FullName:    "Trần Văn Hùng",
		PhoneNumber: "0905333444",
		Email:       "hung.tran@hue-sweet.vn",
		Gender:      "MALE",
		Status:      "ACTIVE",
		Version:     1,
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	require.NoError(t, err)

	// Step 2: Create 5 shipping addresses for this customer (Address 0 is default initially)
	var addressIDs []uuid.UUID
	for i := 0; i < 5; i++ {
		addrID := uuid.New()
		addressIDs = append(addressIDs, addrID)

		isDefault := (i == 0)
		addr := &domain.ShippingAddress{
			ID:            addrID,
			CustomerID:    customerID,
			RecipientName: fmt.Sprintf("Người nhận %d", i+1),
			PhoneNumber:   fmt.Sprintf("090511100%d", i),
			StreetAddress: fmt.Sprintf("%d Lê Lợi", (i+1)*10),
			WardCode:      "WARD-TH-001",
			WardName:      "Phường Thuận Hòa",
			ProvinceCode:  "75",
			ProvinceName:  "Thành phố Huế",
			Label:         "HOME",
			IsDefault:     isDefault,
			IsDeleted:     false,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		err = addrRepo.CreateAddress(ctx, addr)
		require.NoError(t, err)
	}

	// Verify initial default count is 1
	initialCount, err := addrRepo.CountDefaultAddresses(ctx, customerID)
	require.NoError(t, err)
	require.Equal(t, 1, initialCount, "Initially, customer must have exactly 1 default address")

	// Step 3: Trigger 100 concurrent goroutines racing to switch default address randomly
	const concurrency = 100
	var wg sync.WaitGroup
	wg.Add(concurrency)

	startBarrier := make(chan struct{})

	for i := 0; i < concurrency; i++ {
		go func(workerID int) {
			defer wg.Done()

			// Wait for barrier to ensure high-contention simultaneous execution
			<-startBarrier

			r := rand.New(rand.NewSource(time.Now().UnixNano() + int64(workerID)))
			targetIndex := r.Intn(len(addressIDs))
			targetAddrID := addressIDs[targetIndex]

			// Call switch default address
			_ = addrRepo.SwitchDefaultAddress(ctx, customerID, targetAddrID)
		}(i)
	}

	// Release all 100 workers simultaneously
	close(startBarrier)
	wg.Wait()

	// Step 4: Critical Invariant Assertion:
	// Exact count of default addresses for customer must ALWAYS BE 1 (never 0, never > 1)
	finalCount, err := addrRepo.CountDefaultAddresses(ctx, customerID)
	require.NoError(t, err)
	require.Equal(t, 1, finalCount, "Post-concurrency: Customer must have EXACTLY 1 default address")
}
