package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
)

var ErrAssignmentNotFound = errors.New("internal delivery assignment not found")

// Employee IDs are Identity principal UUIDs, not role names or Keycloak subjects.
// The future authenticated transport must supply ActorID and validate DriverID
// against the authoritative employee directory before invoking this command.
type AssignInternalDelivery struct {
	ShipmentID      uuid.UUID `json:"shipment_id"`
	DriverID        uuid.UUID `json:"driver_id"`
	ActorID         uuid.UUID `json:"actor_id"`
	ExpectedVersion int64     `json:"expected_version"`
	Reason          string    `json:"reason"`
	IdempotencyKey  string    `json:"-"`
}

func (c AssignInternalDelivery) Validate() error {
	if c.ShipmentID == uuid.Nil || c.DriverID == uuid.Nil || c.ActorID == uuid.Nil || c.ExpectedVersion < 0 || c.ExpectedVersion == math.MaxInt64 {
		return fmt.Errorf("%w: shipment, driver, actor UUIDs and a valid assignment version are required", ErrInvalidShipment)
	}
	if !utf8.ValidString(c.IdempotencyKey) || len(c.IdempotencyKey) < 8 || len(c.IdempotencyKey) > 128 || strings.TrimSpace(c.IdempotencyKey) != c.IdempotencyKey || strings.ContainsFunc(c.IdempotencyKey, unicode.IsControl) {
		return fmt.Errorf("%w: invalid assignment idempotency key", ErrInvalidShipment)
	}
	if !utf8.ValidString(c.Reason) || strings.TrimSpace(c.Reason) == "" || utf8.RuneCountInString(c.Reason) > 500 || strings.ContainsFunc(c.Reason, unicode.IsControl) {
		return fmt.Errorf("%w: assignment reason must contain 1..500 characters without controls", ErrInvalidShipment)
	}
	return nil
}

func (c AssignInternalDelivery) RequestHash() string {
	data, _ := json.Marshal(c) // Fixed scalar fields only; IdempotencyKey is excluded.
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

type InternalDeliveryAssignment struct {
	ShipmentID uuid.UUID `json:"shipment_id"`
	DriverID   uuid.UUID `json:"driver_id"`
	ActorID    uuid.UUID `json:"assigned_by"`
	Version    int64     `json:"version"`
	Reason     string    `json:"reason"`
	AssignedAt time.Time `json:"assigned_at"`
}

type InternalDeliveryAssignmentEvent struct {
	InternalDeliveryAssignment
	ID               uuid.UUID  `json:"event_id"`
	PreviousDriverID *uuid.UUID `json:"previous_driver_id"`
}

func ValidateInternalAssignmentState(carrierCode string, status ShipmentStatus, codAmount int64) error {
	if carrierCode != "INTERNAL" || codAmount != 0 {
		return fmt.Errorf("%w: only prepaid INTERNAL shipments can be assigned", ErrInvalidShipment)
	}
	if status != StatusCreated && status != StatusReadyForHandover {
		return fmt.Errorf("%w: assignment requires a shipment before handover", ErrInvalidTransition)
	}
	return nil
}
