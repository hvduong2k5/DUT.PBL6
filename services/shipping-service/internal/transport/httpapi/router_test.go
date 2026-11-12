package httpapi

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/omamx/shipping-service/internal/application"
	"github.com/omamx/shipping-service/internal/carrier"
	ghncarrier "github.com/omamx/shipping-service/internal/carrier/ghn"
	"github.com/omamx/shipping-service/internal/domain"
	"github.com/rs/zerolog"
)

type httpMemoryRepository struct {
	shipments         map[uuid.UUID]*domain.Shipment
	keys              map[string]uuid.UUID
	hashes            map[string]string
	actionsByKey      map[string]httpActionBinding
	actionsByShipment map[string]httpActionBinding
	timeline          map[uuid.UUID][]domain.TrackingEvent
	pingErr           error
}

type httpActionBinding struct {
	shipmentID uuid.UUID
	action     string
	key        string
	hash       string
}

func newHTTPMemoryRepository() *httpMemoryRepository {
	return &httpMemoryRepository{
		shipments: make(map[uuid.UUID]*domain.Shipment),
		keys:      make(map[string]uuid.UUID), hashes: make(map[string]string),
		actionsByKey: make(map[string]httpActionBinding), actionsByShipment: make(map[string]httpActionBinding),
		timeline: make(map[uuid.UUID][]domain.TrackingEvent),
	}
}

type fixedStaffAuthenticator struct{}

func (fixedStaffAuthenticator) Authenticate(_ context.Context, authorization string) (StaffPrincipal, error) {
	if authorization != "Bearer staff-jwt-test" {
		return StaffPrincipal{}, errStaffUnauthenticated
	}
	return StaffPrincipal{UserID: "a0000000-0000-0000-0000-000000000001"}, nil
}

func (r *httpMemoryRepository) Ping(context.Context) error { return r.pingErr }

func (r *httpMemoryRepository) BeginCreation(_ context.Context, key, requestHash string, shipment *domain.Shipment) (*domain.Shipment, bool, error) {
	if id, ok := r.keys[key]; ok {
		if r.hashes[key] != requestHash {
			return nil, false, domain.ErrIdempotencyConflict
		}
		return r.clone(r.shipments[id]), false, nil
	}
	r.keys[key] = shipment.ID
	r.hashes[key] = requestHash
	r.shipments[shipment.ID] = r.clone(shipment)
	return r.clone(shipment), true, nil
}

func (r *httpMemoryRepository) BindAction(_ context.Context, shipmentID uuid.UUID, action, key, requestHash string) error {
	binding := httpActionBinding{shipmentID: shipmentID, action: action, key: key, hash: requestHash}
	if previous, exists := r.actionsByKey[key]; exists {
		if previous != binding {
			return domain.ErrIdempotencyConflict
		}
		return nil
	}
	shipmentAction := action + ":" + shipmentID.String()
	if previous, exists := r.actionsByShipment[shipmentAction]; exists {
		if previous != binding {
			return domain.ErrIdempotencyConflict
		}
		return nil
	}
	r.actionsByKey[key] = binding
	r.actionsByShipment[shipmentAction] = binding
	return nil
}

func (r *httpMemoryRepository) Save(_ context.Context, shipment *domain.Shipment, expectedRevision int64, tracking *domain.TrackingEvent, _ *domain.DomainEvent) error {
	stored := r.shipments[shipment.ID]
	if stored == nil {
		return domain.ErrShipmentNotFound
	}
	if stored.Revision != expectedRevision {
		return domain.ErrConcurrentUpdate
	}
	r.shipments[shipment.ID] = r.clone(shipment)
	if tracking != nil {
		r.timeline[shipment.ID] = append(r.timeline[shipment.ID], *tracking)
	}
	return nil
}

func (r *httpMemoryRepository) GetByID(_ context.Context, id uuid.UUID) (*domain.Shipment, error) {
	shipment := r.shipments[id]
	if shipment == nil {
		return nil, domain.ErrShipmentNotFound
	}
	return r.clone(shipment), nil
}

func (r *httpMemoryRepository) GetByTracking(_ context.Context, carrierCode, trackingCode string) (*domain.Shipment, error) {
	for _, shipment := range r.shipments {
		if shipment.CarrierCode == carrierCode && shipment.TrackingCode == trackingCode {
			return r.clone(shipment), nil
		}
	}
	return nil, domain.ErrTrackingNotFound
}

func (r *httpMemoryRepository) ListByOrder(_ context.Context, orderID string) ([]domain.Shipment, error) {
	var items []domain.Shipment
	for _, shipment := range r.shipments {
		if shipment.OrderID == orderID {
			items = append(items, *r.clone(shipment))
		}
	}
	return items, nil
}

func (r *httpMemoryRepository) List(_ context.Context, filter application.ListFilter) ([]domain.Shipment, error) {
	var items []domain.Shipment
	for _, shipment := range r.shipments {
		if filter.Status != "" && shipment.Status != filter.Status ||
			filter.CarrierCode != "" && shipment.CarrierCode != filter.CarrierCode ||
			filter.OrderID != "" && shipment.OrderID != filter.OrderID {
			continue
		}
		items = append(items, *r.clone(shipment))
	}
	if filter.Offset >= len(items) {
		return nil, nil
	}
	items = items[filter.Offset:]
	if filter.Limit < len(items) {
		items = items[:filter.Limit]
	}
	return items, nil
}

func (r *httpMemoryRepository) TrackingTimeline(_ context.Context, id uuid.UUID) ([]domain.TrackingEvent, error) {
	items := append([]domain.TrackingEvent(nil), r.timeline[id]...)
	return items, nil
}

func (r *httpMemoryRepository) clone(shipment *domain.Shipment) *domain.Shipment {
	if shipment == nil {
		return nil
	}
	copy := *shipment
	return &copy
}

type quoteFailureCarrier struct{}

func (quoteFailureCarrier) Code() string { return "QUOTE-FAILURE" }
func (quoteFailureCarrier) Quote(context.Context, carrier.QuoteRequest) (carrier.Quote, error) {
	return carrier.Quote{}, errors.New("provider is unavailable")
}
func (quoteFailureCarrier) CreateShipment(context.Context, carrier.CreateRequest) (carrier.CreateResult, error) {
	return carrier.CreateResult{}, errors.New("not used")
}
func (quoteFailureCarrier) CancelShipment(context.Context, carrier.CancelRequest) error {
	return errors.New("not used")
}
func (quoteFailureCarrier) GenerateLabelURL(context.Context, carrier.LabelRequest) (carrier.LabelResult, error) {
	return carrier.LabelResult{}, errors.New("not used")
}
func (quoteFailureCarrier) ParseWebhook([]byte) (carrier.WebhookUpdate, error) {
	return carrier.WebhookUpdate{}, carrier.ErrInvalidWebhook
}

func TestGHNWebhookIsHiddenWhenDisabled(t *testing.T) {
	handler := NewRouter(nil, "internal-token", "fake-secret", "", zerolog.Nop())
	request := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/logistics/ghn", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", response.Code)
	}
}

func TestQuoteMapsCarrierFailureToBadGateway(t *testing.T) {
	service := application.NewService(newHTTPMemoryRepository(), carrier.NewRegistry(quoteFailureCarrier{}))
	handler := NewRouter(service, "internal-token", "fake-secret", "", zerolog.Nop())
	body := `{"carrier_code":"QUOTE-FAILURE","destination":{"name":"Customer","phone":"0905000000","street":"1 Main Street","ward":"Ward 1","province":"Hue","country_code":"VN"},"parcel":{"weight_grams":1000,"length_cm":30,"width_cm":20,"height_cm":15}}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/internal/shipping/quotes", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer internal-token")
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	var providerError errorBody
	if response.Code != http.StatusBadGateway || json.Unmarshal(response.Body.Bytes(), &providerError) != nil ||
		providerError.Code != "CARRIER_QUOTE_FAILED" || providerError.RequestID == "" {
		t.Fatalf("carrier quote failure should be a safe 502: status=%d body=%s", response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), "provider is unavailable") {
		t.Fatal("provider details must not leak in the public error message")
	}

	invalidBody := strings.Replace(body, `"weight_grams":1000`, `"weight_grams":0`, 1)
	invalidRequest := httptest.NewRequest(http.MethodPost, "/api/v1/internal/shipping/quotes", strings.NewReader(invalidBody))
	invalidRequest.Header.Set("Authorization", "Bearer internal-token")
	invalidRequest.Header.Set("Content-Type", "application/json")
	invalidResponse := httptest.NewRecorder()
	handler.ServeHTTP(invalidResponse, invalidRequest)
	if invalidResponse.Code != http.StatusUnprocessableEntity {
		t.Fatalf("invalid quote input should remain a 422 validation error: status=%d body=%s", invalidResponse.Code, invalidResponse.Body.String())
	}
}

func TestJSONEndpointsRejectUnsupportedContentType(t *testing.T) {
	service := application.NewService(newHTTPMemoryRepository(), carrier.NewRegistry(carrier.NewFake()))
	handler := NewRouter(service, "internal-token", "fake-secret", "", zerolog.Nop())
	tests := []struct {
		name        string
		contentType string
	}{
		{name: "missing content type"},
		{name: "non-json content type", contentType: "text/plain"},
		{name: "malformed content type", contentType: "application/json; charset=\""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/v1/internal/shipping/quotes", strings.NewReader(`{}`))
			request.Header.Set("Authorization", "Bearer internal-token")
			if test.contentType != "" {
				request.Header.Set("Content-Type", test.contentType)
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			var body errorBody
			if response.Code != http.StatusUnsupportedMediaType || json.Unmarshal(response.Body.Bytes(), &body) != nil ||
				body.Code != "UNSUPPORTED_MEDIA_TYPE" || body.RequestID == "" {
				t.Fatalf("expected common 415 error, got status=%d body=%s", response.Code, response.Body.String())
			}
		})
	}

	request := httptest.NewRequest(http.MethodPost, "/api/v1/internal/shipping/quotes", strings.NewReader(`{}`))
	request.Header.Set("Authorization", "Bearer internal-token")
	request.Header.Set("Content-Type", "application/json; charset=utf-8")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code == http.StatusUnsupportedMediaType {
		t.Fatalf("valid JSON media type with charset parameter should be accepted, got %s", response.Body.String())
	}
}

func TestOversizedJSONBodyReturnsPayloadTooLarge(t *testing.T) {
	handler := NewRouter(application.NewService(newHTTPMemoryRepository(), carrier.NewRegistry(carrier.NewFake())), "internal-token", "fake-secret", "", zerolog.Nop())
	request := httptest.NewRequest(http.MethodPost, "/api/v1/internal/shipping/quotes", strings.NewReader(`{"padding":"`+strings.Repeat("x", 1<<20)+`"}`))
	request.Header.Set("Authorization", "Bearer internal-token")
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	var body errorBody
	if response.Code != http.StatusRequestEntityTooLarge || json.Unmarshal(response.Body.Bytes(), &body) != nil ||
		body.Code != "PAYLOAD_TOO_LARGE" || body.RequestID == "" {
		t.Fatalf("oversized JSON body should return 413: status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestEmptyCollectionResponsesAreArrays(t *testing.T) {
	repository := newHTTPMemoryRepository()
	service := application.NewService(repository, carrier.NewRegistry(carrier.NewFake()))
	handler := NewRouter(service, "internal-token", "fake-secret", "", zerolog.Nop(), fixedStaffAuthenticator{})
	tests := []struct {
		name string
		path string
		key  string
	}{
		{name: "admin shipment list", path: "/api/v1/admin/shipping/shipments", key: "items"},
		{name: "order tracking list", path: "/api/v1/orders/order-without-shipments/tracking", key: "shipments"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, test.path, nil)
			if test.name == "admin shipment list" {
				request.Header.Set("Authorization", "Bearer staff-jwt-test")
			} else {
				request.Header.Set("Authorization", "Bearer internal-token")
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			var body map[string]json.RawMessage
			if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &body) != nil {
				t.Fatalf("empty collection request failed: status=%d body=%s", response.Code, response.Body.String())
			}
			var collection []json.RawMessage
			if err := json.Unmarshal(body[test.key], &collection); err != nil || collection == nil {
				t.Fatalf("%s must be an empty JSON array, got %s (err=%v)", test.key, body[test.key], err)
			}
		})
	}
}

func TestGHNWebhookRejectsInvalidSecretBeforeProcessing(t *testing.T) {
	handler := NewRouter(nil, "internal-token", "fake-secret", "expected-secret", zerolog.Nop())
	request := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/logistics/ghn", nil)
	request.Header.Set("X-GHN-Webhook-Secret", "wrong-secret")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", response.Code)
	}
	if response.Header().Get("X-Request-Id") == "" {
		t.Fatal("expected request id response header")
	}
	var body errorBody
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if body.RequestID != response.Header().Get("X-Request-Id") {
		t.Fatalf("error requestId %q does not match response header %q", body.RequestID, response.Header().Get("X-Request-Id"))
	}
}

func TestInternalEndpointsRequireAuthenticationAndReturnCommonErrorEnvelope(t *testing.T) {
	handler := NewRouter(nil, "internal-token", "fake-secret", "", zerolog.Nop())
	request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/shipping/shipments", nil)
	request.Header.Set("Authorization", "Bearer internal-token")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", response.Code)
	}
	var body errorBody
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if body.Code != "UNAUTHORIZED" || body.RequestID == "" {
		t.Fatalf("unexpected common error envelope: %+v", body)
	}
}

func TestInternalShipmentHTTPContractLifecycle(t *testing.T) {
	repository := newHTTPMemoryRepository()
	service := application.NewService(repository, carrier.NewRegistry(carrier.NewFake()))
	handler := NewRouter(service, "internal-token", "fake-secret", "", zerolog.Nop(), fixedStaffAuthenticator{})
	requestBody := `{"idempotency_key":"rest-create-key-0001","order_id":"order-1","package_id":"package-1","carrier_code":"FAKE","service_code":"STANDARD","recipient":{"name":"Customer","phone":"0905000000","street":"1 Main Street","ward":"Ward 1","province":"Hue","country_code":"VN"},"parcel":{"weight_grams":1000,"length_cm":30,"width_cm":20,"height_cm":15}}`
	quoteBody := `{"carrier_code":"FAKE","destination":{"name":"Customer","phone":"0905000000","street":"1 Main Street","ward":"Ward 1","province":"Hue","country_code":"VN"},"parcel":{"weight_grams":1000,"length_cm":30,"width_cm":20,"height_cm":15}}`
	quoteRequest := httptest.NewRequest(http.MethodPost, "/api/v1/internal/shipping/quotes", strings.NewReader(quoteBody))
	quoteRequest.Header.Set("Authorization", "Bearer internal-token")
	quoteRequest.Header.Set("Content-Type", "application/json")
	quoteResponse := httptest.NewRecorder()
	handler.ServeHTTP(quoteResponse, quoteRequest)
	if quoteResponse.Code != http.StatusOK || !strings.Contains(quoteResponse.Body.String(), `"fee_vnd":`) {
		t.Fatalf("quote response = %d, body=%s", quoteResponse.Code, quoteResponse.Body.String())
	}

	create := httptest.NewRequest(http.MethodPost, "/api/v1/internal/shipments", strings.NewReader(requestBody))
	create.Header.Set("Authorization", "Bearer internal-token")
	create.Header.Set("Content-Type", "application/json")
	create.Header.Set("Idempotency-Key", "rest-create-key-0001")
	created := httptest.NewRecorder()
	handler.ServeHTTP(created, create)
	if created.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body=%s", created.Code, created.Body.String())
	}
	var shipment domain.Shipment
	if err := json.Unmarshal(created.Body.Bytes(), &shipment); err != nil {
		t.Fatalf("decode created shipment: %v", err)
	}
	if shipment.ShipmentNumber == "" || shipment.Status != domain.StatusCreated || created.Header().Get("X-Request-Id") == "" {
		t.Fatalf("unexpected create response: shipment=%+v headers=%v", shipment, created.Header())
	}
	replayRequest := httptest.NewRequest(http.MethodPost, "/api/v1/internal/shipments", strings.NewReader(requestBody))
	replayRequest.Header.Set("Authorization", "Bearer internal-token")
	replayRequest.Header.Set("Content-Type", "application/json")
	replayRequest.Header.Set("Idempotency-Key", "rest-create-key-0001")
	replay := httptest.NewRecorder()
	handler.ServeHTTP(replay, replayRequest)
	var replayedShipment domain.Shipment
	if replay.Code != http.StatusCreated || json.Unmarshal(replay.Body.Bytes(), &replayedShipment) != nil || replayedShipment.ID != shipment.ID {
		t.Fatalf("create replay did not return original shipment: status=%d body=%s", replay.Code, replay.Body.String())
	}
	changedPayload := strings.Replace(requestBody, `"name":"Customer"`, `"name":"Different Customer"`, 1)
	conflictingCreate := httptest.NewRequest(http.MethodPost, "/api/v1/internal/shipments", strings.NewReader(changedPayload))
	conflictingCreate.Header.Set("Authorization", "Bearer internal-token")
	conflictingCreate.Header.Set("Content-Type", "application/json")
	conflictingCreate.Header.Set("Idempotency-Key", "rest-create-key-0001")
	conflictResponse := httptest.NewRecorder()
	handler.ServeHTTP(conflictResponse, conflictingCreate)
	var idempotencyError errorBody
	if conflictResponse.Code != http.StatusConflict || json.Unmarshal(conflictResponse.Body.Bytes(), &idempotencyError) != nil || idempotencyError.Code != "IDEMPOTENCY_KEY_REUSED" {
		t.Fatalf("changed payload with reused key must conflict: status=%d body=%s", conflictResponse.Code, conflictResponse.Body.String())
	}

	get := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/admin/shipping/shipments/%s", shipment.ID), nil)
	get.Header.Set("Authorization", "Bearer staff-jwt-test")
	got := httptest.NewRecorder()
	handler.ServeHTTP(got, get)
	if got.Code != http.StatusOK {
		t.Fatalf("get shipment status = %d, body=%s", got.Code, got.Body.String())
	}
	var detail struct {
		Shipment       domain.Shipment        `json:"shipment"`
		TrackingEvents []domain.TrackingEvent `json:"tracking_events"`
	}
	if err := json.Unmarshal(got.Body.Bytes(), &detail); err != nil {
		t.Fatalf("decode shipment detail: %v", err)
	}
	if detail.Shipment.ID != shipment.ID || detail.TrackingEvents == nil {
		t.Fatalf("unexpected detail response: %+v", detail)
	}

	list := httptest.NewRequest(http.MethodGet, "/api/v1/admin/shipping/shipments?page=1&pageSize=20", nil)
	list.Header.Set("Authorization", "Bearer staff-jwt-test")
	listed := httptest.NewRecorder()
	handler.ServeHTTP(listed, list)
	if listed.Code != http.StatusOK || !strings.Contains(listed.Body.String(), `"items":[`) {
		t.Fatalf("shipment list response = %d, body=%s", listed.Code, listed.Body.String())
	}

	labelRequest := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/internal/shipments/%s/label-url", shipment.ID), strings.NewReader(`{"format":"A5"}`))
	labelRequest.Header.Set("Authorization", "Bearer internal-token")
	labelRequest.Header.Set("Content-Type", "application/json")
	labelResponse := httptest.NewRecorder()
	handler.ServeHTTP(labelResponse, labelRequest)
	if labelResponse.Code != http.StatusOK || !strings.Contains(labelResponse.Body.String(), `"label_url":"https://fake-carrier.local/`) {
		t.Fatalf("label URL response = %d, body=%s", labelResponse.Code, labelResponse.Body.String())
	}

	cancelRequest := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/internal/shipments/%s/cancel", shipment.ID), strings.NewReader(`{"reason_code":"OTHER","reason":"customer request"}`))
	cancelRequest.Header.Set("Authorization", "Bearer internal-token")
	cancelRequest.Header.Set("Content-Type", "application/json")
	cancelRequest.Header.Set("Idempotency-Key", "cancel-shipment-key-01")
	cancelled := httptest.NewRecorder()
	handler.ServeHTTP(cancelled, cancelRequest)
	if cancelled.Code != http.StatusOK || !strings.Contains(cancelled.Body.String(), `"status":"CANCELLED"`) {
		t.Fatalf("cancel response = %d, body=%s", cancelled.Code, cancelled.Body.String())
	}
	var cancelledShipment domain.Shipment
	if err := json.Unmarshal(cancelled.Body.Bytes(), &cancelledShipment); err != nil {
		t.Fatalf("decode cancellation response: %v", err)
	}
	retryCancel := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/internal/shipments/%s/cancel", shipment.ID), strings.NewReader(`{"reason_code":"OTHER","reason":"customer request"}`))
	retryCancel.Header.Set("Authorization", "Bearer internal-token")
	retryCancel.Header.Set("Content-Type", "application/json")
	retryCancel.Header.Set("Idempotency-Key", "cancel-shipment-key-01")
	retriedCancel := httptest.NewRecorder()
	handler.ServeHTTP(retriedCancel, retryCancel)
	var retriedShipment domain.Shipment
	if retriedCancel.Code != http.StatusOK || json.Unmarshal(retriedCancel.Body.Bytes(), &retriedShipment) != nil || retriedShipment.Revision != cancelledShipment.Revision {
		t.Fatalf("cancel retry was not idempotent: status=%d body=%s", retriedCancel.Code, retriedCancel.Body.String())
	}
	changedCancel := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/internal/shipments/%s/cancel", shipment.ID), strings.NewReader(`{"reason_code":"OTHER","reason":"different reason"}`))
	changedCancel.Header.Set("Authorization", "Bearer internal-token")
	changedCancel.Header.Set("Content-Type", "application/json")
	changedCancel.Header.Set("Idempotency-Key", "cancel-shipment-key-01")
	changedCancelResponse := httptest.NewRecorder()
	handler.ServeHTTP(changedCancelResponse, changedCancel)
	var changedCancelError errorBody
	if changedCancelResponse.Code != http.StatusConflict || json.Unmarshal(changedCancelResponse.Body.Bytes(), &changedCancelError) != nil || changedCancelError.Code != "IDEMPOTENCY_KEY_REUSED" {
		t.Fatalf("changed cancellation payload with reused key must conflict: status=%d body=%s", changedCancelResponse.Code, changedCancelResponse.Body.String())
	}
	newCancelKey := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/internal/shipments/%s/cancel", shipment.ID), strings.NewReader(`{"reason_code":"OTHER","reason":"customer request"}`))
	newCancelKey.Header.Set("Authorization", "Bearer internal-token")
	newCancelKey.Header.Set("Content-Type", "application/json")
	newCancelKey.Header.Set("Idempotency-Key", "cancel-shipment-key-02")
	newCancelKeyResponse := httptest.NewRecorder()
	handler.ServeHTTP(newCancelKeyResponse, newCancelKey)
	var newCancelKeyError errorBody
	if newCancelKeyResponse.Code != http.StatusConflict || json.Unmarshal(newCancelKeyResponse.Body.Bytes(), &newCancelKeyError) != nil || newCancelKeyError.Code != "IDEMPOTENCY_KEY_REUSED" {
		t.Fatalf("different cancellation key for an already-bound action must conflict: status=%d body=%s", newCancelKeyResponse.Code, newCancelKeyResponse.Body.String())
	}

	tracking := httptest.NewRequest(http.MethodGet, "/api/v1/orders/order-1/tracking", nil)
	tracking.Header.Set("Authorization", "Bearer internal-token")
	tracked := httptest.NewRecorder()
	handler.ServeHTTP(tracked, tracking)
	if tracked.Code != http.StatusOK || !strings.Contains(tracked.Body.String(), `"tracking_events":[`) {
		t.Fatalf("tracking response = %d, body=%s", tracked.Code, tracked.Body.String())
	}

	secondRequestBody := strings.Replace(requestBody, "rest-create-key-0001", "rest-create-key-0002", 1)
	secondCreate := httptest.NewRequest(http.MethodPost, "/api/v1/internal/shipments", strings.NewReader(secondRequestBody))
	secondCreate.Header.Set("Authorization", "Bearer internal-token")
	secondCreate.Header.Set("Content-Type", "application/json")
	secondCreate.Header.Set("Idempotency-Key", "rest-create-key-0002")
	secondCreated := httptest.NewRecorder()
	handler.ServeHTTP(secondCreated, secondCreate)
	if secondCreated.Code != http.StatusCreated {
		t.Fatalf("second create status = %d, body=%s", secondCreated.Code, secondCreated.Body.String())
	}
	var secondShipment domain.Shipment
	if err := json.Unmarshal(secondCreated.Body.Bytes(), &secondShipment); err != nil {
		t.Fatalf("decode second shipment: %v", err)
	}
	handover := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/internal/shipments/%s/handover", secondShipment.ID), nil)
	handover.Header.Set("Authorization", "Bearer internal-token")
	handover.Header.Set("Idempotency-Key", "handover-shipment-key-01")
	handedOver := httptest.NewRecorder()
	handler.ServeHTTP(handedOver, handover)
	if handedOver.Code != http.StatusOK || !strings.Contains(handedOver.Body.String(), `"status":"HANDED_OVER"`) {
		t.Fatalf("handover response = %d, body=%s", handedOver.Code, handedOver.Body.String())
	}
	var handedOverShipment domain.Shipment
	if err := json.Unmarshal(handedOver.Body.Bytes(), &handedOverShipment); err != nil {
		t.Fatalf("decode handover response: %v", err)
	}
	retryHandover := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/internal/shipments/%s/handover", secondShipment.ID), nil)
	retryHandover.Header.Set("Authorization", "Bearer internal-token")
	retryHandover.Header.Set("Idempotency-Key", "handover-shipment-key-01")
	retriedHandover := httptest.NewRecorder()
	handler.ServeHTTP(retriedHandover, retryHandover)
	var retriedHandoverShipment domain.Shipment
	if retriedHandover.Code != http.StatusOK || json.Unmarshal(retriedHandover.Body.Bytes(), &retriedHandoverShipment) != nil || retriedHandoverShipment.Revision != handedOverShipment.Revision {
		t.Fatalf("handover retry was not idempotent: status=%d body=%s", retriedHandover.Code, retriedHandover.Body.String())
	}
	newHandoverKey := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/internal/shipments/%s/handover", secondShipment.ID), nil)
	newHandoverKey.Header.Set("Authorization", "Bearer internal-token")
	newHandoverKey.Header.Set("Idempotency-Key", "handover-shipment-key-02")
	newHandoverKeyResponse := httptest.NewRecorder()
	handler.ServeHTTP(newHandoverKeyResponse, newHandoverKey)
	var newHandoverKeyError errorBody
	if newHandoverKeyResponse.Code != http.StatusConflict || json.Unmarshal(newHandoverKeyResponse.Body.Bytes(), &newHandoverKeyError) != nil || newHandoverKeyError.Code != "IDEMPOTENCY_KEY_REUSED" {
		t.Fatalf("different handover key for an already-bound action must conflict: status=%d body=%s", newHandoverKeyResponse.Code, newHandoverKeyResponse.Body.String())
	}
}

func TestRetryableShipmentOperationsRequireIdempotencyHeader(t *testing.T) {
	handler := NewRouter(nil, "internal-token", "fake-secret", "", zerolog.Nop())
	shipmentID := uuid.NewString()
	requests := []struct {
		name string
		path string
		body string
	}{
		{name: "create", path: "/api/v1/internal/shipments", body: `{}`},
		{name: "cancel", path: "/api/v1/internal/shipments/" + shipmentID + "/cancel", body: `{}`},
		{name: "handover", path: "/api/v1/internal/shipments/" + shipmentID + "/handover"},
	}
	for _, test := range requests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, test.path, strings.NewReader(test.body))
			request.Header.Set("Authorization", "Bearer internal-token")
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("missing idempotency key status = %d, body=%s", response.Code, response.Body.String())
			}
			var errorResponse errorBody
			if err := json.Unmarshal(response.Body.Bytes(), &errorResponse); err != nil {
				t.Fatalf("decode idempotency error: %v", err)
			}
			if errorResponse.Code != "IDEMPOTENCY_KEY_REQUIRED" || errorResponse.RequestID == "" {
				t.Fatalf("unexpected missing-key error: %+v", errorResponse)
			}
		})
	}
}

func TestIdempotencyKeyRejectsLeadingOrTrailingWhitespace(t *testing.T) {
	handler := NewRouter(nil, "internal-token", "fake-secret", "", zerolog.Nop())
	for _, key := range []string{" rest-create-key-0001", "rest-create-key-0001 "} {
		t.Run(fmt.Sprintf("key_%q", key), func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/v1/internal/shipments", strings.NewReader(`{}`))
			request.Header.Set("Authorization", "Bearer internal-token")
			request.Header.Set("Idempotency-Key", key)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			var body errorBody
			if response.Code != http.StatusBadRequest || json.Unmarshal(response.Body.Bytes(), &body) != nil || body.Code != "INVALID_IDEMPOTENCY_KEY" {
				t.Fatalf("whitespace-padded key must be rejected: status=%d body=%s", response.Code, response.Body.String())
			}
		})
	}
}

func TestListShipmentsRejectsOffsetOverflow(t *testing.T) {
	repository := newHTTPMemoryRepository()
	service := application.NewService(repository, carrier.NewRegistry(carrier.NewFake()))
	handler := NewRouter(service, "internal-token", "fake-secret", "", zerolog.Nop(), fixedStaffAuthenticator{})
	tests := []struct {
		name  string
		query string
	}{
		{name: "offset overflow", query: fmt.Sprintf("page=%d&pageSize=100", int(^uint(0)>>1))},
		{name: "non-numeric page", query: "page=abc"},
		{name: "empty page", query: "page="},
		{name: "repeated page", query: "page=1&page=2"},
		{name: "non-positive page", query: "page=0"},
		{name: "page size over maximum", query: "pageSize=101"},
		{name: "empty page size", query: "pageSize="},
		{name: "repeated page size", query: "pageSize=10&pageSize=20"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/shipping/shipments?"+test.query, nil)
			request.Header.Set("Authorization", "Bearer staff-jwt-test")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			var body errorBody
			if response.Code != http.StatusBadRequest || json.Unmarshal(response.Body.Bytes(), &body) != nil || body.Code != "INVALID_PAGINATION" {
				t.Fatalf("invalid pagination must be rejected: status=%d body=%s", response.Code, response.Body.String())
			}
		})
	}
}

func TestListShipmentsRejectsUnknownStatusFilter(t *testing.T) {
	repository := newHTTPMemoryRepository()
	service := application.NewService(repository, carrier.NewRegistry(carrier.NewFake()))
	handler := NewRouter(service, "internal-token", "fake-secret", "", zerolog.Nop(), fixedStaffAuthenticator{})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/shipping/shipments?status=NOT_A_STATUS", nil)
	request.Header.Set("Authorization", "Bearer staff-jwt-test")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	var body errorBody
	if response.Code != http.StatusBadRequest || json.Unmarshal(response.Body.Bytes(), &body) != nil || body.Code != "INVALID_STATUS_FILTER" {
		t.Fatalf("unknown shipment status must be rejected: status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestListShipmentsRejectsRepeatedFilters(t *testing.T) {
	tests := []struct {
		name  string
		query string
		code  string
	}{
		{name: "status", query: "status=CREATED&status=DELIVERED", code: "INVALID_STATUS_FILTER"},
		{name: "carrier", query: "carrier=FAKE&carrier=GHN", code: "INVALID_FILTER"},
		{name: "order id", query: "orderId=order-1&orderId=order-2", code: "INVALID_FILTER"},
	}
	service := application.NewService(newHTTPMemoryRepository(), carrier.NewRegistry(carrier.NewFake()))
	handler := NewRouter(service, "internal-token", "fake-secret", "", zerolog.Nop(), fixedStaffAuthenticator{})
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/shipping/shipments?"+test.query, nil)
			request.Header.Set("Authorization", "Bearer staff-jwt-test")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			var body errorBody
			if response.Code != http.StatusBadRequest || json.Unmarshal(response.Body.Bytes(), &body) != nil || body.Code != test.code {
				t.Fatalf("repeated filter should be rejected as %s: status=%d body=%s", test.code, response.Code, response.Body.String())
			}
		})
	}
}

func TestReadyEndpointReturnsCommonErrorWhenDatabaseIsUnavailable(t *testing.T) {
	repository := newHTTPMemoryRepository()
	repository.pingErr = errors.New("database connection refused")
	service := application.NewService(repository, carrier.NewRegistry(carrier.NewFake()))
	handler := NewRouter(service, "internal-token", "fake-secret", "", zerolog.Nop())
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health/ready", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("readiness status = %d, body=%s", response.Code, response.Body.String())
	}
	var body errorBody
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode readiness error: %v", err)
	}
	if body.Code != "NOT_READY" || body.RequestID == "" {
		t.Fatalf("unexpected readiness error: %+v", body)
	}
}

func TestRecoveredHandlerPanicUsesCommonErrorEnvelope(t *testing.T) {
	handler := NewRouter(nil, "internal-token", "fake-secret", "", zerolog.Nop())
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health/ready", nil))
	var body errorBody
	if response.Code != http.StatusInternalServerError || json.Unmarshal(response.Body.Bytes(), &body) != nil {
		t.Fatalf("panic response must use JSON error envelope: status=%d body=%s", response.Code, response.Body.String())
	}
	if body.Code != "INTERNAL_ERROR" || body.RequestID == "" || body.RequestID != response.Header().Get("X-Request-Id") {
		t.Fatalf("unexpected recovered panic error: body=%+v headers=%v", body, response.Header())
	}
	if strings.Contains(response.Body.String(), "nil pointer") {
		t.Fatal("panic details must not be returned to the caller")
	}
}

func TestFakeWebhookValidatesSignatureAndAppliesTrackingEvent(t *testing.T) {
	repository := newHTTPMemoryRepository()
	service := application.NewService(repository, carrier.NewRegistry(carrier.NewFake()))
	shipment, err := service.CreateShipment(context.Background(), domain.CreateShipmentCommand{
		IdempotencyKey: "fake-webhook-test-key", OrderID: "order-webhook", PackageID: "package-webhook",
		CarrierCode: carrier.FakeCode, ServiceCode: "STANDARD",
		Recipient: domain.Recipient{Name: "Customer", Phone: "0905000000", Street: "1 Main Street", Ward: "Ward 1", Province: "Hue"},
		Parcel:    domain.Parcel{WeightGrams: 1000, LengthCM: 30, WidthCM: 20, HeightCM: 15},
	})
	if err != nil {
		t.Fatalf("create shipment for webhook: %v", err)
	}
	handler := NewRouter(service, "internal-token", "fake-secret", "", zerolog.Nop())
	body := fmt.Sprintf(`{"event_id":"event-1","tracking_code":%q,"status":"IN_TRANSIT","event_at":"2026-11-12T10:00:00Z"}`, shipment.TrackingCode)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/logistics/fake", strings.NewReader(body))
	request.Header.Set("X-Shipping-Signature", "sha256=00")
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("invalid signature status = %d, body=%s", response.Code, response.Body.String())
	}

	mac := hmac.New(sha256.New, []byte("fake-secret"))
	_, _ = mac.Write([]byte(body))
	request = httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/logistics/fake", strings.NewReader(body))
	request.Header.Set("X-Shipping-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"result":"APPLIED"`) || !strings.Contains(response.Body.String(), `"status":"IN_TRANSIT"`) {
		t.Fatalf("signed fake webhook response = %d, body=%s", response.Code, response.Body.String())
	}
}

func TestGHNWebhookAppliesCallbackAndReturnsRetryableStatusBeforeShipmentExists(t *testing.T) {
	ghn, err := ghncarrier.New(ghncarrier.Config{
		BaseURL: "https://dev-online-gateway.ghn.vn", Token: "test-token", ShopID: 123,
		FromDistrictID: 1450, FromWardCode: "11111",
	})
	if err != nil {
		t.Fatalf("create GHN provider: %v", err)
	}
	body := `{"ShopID":123,"Time":"2026-11-12T10:00:00Z","OrderCode":"GHN-TRACK-1","Type":"StatusUpdate","Status":"picked","Description":"Picked up"}`

	t.Run("applies callback for known shipment", func(t *testing.T) {
		repository := newHTTPMemoryRepository()
		shipmentID := uuid.New()
		repository.shipments[shipmentID] = &domain.Shipment{
			ID: shipmentID, OrderID: "order-ghn", PackageID: "package-ghn",
			CarrierCode: ghncarrier.Code, TrackingCode: "GHN-TRACK-1",
			Status: domain.StatusCreated, Revision: 1,
		}
		service := application.NewService(repository, carrier.NewRegistry(carrier.NewFake(), ghn))
		handler := NewRouter(service, "internal-token", "fake-secret", "ghn-webhook-secret", zerolog.Nop())
		request := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/logistics/ghn", strings.NewReader(body))
		request.Header.Set("X-GHN-Webhook-Secret", "ghn-webhook-secret")
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"status":"HANDED_OVER"`) {
			t.Fatalf("valid GHN callback response = %d, body=%s", response.Code, response.Body.String())
		}
	})

	t.Run("asks provider to retry before shipment is committed", func(t *testing.T) {
		repository := newHTTPMemoryRepository()
		service := application.NewService(repository, carrier.NewRegistry(carrier.NewFake(), ghn))
		handler := NewRouter(service, "internal-token", "fake-secret", "ghn-webhook-secret", zerolog.Nop())
		request := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/logistics/ghn", strings.NewReader(body))
		request.Header.Set("X-GHN-Webhook-Secret", "ghn-webhook-secret")
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusServiceUnavailable {
			t.Fatalf("race callback status = %d, body=%s", response.Code, response.Body.String())
		}
		var errorResponse errorBody
		if err := json.Unmarshal(response.Body.Bytes(), &errorResponse); err != nil {
			t.Fatalf("decode retryable GHN error: %v", err)
		}
		if errorResponse.Code != "SHIPMENT_NOT_READY" || errorResponse.RequestID == "" {
			t.Fatalf("unexpected retryable GHN error: %+v", errorResponse)
		}
	})

	t.Run("rejects malformed provider payload with a safe error", func(t *testing.T) {
		repository := newHTTPMemoryRepository()
		service := application.NewService(repository, carrier.NewRegistry(carrier.NewFake(), ghn))
		handler := NewRouter(service, "internal-token", "fake-secret", "ghn-webhook-secret", zerolog.Nop())
		request := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/logistics/ghn", strings.NewReader(`{}`))
		request.Header.Set("X-GHN-Webhook-Secret", "ghn-webhook-secret")
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("malformed GHN callback status = %d, body=%s", response.Code, response.Body.String())
		}
		var errorResponse errorBody
		if err := json.Unmarshal(response.Body.Bytes(), &errorResponse); err != nil {
			t.Fatalf("decode malformed GHN error: %v", err)
		}
		if errorResponse.Code != "INVALID_GHN_WEBHOOK" || errorResponse.Message != "GHN webhook payload is invalid" || errorResponse.RequestID == "" {
			t.Fatalf("unexpected malformed GHN error: %+v", errorResponse)
		}
	})
}

func TestWebhooksRejectUnsupportedContentTypeAfterAuthentication(t *testing.T) {
	body := `{"event_id":"event-1","tracking_code":"track-1","status":"IN_TRANSIT","event_at":"2026-11-12T10:00:00Z"}`
	mac := hmac.New(sha256.New, []byte("fake-secret"))
	_, _ = mac.Write([]byte(body))
	tests := []struct {
		name      string
		path      string
		secretKey string
		secret    string
		signature string
	}{
		{name: "fake carrier", path: "/api/v1/webhooks/logistics/fake", secretKey: "X-Shipping-Signature", signature: "sha256=" + hex.EncodeToString(mac.Sum(nil))},
		{name: "GHN", path: "/api/v1/webhooks/logistics/ghn", secretKey: "X-GHN-Webhook-Secret", secret: "ghn-secret"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := NewRouter(nil, "internal-token", "fake-secret", "ghn-secret", zerolog.Nop())
			request := httptest.NewRequest(http.MethodPost, test.path, strings.NewReader(body))
			request.Header.Set(test.secretKey, test.signature)
			if test.secret != "" {
				request.Header.Set(test.secretKey, test.secret)
			}
			request.Header.Set("Content-Type", "text/plain")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			var errorResponse errorBody
			if response.Code != http.StatusUnsupportedMediaType || json.Unmarshal(response.Body.Bytes(), &errorResponse) != nil ||
				errorResponse.Code != "UNSUPPORTED_MEDIA_TYPE" || errorResponse.RequestID == "" {
				t.Fatalf("authenticated webhook should reject unsupported media type: status=%d body=%s", response.Code, response.Body.String())
			}
		})
	}
}

func TestOversizedAuthenticatedWebhookBodyReturnsPayloadTooLarge(t *testing.T) {
	handler := NewRouter(nil, "internal-token", "fake-secret", "ghn-secret", zerolog.Nop())
	request := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/logistics/ghn", strings.NewReader(strings.Repeat("x", (1<<20)+1)))
	request.Header.Set("X-GHN-Webhook-Secret", "ghn-secret")
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	var body errorBody
	if response.Code != http.StatusRequestEntityTooLarge || json.Unmarshal(response.Body.Bytes(), &body) != nil ||
		body.Code != "PAYLOAD_TOO_LARGE" || body.RequestID == "" {
		t.Fatalf("oversized webhook body should return 413: status=%d body=%s", response.Code, response.Body.String())
	}
}
