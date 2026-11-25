package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
)

type ProgressInternalDelivery struct {
	ShipmentID        uuid.UUID      `json:"shipment_id"`
	ActorID           uuid.UUID      `json:"actor_id"`
	AssignmentVersion int64          `json:"assignment_version"`
	ExpectedRevision  int64          `json:"expected_revision"`
	Status            ShipmentStatus `json:"status"`
	Reason            string         `json:"reason"`
	IdempotencyKey    string         `json:"-"`
}

func (c ProgressInternalDelivery) Validate() error {
	a := AssignInternalDelivery{ShipmentID: c.ShipmentID, DriverID: c.ActorID, ActorID: c.ActorID, ExpectedVersion: c.AssignmentVersion, Reason: c.Reason, IdempotencyKey: c.IdempotencyKey}
	if err := a.Validate(); err != nil {
		return err
	}
	if c.AssignmentVersion < 1 || c.ExpectedRevision < 1 || c.ExpectedRevision == math.MaxInt64 {
		return fmt.Errorf("%w: positive assignment version and shipment revision required", ErrInvalidShipment)
	}
	switch c.Status {
	case StatusInTransit, StatusOutForDelivery, StatusDelivered, StatusDeliveryFailed, StatusReturning, StatusReturned:
		return nil
	}
	return fmt.Errorf("%w: unsupported driver status", ErrInvalidShipment)
}

func (c ProgressInternalDelivery) RequestHash() string {
	data, _ := json.Marshal(c)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// Driver actions cannot manufacture warehouse handover or skip delivery stages.
func ValidateInternalProgress(from, to ShipmentStatus) error {
	allowed := map[ShipmentStatus][]ShipmentStatus{
		StatusHandedOver:     {StatusInTransit},
		StatusInTransit:      {StatusOutForDelivery},
		StatusOutForDelivery: {StatusDelivered, StatusDeliveryFailed},
		StatusDeliveryFailed: {StatusOutForDelivery, StatusReturning},
		StatusReturning:      {StatusReturned},
	}
	for _, target := range allowed[from] {
		if to == target {
			return nil
		}
	}
	return fmt.Errorf("%w: driver cannot change %s to %s", ErrInvalidTransition, from, to)
}

func InternalProgressEventName(status ShipmentStatus) string {
	switch status {
	case StatusDelivered:
		return "delivered"
	case StatusDeliveryFailed:
		return "delivery-failed"
	case StatusReturning:
		return "returning"
	case StatusReturned:
		return "returned"
	default:
		return "status-changed"
	}
}

type InternalDeliveryProgressReceipt struct {
	ShipmentID        uuid.UUID      `json:"shipment_id"`
	ActorID           uuid.UUID      `json:"actor_id"`
	AssignmentVersion int64          `json:"assignment_version"`
	Revision          int64          `json:"revision"`
	Status            ShipmentStatus `json:"status"`
	EventID           uuid.UUID      `json:"event_id"`
	OccurredAt        time.Time      `json:"occurred_at"`
}
