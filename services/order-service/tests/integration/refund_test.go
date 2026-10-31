package integration

import (
	"sync"
	"testing"
)

func TestV12ConcurrentRefundBudgetAndUnknownRecovery(t *testing.T) {
	h := newHarness(t)
	o := h.paid()
	receipt := h.receiptID(o.ID)
	a := h.approval(o, receipt, 150000)
	b := h.approval(o, receipt, 150000)
	start := make(chan struct{})
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for _, id := range []string{a.ID, b.ID} {
		wg.Add(1)
		go func(approval string) { defer wg.Done(); <-start; _, e := h.S.AcceptRefund(h.ctx, approval); errs <- e }(id)
	}
	close(start)
	wg.Wait()
	close(errs)
	success := 0
	for e := range errs {
		if e == nil {
			success++
		} else {
			requireCode(t, e, "REFUND_BUDGET_EXCEEDED")
		}
	}
	if success != 1 {
		t.Fatalf("approved obligations %d", success)
	}
	if e := h.Sim.SetFault(h.ctx, "refund", "lost_ack", 1); e != nil {
		t.Fatal(e)
	}
	if _, e := h.S.WorkJob(h.ctx); e != nil {
		t.Fatal(e)
	}
	if h.count("SELECT count(*) FROM refunds WHERE status='UNKNOWN'") != 1 || h.mockCount("SELECT count(*) FROM provider_refunds WHERE status='SUCCEEDED'") != 1 {
		t.Fatal("lost ACK did not preserve independent financial truth")
	}
	extra := h.approval(o, receipt, 150000)
	_, e := h.S.AcceptRefund(h.ctx, extra.ID)
	requireCode(t, e, "REFUND_BUDGET_EXCEEDED")
	h.jobs()
	if h.order(o.ID).Payment != "PARTIALLY_REFUNDED" || h.mockCount("SELECT count(*) FROM provider_refunds") != 1 {
		t.Fatal("refund query repeated financial effect")
	}
	if h.order(o.ID).Status != "PAID" || h.mockCount("SELECT physical FROM stock WHERE sku='MX-GION-500G'") != 98 {
		t.Fatal("refund changed fulfillment or restocked")
	}
}
func TestV12FullRefundAndReplay(t *testing.T) {
	h := newHarness(t)
	o := h.paid()
	a := h.approval(o, h.receiptID(o.ID), o.Final)
	id, e := h.S.AcceptRefund(h.ctx, a.ID)
	if e != nil {
		t.Fatal(e)
	}
	again, e := h.S.AcceptRefund(h.ctx, a.ID)
	if e != nil || again != id {
		t.Fatal("approval replay not stable")
	}
	h.jobs()
	if h.order(o.ID).Payment != "REFUNDED" || h.mockCount("SELECT count(*) FROM provider_refunds") != 1 {
		t.Fatal("full refund status wrong")
	}
	if h.count("SELECT count(*) FROM outbox WHERE event_type='vn.omama.order.refund_succeeded.v2'") != 1 {
		t.Fatal("duplicate refund fact")
	}
}
func TestV12FailedRefundDoesNotRestartFulfillment(t *testing.T) {
	h := newHarness(t)
	o := h.paid()
	a := h.approval(o, h.receiptID(o.ID), o.Final)
	if e := h.Sim.SetFault(h.ctx, "refund_definitive", "failed", 1); e != nil {
		t.Fatal(e)
	}
	if _, e := h.S.AcceptRefund(h.ctx, a.ID); e != nil {
		t.Fatal(e)
	}
	h.jobs()
	if h.count("SELECT count(*) FROM refunds WHERE status='FAILED'") != 1 || h.order(o.ID).Status != "PAID" || h.order(o.ID).Payment != "CONFIRMED" {
		t.Fatal("definitive failure corrupted order state")
	}
}
func TestV12CareCaseCompletionBarrier(t *testing.T) {
	h := newHarness(t)
	o := h.paid()
	h.acceptTask(o)
	h.apply(h.source(o, "PACKING_ACCEPTED"))
	h.apply(h.source(o, "PACKING_SEALED"))
	h.apply(h.source(o, "SHIPMENT_CREATED"))
	h.apply(h.source(o, "SHIPMENT_DISPATCHED"))
	h.apply(h.source(o, "SHIPMENT_DELIVERED"))
	h.exec("UPDATE orders SET completion_due_at=clock_timestamp()-interval '1 second' WHERE id=$1", o.ID)
	opened := h.source(o, "CASE_OPENED")
	if e := h.S.CompleteOrders(h.ctx); e != nil {
		t.Fatal(e)
	}
	if h.order(o.ID).Status != "DELIVERED" {
		t.Fatal("completion ignored case at Care with event lag")
	}
	h.apply(opened)
	closed := h.source(o, "CASE_CLOSED")
	h.apply(closed)
	if e := h.S.CompleteOrders(h.ctx); e != nil {
		t.Fatal(e)
	}
	if e := h.S.CompleteOrders(h.ctx); e != nil {
		t.Fatal(e)
	}
	if h.order(o.ID).Status != "COMPLETED" || h.count("SELECT count(*) FROM outbox WHERE event_type='vn.omama.order.completed.v2'") != 1 {
		t.Fatal("completion barrier/once failed")
	}
	if h.request(h.Mock.URL, "POST", "/test/event", "", map[string]any{"order_id": o.ID, "type": "CASE_OPENED"}, nil) != 409 {
		t.Fatal("Care did not enforce completion decision barrier")
	}
}
