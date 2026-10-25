package carrier

import (
	"context"
	"time"

	"github.com/omamx/shipping-service/internal/domain"
)

type QuoteRequest struct {
	Destination domain.Recipient `json:"destination"`
	Parcel      domain.Parcel    `json:"parcel"`
	ServiceCode string           `json:"service_code,omitempty"`
}

type Quote struct {
	CarrierCode         string    `json:"carrier_code"`
	ServiceCode         string    `json:"service_code"`
	ServiceName         string    `json:"service_name"`
	FeeVND              int64     `json:"fee_vnd"`
	EstimatedDeliveryAt time.Time `json:"estimated_delivery_at"`
	ExpiresAt           time.Time `json:"expires_at"`
}

type CreateRequest struct {
	IdempotencyKey string
	ShipmentID     string
	OrderID        string
	PackageID      string
	SealCode       string
	ServiceCode    string
	Recipient      domain.Recipient
	Parcel         domain.Parcel
	CODAmountVND   int64
	ShippingNotes  string
}

type CreateResult struct {
	CarrierOrderID      string
	TrackingCode        string
	LabelReference      string
	EstimatedFeeVND     int64
	EstimatedPickupAt   *time.Time
	EstimatedDeliveryAt *time.Time
}

type WebhookUpdate struct {
	ExternalEventID     string
	TrackingCode        string
	ProviderStatusCode  string
	ProviderStatusLabel string
	MappedStatus        domain.ShipmentStatus
	Location            string
	PublicDescription   string
	InternalNote        string
	EventAt             time.Time
}

type Carrier interface {
	Code() string
	Quote(context.Context, QuoteRequest) (Quote, error)
	CreateShipment(context.Context, CreateRequest) (CreateResult, error)
	ParseWebhook(body []byte) (WebhookUpdate, error)
}

type Registry struct {
	carriers map[string]Carrier
}

func NewRegistry(carriers ...Carrier) *Registry {
	r := &Registry{carriers: make(map[string]Carrier, len(carriers))}
	for _, c := range carriers {
		r.carriers[c.Code()] = c
	}
	return r
}

func (r *Registry) Get(code string) (Carrier, bool) {
	c, ok := r.carriers[normalizeCode(code)]
	return c, ok
}

func normalizeCode(code string) string {
	b := []byte(code)
	for i, c := range b {
		if c >= 'a' && c <= 'z' {
			b[i] = c - ('a' - 'A')
		}
	}
	return string(b)
}
