package core

import (
	"errors"
	"github.com/google/uuid"
	"reflect"
	"testing"
	"time"
)

func TestFulfillmentProofChainAndCompletion(t *testing.T) {
	a, stock := allocated(t)
	var e error
	a, e = a.ConfirmPaid(contextFor(a, stock.ExpiresAt), stock)
	if e != nil {
		t.Fatal(e)
	}
	at := a.Record().DecidedAt.Add(time.Second)
	task, shipment := uuid.NewString(), uuid.NewString()
	for _, step := range []struct {
		kind, resource, seal string
		version              int64
	}{{"PACKING_ACCEPTED", task, "", 1}, {"PACKING_SEALED", task, "test-seal", 2}, {"SHIPMENT_CREATED", shipment, "", 1}, {"SHIPMENT_DISPATCHED", shipment, "", 2}, {"SHIPMENT_DELIVERED", shipment, "", 3}} {
		before := a.Record()
		bad := FulfillmentProof{OrderID: uuid.NewString(), ResourceID: step.resource, Kind: step.kind, Version: step.version, Generation: 1, At: at, Seal: step.seal}
		if _, e = a.ApplyFulfillment(contextFor(a, at), bad); !errors.Is(e, ErrProof) || !reflect.DeepEqual(a.Record(), before) {
			t.Fatal("foreign proof mutated root", e)
		}
		proof := bad
		proof.OrderID = a.ID()
		a, e = a.ApplyFulfillment(contextFor(a, at), proof)
		if e != nil {
			t.Fatal(step.kind, e)
		}
		if _, e = Restore(a.Snapshot(), a.Record()); e != nil {
			t.Fatal("restore", step.kind, e)
		}
		at = at.Add(time.Second)
	}
	eligibility := CompletionEligibility{State: Delivered, OrderID: a.ID(), CareOrderID: a.ID(), CareDecisionID: uuid.NewString(), CareAllowed: true, SettlementVerified: true, DeliveredAt: a.Record().Fulfillment.DeliveredAt, WindowEndsAt: a.Record().Fulfillment.DeliveredAt.Add(7 * 24 * time.Hour)}
	eligibility.At = eligibility.WindowEndsAt
	blocked, e := a.Block(contextFor(a, at), HoldBlock)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = blocked.Complete(contextFor(blocked, eligibility.At), eligibility); !errors.Is(e, ErrBlocked) {
		t.Fatal("hold bypass", e)
	}
	eligibility.DeliveredAt = eligibility.DeliveredAt.Add(-time.Second)
	if _, e = a.Complete(contextFor(a, eligibility.At), eligibility); !errors.Is(e, ErrProof) {
		t.Fatal("delivery timestamp changed", e)
	}
	eligibility.DeliveredAt = a.Record().Fulfillment.DeliveredAt
	done, e := a.Complete(contextFor(a, eligibility.At), eligibility)
	if e != nil || done.State() != Completed {
		t.Fatal(done.State(), e)
	}
	if _, e = Restore(done.Snapshot(), done.Record()); e != nil {
		t.Fatal(e)
	}
}
