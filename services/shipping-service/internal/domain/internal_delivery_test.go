package domain

import (
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func assignmentCommand() AssignInternalDelivery {
	return AssignInternalDelivery{ShipmentID: uuid.New(), DriverID: uuid.New(), ActorID: uuid.New(), Reason: "Initial dispatch", IdempotencyKey: "assignment-0001"}
}

func TestInternalAssignmentValidation(t *testing.T) {
	if err := assignmentCommand().Validate(); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		mutate func(*AssignInternalDelivery)
	}{
		{"shipment", func(c *AssignInternalDelivery) { c.ShipmentID = uuid.Nil }},
		{"driver", func(c *AssignInternalDelivery) { c.DriverID = uuid.Nil }},
		{"actor", func(c *AssignInternalDelivery) { c.ActorID = uuid.Nil }},
		{"version", func(c *AssignInternalDelivery) { c.ExpectedVersion = -1 }},
		{"overflow", func(c *AssignInternalDelivery) { c.ExpectedVersion = math.MaxInt64 }},
		{"empty reason", func(c *AssignInternalDelivery) { c.Reason = "  " }},
		{"long reason", func(c *AssignInternalDelivery) { c.Reason = strings.Repeat("x", 501) }},
		{"control reason", func(c *AssignInternalDelivery) { c.Reason = "bad\nreason" }},
		{"short key", func(c *AssignInternalDelivery) { c.IdempotencyKey = "abc" }},
		{"long key", func(c *AssignInternalDelivery) { c.IdempotencyKey = strings.Repeat("x", 129) }},
		{"padded key", func(c *AssignInternalDelivery) { c.IdempotencyKey = " assignment-key" }},
		{"control key", func(c *AssignInternalDelivery) { c.IdempotencyKey = "bad-key\n" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := assignmentCommand()
			tc.mutate(&c)
			if !errors.Is(c.Validate(), ErrInvalidShipment) {
				t.Fatal("invalid assignment accepted")
			}
		})
	}
}

func TestAssignmentRequestHashBindsActorTargetVersionAndReason(t *testing.T) {
	c := assignmentCommand()
	base := c.RequestHash()
	c.IdempotencyKey = "different-action-key"
	if c.RequestHash() != base {
		t.Fatal("key must not be payload")
	}
	for _, mutate := range []func(*AssignInternalDelivery){
		func(c *AssignInternalDelivery) { c.DriverID = uuid.New() }, func(c *AssignInternalDelivery) { c.ShipmentID = uuid.New() }, func(c *AssignInternalDelivery) { c.ActorID = uuid.New() }, func(c *AssignInternalDelivery) { c.ExpectedVersion++ }, func(c *AssignInternalDelivery) { c.Reason = "Reassignment" },
	} {
		changed := c
		mutate(&changed)
		if changed.RequestHash() == base {
			t.Fatal("request hash omitted assignment field")
		}
	}
}

func TestAssignmentOnlyPreHandoverInternalPrepaidShipments(t *testing.T) {
	for _, state := range []ShipmentStatus{StatusCreated, StatusReadyForHandover} {
		if err := ValidateInternalAssignmentState("INTERNAL", state, 0); err != nil {
			t.Fatal(err)
		}
	}
	for _, code := range []string{"GHN", "FAKE", "internal", ""} {
		if err := ValidateInternalAssignmentState(code, StatusCreated, 0); !errors.Is(err, ErrInvalidShipment) {
			t.Fatal("other carrier accepted")
		}
	}
	if err := ValidateInternalAssignmentState("INTERNAL", StatusCreated, 100); !errors.Is(err, ErrInvalidShipment) {
		t.Fatal("COD accepted")
	}
	for _, state := range []ShipmentStatus{StatusCreating, StatusCreationFailed, StatusHandedOver, StatusInTransit, StatusOutForDelivery, StatusDelivered, StatusDeliveryFailed, StatusReturning, StatusReturned, StatusCancelled} {
		if err := ValidateInternalAssignmentState("INTERNAL", state, 0); !errors.Is(err, ErrInvalidTransition) {
			t.Fatalf("state accepted: %s", state)
		}
	}
}
