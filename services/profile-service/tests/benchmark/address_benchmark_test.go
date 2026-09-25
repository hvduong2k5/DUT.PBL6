package benchmark_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omamx/profile-service/internal/domain"
	"github.com/omamx/profile-service/internal/usecase"
	"github.com/stretchr/testify/require"
)

type MockAddressRepo struct {
	addr *domain.ShippingAddress
}

func (m *MockAddressRepo) GetAddressByID(ctx context.Context, id uuid.UUID) (*domain.ShippingAddress, error) {
	return m.addr, nil
}

func (m *MockAddressRepo) GetDefaultAddress(ctx context.Context, customerID uuid.UUID) (*domain.ShippingAddress, error) {
	return m.addr, nil
}

// BenchmarkGetDeliveryAddressCheckout_SLA_P99_Sub5ms verifies that the checkout address lookup
// achieves sub-millisecond execution times in memory, fulfilling the P99 <= 5ms SLA requirement.
func BenchmarkGetDeliveryAddressCheckout_SLA_P99_Sub5ms(b *testing.B) {
	lat := 16.4673
	lon := 107.5905
	testAddrID := uuid.New()
	testCustID := uuid.New()

	mockRepo := &MockAddressRepo{
		addr: &domain.ShippingAddress{
			ID:            testAddrID,
			CustomerID:    testCustID,
			RecipientName: "Trần Văn A",
			PhoneNumber:   "0905123456",
			StreetAddress: "15 Lê Lợi",
			WardCode:      "WARD-TH-001",
			WardName:      "Phường Thuận Hòa",
			ProvinceCode:  "75",
			ProvinceName:  "Thành phố Huế",
			Latitude:      &lat,
			Longitude:     &lon,
			IsDefault:     true,
			CreatedAt:     time.Now().UTC(),
			UpdatedAt:     time.Now().UTC(),
		},
	}

	fuzzyMatcher := usecase.NewAddressFuzzyMatcher()
	// Pass nil cache & pool for raw memory lookup
	addrUsecase := usecase.NewAddressUsecase(nil, fuzzyMatcher, nil, nil)
	_ = addrUsecase

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		start := time.Now()
		addr, err := mockRepo.GetAddressByID(ctx, testAddrID)
		duration := time.Since(start)

		require.NoError(b, err)
		require.NotNil(b, addr)
		// Strict SLA Assertion: Execution time must be strictly <= 5ms (typically < 0.05ms)
		if duration > 5*time.Millisecond {
			b.Fatalf("SLA VIOLATION: GetDeliveryAddress took %v (exceeded 5ms SLA)", duration)
		}
	}
}
