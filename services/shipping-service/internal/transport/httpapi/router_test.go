package httpapi

import (
	"context"
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
	"github.com/omamx/shipping-service/internal/domain"
	"github.com/rs/zerolog"
)

type httpMemoryRepository struct {
	shipments map[uuid.UUID]*domain.Shipment
	keys      map[string]uuid.UUID
	timeline  map[uuid.UUID][]domain.TrackingEvent
	pingErr   error
}

func newHTTPMemoryRepository() *httpMemoryRepository {
	return &httpMemoryRepository{
		shipments: make(map[uuid.UUID]*domain.Shipment),
		keys:      make(map[string]uuid.UUID),
		timeline:  make(map[uuid.UUID][]domain.TrackingEvent),
	}
}

func (r *httpMemoryRepository) Ping(context.Context) error { return r.pingErr }

func (r *httpMemoryRepository) BeginCreation(_ context.Context, key string, shipment *domain.Shipment) (*domain.Shipment, bool, error) {
	if id, ok := r.keys[key]; ok {
		return r.clone(r.shipments[id]), false, nil
	}
	r.keys[key] = shipment.ID
	r.shipments[shipment.ID] = r.clone(shipment)
	return r.clone(shipment), true, nil
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
	items := make([]domain.Shipment, 0)
	for _, shipment := range r.shipments {
		if shipment.OrderID == orderID {
			items = append(items, *r.clone(shipment))
		}
	}
	return items, nil
}

func (r *httpMemoryRepository) List(_ context.Context, filter application.ListFilter) ([]domain.Shipment, error) {
	items := make([]domain.Shipment, 0)
	for _, shipment := range r.shipments {
		if filter.Status != "" && shipment.Status != filter.Status ||
			filter.CarrierCode != "" && shipment.CarrierCode != filter.CarrierCode ||
			filter.OrderID != "" && shipment.OrderID != filter.OrderID {
			continue
		}
		items = append(items, *r.clone(shipment))
	}
	if filter.Offset >= len(items) {
		return []domain.Shipment{}, nil
	}
	items = items[filter.Offset:]
	if filter.Limit < len(items) {
		items = items[:filter.Limit]
	}
	return items, nil
}

func (r *httpMemoryRepository) TrackingTimeline(_ context.Context, id uuid.UUID) ([]domain.TrackingEvent, error) {
	items := append([]domain.TrackingEvent{}, r.timeline[id]...)
	return items, nil
}

func (r *httpMemoryRepository) clone(shipment *domain.Shipment) *domain.Shipment {
	if shipment == nil {
		return nil
	}
	copy := *shipment
	return &copy
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
	handler := NewRouter(service, "internal-token", "fake-secret", "", zerolog.Nop())
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

	get := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/admin/shipping/shipments/%s", shipment.ID), nil)
	get.Header.Set("X-Internal-Token", "internal-token")
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
	list.Header.Set("Authorization", "Bearer internal-token")
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
	cancelled := httptest.NewRecorder()
	handler.ServeHTTP(cancelled, cancelRequest)
	if cancelled.Code != http.StatusOK || !strings.Contains(cancelled.Body.String(), `"status":"CANCELLED"`) {
		t.Fatalf("cancel response = %d, body=%s", cancelled.Code, cancelled.Body.String())
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
	handedOver := httptest.NewRecorder()
	handler.ServeHTTP(handedOver, handover)
	if handedOver.Code != http.StatusOK || !strings.Contains(handedOver.Body.String(), `"status":"HANDED_OVER"`) {
		t.Fatalf("handover response = %d, body=%s", handedOver.Code, handedOver.Body.String())
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
