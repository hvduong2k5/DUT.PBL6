package returns

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omamx/care-service/internal/domain"
	"github.com/omamx/care-service/internal/shipping"
)

func TestCreateUsesOrderSnapshotAndRejectsForgedOwnership(t *testing.T) {
	actorID := uuid.New()
	provider := &fakeOrderProvider{value: validOrderContext(actorID)}
	repository := &fakeRepository{}
	service := NewService(repository, provider, nil)
	input := validCreateInput(actorID)
	created, applied, err := service.Create(context.Background(), input)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if !applied || created.OrderID != provider.value.OrderID || created.Lines[0].RefundableCeiling != 40000 {
		t.Fatalf("Create() did not persist Order-owned values: applied=%v case=%+v", applied, created)
	}
	if provider.calls != 1 {
		t.Fatalf("Order calls = %d; want 1", provider.calls)
	}

	input.ActorID = uuid.New()
	if _, _, err := service.Create(context.Background(), input); err != domain.ErrNotFound {
		t.Fatalf("Create() error = %v; want non-enumerating not found for another customer's order", err)
	}
}

func TestCreateRejectsQuantityAboveAuthoritativeRemainder(t *testing.T) {
	actorID := uuid.New()
	provider := &fakeOrderProvider{value: validOrderContext(actorID)}
	provider.value.Lines[0].PreviouslyCommittedReturnQuantity = 2
	service := NewService(&fakeRepository{}, provider, nil)
	input := validCreateInput(actorID)
	input.Lines[0].Quantity = 2
	if _, _, err := service.Create(context.Background(), input); err == nil {
		t.Fatal("Create() accepted a quantity above Order's remaining quantity")
	}
}

func TestCreateSnapshotsCeilingForRequestedQuantity(t *testing.T) {
	actorID := uuid.New()
	provider := &fakeOrderProvider{value: validOrderContext(actorID)}
	service := NewService(&fakeRepository{}, provider, nil)
	input := validCreateInput(actorID)
	input.Lines[0].Quantity = 2
	created, _, err := service.Create(context.Background(), input)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if got := created.Lines[0].RefundableCeiling; got != 80000 {
		t.Fatalf("requested-quantity ceiling = %d; want 80000", got)
	}
}

func TestCreateRejectsIncompleteOrOversizedOrderSnapshot(t *testing.T) {
	actorID := uuid.New()
	for _, test := range []struct {
		name   string
		mutate func(*OrderContext)
	}{
		{name: "missing customer identity", mutate: func(order *OrderContext) { order.CustomerID = uuid.Nil }},
		{name: "oversized snapshot", mutate: func(order *OrderContext) { order.Channel = strings.Repeat("x", 70_000) }},
		{name: "oversized line field", mutate: func(order *OrderContext) { order.Lines[0].ProductName = strings.Repeat("x", 256) }},
	} {
		t.Run(test.name, func(t *testing.T) {
			order := validOrderContext(actorID)
			test.mutate(&order)
			repository := &fakeRepository{}
			service := NewService(repository, &fakeOrderProvider{value: order}, nil)
			if _, _, err := service.Create(context.Background(), validCreateInput(actorID)); !errors.Is(err, domain.ErrDependencyUnavailable) {
				t.Fatalf("Create() error = %v; want dependency data error", err)
			}
			if repository.value.ID != uuid.Nil {
				t.Fatal("invalid Order snapshot reached repository")
			}
		})
	}
}

func TestBuildDecisionRejectsAmountAboveApprovedQuantityCeiling(t *testing.T) {
	caseValue := Case{Version: 2, Lines: []CaseLine{{ID: uuid.New(), RequestedQuantity: 2, RefundableCeiling: 20000, RefundableUnitCeiling: 10000, Currency: "VND"}}}
	_, err := buildDecision(ActionInput{
		ActorID: uuid.New(), ReasonCode: "DAMAGED", Note: "Verified", ReturnRequired: true,
		Lines: []DecisionLineInput{{ReturnCaseLineID: caseValue.Lines[0].ID, ApprovedQuantity: 1, ApprovedAmountMinor: 10001}},
	}, caseValue, time.Now().UTC())
	if err == nil {
		t.Fatal("buildDecision() accepted an amount above the Order ceiling")
	}
}

func TestBuildDecisionDerivesPartialApprovalAndPreservesReturnFlag(t *testing.T) {
	first, second := uuid.New(), uuid.New()
	caseValue := Case{Version: 4, Lines: []CaseLine{
		{ID: first, RequestedQuantity: 2, RefundableCeiling: 40000, RefundableUnitCeiling: 20000, Currency: "VND"},
		{ID: second, RequestedQuantity: 1, RefundableCeiling: 10000, RefundableUnitCeiling: 10000, Currency: "VND"},
	}}
	decision, err := buildDecision(ActionInput{
		ActorID: uuid.New(), ReasonCode: "QUALITY", Note: "Part of the request is supported", ReturnRequired: true,
		Lines: []DecisionLineInput{
			{ReturnCaseLineID: first, ApprovedQuantity: 1, ApprovedAmountMinor: 20000},
			{ReturnCaseLineID: second, ApprovedQuantity: 0},
		},
	}, caseValue, time.Now().UTC())
	if err != nil {
		t.Fatalf("buildDecision() error = %v", err)
	}
	if decision.Outcome != domain.ReturnCasePartiallyApproved || decision.ApprovedTotalMinor != 20000 || !decision.ReturnRequired {
		t.Fatalf("unexpected decision: %+v", decision)
	}
}

func TestHTTPOrderClientUsesProposedInternalContract(t *testing.T) {
	actorID := uuid.New()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != orderReturnContextPath+"order-1/care-return-context" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("X-Internal-Token") != "order-secret" {
			t.Errorf("internal token header missing")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"order_id":"order-1","customer_id":"` + actorID.String() + `","contract_version":"care-return-context.v1","captured_at":"2026-11-13T10:00:00Z","return_eligible_at":"2026-11-13T09:00:00Z","eligibility_expires_at":"2026-11-15T09:00:00Z","lines":[]}}`))
	}))
	defer server.Close()
	client, err := NewHTTPOrderClient(server.URL, "order-secret", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	value, err := client.GetReturnContext(context.Background(), "order-1")
	if err != nil {
		t.Fatalf("GetReturnContext() error = %v", err)
	}
	if value.CustomerID != actorID || value.ContractVersion != "care-return-context.v1" {
		t.Fatalf("unexpected Order context: %+v", value)
	}
}

func TestCreateRequestDigestIgnoresTransportRetryMetadata(t *testing.T) {
	input := validCreateInput(uuid.New())
	first := createRequestDigest(input)
	input.RequestID = "different-request-id"
	input.IdempotencyKey = "same-or-caller-key"
	if createRequestDigest(input) != first {
		t.Fatal("request ID/idempotency metadata changed business request digest")
	}
	input.CustomerNote = "changed business payload"
	if createRequestDigest(input) == first {
		t.Fatal("business payload change did not change digest")
	}
	if strings.TrimSpace(input.OrderID) == "" {
		t.Fatal("test setup requires order ID")
	}
}

func TestPhysicalReturnCallsShippingAfterDecisionIsRecorded(t *testing.T) {
	actorID, caseID, lineID := uuid.New(), uuid.New(), uuid.New()
	caseValue := Case{ID: caseID, Code: "RET-TEST", CustomerID: actorID, OrderID: "order-1", Status: domain.ReturnCaseReviewing, Version: 2,
		ReasonCode: "DAMAGED", OrderContext: validOrderContext(actorID), Lines: []CaseLine{{ID: lineID, OrderLineID: "line-1", RequestedQuantity: 1, RefundableCeiling: 40000, RefundableUnitCeiling: 40000, Currency: "VND"}}}
	shippingCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		shippingCalls++
		if r.Header.Get("Authorization") != "Bearer ship-secret" {
			t.Errorf("Shipping internal bearer token was not sent")
		}
		if r.Header.Get("Idempotency-Key") == "" {
			t.Errorf("Shipping request has no idempotency key")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"return_shipment_id":"reverse-1","return_case_id":"` + caseID.String() + `","order_id":"order-1","package_id":"return-package:` + caseID.String() + `","status":"CREATED","cod_amount_vnd":0,"fee_payer":"SHOP"}`))
	}))
	defer server.Close()
	client, err := shipping.NewClient(server.URL, "ship-secret", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	repository := &fakeRepository{value: caseValue}
	service := NewService(repository, nil, client)
	result, _, err := service.Action(context.Background(), ActionInput{
		ActorID: actorID, CaseID: caseID, Action: "DECIDE", ExpectedVersion: 2, ReasonCode: "DAMAGED", Note: "Verified damage", ReturnRequired: true,
		Lines: []DecisionLineInput{{ReturnCaseLineID: lineID, ApprovedQuantity: 1, ApprovedAmountMinor: 35000}}, IdempotencyKey: "decide-return-0001",
	})
	if err != nil {
		t.Fatalf("Action() error = %v", err)
	}
	if shippingCalls != 1 || result.ReverseShipmentID != "reverse-1" || result.Status != domain.ReturnCaseInFulfillment {
		t.Fatalf("unexpected result: calls=%d case=%+v", shippingCalls, result)
	}
	if repository.value.Decision == nil {
		t.Fatal("decision was not recorded before Shipping call")
	}
}

type fakeOrderProvider struct {
	value OrderContext
	calls int
}

func (f *fakeOrderProvider) GetReturnContext(_ context.Context, _ string) (OrderContext, error) {
	f.calls++
	return f.value, nil
}

type fakeRepository struct{ value Case }

func (f *fakeRepository) ReplayCreate(context.Context, uuid.UUID, string, string) (Case, bool, error) {
	return Case{}, false, nil
}

func (f *fakeRepository) Create(_ context.Context, record CreateRecord) (Case, bool, error) {
	f.value = record.Case
	return f.value, true, nil
}
func (f *fakeRepository) GetOwned(_ context.Context, _, _ uuid.UUID) (Case, error) {
	return f.value, nil
}
func (f *fakeRepository) ListOwned(context.Context, uuid.UUID, domain.ReturnCaseStatus, int, int) ([]Case, int, error) {
	return []Case{f.value}, 1, nil
}
func (f *fakeRepository) Get(_ context.Context, _ uuid.UUID) (Case, error) { return f.value, nil }
func (f *fakeRepository) List(context.Context, domain.ReturnCaseStatus, int, int) ([]Case, int, error) {
	return nil, 0, nil
}
func (f *fakeRepository) StatusCounts(context.Context) (map[domain.ReturnCaseStatus]int, error) {
	return map[domain.ReturnCaseStatus]int{}, nil
}
func (f *fakeRepository) Action(_ context.Context, record ActionRecord) (Case, bool, error) {
	if record.Decision != nil {
		f.value.Decision = record.Decision
		f.value.Status = record.Decision.Outcome
	} else if record.Action == "START_REVIEW" {
		f.value.Status = domain.ReturnCaseReviewing
	} else if record.Action == "REQUEST_INFORMATION" {
		f.value.Status = domain.ReturnCaseNeedsInformation
	}
	f.value.Version++
	return f.value, true, nil
}
func (f *fakeRepository) AttachReverseShipment(_ context.Context, _, _ uuid.UUID, shipmentID string, _ time.Time) (Case, error) {
	f.value.ReverseShipmentID = shipmentID
	f.value.Status = domain.ReturnCaseInFulfillment
	f.value.Version++
	return f.value, nil
}

func validCreateInput(actorID uuid.UUID) CreateInput {
	return CreateInput{
		ActorID: actorID, OrderID: "order-1", Resolution: "RETURN_AND_REFUND", ReasonCode: "DAMAGED", CustomerNote: "Product arrived damaged",
		Lines: []RequestedLine{{OrderLineID: "line-1", Quantity: 1}}, IdempotencyKey: "return-create-key-01",
	}
}

func validOrderContext(customerID uuid.UUID) OrderContext {
	now := time.Now().UTC()
	return OrderContext{
		OrderID: "order-1", OrderNumber: "ORD-1", CustomerID: customerID, Channel: "D2C_WEB", Status: "DELIVERED", ContractVersion: "care-return-context.v1",
		CapturedAt: now, ReturnEligibleAt: now.Add(-time.Hour), EligibilityExpiresAt: now.Add(24 * time.Hour),
		PickupAddress: Address{Name: "Customer", Phone: "+84900000000", Street: "1 Example", Ward: "Ward 1", Province: "Da Nang", CountryCode: "VN"},
		Destination:   Address{Name: "Warehouse", Phone: "+84280000000", Street: "2 Warehouse", Ward: "Ward 2", Province: "Ho Chi Minh City", CountryCode: "VN"},
		Parcel:        Parcel{WeightGrams: 1000, LengthCM: 20, WidthCM: 15, HeightCM: 10},
		Lines:         []OrderLine{{OrderLineID: "line-1", ProductID: "product-1", SKU: "SKU-1", ProductName: "Test product", PurchasedQuantity: 3, RefundableUnitCeilingMinor: 40000, Currency: "VND"}},
	}
}
