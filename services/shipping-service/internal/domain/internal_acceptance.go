package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

var ErrDriverNotAssigned = fmt.Errorf("driver is not assigned to shipment")

// ActorID must come from an authenticated employee principal, never a request body.
type AcceptInternalDelivery struct {
	ShipmentID      uuid.UUID `json:"shipment_id"`
	ActorID         uuid.UUID `json:"actor_id"`
	ExpectedVersion int64     `json:"expected_version"`
	IdempotencyKey  string    `json:"-"`
}

func (c AcceptInternalDelivery) Validate() error {
	// Reuse the assignment key/UUID/version constraints without trusting a driver ID.
	a := AssignInternalDelivery{ShipmentID: c.ShipmentID, DriverID: c.ActorID, ActorID: c.ActorID, ExpectedVersion: c.ExpectedVersion, Reason: "Accept assigned delivery", IdempotencyKey: c.IdempotencyKey}
	if err := a.Validate(); err != nil {
		return err
	}
	if c.ExpectedVersion < 1 {
		return fmt.Errorf("%w: assignment version must be positive", ErrInvalidShipment)
	}
	return nil
}

func (c AcceptInternalDelivery) RequestHash() string {
	return (AssignInternalDelivery{ShipmentID: c.ShipmentID, DriverID: c.ActorID, ActorID: c.ActorID, ExpectedVersion: c.ExpectedVersion, Reason: "Accept assigned delivery"}).RequestHash()
}

type InternalDeliveryAcceptance struct {
	ShipmentID        uuid.UUID `json:"shipment_id"`
	DriverID          uuid.UUID `json:"driver_id"`
	AssignmentVersion int64     `json:"assignment_version"`
	AcceptedAt        time.Time `json:"accepted_at"`
}
