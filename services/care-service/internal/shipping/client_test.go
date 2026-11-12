package shipping

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/omamx/care-service/internal/domain"
)

func TestNewClientValidatesConfiguration(t *testing.T) {
	for _, test := range []struct {
		name, baseURL, token string
	}{
		{name: "missing URL", token: "internal-secret"},
		{name: "relative URL", baseURL: "/shipping", token: "internal-secret"},
		{name: "credentials in URL", baseURL: "https://user:pass@shipping.local", token: "internal-secret"},
		{name: "missing token", baseURL: "http://shipping.local"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := NewClient(test.baseURL, test.token, nil); err == nil {
				t.Fatal("NewClient() error = nil, want configuration error")
			}
		})
	}
}

func TestCreateForApprovedDecisionRejectsBeforeCallingShipping(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls++ }))
	defer server.Close()
	client, err := NewClient(server.URL, "internal-secret", server.Client())
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		decision ApprovedDecision
	}{
		{name: "still reviewing", decision: validDecision(domain.ReturnCaseReviewing)},
		{name: "rejected", decision: validDecision(domain.ReturnCaseRejected)},
		{name: "no physical return", decision: func() ApprovedDecision {
			d := validDecision(domain.ReturnCaseApproved)
			d.ReturnRequired = false
			return d
		}()},
		{name: "missing package", decision: func() ApprovedDecision { d := validDecision(domain.ReturnCaseApproved); d.PackageID = " "; return d }()},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := client.CreateForApprovedDecision(context.Background(), test.decision); err == nil {
				t.Fatal("CreateForApprovedDecision() error = nil, want precondition error")
			}
		})
	}
	if calls != 0 {
		t.Fatalf("Shipping was called %d times for invalid decisions; want 0", calls)
	}
}

func TestCreateForApprovedDecisionSendsAuthenticatedIdempotentRequest(t *testing.T) {
	decision := validDecision(domain.ReturnCaseApproved)
	var requestBody map[string]any
	var firstKey string
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != http.MethodPost || r.URL.Path != createReturnShipmentPath {
			t.Errorf("request = %s %s; want POST %s", r.Method, r.URL.Path, createReturnShipmentPath)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer internal-secret" {
			t.Errorf("Authorization = %q; want bearer internal token", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q; want application/json", got)
		}
		key := r.Header.Get("Idempotency-Key")
		if !strings.HasPrefix(key, "care-return:") {
			t.Errorf("Idempotency-Key = %q; want care-return prefix", key)
		}
		if calls == 1 {
			firstKey = key
		} else if key != firstKey {
			t.Errorf("retry Idempotency-Key = %q; want stable key %q", key, firstKey)
		}
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Errorf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"return_shipment_id":"ret-123","shipment_number":"RS-123","return_case_id":"case-123","order_id":"order-123","package_id":"package-123","carrier_code":"FAKE","service_code":"STANDARD","status":"CREATED","cod_amount_vnd":0,"fee_payer":"SHOP","provider_estimated_fee_vnd":0,"reason_code":"DAMAGED","revision":1}`))
	}))
	defer server.Close()
	client, err := NewClient(server.URL, "internal-secret", server.Client())
	if err != nil {
		t.Fatal(err)
	}

	for range 2 { // a retry of the same immutable decision must reuse its key.
		shipment, err := client.CreateForApprovedDecision(context.Background(), decision)
		if err != nil {
			t.Fatalf("CreateForApprovedDecision() error = %v", err)
		}
		if shipment.ID != "ret-123" || shipment.CODAmountVND != 0 || shipment.FeePayer != "SHOP" {
			t.Fatalf("unexpected return shipment response: %+v", shipment)
		}
	}
	if requestBody["return_case_id"] != decision.CaseID || requestBody["order_id"] != decision.OrderID || requestBody["package_id"] != decision.PackageID {
		t.Fatalf("request body missing decision linkage: %#v", requestBody)
	}
	if requestBody["carrier_code"] != returnCarrierCode || requestBody["service_code"] != returnServiceCode {
		t.Fatalf("request did not select MVP fake carrier: %#v", requestBody)
	}
}

func TestCreateForApprovedDecisionRejectsMismatchedShippingResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"return_shipment_id":"ret-123","return_case_id":"another-case","order_id":"order-123","package_id":"package-123","status":"CREATED","cod_amount_vnd":0,"fee_payer":"SHOP"}`))
	}))
	defer server.Close()
	client, err := NewClient(server.URL, "internal-secret", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.CreateForApprovedDecision(context.Background(), validDecision(domain.ReturnCasePartiallyApproved)); err == nil {
		t.Fatal("CreateForApprovedDecision() error = nil, want mismatched response error")
	}
}

func TestCreateForApprovedDecisionReturnsTypedShippingError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"error":{"code":"IDEMPOTENCY_CONFLICT"}}`))
	}))
	defer server.Close()
	client, err := NewClient(server.URL, "internal-secret", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.CreateForApprovedDecision(context.Background(), validDecision(domain.ReturnCaseApproved))
	var shippingErr *Error
	if !errors.As(err, &shippingErr) {
		t.Fatalf("error = %T (%v); want *shipping.Error", err, err)
	}
	if shippingErr.StatusCode != http.StatusConflict || shippingErr.Code != "IDEMPOTENCY_CONFLICT" {
		t.Fatalf("shipping error = %+v; want conflict/IDEMPOTENCY_CONFLICT", shippingErr)
	}
}

func validDecision(status domain.ReturnCaseStatus) ApprovedDecision {
	return ApprovedDecision{
		CaseID: "case-123", DecisionVersion: 2, Status: status, ReturnRequired: true,
		OrderID: "order-123", PackageID: "package-123", ReasonCode: "DAMAGED",
		PickupAddress: Address{Name: "Customer", Phone: "+84900000000", Street: "1 Example St", Ward: "Ward 1", Province: "Da Nang", CountryCode: "VN"},
		Destination:   Address{Name: "Warehouse", Phone: "+842800000000", Street: "2 Warehouse Rd", Ward: "Ward 2", Province: "Ho Chi Minh City", CountryCode: "VN"},
		Parcel:        Parcel{WeightGrams: 1000, LengthCM: 20, WidthCM: 15, HeightCM: 10},
	}
}
