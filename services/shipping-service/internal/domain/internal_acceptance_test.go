package domain

import (
	"github.com/google/uuid"
	"testing"
)

func TestAcceptInternalDeliveryValidationAndHash(t *testing.T) {
	c := AcceptInternalDelivery{ShipmentID: uuid.New(), ActorID: uuid.New(), ExpectedVersion: 1, IdempotencyKey: "accept-key-123456"}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*AcceptInternalDelivery){
		func(x *AcceptInternalDelivery) { x.ShipmentID = uuid.Nil },
		func(x *AcceptInternalDelivery) { x.ActorID = uuid.Nil },
		func(x *AcceptInternalDelivery) { x.ExpectedVersion = 0 },
		func(x *AcceptInternalDelivery) { x.ExpectedVersion = -1 },
		func(x *AcceptInternalDelivery) { x.IdempotencyKey = "short" },
	} {
		bad := c
		mutate(&bad)
		if bad.Validate() == nil {
			t.Fatalf("accepted invalid command: %+v", bad)
		}
	}
	other := c
	other.IdempotencyKey = "another-key-123456"
	if other.RequestHash() != c.RequestHash() {
		t.Fatal("key changes payload hash")
	}
	other = c
	other.ActorID = uuid.New()
	if other.RequestHash() == c.RequestHash() {
		t.Fatal("actor not bound")
	}
	other = c
	other.ExpectedVersion++
	if other.RequestHash() == c.RequestHash() {
		t.Fatal("version not bound")
	}
}
