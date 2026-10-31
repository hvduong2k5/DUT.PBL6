package integration

import (
	"github.com/google/uuid"
	"github.com/omamx/order-service/internal/domain"
	"testing"
	"time"
)

func TestV10FencedWorkerCannotOverwriteNewLease(t *testing.T) {
	h := newHarness(t)
	q := h.quote("VIETQR")
	op, _, e := h.accept(q, uuid.NewString())
	if e != nil {
		t.Fatal(e)
	}
	if _, e = h.S.WorkPlacement(h.ctx); e != nil {
		t.Fatal(e)
	}
	if e = h.Sim.SetFault(h.ctx, "reserve", "delay_ack", 1); e != nil {
		t.Fatal(e)
	}
	done := make(chan error, 1)
	go func() { _, err := h.S.WorkPlacement(h.ctx); done <- err }()
	deadline := time.Now().Add(5 * time.Second)
	for h.mockCount("SELECT count(*) FROM reservations") == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if h.mockCount("SELECT count(*) FROM reservations") != 1 {
		t.Fatal("reserve fault did not commit")
	}
	h.exec("UPDATE checkout_operations SET lease_owner='replacement-worker',lease_epoch=lease_epoch+1,lease_until=NULL WHERE id=$1", op.ID)
	settled := h.driveOperation(op.ID)
	if settled.Status != "SUCCEEDED" {
		t.Fatalf("takeover failed %+v", settled)
	}
	if e = <-done; e != nil {
		t.Fatal(e)
	}
	after, e := h.S.Operation(h.ctx, op.ID, h.Principal)
	if e != nil || after.Status != "SUCCEEDED" {
		t.Fatal("stale worker overwrote terminal operation")
	}
	if h.mockCount("SELECT available FROM stock WHERE sku='MX-GION-500G'") != 98 {
		t.Fatal("takeover caused duplicate reserve")
	}
}
func TestV10OrderHistoryOutboxCommitIsAtomic(t *testing.T) {
	h := newHarness(t)
	q := h.quote("VIETQR")
	op, _, e := h.accept(q, uuid.NewString())
	if e != nil {
		t.Fatal(e)
	}
	for i := 0; i < 2; i++ {
		if _, e = h.S.WorkPlacement(h.ctx); e != nil {
			t.Fatal(e)
		}
	}
	h.exec(`CREATE FUNCTION fail_history() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected history failure'; END $$`)
	h.exec(`CREATE TRIGGER fail_history BEFORE INSERT ON order_history FOR EACH ROW EXECUTE FUNCTION fail_history()`)
	if _, e = h.S.WorkPlacement(h.ctx); e == nil {
		t.Fatal("transaction fault not injected")
	}
	if h.count("SELECT count(*) FROM orders") != 0 || h.count("SELECT count(*) FROM outbox") != 0 {
		t.Fatal("Order/outbox partially committed")
	}
	if h.mockCount("SELECT count(*) FROM reservations WHERE state='RESERVED'") != 1 {
		t.Fatal("durable reservation lost")
	}
	h.exec("DROP TRIGGER fail_history ON order_history")
	settled := h.driveOperation(op.ID)
	if settled.Status != "SUCCEEDED" || h.count("SELECT count(*) FROM orders") != 1 {
		t.Fatal("placement failed to recover from rollback")
	}
}
func TestV12ApprovedPaidCancellationKeepsRefundIndependent(t *testing.T) {
	h := newHarness(t)
	o := h.paid()
	a := domain.Approval{ID: uuid.NewString(), OrderID: o.ID, Type: "CANCEL", State: "APPROVED", Actor: "SALES_MANAGER"}
	if h.request(h.Mock.URL, "POST", "/test/approval", "", a, nil) != 200 {
		t.Fatal("approval fixture failed")
	}
	if e := h.S.ApproveCancellation(h.ctx, a.ID); e != nil {
		t.Fatal(e)
	}
	before := h.order(o.ID).Version
	if e := h.S.ApproveCancellation(h.ctx, a.ID); e != nil || h.order(o.ID).Version != before {
		t.Fatal("approved cancellation replay mutated twice")
	}
	h.jobs()
	current := h.order(o.ID)
	if current.Status != "CANCELLED_BY_ADMIN" || current.Payment != "CONFIRMED" || current.Stock != "RELEASED" {
		t.Fatalf("cancelled paid order %+v", current)
	}
	if h.mockCount("SELECT count(*) FROM provider_refunds") != 0 || h.count("SELECT count(*) FROM refunds") != 0 {
		t.Fatal("cancellation auto-refunded without approval")
	}
}
