package catalogprovider

import (
	"context"
	"testing"
	"time"

	"github.com/omamx/order-service/internal/domain"
	catalog "github.com/omamx/order-service/internal/gen/catalog/v1"
	common "github.com/omamx/order-service/internal/gen/common/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type fakeCatalogClient struct {
	catalog.CatalogServiceClient
	lastMD  metadata.MD
	lastReq *catalog.ValidateCheckoutPricingRequest
	resp    *catalog.ValidateCheckoutPricingResponse
	err     error
}

func (f *fakeCatalogClient) ValidateCheckoutPricing(ctx context.Context, in *catalog.ValidateCheckoutPricingRequest, _ ...grpc.CallOption) (*catalog.ValidateCheckoutPricingResponse, error) {
	f.lastMD, _ = metadata.FromOutgoingContext(ctx)
	f.lastReq = in
	return f.resp, f.err
}

func TestValidateCheckoutPricingSendsTokenAndMapsSnapshot(t *testing.T) {
	client := &fakeCatalogClient{
		resp: &catalog.ValidateCheckoutPricingResponse{
			IsValid: true,
			Items: []*catalog.ValidatedCheckoutItem{
				{
					SkuCode:                "PEER",
					ProductName:            "Me Xung Hue",
					Quantity:               2,
					AuthoritativeUnitPrice: &common.Money{CurrencyCode: "VND", Units: 50000, Nanos: 0},
					NetWeight:              &common.Weight{DecimalValue: "0.5", Unit: common.WeightUnit_WEIGHT_UNIT_KILOGRAM},
					PackagingType:          catalog.PackagingType_PACKAGING_TYPE_HOP_GIAY_KRAFT,
					IsSellable:             true,
					VariantVersion:         1,
					PriceVersion:           0,
				},
			},
			CanonicalSubtotal: &common.Money{CurrencyCode: "VND", Units: 100000, Nanos: 0},
			EvaluatedAt:       timestamppb.New(time.Now().UTC()),
		},
	}
	adapter := &Adapter{Client: client, Token: "token-012345678901234567890123456789"}
	snap, err := adapter.ValidateCheckoutPricing(context.Background(), []domain.Item{{SKU: "PEER", Quantity: 2}}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := client.lastMD.Get("x-internal-token"); len(got) != 1 || got[0] != adapter.Token {
		t.Fatalf("expected x-internal-token metadata, got %v", got)
	}
	if client.lastReq.Items[0].ClientUnitPrice != nil {
		t.Fatalf("did not expect client_unit_price when sendClientPrice=false")
	}
	if snap.Subtotal != 100000 || snap.Version < 1 || snap.Items[0].Weight != 500 || snap.Items[0].Packaging != "HOP_GIAY_KRAFT" {
		t.Fatalf("unexpected snapshot: %+v", snap)
	}
}

func TestValidateCheckoutPricingMapsPriceChanged(t *testing.T) {
	client := &fakeCatalogClient{
		resp: &catalog.ValidateCheckoutPricingResponse{
			IsValid: false,
			Discrepancies: []*catalog.PriceDiscrepancy{
				{
					SkuCode:           "PEER",
					DiscrepancyReason: catalog.ValidationIssue_VALIDATION_ISSUE_PRICE_CHANGED,
				},
			},
		},
	}
	adapter := &Adapter{Client: client, Token: "token-012345678901234567890123456789"}
	_, err := adapter.ValidateCheckoutPricing(context.Background(), []domain.Item{{SKU: "PEER", Quantity: 2, UnitPrice: 1000}}, true)
	if err == nil {
		t.Fatal("expected error for changed price")
	}
	if d, ok := err.(*domain.Error); !ok || d.HTTP != 409 {
		t.Fatalf("expected 409 conflict, got %v", err)
	}
}
