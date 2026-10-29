package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type ShipmentStatus string

const (
	StatusCreating         ShipmentStatus = "CREATING"
	StatusCreationFailed   ShipmentStatus = "CREATION_FAILED"
	StatusCreated          ShipmentStatus = "CREATED"
	StatusReadyForHandover ShipmentStatus = "READY_FOR_HANDOVER"
	StatusHandedOver       ShipmentStatus = "HANDED_OVER"
	StatusInTransit        ShipmentStatus = "IN_TRANSIT"
	StatusOutForDelivery   ShipmentStatus = "OUT_FOR_DELIVERY"
	StatusDelivered        ShipmentStatus = "DELIVERED"
	StatusDeliveryFailed   ShipmentStatus = "DELIVERY_FAILED"
	StatusReturning        ShipmentStatus = "RETURNING"
	StatusReturned         ShipmentStatus = "RETURNED"
	StatusCancelled        ShipmentStatus = "CANCELLED"
)

var (
	ErrInvalidShipment    = errors.New("invalid shipment")
	ErrInvalidTransition  = errors.New("invalid shipment status transition")
	ErrConcurrentUpdate   = errors.New("shipment was updated concurrently")
	ErrShipmentNotFound   = errors.New("shipment not found")
	ErrTrackingNotFound   = errors.New("tracking code not found")
	ErrDuplicateEvent     = errors.New("tracking event already processed")
	ErrUnsupportedCarrier = errors.New("unsupported carrier")
)

type Recipient struct {
	Name        string `json:"name"`
	Phone       string `json:"phone"`
	Street      string `json:"street"`
	Ward        string `json:"ward"`
	District    string `json:"district"`
	Province    string `json:"province"`
	CountryCode string `json:"country_code"`
	PostalCode  string `json:"postal_code,omitempty"`
}

func (r Recipient) Validate() error {
	if strings.TrimSpace(r.Name) == "" || strings.TrimSpace(r.Phone) == "" ||
		strings.TrimSpace(r.Street) == "" || strings.TrimSpace(r.Ward) == "" ||
		strings.TrimSpace(r.District) == "" || strings.TrimSpace(r.Province) == "" {
		return fmt.Errorf("%w: recipient name, phone and complete address are required", ErrInvalidShipment)
	}
	if r.CountryCode != "" && !strings.EqualFold(r.CountryCode, "VN") {
		return fmt.Errorf("%w: only VN destinations are currently supported", ErrInvalidShipment)
	}
	return nil
}

type Parcel struct {
	WeightGrams int64 `json:"weight_grams"`
	LengthCM    int32 `json:"length_cm"`
	WidthCM     int32 `json:"width_cm"`
	HeightCM    int32 `json:"height_cm"`
	Fragile     bool  `json:"fragile"`
}

func (p Parcel) Validate() error {
	if p.WeightGrams <= 0 {
		return fmt.Errorf("%w: parcel weight must be positive", ErrInvalidShipment)
	}
	if p.LengthCM <= 0 || p.WidthCM <= 0 || p.HeightCM <= 0 {
		return fmt.Errorf("%w: parcel dimensions must be positive", ErrInvalidShipment)
	}
	return nil
}

type CreateShipmentCommand struct {
	IdempotencyKey       string    `json:"idempotency_key"`
	OrderID              string    `json:"order_id"`
	PackageID            string    `json:"package_id"`
	SealCode             string    `json:"seal_code"`
	CarrierCode          string    `json:"carrier_code"`
	ServiceCode          string    `json:"service_code"`
	Recipient            Recipient `json:"recipient"`
	Parcel               Parcel    `json:"parcel"`
	CODAmountVND         int64     `json:"cod_amount_vnd"`
	CheckoutFeeVND       int64     `json:"checkout_fee_vnd"`
	QuoteID              string    `json:"quote_id,omitempty"`
	ReadyGeneration      int64     `json:"ready_generation,omitempty"`
	ItemsDigest          string    `json:"items_digest,omitempty"`
	OrderSnapshotVersion int64     `json:"order_snapshot_version,omitempty"`
	ShippingNotes        string    `json:"shipping_notes,omitempty"`
}

func (c CreateShipmentCommand) Validate() error {
	if strings.TrimSpace(c.IdempotencyKey) == "" || strings.TrimSpace(c.OrderID) == "" ||
		strings.TrimSpace(c.PackageID) == "" || strings.TrimSpace(c.CarrierCode) == "" ||
		strings.TrimSpace(c.ServiceCode) == "" {
		return fmt.Errorf("%w: idempotency_key, order_id, package_id, carrier_code and service_code are required", ErrInvalidShipment)
	}
	if c.CODAmountVND < 0 || c.CheckoutFeeVND < 0 {
		return fmt.Errorf("%w: monetary values cannot be negative", ErrInvalidShipment)
	}
	if c.ReadyGeneration < 0 || c.OrderSnapshotVersion < 0 {
		return fmt.Errorf("%w: proof versions cannot be negative", ErrInvalidShipment)
	}
	if err := c.Recipient.Validate(); err != nil {
		return err
	}
	return c.Parcel.Validate()
}

type Shipment struct {
	ID                      uuid.UUID      `json:"shipment_id"`
	ShipmentNumber          string         `json:"shipment_number"`
	OrderID                 string         `json:"order_id"`
	PackageID               string         `json:"package_id"`
	SealCode                string         `json:"seal_code"`
	CarrierCode             string         `json:"carrier_code"`
	ServiceCode             string         `json:"service_code"`
	CarrierOrderID          string         `json:"carrier_order_id,omitempty"`
	TrackingCode            string         `json:"tracking_code,omitempty"`
	LabelReference          string         `json:"label_reference,omitempty"`
	Status                  ShipmentStatus `json:"status"`
	Recipient               Recipient      `json:"recipient"`
	Parcel                  Parcel         `json:"parcel"`
	CODAmountVND            int64          `json:"cod_amount_vnd"`
	CheckoutFeeVND          int64          `json:"checkout_fee_vnd"`
	QuoteID                 string         `json:"quote_id,omitempty"`
	ReadyGeneration         int64          `json:"ready_generation,omitempty"`
	ItemsDigest             string         `json:"items_digest,omitempty"`
	OrderSnapshotVersion    int64          `json:"order_snapshot_version,omitempty"`
	ProviderEstimatedFeeVND int64          `json:"provider_estimated_fee_vnd"`
	ActualFeeVND            *int64         `json:"actual_fee_vnd,omitempty"`
	FailureCode             string         `json:"failure_code,omitempty"`
	FailureReason           string         `json:"failure_reason,omitempty"`
	EstimatedPickupAt       *time.Time     `json:"estimated_pickup_at,omitempty"`
	EstimatedDeliveryAt     *time.Time     `json:"estimated_delivery_at,omitempty"`
	HandedOverAt            *time.Time     `json:"handed_over_at,omitempty"`
	DeliveredAt             *time.Time     `json:"delivered_at,omitempty"`
	LastEventAt             *time.Time     `json:"last_event_at,omitempty"`
	CreatedAt               time.Time      `json:"created_at"`
	UpdatedAt               time.Time      `json:"updated_at"`
	Revision                int64          `json:"revision"`
}

func NewCreatingShipment(cmd CreateShipmentCommand, now time.Time) (*Shipment, error) {
	if err := cmd.Validate(); err != nil {
		return nil, err
	}
	id := uuid.New()
	now = now.UTC()
	return &Shipment{
		ID:                   id,
		ShipmentNumber:       "SHP-" + strings.ToUpper(strings.ReplaceAll(id.String()[:13], "-", "")),
		OrderID:              cmd.OrderID,
		PackageID:            cmd.PackageID,
		SealCode:             cmd.SealCode,
		CarrierCode:          strings.ToUpper(cmd.CarrierCode),
		ServiceCode:          cmd.ServiceCode,
		Status:               StatusCreating,
		Recipient:            cmd.Recipient,
		Parcel:               cmd.Parcel,
		CODAmountVND:         cmd.CODAmountVND,
		CheckoutFeeVND:       cmd.CheckoutFeeVND,
		QuoteID:              cmd.QuoteID,
		ReadyGeneration:      cmd.ReadyGeneration,
		ItemsDigest:          cmd.ItemsDigest,
		OrderSnapshotVersion: cmd.OrderSnapshotVersion,
		CreatedAt:            now,
		UpdatedAt:            now,
		Revision:             1,
	}, nil
}

var allowedTransitions = map[ShipmentStatus]map[ShipmentStatus]struct{}{
	StatusCreating: {
		StatusCreated: {}, StatusCreationFailed: {},
	},
	StatusCreationFailed: {
		StatusCreating: {}, StatusCancelled: {},
	},
	StatusCreated: {
		StatusReadyForHandover: {}, StatusHandedOver: {}, StatusInTransit: {}, StatusCancelled: {},
	},
	StatusReadyForHandover: {
		StatusHandedOver: {}, StatusInTransit: {}, StatusCancelled: {},
	},
	StatusHandedOver: {
		StatusInTransit: {}, StatusOutForDelivery: {}, StatusDelivered: {}, StatusDeliveryFailed: {}, StatusReturning: {},
	},
	StatusInTransit: {
		StatusOutForDelivery: {}, StatusDelivered: {}, StatusDeliveryFailed: {}, StatusReturning: {},
	},
	StatusOutForDelivery: {
		StatusDelivered: {}, StatusDeliveryFailed: {}, StatusReturning: {},
	},
	StatusDeliveryFailed: {
		StatusOutForDelivery: {}, StatusReturning: {}, StatusReturned: {},
	},
	StatusReturning: {
		StatusReturned: {},
	},
}

func (s *Shipment) Transition(to ShipmentStatus, now time.Time) error {
	if s.Status == to {
		s.UpdatedAt = now.UTC()
		s.Revision++
		return nil
	}
	if _, ok := allowedTransitions[s.Status][to]; !ok {
		return fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, s.Status, to)
	}
	s.Status = to
	s.UpdatedAt = now.UTC()
	s.Revision++
	if to == StatusHandedOver {
		t := now.UTC()
		s.HandedOverAt = &t
	}
	if to == StatusDelivered {
		t := now.UTC()
		s.DeliveredAt = &t
	}
	return nil
}

func (s *Shipment) MarkCreated(carrierOrderID, trackingCode, labelReference string, estimatedFee int64, pickupAt, deliveryAt *time.Time, now time.Time) error {
	if strings.TrimSpace(carrierOrderID) == "" || strings.TrimSpace(trackingCode) == "" {
		return fmt.Errorf("%w: carrier order id and tracking code are required", ErrInvalidShipment)
	}
	if err := s.Transition(StatusCreated, now); err != nil {
		return err
	}
	s.CarrierOrderID = carrierOrderID
	s.TrackingCode = trackingCode
	s.LabelReference = labelReference
	s.ProviderEstimatedFeeVND = estimatedFee
	s.EstimatedPickupAt = pickupAt
	s.EstimatedDeliveryAt = deliveryAt
	s.FailureCode = ""
	s.FailureReason = ""
	return nil
}

func (s *Shipment) MarkCreationFailed(code, reason string, now time.Time) error {
	if err := s.Transition(StatusCreationFailed, now); err != nil {
		return err
	}
	s.FailureCode = code
	s.FailureReason = reason
	return nil
}

type TrackingEvent struct {
	ID                  uuid.UUID      `json:"id"`
	ShipmentID          uuid.UUID      `json:"shipment_id"`
	ExternalEventID     string         `json:"external_event_id"`
	CarrierCode         string         `json:"carrier_code"`
	ProviderStatusCode  string         `json:"provider_status_code"`
	ProviderStatusLabel string         `json:"provider_status_label,omitempty"`
	MappedStatus        ShipmentStatus `json:"mapped_status"`
	MappingResult       string         `json:"mapping_result"`
	Location            string         `json:"location,omitempty"`
	PublicDescription   string         `json:"public_description,omitempty"`
	InternalNote        string         `json:"internal_note,omitempty"`
	EventAt             time.Time      `json:"event_at"`
	ReceivedAt          time.Time      `json:"received_at"`
}

type DomainEvent struct {
	SpecVersion     string            `json:"specversion"`
	ID              uuid.UUID         `json:"id"`
	Source          string            `json:"source"`
	Type            string            `json:"type"`
	Subject         string            `json:"subject"`
	Time            time.Time         `json:"time"`
	DataContentType string            `json:"datacontenttype"`
	TraceParent     string            `json:"traceparent,omitempty"`
	Data            ShipmentEventData `json:"data"`
	AggregateID     uuid.UUID         `json:"-"`
	OccurredAt      time.Time         `json:"-"`
}

type ShipmentEventData struct {
	ShipmentID   uuid.UUID `json:"shipment_id"`
	OrderID      string    `json:"order_id"`
	PackageID    string    `json:"package_id"`
	CarrierCode  string    `json:"carrier_code"`
	TrackingCode string    `json:"tracking_code,omitempty"`
	Status       string    `json:"status"`
}

func NewDomainEvent(eventType string, shipment *Shipment, now time.Time) DomainEvent {
	now = now.UTC()
	traceID := strings.ReplaceAll(uuid.NewString(), "-", "")
	spanID := strings.ReplaceAll(uuid.NewString(), "-", "")[:16]
	return DomainEvent{
		SpecVersion: "1.0", ID: uuid.New(), Source: "https://omama.vn/services/shipping-service",
		Type: eventType, Subject: "shipment_id:" + shipment.ID.String(), Time: now,
		DataContentType: "application/json", TraceParent: "00-" + traceID + "-" + spanID + "-01",
		AggregateID: shipment.ID, OccurredAt: now,
		Data: ShipmentEventData{ShipmentID: shipment.ID, OrderID: shipment.OrderID,
			PackageID: shipment.PackageID, CarrierCode: shipment.CarrierCode,
			TrackingCode: shipment.TrackingCode, Status: string(shipment.Status)},
	}
}
