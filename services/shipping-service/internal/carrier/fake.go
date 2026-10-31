package carrier

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/omamx/shipping-service/internal/domain"
)

const FakeCode = "FAKE"

type Fake struct {
	now func() time.Time
}

func NewFake() *Fake {
	return &Fake{now: time.Now}
}

func (f *Fake) Code() string { return FakeCode }

func (f *Fake) Quote(_ context.Context, req QuoteRequest) (Quote, error) {
	if err := req.Destination.Validate(); err != nil {
		return Quote{}, err
	}
	if err := req.Parcel.Validate(); err != nil {
		return Quote{}, err
	}
	now := f.now().UTC()
	fee := int64(22000) + ((req.Parcel.WeightGrams+499)/500)*5000
	if req.Parcel.Fragile {
		fee += 7000
	}
	serviceCode := req.ServiceCode
	if serviceCode == "" {
		serviceCode = "STANDARD"
	}
	return Quote{
		CarrierCode: FakeCode, ServiceCode: strings.ToUpper(serviceCode), ServiceName: "Giao hàng tiêu chuẩn (local)",
		FeeVND: fee, EstimatedDeliveryAt: now.Add(72 * time.Hour), ExpiresAt: now.Add(15 * time.Minute),
	}, nil
}

func (f *Fake) CreateShipment(_ context.Context, req CreateRequest) (CreateResult, error) {
	if strings.TrimSpace(req.IdempotencyKey) == "" {
		return CreateResult{}, fmt.Errorf("idempotency key is required")
	}
	if strings.Contains(strings.ToUpper(req.Recipient.Street), "REJECT") {
		return CreateResult{}, fmt.Errorf("FAKE_ADDRESS_REJECTED: carrier rejected destination address")
	}
	sum := sha256.Sum256([]byte(req.IdempotencyKey))
	suffix := strings.ToUpper(hex.EncodeToString(sum[:6]))
	now := f.now().UTC()
	pickup := now.Add(4 * time.Hour)
	delivery := now.Add(72 * time.Hour)
	fee := int64(22000) + ((req.Parcel.WeightGrams+499)/500)*5000
	if req.Parcel.Fragile {
		fee += 7000
	}
	return CreateResult{
		CarrierOrderID:      "FAKE-ORDER-" + suffix,
		TrackingCode:        "FAKE" + suffix,
		LabelReference:      "labels/fake/" + suffix + ".pdf",
		EstimatedFeeVND:     fee,
		EstimatedPickupAt:   &pickup,
		EstimatedDeliveryAt: &delivery,
	}, nil
}

func (f *Fake) CancelShipment(_ context.Context, req CancelRequest) error {
	if strings.TrimSpace(req.CarrierOrderID) == "" {
		return fmt.Errorf("carrier order id is required")
	}
	return nil
}

func (f *Fake) GenerateLabelURL(_ context.Context, req LabelRequest) (LabelResult, error) {
	if strings.TrimSpace(req.CarrierOrderID) == "" {
		return LabelResult{}, fmt.Errorf("carrier order id is required")
	}
	format, err := NormalizeLabelFormat(req.Format)
	if err != nil {
		return LabelResult{}, err
	}
	now := f.now().UTC()
	return LabelResult{
		URL:       "https://fake-carrier.local/labels/" + url.PathEscape(req.CarrierOrderID) + "?format=" + url.QueryEscape(format),
		ExpiresAt: now.Add(15 * time.Minute),
	}, nil
}

type fakeWebhookPayload struct {
	EventID      string    `json:"event_id"`
	TrackingCode string    `json:"tracking_code"`
	Status       string    `json:"status"`
	StatusLabel  string    `json:"status_label"`
	Location     string    `json:"location"`
	Description  string    `json:"description"`
	InternalNote string    `json:"internal_note"`
	EventAt      time.Time `json:"event_at"`
}

func (f *Fake) ParseWebhook(body []byte) (WebhookUpdate, error) {
	var payload fakeWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return WebhookUpdate{}, fmt.Errorf("decode fake webhook: %w", err)
	}
	if payload.EventID == "" || payload.TrackingCode == "" || payload.Status == "" || payload.EventAt.IsZero() {
		return WebhookUpdate{}, fmt.Errorf("event_id, tracking_code, status and event_at are required")
	}
	mapped, ok := fakeStatusMap[strings.ToUpper(payload.Status)]
	if !ok {
		return WebhookUpdate{}, fmt.Errorf("unmapped fake carrier status %q", payload.Status)
	}
	return WebhookUpdate{
		ExternalEventID: payload.EventID, TrackingCode: payload.TrackingCode,
		ProviderStatusCode: strings.ToUpper(payload.Status), ProviderStatusLabel: payload.StatusLabel,
		MappedStatus: mapped, Location: payload.Location, PublicDescription: payload.Description,
		InternalNote: payload.InternalNote, EventAt: payload.EventAt.UTC(),
	}, nil
}

var fakeStatusMap = map[string]domain.ShipmentStatus{
	"ORDER_CREATED":    domain.StatusCreated,
	"READY_FOR_PICKUP": domain.StatusReadyForHandover,
	"PICKED_UP":        domain.StatusHandedOver,
	"IN_TRANSIT":       domain.StatusInTransit,
	"OUT_FOR_DELIVERY": domain.StatusOutForDelivery,
	"DELIVERED":        domain.StatusDelivered,
	"DELIVERY_FAILED":  domain.StatusDeliveryFailed,
	"RETURNING":        domain.StatusReturning,
	"RETURNED":         domain.StatusReturned,
}
