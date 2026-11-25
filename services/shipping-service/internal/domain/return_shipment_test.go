package domain

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewReturnShipmentUsesReturnOnlyDefaults(t *testing.T) {
	command := validReturnShipmentCommand()
	shipment, err := NewReturnShipment(command, time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if shipment.Status != ReturnShipmentCreating || shipment.CODAmountVND != 0 || shipment.FeePayer != ReturnShipmentFeePayerShop || len(shipment.ShipmentNumber) <= 4 {
		t.Fatalf("unexpected return shipment defaults: %+v", shipment)
	}
}

func TestReturnShipmentRequestHashExcludesIdempotencyKeyButBindsAddresses(t *testing.T) {
	command := validReturnShipmentCommand()
	first, err := command.RequestHash()
	if err != nil {
		t.Fatal(err)
	}
	command.IdempotencyKey = "different-return-key-12345"
	second, err := command.RequestHash()
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatal("idempotency key must not change request fingerprint")
	}
	command.PickupAddress.Street = "changed pickup"
	third, err := command.RequestHash()
	if err != nil {
		t.Fatal(err)
	}
	if first == third {
		t.Fatal("changed pickup address must change request fingerprint")
	}
}

func TestReturnShipmentRequiresPickupDestinationParcelAndReason(t *testing.T) {
	command := validReturnShipmentCommand()
	command.PickupAddress.Phone = ""
	_, err := NewReturnShipment(command, time.Now())
	if !errors.Is(err, ErrInvalidShipment) {
		t.Fatalf("expected ErrInvalidShipment, got %v", err)
	}

	command = validReturnShipmentCommand()
	command.Destination.Street = ""
	_, err = NewReturnShipment(command, time.Now())
	if !errors.Is(err, ErrInvalidShipment) {
		t.Fatalf("expected ErrInvalidShipment, got %v", err)
	}

	command = validReturnShipmentCommand()
	command.ReasonCode = ""
	_, err = NewReturnShipment(command, time.Now())
	if !errors.Is(err, ErrInvalidShipment) {
		t.Fatalf("expected ErrInvalidShipment, got %v", err)
	}
}

func validReturnShipmentCommand() CreateReturnShipmentCommand {
	return CreateReturnShipmentCommand{
		IdempotencyKey: "return-idempotency-123456", ReturnCaseID: uuid.NewString(), OrderID: "ORD-1",
		PackageID: "PKG-1", CarrierCode: "FAKE", ServiceCode: "STANDARD", ReasonCode: "DAMAGED",
		PickupAddress: Recipient{Name: "Customer", Phone: "0900000000", Street: "Customer address", Ward: "Ward 1", District: "District 1", Province: "Hue", CountryCode: "VN"},
		Destination:   Recipient{Name: "Warehouse", Phone: "0900000001", Street: "Warehouse address", Ward: "Ward 2", District: "District 1", Province: "Hue", CountryCode: "VN"},
		Parcel:        Parcel{WeightGrams: 500, LengthCM: 10, WidthCM: 10, HeightCM: 10},
	}
}
