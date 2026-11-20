package carrier

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const InternalCode = "INTERNAL"

// Internal is the company fleet, not a simulator or an external webhook provider.
// Assignment and proof-bearing delivery actions belong to the Shipping workbench.
type Internal struct {
	feeVND  int64
	transit time.Duration
	now     func() time.Time
}

func NewInternal(feeVND int64, estimatedTransit time.Duration) (*Internal, error) {
	if feeVND < 0 || feeVND > 1_000_000_000 || estimatedTransit <= 0 || estimatedTransit > 30*24*time.Hour {
		return nil, fmt.Errorf("internal delivery requires a fee between 0 and 1000000000 VND and transit time within 30 days")
	}
	return &Internal{feeVND: feeVND, transit: estimatedTransit, now: time.Now}, nil
}

func (*Internal) Code() string { return InternalCode }

func internalService(code string) error {
	if code != "" && !strings.EqualFold(code, "STANDARD") {
		return fmt.Errorf("unsupported internal delivery service")
	}
	return nil
}

func (c *Internal) Quote(_ context.Context, req QuoteRequest) (Quote, error) {
	if err := req.Destination.Validate(); err != nil {
		return Quote{}, err
	}
	if err := req.Parcel.Validate(); err != nil {
		return Quote{}, err
	}
	if err := internalService(req.ServiceCode); err != nil {
		return Quote{}, err
	}
	now := c.now().UTC()
	return Quote{CarrierCode: InternalCode, ServiceCode: "STANDARD", ServiceName: "Company delivery",
		FeeVND: c.feeVND, EstimatedDeliveryAt: now.Add(c.transit), ExpiresAt: now.Add(15 * time.Minute)}, nil
}

func (c *Internal) CreateShipment(ctx context.Context, req CreateRequest) (CreateResult, error) {
	id, err := uuid.Parse(req.ShipmentID)
	if err != nil || id == uuid.Nil || strings.TrimSpace(req.IdempotencyKey) == "" || strings.TrimSpace(req.OrderID) == "" || strings.TrimSpace(req.PackageID) == "" {
		return CreateResult{}, fmt.Errorf("internal delivery requires shipment UUID, order, package and idempotency key")
	}
	if req.CODAmountVND != 0 {
		return CreateResult{}, fmt.Errorf("internal COD collection is not enabled")
	}
	quote, err := c.Quote(ctx, QuoteRequest{Destination: req.Recipient, Parcel: req.Parcel, ServiceCode: req.ServiceCode})
	if err != nil {
		return CreateResult{}, err
	}
	delivery := quote.EstimatedDeliveryAt
	// Stable per shipment, regardless of a later retry's clock or action key.
	return CreateResult{CarrierOrderID: "INTERNAL-" + id.String(), TrackingCode: "INT-" + id.String(),
		EstimatedFeeVND: c.feeVND, EstimatedDeliveryAt: &delivery}, nil
}

func (*Internal) CancelShipment(_ context.Context, req CancelRequest) error {
	if !strings.HasPrefix(req.CarrierOrderID, "INTERNAL-") {
		return fmt.Errorf("invalid internal carrier reference")
	}
	id, err := uuid.Parse(strings.TrimPrefix(req.CarrierOrderID, "INTERNAL-"))
	if err != nil || id == uuid.Nil {
		return fmt.Errorf("invalid internal carrier reference")
	}
	// No external side effect: application lifecycle/idempotency owns cancellation.
	return nil
}

func (*Internal) GenerateLabelURL(context.Context, LabelRequest) (LabelResult, error) {
	return LabelResult{}, fmt.Errorf("internal delivery print labels are not implemented")
}

func (*Internal) ParseWebhook([]byte) (WebhookUpdate, error) {
	return WebhookUpdate{}, fmt.Errorf("%w: internal deliveries require authenticated employee actions", ErrInvalidWebhook)
}
