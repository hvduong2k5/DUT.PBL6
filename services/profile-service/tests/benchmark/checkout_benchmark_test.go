package benchmark_test

import (
	"context"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/profile-service/internal/domain"
	"github.com/omamx/profile-service/internal/repository"
	"github.com/omamx/profile-service/internal/usecase"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
	"time"
)

// This measures the real usecase and PostgreSQL lookup; ns/op is not P99 or an SLA proof.
func BenchmarkCheckoutPostgres(b *testing.B) {
	url := os.Getenv("TEST_PROFILE_DATABASE_URL")
	if url == "" {
		b.Skip("set TEST_PROFILE_DATABASE_URL to an isolated migrated test DB")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	require.NoError(b, err)
	defer pool.Close()
	customers := repository.NewCustomerRepository(pool)
	addresses := repository.NewAddressRepository(pool)
	id := uuid.New()
	now := time.Now().UTC()
	require.NoError(b, customers.Create(ctx, &domain.CustomerProfile{ID: id, UserID: uuid.New(), FullName: "Benchmark", PhoneNumber: "0905111111", Status: "ACTIVE", Version: 1, CreatedAt: now, UpdatedAt: now}))
	defer func() {
		_, err := pool.Exec(ctx, `DELETE FROM customer_profiles WHERE id=$1`, id)
		require.NoError(b, err)
	}()
	a := &domain.ShippingAddress{ID: uuid.New(), CustomerID: id, RecipientName: "Benchmark", PhoneNumber: "0905111111", StreetAddress: "15 Lê Lợi", WardCode: "WARD-TH-001", ProvinceCode: "75", Label: "HOME", CreatedAt: now, UpdatedAt: now}
	require.NoError(b, addresses.CreateAddress(ctx, a))
	u := usecase.NewAddressUsecase(addresses, usecase.NewAddressFuzzyMatcher(), nil, pool)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		result, err := u.GetDeliveryAddressCheckout(ctx, a.ID, id)
		if err != nil || result.ID != a.ID {
			b.Fatalf("lookup failed: %v", err)
		}
	}
	b.StopTimer()
}
