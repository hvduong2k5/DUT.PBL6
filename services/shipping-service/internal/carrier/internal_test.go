package carrier

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/omamx/shipping-service/internal/domain"
)

func internalRequest() CreateRequest {
	return CreateRequest{ShipmentID: "a0000000-0000-0000-0000-000000000001", IdempotencyKey: "internal-create-0001", OrderID: "order-1", PackageID: "package-1", ServiceCode: "STANDARD",
		Recipient: domain.Recipient{Name: "Test", Phone: "0900000001", Street: "Test street", Ward: "Test ward", Province: "Test province", CountryCode: "VN"},
		Parcel:    domain.Parcel{WeightGrams: 500, LengthCM: 20, WidthCM: 15, HeightCM: 10}}
}

func TestInternalCarrierQuoteAndStableCreation(t *testing.T) {
	c, err := NewInternal(25000, 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	c.now = func() time.Time { return time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC) }
	req := internalRequest()
	q, err := c.Quote(context.Background(), QuoteRequest{Destination: req.Recipient, Parcel: req.Parcel})
	if err != nil || q.CarrierCode != InternalCode || q.FeeVND != 25000 || q.ServiceCode != "STANDARD" || !q.ExpiresAt.Equal(c.now().Add(15*time.Minute)) {
		t.Fatalf("quote: %+v %v", q, err)
	}
	first, err := c.CreateShipment(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	req.IdempotencyKey = "another-retry-key"
	second, err := c.CreateShipment(context.Background(), req)
	if err != nil || first.TrackingCode != second.TrackingCode || first.CarrierOrderID != second.CarrierOrderID || first.EstimatedFeeVND != q.FeeVND || first.LabelReference != "" {
		t.Fatalf("unstable internal identity: %+v %+v %v", first, second, err)
	}
	if _, ok := NewRegistry(NewFake(), c).Get("internal"); !ok {
		t.Fatal("internal carrier not registered independently")
	}
}

func TestInternalCarrierRejectsInvalidCommands(t *testing.T) {
	c, _ := NewInternal(0, time.Hour)
	for _, tc := range []struct {
		name   string
		mutate func(*CreateRequest)
	}{
		{"nonzero COD", func(r *CreateRequest) { r.CODAmountVND = 1 }},
		{"negative COD", func(r *CreateRequest) { r.CODAmountVND = -1 }},
		{"invalid shipment", func(r *CreateRequest) { r.ShipmentID = "GHN-ORDER" }},
		{"nil shipment", func(r *CreateRequest) { r.ShipmentID = "00000000-0000-0000-0000-000000000000" }},
		{"missing key", func(r *CreateRequest) { r.IdempotencyKey = "" }},
		{"missing order", func(r *CreateRequest) { r.OrderID = "" }},
		{"missing package", func(r *CreateRequest) { r.PackageID = "" }},
		{"unsupported service", func(r *CreateRequest) { r.ServiceCode = "GHN_LIGHT" }},
		{"invalid address", func(r *CreateRequest) { r.Recipient.Street = "" }},
		{"invalid parcel", func(r *CreateRequest) { r.Parcel.WeightGrams = 0 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := internalRequest()
			tc.mutate(&r)
			if _, err := c.CreateShipment(context.Background(), r); err == nil {
				t.Fatal("invalid command accepted")
			}
		})
	}
	for _, fee := range []int64{-1, 1_000_000_001} {
		if _, err := NewInternal(fee, time.Hour); err == nil {
			t.Fatal("invalid fee accepted")
		}
	}
	for _, d := range []time.Duration{0, -time.Second, 31 * 24 * time.Hour} {
		if _, err := NewInternal(0, d); err == nil {
			t.Fatal("invalid transit accepted")
		}
	}
}

func TestInternalCarrierCannotConsumeWebhooksOrFakeLabels(t *testing.T) {
	c, _ := NewInternal(0, time.Hour)
	if _, err := c.ParseWebhook([]byte(`{"status":"DELIVERED"}`)); !errors.Is(err, ErrInvalidWebhook) {
		t.Fatalf("external update accepted: %v", err)
	}
	if _, err := c.GenerateLabelURL(context.Background(), LabelRequest{}); err == nil {
		t.Fatal("invented print URL")
	}
	if err := c.CancelShipment(context.Background(), CancelRequest{CarrierOrderID: "INTERNAL-" + internalRequest().ShipmentID}); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"GHN-ORDER", "FAKE-ORDER", "INTERNAL-invalid", "INTERNAL-00000000-0000-0000-0000-000000000000"} {
		if err := c.CancelShipment(context.Background(), CancelRequest{CarrierOrderID: id}); err == nil {
			t.Fatal("accepted other carrier reference")
		}
	}
}
