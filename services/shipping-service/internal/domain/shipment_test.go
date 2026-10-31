package domain

import (
	"errors"
	"testing"
	"time"
)

func validCommand() CreateShipmentCommand {
	return CreateShipmentCommand{
		IdempotencyKey: "packing:pkg-1", OrderID: "ord-1", PackageID: "pkg-1",
		CarrierCode: "fake", ServiceCode: "standard",
		Recipient: Recipient{Name: "Nguyen Van A", Phone: "0905000000", Street: "1 Nguyen Hue", Ward: "Thuan Hoa", District: "Phu Xuan", Province: "Hue", CountryCode: "VN"},
		Parcel:    Parcel{WeightGrams: 1200, LengthCM: 30, WidthCM: 20, HeightCM: 15},
	}
}

func TestShipmentLifecycle(t *testing.T) {
	now := time.Date(2026, 10, 17, 10, 0, 0, 0, time.UTC)
	s, err := NewCreatingShipment(validCommand(), now)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.MarkCreated("carrier-1", "TRACK-1", "label-1", 42000, nil, nil, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	for _, status := range []ShipmentStatus{StatusReadyForHandover, StatusHandedOver, StatusInTransit, StatusOutForDelivery, StatusDelivered} {
		if err := s.Transition(status, now.Add(time.Minute)); err != nil {
			t.Fatalf("transition to %s: %v", status, err)
		}
	}
	if s.DeliveredAt == nil || s.Status != StatusDelivered {
		t.Fatalf("expected delivered shipment, got %s", s.Status)
	}
}

func TestShipmentRejectsInvalidTransition(t *testing.T) {
	s, err := NewCreatingShipment(validCommand(), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Transition(StatusDelivered, time.Now()); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expected invalid transition, got %v", err)
	}
}

func TestCreateCommandRequiresPackageDimensions(t *testing.T) {
	cmd := validCommand()
	cmd.Parcel.HeightCM = 0
	if _, err := NewCreatingShipment(cmd, time.Now()); !errors.Is(err, ErrInvalidShipment) {
		t.Fatalf("expected invalid shipment, got %v", err)
	}
}
