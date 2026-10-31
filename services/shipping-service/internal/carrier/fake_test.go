package carrier

import (
	"context"
	"testing"
	"time"

	"github.com/omamx/shipping-service/internal/domain"
)

func TestFakeCreateIsDeterministicForIdempotencyKey(t *testing.T) {
	fake := NewFake()
	req := CreateRequest{IdempotencyKey: "packing:pkg-1", Parcel: domain.Parcel{WeightGrams: 1000}}
	first, err := fake.CreateShipment(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	second, err := fake.CreateShipment(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if first.TrackingCode != second.TrackingCode || first.CarrierOrderID != second.CarrierOrderID {
		t.Fatal("fake carrier must be idempotent")
	}
}

func TestFakeParsesWebhook(t *testing.T) {
	fake := NewFake()
	payload := []byte(`{"event_id":"evt-1","tracking_code":"FAKE1","status":"PICKED_UP","event_at":"2026-10-17T10:00:00Z"}`)
	update, err := fake.ParseWebhook(payload)
	if err != nil {
		t.Fatal(err)
	}
	if update.MappedStatus != domain.StatusHandedOver || !update.EventAt.Equal(time.Date(2026, 10, 17, 10, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected update: %+v", update)
	}
}

func TestFakeQuoteMatchesCreatedShipmentFee(t *testing.T) {
	fake := NewFake()
	parcel := domain.Parcel{WeightGrams: 1200, LengthCM: 30, WidthCM: 20, HeightCM: 15, Fragile: true}
	recipient := domain.Recipient{Name: "A", Phone: "0905000000", Street: "1 Hue", Ward: "A", District: "B", Province: "Hue", CountryCode: "VN"}
	quote, err := fake.Quote(context.Background(), QuoteRequest{Destination: recipient, Parcel: parcel, ServiceCode: "STANDARD"})
	if err != nil {
		t.Fatal(err)
	}
	created, err := fake.CreateShipment(context.Background(), CreateRequest{IdempotencyKey: "fee-check", Parcel: parcel})
	if err != nil {
		t.Fatal(err)
	}
	if quote.FeeVND != created.EstimatedFeeVND {
		t.Fatalf("quote fee %d differs from created fee %d", quote.FeeVND, created.EstimatedFeeVND)
	}
}
