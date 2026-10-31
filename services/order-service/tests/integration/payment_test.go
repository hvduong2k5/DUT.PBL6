package integration

import (
	"github.com/google/uuid"
	"github.com/omamx/order-service/internal/domain"
	"sync"
	"testing"
)

func TestV10PaymentLostCallbackAndFinalizeACK(t *testing.T) {
	h := newHarness(t)
	o := h.place("VIETQR")
	if e := h.Sim.SetFault(h.ctx, "finalize", "lost_ack", 1); e != nil {
		t.Fatal(e)
	}
	receipt := h.pay(o, o.Final)
	if h.count("SELECT count(*) FROM receipts") != 0 {
		t.Fatal("fixture unexpectedly delivered callback")
	}
	if e := h.S.ReconcilePayments(h.ctx); e != nil {
		t.Fatal(e)
	}
	if h.order(o.ID).Status != "PAYMENT_FINALIZING" {
		t.Fatal("paid before stock acknowledgement")
	}
	if h.count("SELECT count(*) FROM outbox WHERE event_type='vn.omama.order.paid.v2'") != 0 {
		t.Fatal("early paid event")
	}
	h.jobs()
	current := h.order(o.ID)
	if current.Status != "PAID" || current.Stock != "COMMITTED" || current.Payment != "CONFIRMED" {
		t.Fatal("paid invariant not reached")
	}
	if h.mockCount("SELECT physical FROM stock WHERE sku='MX-GION-500G'") != 98 || h.mockCount("SELECT available FROM stock WHERE sku='MX-GION-500G'") != 98 {
		t.Fatal("double stock deduction")
	}
	if e := h.S.RecordReceipt(h.ctx, receipt); e != nil {
		t.Fatal(e)
	}
	if h.count("SELECT count(*) FROM receipts") != 1 || h.count("SELECT count(*) FROM outbox WHERE event_type='vn.omama.order.paid.v2'") != 1 {
		t.Fatal("duplicate receipt/paid event")
	}
}
func TestV10ConcurrentReceiptAndAlteredPayload(t *testing.T) {
	h := newHarness(t)
	o := h.place("VIETQR")
	r := h.pay(o, o.Final)
	var wg sync.WaitGroup
	errs := make(chan error, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if e := h.S.RecordReceipt(h.ctx, r); e != nil {
				errs <- e
			}
		}()
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Error(e)
	}
	h.jobs()
	changed := r
	changed.Amount++
	requireCode(t, h.S.RecordReceipt(h.ctx, changed), "RECEIPT_CONFLICT")
	if h.count("SELECT count(*) FROM receipts") != 1 || h.count("SELECT count(*) FROM allocations") != 1 {
		t.Fatal("receipt was allocated twice")
	}
	if h.count("SELECT count(*) FROM audit_records WHERE action='RECEIPT_CONFLICT'") != 1 {
		t.Fatal("altered receipt not audited")
	}
}
func TestV10UnderOverUnmatchedAndLateReceipts(t *testing.T) {
	h := newHarness(t)
	o := h.place("VIETQR")
	for _, amount := range []int64{o.Final - 1, o.Final + 1} {
		r := h.pay(o, amount)
		if e := h.S.RecordReceipt(h.ctx, r); e != nil {
			t.Fatal(e)
		}
	}
	if h.order(o.ID).Status != "PENDING_PAYMENT" || h.count("SELECT count(*) FROM allocations") != 0 {
		t.Fatal("mismatched receipt auto-allocated")
	}
	r := h.pay(domain.Order{Reference: "UNMATCHED-SYNTHETIC"}, 42)
	if e := h.S.RecordReceipt(h.ctx, r); e != nil {
		t.Fatal(e)
	}
	if h.count("SELECT count(*) FROM receipts WHERE order_id IS NULL") != 1 || h.count("SELECT count(*) FROM outbox WHERE aggregate_type='PAYMENT'") != 3 {
		t.Fatal("unmatched money lost")
	}
	h.exec("UPDATE orders SET payment_expires_at=clock_timestamp()-interval '1 second' WHERE id=$1", o.ID)
	if e := h.S.SweepTimeouts(h.ctx); e != nil {
		t.Fatal(e)
	}
	h.jobs()
	r = h.pay(o, o.Final)
	if e := h.S.RecordReceipt(h.ctx, r); e != nil {
		t.Fatal(e)
	}
	if h.order(o.ID).Status != "CANCELLED_TIMEOUT" || h.count("SELECT count(*) FROM allocations") != 0 {
		t.Fatal("late receipt resurrected cancelled Order")
	}
}
func TestV10InventoryExpiryWinsFinalization(t *testing.T) {
	h := newHarness(t)
	o := h.place("VIETQR")
	r := h.pay(o, o.Final)
	if e := h.S.RecordReceipt(h.ctx, r); e != nil {
		t.Fatal(e)
	}
	h.mockExec("UPDATE reservations SET expires_at=clock_timestamp()-interval '1 second' WHERE order_id=$1", o.ID)
	h.jobs()
	current := h.order(o.ID)
	if current.Status != "CANCELLED_TIMEOUT" || current.Payment != "RECONCILIATION_REQUIRED" || current.Stock != "EXPIRED" {
		t.Fatalf("expiry outcome %+v", current)
	}
	if h.count("SELECT count(*) FROM outbox WHERE event_type IN ('vn.omama.order.paid.v2','vn.omama.order.fulfillment_ready.v2')") != 0 {
		t.Fatal("expiry produced paid/ready")
	}
	if h.mockCount("SELECT available FROM stock WHERE sku='MX-GION-500G'") != 100 {
		t.Fatal("expiry did not restore availability")
	}
}
func TestV10PaymentTimeoutRace(t *testing.T) {
	h := newHarness(t)
	o := h.place("VIETQR")
	r := h.pay(o, o.Final)
	h.exec("UPDATE orders SET payment_expires_at=clock_timestamp()-interval '1 microsecond' WHERE id=$1", o.ID)
	start := make(chan struct{})
	errs := make(chan error, 2)
	go func() { <-start; errs <- h.S.RecordReceipt(h.ctx, r) }()
	go func() { <-start; errs <- h.S.SweepTimeouts(h.ctx) }()
	close(start)
	for i := 0; i < 2; i++ {
		if e := <-errs; e != nil {
			t.Fatal(e)
		}
	}
	h.jobs()
	if h.order(o.ID).Status != "CANCELLED_TIMEOUT" || h.count("SELECT count(*) FROM allocations") != 0 {
		t.Fatal("expired local guard lost race")
	}
}
func TestV10CancelOwnershipCASAndReplay(t *testing.T) {
	h := newHarness(t)
	o := h.place("VIETQR")
	other := domain.Principal{ID: uuid.NewString(), Role: "CUSTOMER"}
	_, e := h.S.Cancel(h.ctx, other, o.ID, uuid.NewString(), o.Version, "test", false)
	requireCode(t, e, "ORDER_NOT_FOUND")
	_, e = h.S.Cancel(h.ctx, h.Principal, o.ID, uuid.NewString(), o.Version+1, "test", false)
	requireCode(t, e, "VERSION_CONFLICT")
	key := uuid.NewString()
	first, e := h.S.Cancel(h.ctx, h.Principal, o.ID, key, o.Version, "test", false)
	if e != nil {
		t.Fatal(e)
	}
	h.jobs()
	second, e := h.S.Cancel(h.ctx, h.Principal, o.ID, key, o.Version, "test", false)
	if e != nil || domain.Hash(first) != domain.Hash(second) {
		t.Fatal("cancel replay not stable")
	}
	if h.order(o.ID).Status != "CANCELLED_BY_USER" || h.mockCount("SELECT available FROM stock WHERE sku='MX-GION-500G'") != 100 {
		t.Fatal("cancel did not restore one reservation")
	}
}
func TestV10CODIsUnpaidAndCanCancel(t *testing.T) {
	h := newHarness(t)
	o := h.place("COD")
	h.jobs()
	o = h.order(o.ID)
	if o.Status != "CONFIRMED_COD" || o.Payment != "UNPAID" || o.Stock != "COMMITTED" || !o.Ready {
		t.Fatal("COD guard incorrect")
	}
	if h.count("SELECT count(*) FROM receipts") != 0 || h.count("SELECT count(*) FROM outbox WHERE event_type='vn.omama.order.paid.v2'") != 0 {
		t.Fatal("COD forged payment")
	}
	_, e := h.S.Cancel(h.ctx, h.Principal, o.ID, uuid.NewString(), o.Version, "test", false)
	if e != nil {
		t.Fatal(e)
	}
	h.jobs()
	if h.order(o.ID).Status != "CANCELLED_BY_USER" || h.mockCount("SELECT physical FROM stock WHERE sku='MX-GION-500G'") != 100 {
		t.Fatal("COD reversal incorrect")
	}
}
func TestV10SignedCallbackValidation(t *testing.T) {
	h := newHarness(t)
	o := h.place("VIETQR")
	r := h.pay(o, o.Final)
	if code := h.request(h.API.URL, "POST", "/api/v1/payments/vietqr/callback", "", r, nil); code != 401 {
		t.Fatalf("unsigned callback accepted: %d", code)
	}
	if h.count("SELECT count(*) FROM receipts") != 0 {
		t.Fatal("invalid signature wrote receipt")
	}
	if e := h.Sim.Callback(h.ctx, r); e != nil {
		t.Fatal(e)
	}
	h.jobs()
	if h.order(o.ID).Status != "PAID" {
		t.Fatal("valid simulator callback not applied")
	}
}
func TestV10AcceptedPreviewIsNotCommercialOverride(t *testing.T) {
	h := newHarness(t)
	o := h.place("VIETQR")
	if o.Final != 245000 {
		t.Fatalf("expected canonical 245000 got %d", o.Final)
	}
	h.mockExec("UPDATE catalog SET price=1 WHERE sku='MX-GION-500G'")
	if h.order(o.ID).Snapshot.Items[0].UnitPrice != 110000 {
		t.Fatal("catalog change altered old snapshot")
	}
	_, e := h.S.DB.Exec(h.ctx, "UPDATE order_items SET unit_price=1,line_total=quantity WHERE order_id=$1", o.ID)
	if e == nil {
		t.Fatal("snapshot UPDATE permitted")
	}
}

func TestV10StatementFindsCancelledAndUnmatchedMoney(t *testing.T) {
	h := newHarness(t)
	o := h.place("VIETQR")
	_, e := h.S.Cancel(h.ctx, h.Principal, o.ID, uuid.NewString(), o.Version, "test", false)
	if e != nil {
		t.Fatal(e)
	}
	h.jobs()
	h.pay(o, o.Final)
	h.pay(domain.Order{Reference: "unmatched-lost-callback"}, 50)
	if e = h.S.ReconcilePayments(h.ctx); e != nil {
		t.Fatal(e)
	}
	if h.count("SELECT count(*) FROM receipts") != 2 || h.count("SELECT count(*) FROM reconciliation_cases") != 2 {
		t.Fatal("statement lost late/unmatched financial truth")
	}
	if h.order(o.ID).Status != "CANCELLED_BY_USER" {
		t.Fatal("statement resurrected order")
	}
	if e = h.S.ReconcilePayments(h.ctx); e != nil {
		t.Fatal(e)
	}
	if h.count("SELECT count(*) FROM receipts") != 2 {
		t.Fatal("statement replay duplicated money")
	}
}
