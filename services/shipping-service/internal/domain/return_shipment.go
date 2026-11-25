package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	ReturnShipmentCreating       = "CREATING"
	ReturnShipmentCreated        = "CREATED"
	ReturnShipmentCreationFailed = "CREATION_FAILED"
	ReturnShipmentFeePayerShop   = "SHOP"
)

var ErrReturnShipmentNotFound = errors.New("return shipment not found")

// CreateReturnShipmentCommand is an approved Care return case's request for a
// customer-to-warehouse pickup. Shipping records the request but does not
// approve the return, calculate a refund, or mutate the Order.
type CreateReturnShipmentCommand struct {
	IdempotencyKey string    `json:"-"`
	ReturnCaseID   string    `json:"return_case_id"`
	OrderID        string    `json:"order_id"`
	PackageID      string    `json:"package_id"`
	CarrierCode    string    `json:"carrier_code"`
	ServiceCode    string    `json:"service_code"`
	PickupAddress  Recipient `json:"pickup_address"`
	Destination    Recipient `json:"destination"`
	Parcel         Parcel    `json:"parcel"`
	ReasonCode     string    `json:"reason_code"`
	ShippingNotes  string    `json:"shipping_notes,omitempty"`
}

func (c CreateReturnShipmentCommand) RequestHash() (string, error) {
	c.IdempotencyKey = ""
	encoded, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}

func (c CreateReturnShipmentCommand) Validate() error {
	if strings.TrimSpace(c.IdempotencyKey) == "" || strings.TrimSpace(c.ReturnCaseID) == "" ||
		strings.TrimSpace(c.OrderID) == "" || strings.TrimSpace(c.PackageID) == "" ||
		strings.TrimSpace(c.CarrierCode) == "" || strings.TrimSpace(c.ServiceCode) == "" ||
		strings.TrimSpace(c.ReasonCode) == "" {
		return fmt.Errorf("%w: idempotency key, return case, order, package, carrier, service and reason are required", ErrInvalidShipment)
	}
	if len(c.ReturnCaseID) > 128 || len(c.OrderID) > 128 || len(c.PackageID) > 128 || len(c.ReasonCode) > 80 {
		return fmt.Errorf("%w: return shipment identifiers or reason exceed supported length", ErrInvalidShipment)
	}
	if err := c.PickupAddress.Validate(); err != nil {
		return fmt.Errorf("%w: invalid customer pickup address", err)
	}
	if err := c.Destination.Validate(); err != nil {
		return fmt.Errorf("%w: invalid return warehouse address", err)
	}
	return c.Parcel.Validate()
}

type ReturnShipment struct {
	ID                      uuid.UUID `json:"return_shipment_id"`
	ShipmentNumber          string    `json:"shipment_number"`
	ReturnCaseID            string    `json:"return_case_id"`
	OrderID                 string    `json:"order_id"`
	PackageID               string    `json:"package_id"`
	CarrierCode             string    `json:"carrier_code"`
	ServiceCode             string    `json:"service_code"`
	CarrierOrderID          string    `json:"carrier_order_id,omitempty"`
	TrackingCode            string    `json:"tracking_code,omitempty"`
	Status                  string    `json:"status"`
	PickupAddress           Recipient `json:"pickup_address"`
	Destination             Recipient `json:"destination"`
	Parcel                  Parcel    `json:"parcel"`
	ShippingNotes           string    `json:"shipping_notes,omitempty"`
	CODAmountVND            int64     `json:"cod_amount_vnd"`
	FeePayer                string    `json:"fee_payer"`
	ProviderEstimatedFeeVND int64     `json:"provider_estimated_fee_vnd"`
	ReasonCode              string    `json:"reason_code"`
	FailureCode             string    `json:"failure_code,omitempty"`
	FailureReason           string    `json:"failure_reason,omitempty"`
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`
	Revision                int64     `json:"revision"`
}

func NewReturnShipment(cmd CreateReturnShipmentCommand, now time.Time) (*ReturnShipment, error) {
	if err := cmd.Validate(); err != nil {
		return nil, err
	}
	id := uuid.New()
	now = now.UTC()
	return &ReturnShipment{
		ID: id, ShipmentNumber: "RTN-" + strings.ToUpper(strings.ReplaceAll(id.String()[:13], "-", "")),
		ReturnCaseID: strings.TrimSpace(cmd.ReturnCaseID), OrderID: strings.TrimSpace(cmd.OrderID),
		PackageID: strings.TrimSpace(cmd.PackageID), CarrierCode: strings.ToUpper(strings.TrimSpace(cmd.CarrierCode)),
		ServiceCode: strings.ToUpper(strings.TrimSpace(cmd.ServiceCode)), Status: ReturnShipmentCreating,
		PickupAddress: cmd.PickupAddress, Destination: cmd.Destination, Parcel: cmd.Parcel,
		ShippingNotes: strings.TrimSpace(cmd.ShippingNotes),
		CODAmountVND:  0, FeePayer: ReturnShipmentFeePayerShop, ReasonCode: strings.ToUpper(strings.TrimSpace(cmd.ReasonCode)),
		CreatedAt: now, UpdatedAt: now, Revision: 1,
	}, nil
}

func (s *ReturnShipment) MarkCreated(carrierOrderID, trackingCode string, feeVND int64, now time.Time) error {
	if s == nil || strings.TrimSpace(carrierOrderID) == "" || strings.TrimSpace(trackingCode) == "" || feeVND < 0 {
		return fmt.Errorf("%w: carrier order, tracking code and non-negative fee are required", ErrInvalidShipment)
	}
	if s.Status != ReturnShipmentCreating && s.Status != ReturnShipmentCreationFailed {
		return fmt.Errorf("%w: cannot create return shipment from status %s", ErrInvalidTransition, s.Status)
	}
	s.CarrierOrderID = strings.TrimSpace(carrierOrderID)
	s.TrackingCode = strings.TrimSpace(trackingCode)
	s.ProviderEstimatedFeeVND = feeVND
	s.Status = ReturnShipmentCreated
	s.FailureCode = ""
	s.FailureReason = ""
	s.UpdatedAt = now.UTC()
	s.Revision++
	return nil
}

func (s *ReturnShipment) MarkCreationFailed(code, reason string, now time.Time) error {
	if s == nil || (s.Status != ReturnShipmentCreating && s.Status != ReturnShipmentCreationFailed) {
		return fmt.Errorf("%w: cannot fail return shipment creation", ErrInvalidTransition)
	}
	s.Status = ReturnShipmentCreationFailed
	s.FailureCode = strings.TrimSpace(code)
	s.FailureReason = strings.TrimSpace(reason)
	s.UpdatedAt = now.UTC()
	s.Revision++
	return nil
}

type ReturnShipmentEvent struct {
	SpecVersion     string                  `json:"specversion"`
	ID              uuid.UUID               `json:"id"`
	Source          string                  `json:"source"`
	Type            string                  `json:"type"`
	Subject         string                  `json:"subject"`
	Time            time.Time               `json:"time"`
	DataContentType string                  `json:"datacontenttype"`
	TraceParent     string                  `json:"traceparent"`
	Data            ReturnShipmentEventData `json:"data"`
	AggregateID     uuid.UUID               `json:"-"`
	OccurredAt      time.Time               `json:"-"`
	Topic           string                  `json:"-"`
}

type ReturnShipmentEventData struct {
	ReturnShipmentID uuid.UUID `json:"return_shipment_id"`
	ReturnCaseID     string    `json:"return_case_id"`
	OrderID          string    `json:"order_id"`
	PackageID        string    `json:"package_id"`
	CarrierCode      string    `json:"carrier_code"`
	TrackingCode     string    `json:"tracking_code,omitempty"`
	Status           string    `json:"status"`
	FeePayer         string    `json:"fee_payer"`
	CODAmountVND     int64     `json:"cod_amount_vnd"`
	ResourceVersion  int64     `json:"resource_version"`
}

func NewReturnShipmentEvent(name string, shipment *ReturnShipment, now time.Time) ReturnShipmentEvent {
	now = now.UTC()
	traceID := strings.ReplaceAll(uuid.NewString(), "-", "")
	spanID := strings.ReplaceAll(uuid.NewString(), "-", "")[:16]
	return ReturnShipmentEvent{
		SpecVersion: "1.0", ID: uuid.New(), Source: "https://omama.vn/services/shipping-service",
		Type:    "vn.omama.shipping.return-shipment." + name + ".v1",
		Subject: "return_shipment_id:" + shipment.ID.String(), Time: now,
		DataContentType: "application/json", TraceParent: "00-" + traceID + "-" + spanID + "-01",
		AggregateID: shipment.ID, OccurredAt: now, Topic: "shipping.return.events.v1",
		Data: ReturnShipmentEventData{
			ReturnShipmentID: shipment.ID, ReturnCaseID: shipment.ReturnCaseID, OrderID: shipment.OrderID,
			PackageID: shipment.PackageID, CarrierCode: shipment.CarrierCode, TrackingCode: shipment.TrackingCode,
			Status: shipment.Status, FeePayer: shipment.FeePayer, CODAmountVND: shipment.CODAmountVND,
			ResourceVersion: shipment.Revision,
		},
	}
}
