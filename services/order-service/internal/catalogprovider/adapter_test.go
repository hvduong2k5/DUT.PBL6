package catalogprovider

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/omamx/order-service/internal/domain"
	catalog "github.com/omamx/order-service/internal/gen/catalog/v1"
	common "github.com/omamx/order-service/internal/gen/common/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type flakyCatalogClient struct {
	catalog.CatalogServiceClient
	calls     atomic.Int32
	failUntil int32
	alwaysErr error
}

func (f *flakyCatalogClient) ValidateCheckoutPricing(_ context.Context, in *catalog.ValidateCheckoutPricingRequest, _ ...grpc.CallOption) (*catalog.ValidateCheckoutPricingResponse, error) {
	n := f.calls.Add(1)
	if f.alwaysErr != nil {
		return nil, f.alwaysErr
	}
	if n <= f.failUntil {
		return nil, status.Error(codes.Unavailable, "transient")
	}
	return &catalog.ValidateCheckoutPricingResponse{
		IsValid: true,
		Items: []*catalog.ValidatedCheckoutItem{
			{
				SkuCode:                in.Items[0].SkuCode,
				ProductName:            "Me Xung",
				Quantity:               in.Items[0].Quantity,
				AuthoritativeUnitPrice: &common.Money{CurrencyCode: "VND", Units: 50000, Nanos: 0},
				NetWeight:              &common.Weight{DecimalValue: "500", Unit: common.WeightUnit_WEIGHT_UNIT_GRAM},
				PackagingType:          catalog.PackagingType_PACKAGING_TYPE_HOP_GIAY_KRAFT,
				IsSellable:             true,
				VariantVersion:         1,
				PriceVersion:           3,
			},
		},
		CanonicalSubtotal: &common.Money{CurrencyCode: "VND", Units: 50000 * int64(in.Items[0].Quantity), Nanos: 0},
		EvaluatedAt:       timestamppb.New(time.Now().UTC()),
	}, nil
}

func TestValidateCheckoutPricingRetriesTransientUnavailable(t *testing.T) {
	client := &flakyCatalogClient{failUntil: 2}
	adapter := &Adapter{Client: client, Token: "token-012345678901234567890123456789"}

	snap, err := adapter.ValidateCheckoutPricing(context.Background(), []domain.Item{{SKU: "MX-500", Quantity: 1}}, false)
	if err != nil {
		t.Fatalf("expected retry to succeed, got %v", err)
	}
	if got := client.calls.Load(); got != 3 {
		t.Fatalf("expected 3 attempts, got %d", got)
	}
	if snap.Subtotal != 50000 || snap.Items[0].PriceVersion != 3 {
		t.Fatalf("unexpected snapshot: %+v", snap)
	}
}

func TestValidateCheckoutPricingCircuitBreakerOpens(t *testing.T) {
	client := &flakyCatalogClient{alwaysErr: status.Error(codes.Unavailable, "down")}
	adapter := &Adapter{Client: client, Token: "token-012345678901234567890123456789"}
	items := []domain.Item{{SKU: "MX-500", Quantity: 1}}

	for i := 0; i < 5; i++ {
		_, err := adapter.ValidateCheckoutPricing(context.Background(), items, false)
		if err == nil {
			t.Fatal("expected unavailable error")
		}
	}
	callsAtOpen := client.calls.Load()

	// Next call must short-circuit without invoking the gRPC peer.
	_, err := adapter.ValidateCheckoutPricing(context.Background(), items, false)
	if err == nil {
		t.Fatal("expected circuit-open error")
	}
	if client.calls.Load() != callsAtOpen {
		t.Fatalf("circuit breaker did not short-circuit: calls=%d want=%d", client.calls.Load(), callsAtOpen)
	}
}
