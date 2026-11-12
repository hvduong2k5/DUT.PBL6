package application

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/omamx/shipping-service/internal/carrier"
	"github.com/omamx/shipping-service/internal/domain"
)

type memoryReturnRepository struct {
	*memoryRepository
	muReturn sync.Mutex
	byID     map[uuid.UUID]*domain.ReturnShipment
	byKey    map[string]uuid.UUID
	hashes   map[string]string
	events   []domain.ReturnShipmentEvent
}

func newMemoryReturnRepository() *memoryReturnRepository {
	return &memoryReturnRepository{
		memoryRepository: newMemoryRepository(),
		byID:             map[uuid.UUID]*domain.ReturnShipment{},
		byKey:            map[string]uuid.UUID{},
		hashes:           map[string]string{},
	}
}

func cloneReturnShipment(value *domain.ReturnShipment) *domain.ReturnShipment {
	data, _ := json.Marshal(value)
	var result domain.ReturnShipment
	_ = json.Unmarshal(data, &result)
	return &result
}

func (r *memoryReturnRepository) BeginReturnCreation(_ context.Context, key, requestHash string, shipment *domain.ReturnShipment) (*domain.ReturnShipment, bool, error) {
	r.muReturn.Lock()
	defer r.muReturn.Unlock()
	for _, existing := range r.byID {
		if existing.ReturnCaseID == shipment.ReturnCaseID && existing.PackageID == shipment.PackageID && existing.ID.String() != "" {
			if id, exists := r.byKey[key]; exists && existing.ID == id && r.hashes[key] == requestHash {
				return cloneReturnShipment(existing), false, nil
			}
			return nil, false, domain.ErrIdempotencyConflict
		}
	}
	if id, ok := r.byKey[key]; ok {
		if r.hashes[key] != requestHash {
			return nil, false, domain.ErrIdempotencyConflict
		}
		return cloneReturnShipment(r.byID[id]), false, nil
	}
	r.byKey[key] = shipment.ID
	r.hashes[key] = requestHash
	r.byID[shipment.ID] = cloneReturnShipment(shipment)
	return cloneReturnShipment(shipment), true, nil
}

func (r *memoryReturnRepository) SaveReturnShipment(_ context.Context, shipment *domain.ReturnShipment, expectedRevision int64, event *domain.ReturnShipmentEvent) error {
	r.muReturn.Lock()
	defer r.muReturn.Unlock()
	stored := r.byID[shipment.ID]
	if stored == nil {
		return domain.ErrReturnShipmentNotFound
	}
	if stored.Revision != expectedRevision {
		return domain.ErrConcurrentUpdate
	}
	r.byID[shipment.ID] = cloneReturnShipment(shipment)
	if event != nil {
		r.events = append(r.events, *event)
	}
	return nil
}

func (r *memoryReturnRepository) GetReturnShipment(_ context.Context, id uuid.UUID) (*domain.ReturnShipment, error) {
	r.muReturn.Lock()
	defer r.muReturn.Unlock()
	shipment := r.byID[id]
	if shipment == nil {
		return nil, domain.ErrReturnShipmentNotFound
	}
	return cloneReturnShipment(shipment), nil
}

func TestCreateReturnShipmentUsesSeparateIdempotentAggregateAndEvent(t *testing.T) {
	repository := newMemoryReturnRepository()
	service := NewService(repository, carrier.NewRegistry(carrier.NewFake()))
	command := validReturnCommand()

	created, err := service.CreateReturnShipment(context.Background(), command)
	if err != nil {
		t.Fatal(err)
	}
	if created.Status != domain.ReturnShipmentCreated || created.CODAmountVND != 0 || created.FeePayer != domain.ReturnShipmentFeePayerShop || created.PickupAddress != command.PickupAddress || created.Destination != command.Destination {
		t.Fatalf("unexpected return shipment result: %+v", created)
	}
	if len(repository.events) != 1 || repository.events[0].Topic != "shipping.return.events.v1" || repository.events[0].Type != "vn.omama.shipping.return-shipment.created.v1" {
		t.Fatalf("unexpected return event: %+v", repository.events)
	}

	replayed, err := service.CreateReturnShipment(context.Background(), command)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.ID != created.ID || len(repository.events) != 1 {
		t.Fatalf("idempotent replay mismatch: replay=%+v events=%d", replayed, len(repository.events))
	}

	command.ReasonCode = "CHANGED_REASON"
	_, err = service.CreateReturnShipment(context.Background(), command)
	if !errors.Is(err, domain.ErrIdempotencyConflict) {
		t.Fatalf("expected idempotency conflict, got %v", err)
	}
}

func TestCreateReturnShipmentFailsClosedForUnsupportedCarrier(t *testing.T) {
	repository := newMemoryReturnRepository()
	service := NewService(repository, carrier.NewRegistry(carrier.NewFake()))
	command := validReturnCommand()
	command.CarrierCode = "GHN"
	_, err := service.CreateReturnShipment(context.Background(), command)
	if !errors.Is(err, domain.ErrUnsupportedCarrier) {
		t.Fatalf("expected unsupported carrier, got %v", err)
	}
	if len(repository.byID) != 0 {
		t.Fatalf("unsupported carrier should not persist a return shipment: %+v", repository.byID)
	}
}

func TestCreateReturnShipmentPersistsFailureEventForFakeProviderRejection(t *testing.T) {
	repository := newMemoryReturnRepository()
	service := NewService(repository, carrier.NewRegistry(carrier.NewFake()))
	command := validReturnCommand()
	command.PickupAddress.Street = "REJECT address"
	shipment, err := service.CreateReturnShipment(context.Background(), command)
	var providerErr *ReturnProviderError
	if !errors.As(err, &providerErr) {
		t.Fatalf("expected ReturnProviderError, got %v", err)
	}
	if shipment.Status != domain.ReturnShipmentCreationFailed || len(repository.events) != 1 || repository.events[0].Type != "vn.omama.shipping.return-shipment.creation-failed.v1" {
		t.Fatalf("unexpected provider failure state/event: shipment=%+v events=%+v", shipment, repository.events)
	}
}

func validReturnCommand() domain.CreateReturnShipmentCommand {
	return domain.CreateReturnShipmentCommand{
		IdempotencyKey: "return-case-1234567890",
		ReturnCaseID:   uuid.NewString(), OrderID: "ORD-RETURN-1", PackageID: "PKG-1",
		CarrierCode: carrier.FakeCode, ServiceCode: "STANDARD", ReasonCode: "DAMAGED",
		PickupAddress: domain.Recipient{Name: "Customer", Phone: "0900000000", Street: "1 Customer Street", Ward: "Ward 1", District: "District 1", Province: "Hue", CountryCode: "VN"},
		Destination:   domain.Recipient{Name: "Warehouse", Phone: "0900000001", Street: "2 Warehouse Street", Ward: "Ward 2", District: "District 1", Province: "Hue", CountryCode: "VN"},
		Parcel:        domain.Parcel{WeightGrams: 500, LengthCM: 10, WidthCM: 10, HeightCM: 10},
	}
}
