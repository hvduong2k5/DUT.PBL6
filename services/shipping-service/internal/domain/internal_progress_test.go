package domain

import (
	"github.com/google/uuid"
	"testing"
)

func TestInternalProgressPolicy(t *testing.T) {
	valid := map[ShipmentStatus][]ShipmentStatus{StatusHandedOver: {StatusInTransit}, StatusInTransit: {StatusOutForDelivery}, StatusOutForDelivery: {StatusDelivered, StatusDeliveryFailed}, StatusDeliveryFailed: {StatusOutForDelivery, StatusReturning}, StatusReturning: {StatusReturned}}
	states := []ShipmentStatus{StatusCreated, StatusReadyForHandover, StatusHandedOver, StatusInTransit, StatusOutForDelivery, StatusDelivered, StatusDeliveryFailed, StatusReturning, StatusReturned, StatusCancelled}
	for _, from := range states {
		for _, to := range states {
			want := false
			for _, v := range valid[from] {
				want = want || v == to
			}
			if (ValidateInternalProgress(from, to) == nil) != want {
				t.Fatalf("transition %s -> %s", from, to)
			}
		}
	}
	c := ProgressInternalDelivery{ShipmentID: uuid.New(), ActorID: uuid.New(), AssignmentVersion: 1, ExpectedRevision: 3, Status: StatusInTransit, Reason: "Started route", IdempotencyKey: "progress-key-0001"}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*ProgressInternalDelivery){func(x *ProgressInternalDelivery) { x.ActorID = uuid.Nil }, func(x *ProgressInternalDelivery) { x.AssignmentVersion = 0 }, func(x *ProgressInternalDelivery) { x.ExpectedRevision = 0 }, func(x *ProgressInternalDelivery) { x.Status = StatusHandedOver }, func(x *ProgressInternalDelivery) { x.Reason = "" }} {
		bad := c
		change(&bad)
		if bad.Validate() == nil {
			t.Fatalf("invalid command accepted: %+v", bad)
		}
		if bad.RequestHash() == c.RequestHash() {
			t.Fatal("payload not bound")
		}
	}
	other := c
	other.IdempotencyKey = "other-progress-key"
	if other.RequestHash() != c.RequestHash() {
		t.Fatal("key in hash")
	}
}
