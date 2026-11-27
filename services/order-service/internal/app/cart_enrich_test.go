package app

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/omamx/order-service/internal/catalogprovider"
	"github.com/omamx/order-service/internal/domain"
	catalogpb "github.com/omamx/order-service/internal/gen/catalog/v1"
	common "github.com/omamx/order-service/internal/gen/common/v1"
	goredis "github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type cartCatalogMock struct {
	catalogpb.CatalogServiceClient
	err error
}

func (m cartCatalogMock) ValidateCheckoutPricing(_ context.Context, in *catalogpb.ValidateCheckoutPricingRequest, _ ...grpc.CallOption) (*catalogpb.ValidateCheckoutPricingResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &catalogpb.ValidateCheckoutPricingResponse{
		IsValid: true,
		Items: []*catalogpb.ValidatedCheckoutItem{
			{
				SkuCode:                in.Items[0].SkuCode,
				ProductName:            "Me Xung Gion",
				Quantity:               in.Items[0].Quantity,
				AuthoritativeUnitPrice: &common.Money{CurrencyCode: "VND", Units: 345000, Nanos: 0},
				NetWeight:              &common.Weight{DecimalValue: "500", Unit: common.WeightUnit_WEIGHT_UNIT_GRAM},
				PackagingType:          catalogpb.PackagingType_PACKAGING_TYPE_HOP_GIAY_KRAFT,
				IsSellable:             true,
				VariantVersion:         7,
				PriceVersion:           7,
			},
		},
		CanonicalSubtotal: &common.Money{CurrencyCode: "VND", Units: 345000 * int64(in.Items[0].Quantity), Nanos: 0},
		EvaluatedAt:       timestamppb.New(time.Now().UTC()),
	}, nil
}

func cartTestService(t *testing.T, catalog *catalogprovider.Adapter) (*Service, domain.Principal) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	principal := domain.Principal{ID: "customer-1", Role: "CUSTOMER"}
	cart := domain.Cart{
		Revision: 3,
		Items:    []domain.Item{{ItemID: "MX-500", SKU: "MX-500", Quantity: 2}},
	}
	raw, err := json.Marshal(cart)
	if err != nil {
		t.Fatal(err)
	}
	mr.Set("cart:"+principal.Scope(), string(raw))
	return &Service{
		C:       Config{InternalToken: "test-internal-token"},
		Redis:   client,
		Catalog: catalog,
	}, principal
}

// F-35: a cart read falls back to Catalog gRPC when projection rows are missing
// while keeping the cart itself SKU + quantity only.
func TestCartEnrichesItemsFromCatalogGRPCFallback(t *testing.T) {
	adapter := &catalogprovider.Adapter{Client: cartCatalogMock{}, Token: "token-012345678901234567890123456789"}
	service, principal := cartTestService(t, adapter)

	cart, err := service.Cart(context.Background(), principal)
	if err != nil {
		t.Fatal(err)
	}
	if cart.Revision != 3 || len(cart.Items) != 1 {
		t.Fatalf("unexpected cart: %+v", cart)
	}
	item := cart.Items[0]
	if item.Name != "Me Xung Gion" || item.UnitPrice != 345000 || item.Weight != 500 || item.Version != 7 {
		t.Fatalf("item not enriched: %+v", item)
	}
}

// F-35: when enrichment is impossible (catalog failing), the cart read
// degrades to the SKU-only cart instead of failing.
func TestCartDegradesWhenCatalogFails(t *testing.T) {
	adapter := &catalogprovider.Adapter{Client: cartCatalogMock{err: status.Error(codes.Unavailable, "down")}, Token: "token-012345678901234567890123456789"}
	service, principal := cartTestService(t, adapter)

	cart, err := service.Cart(context.Background(), principal)
	if err != nil {
		t.Fatal(err)
	}
	if len(cart.Items) != 1 {
		t.Fatalf("unexpected cart: %+v", cart)
	}
	item := cart.Items[0]
	if item.SKU != "MX-500" || item.Quantity != 2 || item.Name != "" || item.UnitPrice != 0 {
		t.Fatalf("expected untouched SKU-only item, got %+v", item)
	}
}

// F-35: without a catalog provider configured there is nothing to enrich
// with; the cart read must behave cleanly as SKU-only.
func TestCartWithoutCatalogConfigured(t *testing.T) {
	service, principal := cartTestService(t, nil)

	cart, err := service.Cart(context.Background(), principal)
	if err != nil {
		t.Fatal(err)
	}
	if len(cart.Items) != 1 || cart.Items[0].Name != "" || cart.Items[0].UnitPrice != 0 {
		t.Fatalf("expected SKU-only cart, got %+v", cart)
	}
}

func TestProcessCatalogPriceEventValidation(t *testing.T) {
	s := &Service{}
	if _, err := s.ProcessCatalogPriceEvent(context.Background(), []byte(`not-json`)); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	statusCode, err := s.ProcessCatalogPriceEvent(context.Background(), []byte(`{"specversion":"1.0","id":"11111111-1111-1111-1111-111111111111","type":"vn.omama.catalog.product.updated.v1","data":{}}`))
	if err != nil || statusCode != "IGNORED" {
		t.Fatalf("expected IGNORED for non-price event, got status=%q err=%v", statusCode, err)
	}
	_, err = s.ProcessCatalogPriceEvent(context.Background(), []byte(`{"specversion":"1.0","id":"11111111-1111-1111-1111-111111111111","type":"vn.omama.catalog.product.price.changed.v1","data":{"sku_code":"MX-500","prices":[{"channel":"D2C_WEB","currency_code":"VND","amount_units":50000,"status":"ACTIVE"}]}}`))
	if err == nil || !errors.Is(err, err) {
		t.Fatal("expected DATABASE_UNAVAILABLE when DB is nil")
	}
}
