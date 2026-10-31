package domain

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

const ShippingV2ContractVersion = "order-3.2"

var (
	ErrQuoteNotFound       = errors.New("shipping quote not found")
	ErrIdempotencyConflict = errors.New("idempotency key was already used with a different request")
	requestHashPattern     = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

type BoundQuoteCommand struct {
	IdempotencyKey    string
	ContractVersion   string
	RequestHash       string
	CanonicalHash     string
	Destination       Recipient
	Parcel            Parcel
	PreferredCarrier  string
	ServiceCode       string
	OriginWarehouseID string
}

func (c BoundQuoteCommand) Validate() error {
	if c.ContractVersion != ShippingV2ContractVersion {
		return fmt.Errorf("%w: contract_version must be %s", ErrInvalidShipment, ShippingV2ContractVersion)
	}
	if !validIdempotencyKey(c.IdempotencyKey) {
		return fmt.Errorf("%w: idempotency_key must contain 16-128 printable ASCII characters without spaces", ErrInvalidShipment)
	}
	if !requestHashPattern.MatchString(c.RequestHash) || c.RequestHash != c.CanonicalHash {
		return fmt.Errorf("%w: request_hash does not match the canonical business request", ErrInvalidShipment)
	}
	if err := c.Destination.Validate(); err != nil {
		return err
	}
	return c.Parcel.Validate()
}

func validIdempotencyKey(value string) bool {
	if len(value) < 16 || len(value) > 128 {
		return false
	}
	for _, character := range []byte(value) {
		if character < 0x21 || character > 0x7e {
			return false
		}
	}
	return true
}

type ShippingQuote struct {
	ID                  uuid.UUID
	IdempotencyKey      string
	ContractVersion     string
	RequestHash         string
	Destination         Recipient
	Parcel              Parcel
	PreferredCarrier    string
	RequestedService    string
	OriginWarehouseID   string
	CarrierCode         string
	ServiceCode         string
	ServiceName         string
	FeeVND              int64
	EstimatedDeliveryAt time.Time
	ExpiresAt           time.Time
	CreatedAt           time.Time
	Version             int64
}

func NewShippingQuote(command BoundQuoteCommand, carrierCode, serviceCode, serviceName string, feeVND int64, estimatedDeliveryAt, expiresAt, now time.Time) (*ShippingQuote, error) {
	if err := command.Validate(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(carrierCode) == "" || strings.TrimSpace(serviceCode) == "" || feeVND < 0 {
		return nil, fmt.Errorf("%w: carrier result is incomplete", ErrInvalidShipment)
	}
	now = now.UTC()
	if !expiresAt.After(now) || estimatedDeliveryAt.IsZero() {
		return nil, fmt.Errorf("%w: carrier quote expiry and delivery time are invalid", ErrInvalidShipment)
	}
	return &ShippingQuote{
		ID: uuid.New(), IdempotencyKey: command.IdempotencyKey,
		ContractVersion: command.ContractVersion, RequestHash: command.RequestHash,
		Destination: command.Destination, Parcel: command.Parcel,
		PreferredCarrier: command.PreferredCarrier, RequestedService: command.ServiceCode,
		OriginWarehouseID: command.OriginWarehouseID, CarrierCode: carrierCode,
		ServiceCode: serviceCode, ServiceName: serviceName, FeeVND: feeVND,
		EstimatedDeliveryAt: estimatedDeliveryAt.UTC(), ExpiresAt: expiresAt.UTC(), CreatedAt: now,
	}, nil
}
