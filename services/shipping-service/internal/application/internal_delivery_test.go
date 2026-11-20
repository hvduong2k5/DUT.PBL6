package application

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/omamx/shipping-service/internal/carrier"
	"github.com/omamx/shipping-service/internal/domain"
)

type assignmentRepositoryStub struct {
	*memoryRepository
	calls int
	last  domain.AssignInternalDelivery
}

func (r *assignmentRepositoryStub) AssignInternalDelivery(_ context.Context, c domain.AssignInternalDelivery) (*domain.InternalDeliveryAssignment, error) {
	r.calls++
	r.last = c
	return &domain.InternalDeliveryAssignment{ShipmentID: c.ShipmentID, DriverID: c.DriverID, ActorID: c.ActorID, Version: 1, Reason: c.Reason}, nil
}
func (*assignmentRepositoryStub) GetInternalDeliveryAssignment(context.Context, uuid.UUID) (*domain.InternalDeliveryAssignment, error) {
	return nil, domain.ErrAssignmentNotFound
}
func (*assignmentRepositoryStub) InternalDeliveryAssignmentHistory(context.Context, uuid.UUID) ([]domain.InternalDeliveryAssignmentEvent, error) {
	return []domain.InternalDeliveryAssignmentEvent{}, nil
}

func TestInternalAssignmentServiceValidatesBeforePersistence(t *testing.T) {
	repo := &assignmentRepositoryStub{memoryRepository: newMemoryRepository()}
	svc := NewService(repo, carrier.NewRegistry())
	if _, err := svc.AssignInternalDelivery(context.Background(), domain.AssignInternalDelivery{}); !errors.Is(err, domain.ErrInvalidShipment) || repo.calls != 0 {
		t.Fatal("invalid assignment reached storage")
	}
	c := domain.AssignInternalDelivery{ShipmentID: uuid.New(), DriverID: uuid.New(), ActorID: uuid.New(), IdempotencyKey: "assign-service-001", Reason: "Dispatch"}
	result, err := svc.AssignInternalDelivery(context.Background(), c)
	if err != nil || result.DriverID != c.DriverID || repo.calls != 1 || repo.last != c {
		t.Fatalf("command not preserved: %+v %v", result, err)
	}
	if _, err := svc.GetInternalDeliveryAssignment(context.Background(), uuid.Nil); !errors.Is(err, domain.ErrInvalidShipment) {
		t.Fatal("nil shipment accepted")
	}
	if _, err := svc.GetInternalDeliveryAssignment(context.Background(), c.ShipmentID); !errors.Is(err, domain.ErrAssignmentNotFound) {
		t.Fatalf("missing assignment hidden: %v", err)
	}
	withoutStorage := NewService(newMemoryRepository(), carrier.NewRegistry())
	if _, err := withoutStorage.AssignInternalDelivery(context.Background(), c); err == nil {
		t.Fatal("unconfigured repository accepted")
	}
}
