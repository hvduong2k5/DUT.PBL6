package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/omamx/shipping-service/internal/carrier"
	"github.com/omamx/shipping-service/internal/domain"
)

func TestInternalCarrierUsesShipmentLifecycleAndCannotMutateFake(t *testing.T) {
	repo := newMemoryRepository()
	internal, err := carrier.NewInternal(25000, 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(repo, carrier.NewRegistry(carrier.NewFake(), internal))
	cmd := applicationCommand()
	cmd.CarrierCode = carrier.InternalCode
	cmd.CODAmountVND = 0
	created, err := svc.CreateShipment(context.Background(), cmd)
	if err != nil || created.Status != domain.StatusCreated || created.CarrierCode != carrier.InternalCode {
		t.Fatalf("internal creation: %+v %v", created, err)
	}
	replay, err := svc.CreateShipment(context.Background(), cmd)
	if err != nil || replay.ID != created.ID || replay.Revision != created.Revision {
		t.Fatalf("creation replay changed identity: %v", err)
	}
	cmd.Parcel.WeightGrams++
	if _, err := svc.CreateShipment(context.Background(), cmd); !errors.Is(err, domain.ErrIdempotencyConflict) {
		t.Fatalf("changed request accepted: %v", err)
	}
	cancelled, err := svc.CancelShipment(context.Background(), created.ID, "internal-cancel-0001", carrier.CancelReasonOther, "cancel before assignment")
	if err != nil || cancelled.Status != domain.StatusCancelled {
		t.Fatalf("cancel: %+v %v", cancelled, err)
	}
	if _, _, err := svc.ApplyWebhook(context.Background(), carrier.InternalCode, []byte(`{"status":"DELIVERED"}`)); !errors.Is(err, carrier.ErrInvalidWebhook) {
		t.Fatalf("internal webhook accepted: %v", err)
	}
	fakeCommand := applicationCommand()
	fakeCommand.IdempotencyKey = "separate-fake-command-0001"
	fakeCommand.PackageID = "separate-fake-package"
	fake, err := svc.CreateShipment(context.Background(), fakeCommand)
	if err != nil || fake.CarrierCode != carrier.FakeCode || fake.ID == created.ID {
		t.Fatalf("fake isolation: %+v %v", fake, err)
	}
}
