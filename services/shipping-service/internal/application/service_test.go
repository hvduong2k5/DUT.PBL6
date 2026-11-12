package application

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/omamx/shipping-service/internal/carrier"
	"github.com/omamx/shipping-service/internal/domain"
)

type memoryRepository struct {
	mu                sync.Mutex
	byID              map[uuid.UUID]*domain.Shipment
	byKey             map[string]uuid.UUID
	hashes            map[string]string
	timeline          map[uuid.UUID][]domain.TrackingEvent
	eventIDs          map[string]struct{}
	actionsByKey      map[string]actionBinding
	actionsByShipment map[string]actionBinding
}

type actionBinding struct {
	shipmentID uuid.UUID
	action     string
	key        string
	hash       string
}

type snapshotStub struct {
	snapshot OrderShippingSnapshot
	err      error
}

func (s snapshotStub) GetShippingSnapshot(context.Context, string, string) (OrderShippingSnapshot, error) {
	return s.snapshot, s.err
}

func newMemoryRepository() *memoryRepository {
	return &memoryRepository{
		byID: map[uuid.UUID]*domain.Shipment{}, byKey: map[string]uuid.UUID{}, hashes: map[string]string{},
		timeline: map[uuid.UUID][]domain.TrackingEvent{}, eventIDs: map[string]struct{}{},
		actionsByKey: map[string]actionBinding{}, actionsByShipment: map[string]actionBinding{},
	}
}

func cloneShipment(value *domain.Shipment) *domain.Shipment {
	data, _ := json.Marshal(value)
	var result domain.Shipment
	_ = json.Unmarshal(data, &result)
	return &result
}

func (r *memoryRepository) Ping(context.Context) error { return nil }

func (r *memoryRepository) BeginCreation(_ context.Context, key, requestHash string, shipment *domain.Shipment) (*domain.Shipment, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if id, ok := r.byKey[key]; ok {
		if r.hashes[key] != requestHash {
			return nil, false, domain.ErrIdempotencyConflict
		}
		return cloneShipment(r.byID[id]), false, nil
	}
	r.byKey[key] = shipment.ID
	r.hashes[key] = requestHash
	r.byID[shipment.ID] = cloneShipment(shipment)
	return cloneShipment(shipment), true, nil
}

func (r *memoryRepository) BindAction(_ context.Context, shipmentID uuid.UUID, action, key, requestHash string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	binding := actionBinding{shipmentID: shipmentID, action: action, key: key, hash: requestHash}
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

func (r *memoryRepository) Save(_ context.Context, shipment *domain.Shipment, expectedRevision int64, tracking *domain.TrackingEvent, _ *domain.DomainEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	stored := r.byID[shipment.ID]
	if stored == nil {
		return domain.ErrShipmentNotFound
	}
	if stored.Revision != expectedRevision {
		return domain.ErrConcurrentUpdate
	}
	if tracking != nil {
		key := tracking.CarrierCode + ":" + tracking.ExternalEventID
		if _, exists := r.eventIDs[key]; exists {
			return domain.ErrDuplicateEvent
		}
		r.eventIDs[key] = struct{}{}
		r.timeline[shipment.ID] = append(r.timeline[shipment.ID], *tracking)
	}
	r.byID[shipment.ID] = cloneShipment(shipment)
	return nil
}

func (r *memoryRepository) GetByID(_ context.Context, id uuid.UUID) (*domain.Shipment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	value := r.byID[id]
	if value == nil {
		return nil, domain.ErrShipmentNotFound
	}
	return cloneShipment(value), nil
}

func (r *memoryRepository) GetByTracking(_ context.Context, carrierCode, trackingCode string) (*domain.Shipment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, value := range r.byID {
		if value.CarrierCode == carrierCode && value.TrackingCode == trackingCode {
			return cloneShipment(value), nil
		}
	}
	return nil, domain.ErrTrackingNotFound
}

func (r *memoryRepository) ListByOrder(_ context.Context, orderID string) ([]domain.Shipment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var result []domain.Shipment
	for _, value := range r.byID {
		if value.OrderID == orderID {
			result = append(result, *cloneShipment(value))
		}
	}
	return result, nil
}

func (r *memoryRepository) List(_ context.Context, _ ListFilter) ([]domain.Shipment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var result []domain.Shipment
	for _, value := range r.byID {
		result = append(result, *cloneShipment(value))
	}
	return result, nil
}

func TestReadCollectionsAreNonNilWhenEmpty(t *testing.T) {
	repository := newMemoryRepository()
	service := NewService(repository, carrier.NewRegistry(carrier.NewFake()))
	ctx := context.Background()

	listed, err := service.ListShipments(ctx, ListFilter{Limit: 20})
	if err != nil || listed == nil {
		t.Fatalf("empty shipment list must be a non-nil slice: result=%v err=%v", listed, err)
	}
	ordered, err := service.TrackOrder(ctx, "order-without-shipments")
	if err != nil || ordered == nil {
		t.Fatalf("empty order shipment list must be a non-nil slice: result=%v err=%v", ordered, err)
	}

	shipment, err := service.CreateShipment(ctx, applicationCommand())
	if err != nil {
		t.Fatal(err)
	}
	_, events, err := service.GetShipment(ctx, shipment.ID)
	if err != nil || events == nil {
		t.Fatalf("empty shipment timeline must be a non-nil slice: result=%v err=%v", events, err)
	}
	events, err = service.GetShipmentTimeline(ctx, shipment.ID)
	if err != nil || events == nil {
		t.Fatalf("empty tracking timeline must be a non-nil slice: result=%v err=%v", events, err)
	}
}

func (r *memoryRepository) TrackingTimeline(_ context.Context, id uuid.UUID) ([]domain.TrackingEvent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]domain.TrackingEvent(nil), r.timeline[id]...), nil
}

func applicationCommand() domain.CreateShipmentCommand {
	return domain.CreateShipmentCommand{
		IdempotencyKey: "packing:pkg-1", OrderID: "ord-1", PackageID: "pkg-1", CarrierCode: carrier.FakeCode, ServiceCode: "STANDARD",
		Recipient: domain.Recipient{Name: "Nguyen Van A", Phone: "0905000000", Street: "1 Nguyen Hue", Ward: "Thuan Hoa", District: "Phu Xuan", Province: "Hue", CountryCode: "VN"},
		Parcel:    domain.Parcel{WeightGrams: 1000, LengthCM: 30, WidthCM: 20, HeightCM: 15},
	}
}

func TestCreateShipmentIsIdempotent(t *testing.T) {
	repository := newMemoryRepository()
	service := NewService(repository, carrier.NewRegistry(carrier.NewFake()))
	first, err := service.CreateShipment(context.Background(), applicationCommand())
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.CreateShipment(context.Background(), applicationCommand())
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID || first.TrackingCode != second.TrackingCode {
		t.Fatal("idempotent replay returned a different shipment")
	}
	if len(repository.byID) != 1 {
		t.Fatalf("expected one shipment, got %d", len(repository.byID))
	}
}

func TestCreateShipmentRejectsReusedKeyWithDifferentPayload(t *testing.T) {
	repository := newMemoryRepository()
	service := NewService(repository, carrier.NewRegistry(carrier.NewFake()))
	command := applicationCommand()
	if _, err := service.CreateShipment(context.Background(), command); err != nil {
		t.Fatal(err)
	}
	command.Recipient.Name = "Different Recipient"
	if _, err := service.CreateShipment(context.Background(), command); !errors.Is(err, domain.ErrIdempotencyConflict) {
		t.Fatalf("expected idempotency conflict, got %v", err)
	}
	if len(repository.byID) != 1 {
		t.Fatalf("expected original shipment to remain the only shipment, got %d", len(repository.byID))
	}
}

func TestCancelShipmentTransitionsAndIsIdempotent(t *testing.T) {
	repository := newMemoryRepository()
	service := NewService(repository, carrier.NewRegistry(carrier.NewFake()))
	shipment, err := service.CreateShipment(context.Background(), applicationCommand())
	if err != nil {
		t.Fatal(err)
	}
	cancelled, err := service.CancelShipment(context.Background(), shipment.ID, "cancel-shipment-key-01", carrier.CancelReasonOther, "Không còn nhu cầu giao")
	if err != nil {
		t.Fatal(err)
	}
	if cancelled.Status != domain.StatusCancelled {
		t.Fatalf("expected cancelled shipment, got %s", cancelled.Status)
	}
	replayed, err := service.CancelShipment(context.Background(), shipment.ID, "cancel-shipment-key-01", carrier.CancelReasonOther, "Không còn nhu cầu giao")
	if err != nil {
		t.Fatal(err)
	}
	if replayed.Status != domain.StatusCancelled || replayed.Revision != cancelled.Revision {
		t.Fatalf("idempotent cancellation changed shipment: %+v", replayed)
	}
	if _, err := service.CancelShipment(context.Background(), shipment.ID, "cancel-shipment-key-01", carrier.CancelReasonOther, "different payload"); !errors.Is(err, domain.ErrIdempotencyConflict) {
		t.Fatalf("reusing cancellation key with changed reason must conflict, got %v", err)
	}
	timeline, err := repository.TrackingTimeline(context.Background(), shipment.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(timeline) != 1 || timeline[0].MappedStatus != domain.StatusCancelled {
		t.Fatalf("unexpected cancellation timeline: %+v", timeline)
	}
}

func TestCancelShipmentRejectsAfterHandover(t *testing.T) {
	repository := newMemoryRepository()
	service := NewService(repository, carrier.NewRegistry(carrier.NewFake()))
	shipment, err := service.CreateShipment(context.Background(), applicationCommand())
	if err != nil {
		t.Fatal(err)
	}
	shipment, err = service.ConfirmHandover(context.Background(), shipment.ID, "handover-shipment-key-01")
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.CancelShipment(context.Background(), shipment.ID, "cancel-after-handover-key-01", carrier.CancelReasonOther, "too late")
	if !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("expected invalid transition, got %v", err)
	}
}

func TestCancelShipmentRejectsUnknownReasonCode(t *testing.T) {
	repository := newMemoryRepository()
	service := NewService(repository, carrier.NewRegistry(carrier.NewFake()))
	shipment, err := service.CreateShipment(context.Background(), applicationCommand())
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.CancelShipment(context.Background(), shipment.ID, "invalid-cancel-key-01", "UNKNOWN", "invalid")
	if !errors.Is(err, domain.ErrInvalidShipment) {
		t.Fatalf("expected invalid shipment, got %v", err)
	}
}

func TestGenerateLabelURLDoesNotMutateShipment(t *testing.T) {
	repository := newMemoryRepository()
	service := NewService(repository, carrier.NewRegistry(carrier.NewFake()))
	shipment, err := service.CreateShipment(context.Background(), applicationCommand())
	if err != nil {
		t.Fatal(err)
	}
	access, err := service.GenerateLabelURL(context.Background(), shipment.ID, "80x80")
	if err != nil {
		t.Fatal(err)
	}
	if access.ShipmentID != shipment.ID || access.LabelReference != shipment.LabelReference ||
		access.Format != carrier.LabelFormat80X80 || !strings.Contains(access.URL, "format=80X80") {
		t.Fatalf("unexpected label access: %+v", access)
	}
	stored, err := repository.GetByID(context.Background(), shipment.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Revision != shipment.Revision || stored.LabelReference != shipment.LabelReference {
		t.Fatalf("label generation mutated shipment: %+v", stored)
	}
}

func TestGenerateLabelURLRejectsCancelledShipment(t *testing.T) {
	repository := newMemoryRepository()
	service := NewService(repository, carrier.NewRegistry(carrier.NewFake()))
	shipment, err := service.CreateShipment(context.Background(), applicationCommand())
	if err != nil {
		t.Fatal(err)
	}
	shipment, err = service.CancelShipment(context.Background(), shipment.ID, "label-cancel-key-0001", carrier.CancelReasonOther, "cancel")
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.GenerateLabelURL(context.Background(), shipment.ID, carrier.LabelFormatA5)
	if !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("expected invalid transition, got %v", err)
	}
}

func TestWebhookIsDeduplicated(t *testing.T) {
	repository := newMemoryRepository()
	service := NewService(repository, carrier.NewRegistry(carrier.NewFake()))
	shipment, err := service.CreateShipment(context.Background(), applicationCommand())
	if err != nil {
		t.Fatal(err)
	}
	shipment, err = service.ConfirmHandover(context.Background(), shipment.ID, "webhook-handover-key-01")
	if err != nil {
		t.Fatal(err)
	}
	body := []byte(`{"event_id":"event-1","tracking_code":"` + shipment.TrackingCode + `","status":"IN_TRANSIT","event_at":"2026-10-17T10:00:00Z"}`)
	updated, result, err := service.ApplyWebhook(context.Background(), carrier.FakeCode, body)
	if err != nil {
		t.Fatal(err)
	}
	if result != "APPLIED" || updated.Status != domain.StatusInTransit {
		t.Fatalf("unexpected first result: %s %s", result, updated.Status)
	}
	_, result, err = service.ApplyWebhook(context.Background(), carrier.FakeCode, body)
	if err != nil {
		t.Fatal(err)
	}
	if result != "DUPLICATE" {
		t.Fatalf("expected duplicate, got %s", result)
	}
	timeline, err := repository.TrackingTimeline(context.Background(), shipment.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(timeline) != 1 {
		t.Fatalf("expected one tracking event, got %d", len(timeline))
	}
}

func TestOlderWebhookDoesNotRegressShipment(t *testing.T) {
	repository := newMemoryRepository()
	service := NewService(repository, carrier.NewRegistry(carrier.NewFake()))
	shipment, err := service.CreateShipment(context.Background(), applicationCommand())
	if err != nil {
		t.Fatal(err)
	}
	shipment, err = service.ConfirmHandover(context.Background(), shipment.ID, "webhook-handover-key-02")
	if err != nil {
		t.Fatal(err)
	}
	newer := []byte(`{"event_id":"newer","tracking_code":"` + shipment.TrackingCode + `","status":"IN_TRANSIT","event_at":"2026-10-17T10:00:00Z"}`)
	if _, _, err := service.ApplyWebhook(context.Background(), carrier.FakeCode, newer); err != nil {
		t.Fatal(err)
	}
	older := []byte(`{"event_id":"older","tracking_code":"` + shipment.TrackingCode + `","status":"PICKED_UP","event_at":"2026-10-17T09:00:00Z"}`)
	updated, result, err := service.ApplyWebhook(context.Background(), carrier.FakeCode, older)
	if err != nil {
		t.Fatal(err)
	}
	if result != "STALE" || updated.Status != domain.StatusInTransit {
		t.Fatalf("stale event regressed shipment: %s %s", result, updated.Status)
	}
}

func TestCreateFromPackingUsesImmutableOrderSnapshotAndPackageIdempotency(t *testing.T) {
	repository := newMemoryRepository()
	service := NewService(repository, carrier.NewRegistry(carrier.NewFake()))
	event := PackingCompleted{
		EventID: "event-1", OrderID: "ord-1", PackageID: "pkg-1", SealCode: "seal-1",
		Parcel: domain.Parcel{WeightGrams: 1200, LengthCM: 30, WidthCM: 20, HeightCM: 15, Fragile: true},
	}
	snapshots := snapshotStub{snapshot: OrderShippingSnapshot{
		OrderID: "ord-1", PackageID: "pkg-1", EligibleForShipping: true,
		Recipient:      domain.Recipient{Name: "Nguyen Van A", Phone: "0905000000", Street: "1 Nguyen Hue", Ward: "Thuan Hoa", District: "Phu Xuan", Province: "Hue", CountryCode: "VN"},
		CheckoutFeeVND: 30_000, CODAmountVND: 250_000,
	}}
	first, err := service.CreateFromPacking(context.Background(), event, snapshots, carrier.FakeCode, "STANDARD")
	if err != nil {
		t.Fatal(err)
	}
	event.EventID = "event-2"
	second, err := service.CreateFromPacking(context.Background(), event, snapshots, carrier.FakeCode, "STANDARD")
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID || len(repository.byID) != 1 {
		t.Fatal("re-emitted packing event created a duplicate shipment")
	}
	if first.CODAmountVND != 250_000 || first.CheckoutFeeVND != 30_000 || first.Parcel != event.Parcel {
		t.Fatal("shipment did not preserve order snapshot and fulfillment parcel data")
	}
}

func TestCreateFromPackingRejectsSnapshotIdentityMismatch(t *testing.T) {
	service := NewService(newMemoryRepository(), carrier.NewRegistry(carrier.NewFake()))
	event := PackingCompleted{
		EventID: "event-1", OrderID: "ord-1", PackageID: "pkg-1", SealCode: "seal-1",
		Parcel: domain.Parcel{WeightGrams: 1200, LengthCM: 30, WidthCM: 20, HeightCM: 15},
	}
	_, err := service.CreateFromPacking(context.Background(), event, snapshotStub{snapshot: OrderShippingSnapshot{
		OrderID: "ord-1", PackageID: "another-package", EligibleForShipping: true,
	}}, carrier.FakeCode, "STANDARD")
	if err == nil {
		t.Fatal("expected mismatched snapshot to be rejected")
	}
}
