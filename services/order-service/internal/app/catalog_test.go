package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omamx/order-service/internal/domain"
	common "github.com/omamx/order-service/internal/gen/common/v1"
	shipping "github.com/omamx/order-service/internal/gen/shipping/v2"
	"github.com/omamx/order-service/internal/shippingprovider"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type codShippingPeer struct{ shipping.ShippingServiceClient }

func (codShippingPeer) CalculateShippingFee(_ context.Context, in *shipping.CalculateShippingFeeRequest, _ ...grpc.CallOption) (*shipping.CalculateShippingFeeResponse, error) {
	return &shipping.CalculateShippingFeeResponse{ContractVersion: "order-3.2", RequestHash: in.RequestHash, QuoteVersion: 1, QuoteId: uuid.NewString(), QuoteExpiresAt: timestamppb.New(time.Now().Add(time.Minute)), ShippingFee: &common.Money{Units: 30000, CurrencyCode: "VND"}, Carrier: "FAKE", ServiceCode: "STANDARD"}, nil
}

func TestCatalogCODSnapshotsCollectionPolicyWithoutClaimingSettlement(t *testing.T) {
	calls := 0
	catalog := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		_ = json.NewEncoder(w).Encode(map[string]any{"snapshot_id": uuid.NewString(), "catalog_version": 1, "valid_until": time.Now().Add(time.Minute), "contract_version": "catalog-checkout-1.0", "channel": "D2C_WEB", "subtotal": 1000, "items": []map[string]any{{"sku_code": "SKU-A", "quantity": 1, "product_name": "Fresh Catalog name", "unit_price": 1000, "weight_grams": 500, "packaging_type": "BOX", "is_active": true}}})
	}))
	defer catalog.Close()
	s := &Service{C: Config{CatalogURL: catalog.URL}, Client: catalog.Client(), Shipping: &shippingprovider.Adapter{Client: codShippingPeer{}}}
	quote := domain.Quote{ID: uuid.NewString(), Method: "COD", Items: []domain.Item{{SKU: "SKU-A", Quantity: 1}}, Address: domain.Address{Name: "TEST", Phone: "+84905123456", Province: "75", Ward: "HUE-01", Street: "TEST"}}
	for _, policy := range []string{" blank ", strings.Repeat("x", 256)} {
		s.C.CODPolicyReference = policy
		if _, err := s.catalogTerms(context.Background(), quote); err == nil {
			t.Fatal("COD admitted without an explicit valid collection policy")
		}
	}
	if calls != 0 {
		t.Fatal("disabled COD contacted a provider")
	}
	s.C.CODPolicyReference = ""
	terms, err := s.catalogTerms(context.Background(), quote)
	if err != nil || terms.Payment.CODPolicyReference == nil || *terms.Payment.CODPolicyReference != "cod-collection-1.0" || terms.Payment.CollectAmount == nil || *terms.Payment.CollectAmount != 31000 || terms.Payment.ExpiresAt != nil || terms.Payment.Provider != nil || terms.Payment.ReceiverProfileID != nil {
		t.Fatal("COD must snapshot only the outstanding collection obligation", terms.Payment, err)
	}
	s.C.CODPolicyReference = "cod-custom-2"
	terms, err = s.catalogTerms(context.Background(), quote)
	if err != nil || terms.Payment.CODPolicyReference == nil || *terms.Payment.CODPolicyReference != "cod-custom-2" {
		t.Fatal("explicit COD collection policy was not snapshotted", terms.Payment, err)
	}
	quote.Method = "VIETQR"
	s.C.CODPolicyReference = ""
	terms, err = s.catalogTerms(context.Background(), quote)
	if err != nil || terms.Payment.CODPolicyReference != nil || terms.Payment.CollectAmount != nil {
		t.Fatal("COD admission policy affected VietQR", terms.Payment, err)
	}
}
