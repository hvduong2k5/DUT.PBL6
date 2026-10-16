package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/omamx/order-service/internal/app"
	"github.com/omamx/order-service/internal/domain"
	"github.com/omamx/order-service/internal/sepay"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func newMVP(t *testing.T) *harness {
	return configuredHarness(t, func(c *app.Config) {
		c.ReleaseProfile = "mvp1"
		c.AutoComplete = false
		c.SePay = sepay.Config{Environment: "sandbox", MerchantID: "TEST-MERCHANT", SecretKey: "synthetic-merchant-secret", IPNSecret: "synthetic-ipn-secret", ReturnURL: "https://shop.example/payment"}
	})
}

func mvpEvidence(t *testing.T, name string, value any) {
	t.Helper()
	dir := os.Getenv("ORDER_MVP_EVIDENCE_DIR")
	if dir == "" {
		return
	}
	if e := os.MkdirAll(dir, 0700); e != nil {
		t.Fatal(e)
	}
	b, e := json.MarshalIndent(value, "", "  ")
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(filepath.Join(dir, name+".json"), append(b, '\n'), 0600); e != nil {
		t.Fatal(e)
	}
}
func mvpNotification(o domain.Order, transaction string, amount int64) sepay.Notification {
	return sepay.Notification{Timestamp: time.Now().Unix(), Type: "ORDER_PAID", Order: sepay.Order{ID: "resource-order", OrderID: "REMOTE-" + o.ID, Invoice: o.Reference, Status: "CAPTURED", Amount: sepay.Scalar(fmt.Sprint(o.Final)), Currency: "VND"}, Transaction: sepay.Transaction{ID: sepay.Scalar(transaction), Method: "BANK_TRANSFER", Type: "PAYMENT", Status: "APPROVED", Amount: sepay.Scalar(fmt.Sprint(amount)), Currency: "VND", Date: "2026-10-15 10:30:00"}}
}
func receiveMVP(t *testing.T, h *harness, n sepay.Notification) {
	t.Helper()
	raw, e := json.Marshal(n)
	if e != nil {
		t.Fatal(e)
	}
	if e = h.S.ReceiveSePay(h.ctx, raw); e != nil {
		t.Fatal(e)
	}
	drainMVP(t, h)
}
func drainMVP(t *testing.T, h *harness) {
	t.Helper()
	for i := 0; i < 50; i++ {
		worked, e := h.S.WorkSePayNotification(h.ctx)
		if e != nil {
			t.Fatal(e)
		}
		if !worked {
			return
		}
	}
	t.Fatal("notifications did not drain")
}
func TestMVPScopeRejectsBeforeEffects(t *testing.T) {
	h := newMVP(t)
	if code := h.request(h.API.URL, "POST", "/api/v1/guest-sessions", "", nil, nil); code != 409 {
		t.Fatalf("guest session %d", code)
	}
	guest := domain.Principal{ID: uuid.NewString(), Role: "GUEST"}
	_, e := h.S.CreateQuote(h.ctx, guest, domain.QuoteInput{Method: "VIETQR"})
	requireCode(t, e, "CUSTOMER_LOGIN_REQUIRED")
	_, e = h.S.CreateQuote(h.ctx, h.Principal, domain.QuoteInput{Method: "COD"})
	requireCode(t, e, "PAYMENT_METHOD_DISABLED")
	_, _, e = h.S.Accept(h.ctx, h.Principal, uuid.NewString(), domain.CheckoutInput{Method: "COD"})
	requireCode(t, e, "PAYMENT_METHOD_DISABLED")
	if h.count("SELECT count(*) FROM checkout_operations") != 0 || h.mockCount("SELECT count(*) FROM reservations") != 0 {
		t.Fatal("disabled method caused side effects")
	}
	if _, e = h.S.AcceptRefund(h.ctx, uuid.NewString()); e == nil {
		t.Fatal("automatic refund enabled")
	}
}

func TestMVPBackgroundWorkersFinalizeDurableIPN(t *testing.T) {
	h := newMVP(t)
	o := h.place("VIETQR")
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"data":[],"meta":{"current_page":1,"page_count":0,"has_more":false}}`)
	}))
	defer provider.Close()
	h.S.SePay.BaseURL = provider.URL
	h.S.SePay.HTTP = provider.Client()
	ctx, cancel := context.WithCancel(h.ctx)
	h.cancel = cancel
	h.S.Start(ctx)
	n := mvpNotification(o, "background-tx", o.Final)
	if code := h.request(h.API.URL, "POST", "/api/v1/payments/sepay/ipn", "", n, nil, "X-Secret-Key", h.Config.SePay.IPNSecret); code != 200 {
		t.Fatal(code)
	}
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		if current := h.order(o.ID); current.Status == "PAID" && current.Ready {
			cancel()
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("HTTP IPN did not progress without manually driving workers")
}

func TestMVPInvalidAuthenticatedPayloadIsReviewable(t *testing.T) {
	h := newMVP(t)
	o := h.place("VIETQR")
	bad := map[string]any{"timestamp": time.Now().Unix(), "notification_type": "ORDER_PAID", "order": map[string]any{"order_invoice_number": o.Reference, "order_amount": nil}, "transaction": map[string]any{"id": false}}
	if code := h.request(h.API.URL, "POST", "/api/v1/payments/sepay/ipn", "", bad, nil, "X-Secret-Key", h.Config.SePay.IPNSecret); code != 200 {
		t.Fatalf("authenticated payload was lost: %d", code)
	}
	drainMVP(t, h)
	if h.count("SELECT count(*) FROM sepay_notifications WHERE status='MANUAL_REVIEW' AND reason='INVALID_PROVIDER_PAYLOAD'") != 1 || h.count("SELECT count(*) FROM receipts") != 0 {
		t.Fatal("malformed provider data treated as money or discarded")
	}
	if !h.order(o.ID).Hold {
		t.Fatal("financial review did not hold matching order")
	}
	var id string
	if e := h.S.DB.QueryRow(h.ctx, "SELECT id FROM sepay_notifications").Scan(&id); e != nil {
		t.Fatal(e)
	}
	admin := domain.Principal{ID: uuid.NewString(), Role: "ADMIN"}
	if code := h.request(h.API.URL, "GET", "/api/v1/admin/payments/notifications/"+id, h.token(admin), nil, nil); code != 200 {
		t.Fatal("operator cannot inspect preserved data")
	}
}
func TestMVPCheckoutIPNAndLostFinalizeACK(t *testing.T) {
	h := newMVP(t)
	o := h.place("VIETQR")
	if h.count("SELECT count(*) FROM payment_attempts WHERE order_id=$1", o.ID) != 1 {
		t.Fatal("attempt not atomic with placement")
	}
	var form, again sepay.Checkout
	path := "/api/v1/orders/" + o.ID + "/payment-checkout"
	if code := h.request(h.API.URL, "POST", path, h.Token, nil, &form); code != 200 {
		t.Fatalf("checkout %d", code)
	}
	if code := h.request(h.API.URL, "POST", path, h.Token, nil, &again); code != 200 {
		t.Fatal(code)
	}
	if domain.Hash(form) != domain.Hash(again) || form.Invoice != o.Reference || form.Action != "https://pay-sandbox.sepay.vn/v1/checkout/init" {
		t.Fatal("reload changed provider request")
	}
	mvpEvidence(t, "checkout", form)
	other := h.token(domain.Principal{ID: uuid.NewString(), Role: "CUSTOMER"})
	if code := h.request(h.API.URL, "POST", path, other, nil, nil); code != 404 {
		t.Fatalf("owner guard %d", code)
	}
	n := mvpNotification(o, "remote-tx-1", o.Final)
	if code := h.request(h.API.URL, "POST", "/api/v1/payments/sepay/ipn", "", n, nil); code != 401 {
		t.Fatalf("unauthenticated IPN %d", code)
	}
	if h.count("SELECT count(*) FROM sepay_notifications") != 0 {
		t.Fatal("unauthenticated message persisted")
	}
	if code := h.request(h.API.URL, "POST", "/api/v1/payments/sepay/ipn", "", n, nil, "X-Secret-Key", h.Config.SePay.IPNSecret); code != 200 {
		t.Fatalf("IPN %d", code)
	}
	if h.order(o.ID).Status != "PENDING_PAYMENT" {
		t.Fatal("IPN ACK incorrectly means PAID")
	}
	if e := h.Sim.SetFault(h.ctx, "finalize", "lost_ack", 1); e != nil {
		t.Fatal(e)
	}
	drainMVP(t, h)
	if h.order(o.ID).Status != "PAYMENT_FINALIZING" {
		t.Fatal("not waiting for stock proof")
	}
	h.jobs()
	if current := h.order(o.ID); current.Status != "PAID" || !current.Ready || current.Stock != "COMMITTED" {
		t.Fatalf("paid guard: %+v", current)
	}
	receiveMVP(t, h, n)
	if h.count("SELECT count(*) FROM receipts") != 1 || h.mockCount("SELECT physical FROM stock WHERE sku='MX-GION-500G'") != 98 {
		t.Fatal("duplicate deducted money/stock")
	}
	if code := h.request(h.API.URL, "POST", "/api/v1/payments/vietqr/callback", "", nil, nil); code != 404 {
		t.Fatal("simulator callback still enabled")
	}
	if code := h.request(h.API.URL, "POST", path, h.Token, nil, nil); code != 409 {
		t.Fatal("paid order offered payment again")
	}
	sim := domain.Receipt{Provider: "SIMULATOR"}
	requireCode(t, h.S.RecordReceipt(h.ctx, sim), "PAYMENT_PROVIDER_MISMATCH")
	n.Transaction.ID = "remote-tx-2"
	receiveMVP(t, h, n)
	if h.count("SELECT count(*) FROM receipts") != 2 || h.count("SELECT count(*) FROM allocations") != 1 || h.count("SELECT count(*) FROM reconciliation_cases") != 1 {
		t.Fatal("second transfer was lost or allocated twice")
	}
	n.Transaction.Amount = sepay.Scalar(fmt.Sprint(o.Final + 1))
	receiveMVP(t, h, n)
	if h.count("SELECT count(*) FROM sepay_notifications WHERE status='MANUAL_REVIEW'") != 1 || !h.order(o.ID).Hold {
		t.Fatal("altered receipt not reviewed/held")
	}
}

func TestMVPConcurrentClaimAndEvidenceReuse(t *testing.T) {
	h := newMVP(t)
	o := h.place("VIETQR")
	receiveMVP(t, h, mvpNotification(o, "sale", o.Final))
	h.jobs()
	managers := []domain.Principal{{ID: uuid.NewString(), Role: "ADMIN"}, {ID: uuid.NewString(), Role: "ADMIN"}}
	refunds := []app.ManualRefund{}
	for _, tid := range []string{"extra-a", "extra-b"} {
		receiveMVP(t, h, mvpNotification(o, tid, o.Final))
		var receipt string
		if e := h.S.DB.QueryRow(h.ctx, "SELECT id FROM receipts WHERE transaction_id=$1", tid).Scan(&receipt); e != nil {
			t.Fatal(e)
		}
		f, e := h.S.ApproveManualRefund(h.ctx, managers[0], uuid.NewString(), app.ManualRefundInput{ReceiptID: receipt, Amount: o.Final, Reason: "Duplicate payment", Payee: app.RefundPayee{Bank: "TEST", Account: "123456", Name: "Synthetic", Verification: "owner-verified"}})
		if e != nil {
			t.Fatal(e)
		}
		refunds = append(refunds, f)
	}
	var wg sync.WaitGroup
	winners := make(chan domain.Principal, 2)
	losers := make(chan error, 2)
	for _, p := range managers {
		p := p
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, e := h.S.ClaimManualRefund(h.ctx, p, refunds[0].ID); e != nil {
				losers <- e
			} else {
				winners <- p
			}
		}()
	}
	wg.Wait()
	close(winners)
	close(losers)
	if len(winners) != 1 || len(losers) != 1 {
		t.Fatal("concurrent claim did not select one executor")
	}
	winner := <-winners
	for e := range losers {
		requireCode(t, e, "REFUND_ALREADY_CLAIMED")
	}
	evidence := app.RefundEvidence{Bank: "TEST", Account: "source", TransactionID: "same-bank-reference", Amount: o.Final, PaidAt: time.Now().UTC(), Statement: "verified-statement"}
	if _, e := h.S.SubmitManualRefundEvidence(h.ctx, winner, refunds[0].ID, evidence); e != nil {
		t.Fatal(e)
	}
	if _, e := h.S.ClaimManualRefund(h.ctx, winner, refunds[1].ID); e != nil {
		t.Fatal(e)
	}
	_, e := h.S.SubmitManualRefundEvidence(h.ctx, winner, refunds[1].ID, evidence)
	requireCode(t, e, "SETTLEMENT_ALREADY_USED")
}
func TestMVPLateAndUnsupportedNotifications(t *testing.T) {
	h := newMVP(t)
	o := h.place("VIETQR")
	h.exec("UPDATE orders SET payment_expires_at=clock_timestamp()-interval '1 second' WHERE id=$1", o.ID)
	if e := h.S.SweepTimeouts(h.ctx); e != nil {
		t.Fatal(e)
	}
	h.jobs()
	n := mvpNotification(o, "late-tx", o.Final)
	receiveMVP(t, h, n)
	if h.order(o.ID).Status != "CANCELLED_TIMEOUT" || h.count("SELECT count(*) FROM allocations") != 0 || h.count("SELECT count(*) FROM receipts") != 1 {
		t.Fatal("late transfer resurrected order")
	}
	n.Type = "TRANSACTION_VOID"
	receiveMVP(t, h, n)
	if h.count("SELECT count(*) FROM sepay_notifications WHERE status='MANUAL_REVIEW'") != 1 {
		t.Fatal("void notification silently ignored")
	}
	if !h.order(o.ID).Hold {
		t.Fatal("adverse financial evidence did not hold order")
	}
}
func TestMVPReconciliationUsesInvoiceAndReplaysAfterRestart(t *testing.T) {
	h := newMVP(t)
	o := h.place("VIETQR")
	n := mvpNotification(o, "query-tx", o.Final)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != h.Config.SePay.MerchantID || pass != h.Config.SePay.SecretKey {
			t.Error("query authentication missing")
		}
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/order" {
			json.NewEncoder(w).Encode(map[string]any{"data": []sepay.Order{n.Order}, "meta": map[string]any{"current_page": 1, "page_count": 1, "has_more": false}})
			return
		}
		if r.URL.Path != "/order/detail/"+n.Order.Invoice {
			t.Error("query must use the merchant invoice accepted by sandbox")
		}
		detail := n.Order
		detail.Transactions = []sepay.Transaction{n.Transaction}
		json.NewEncoder(w).Encode(map[string]any{"data": detail})
	}))
	defer server.Close()
	h.S.SePay.BaseURL = server.URL
	h.S.SePay.HTTP = server.Client()
	if e := h.S.ReconcileSePay(h.ctx); e != nil {
		t.Fatal(e)
	}
	// A new service instance reads the same durable queue after a simulated restart.
	restarted, e := app.New(h.ctx, h.Config)
	if e != nil {
		t.Fatal(e)
	}
	defer restarted.Close()
	if worked, e := restarted.WorkSePayNotification(h.ctx); e != nil || !worked {
		t.Fatalf("resume %v %v", worked, e)
	}
	h.jobs()
	receiveMVP(t, h, n)
	if h.count("SELECT count(*) FROM receipts") != 1 || h.count("SELECT count(*) FROM allocations") != 1 {
		t.Fatal("query and IPN used inconsistent transaction identity")
	}
}
func TestMVPManualRefundBudgetClaimAndIndependentVerification(t *testing.T) {
	h := newMVP(t)
	o := h.place("VIETQR")
	receiveMVP(t, h, mvpNotification(o, "original", o.Final))
	h.jobs()
	receiveMVP(t, h, mvpNotification(o, "extra", o.Final))
	var receiptID string
	if e := h.S.DB.QueryRow(h.ctx, "SELECT id FROM receipts WHERE transaction_id='extra'").Scan(&receiptID); e != nil {
		t.Fatal(e)
	}
	a := domain.Principal{ID: uuid.NewString(), Role: "ADMIN"}
	b := domain.Principal{ID: uuid.NewString(), Role: "ADMIN"}
	in := app.ManualRefundInput{ReceiptID: receiptID, Amount: o.Final, Reason: "Duplicate bank transfer", Payee: app.RefundPayee{Bank: "TEST", Account: "12345678", Name: "Synthetic receiver", Verification: "case-verified-owner"}}
	var wg sync.WaitGroup
	results := make(chan app.ManualRefund, 2)
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			f, e := h.S.ApproveManualRefund(h.ctx, a, uuid.NewString(), in)
			if e != nil {
				errs <- e
			} else {
				results <- f
			}
		}()
	}
	wg.Wait()
	close(results)
	close(errs)
	if len(results) != 1 || len(errs) != 1 {
		t.Fatalf("refund budget accepted %d errors %d", len(results), len(errs))
	}
	for e := range errs {
		requireCode(t, e, "REFUND_BUDGET_EXCEEDED")
	}
	f := <-results
	if _, e := h.S.ClaimManualRefund(h.ctx, a, f.ID); e != nil {
		t.Fatal(e)
	}
	_, e := h.S.ClaimManualRefund(h.ctx, b, f.ID)
	requireCode(t, e, "REFUND_ALREADY_CLAIMED")
	if _, e = h.S.MarkManualRefundUnknown(h.ctx, a, f.ID); e != nil {
		t.Fatal(e)
	}
	_, e = h.S.ApproveManualRefund(h.ctx, a, uuid.NewString(), in)
	requireCode(t, e, "REFUND_BUDGET_EXCEEDED")
	evidence := app.RefundEvidence{Bank: "TEST", Account: "source-123", TransactionID: "bank-refund-123", Amount: o.Final, PaidAt: time.Now().UTC(), Statement: "verified-bank-statement-123"}
	f, e = h.S.SubmitManualRefundEvidence(h.ctx, a, f.ID, evidence)
	if e != nil {
		t.Fatal(e)
	}
	_, e = h.S.ConfirmManualRefund(h.ctx, a, f.ID, *f.EvidenceHash)
	requireCode(t, e, "INDEPENDENT_VERIFIER_REQUIRED")
	var verified app.ManualRefund
	bt := h.token(b)
	if code := h.request(h.API.URL, "POST", "/api/v1/admin/manual-refunds/"+f.ID+"/confirm", bt, map[string]string{"evidence_hash": *f.EvidenceHash}, &verified); code != 200 {
		t.Fatalf("verification %d", code)
	}
	if verified.Status != "SUCCEEDED" {
		t.Fatal("not settled")
	}
	mvpEvidence(t, "manual-refund", verified)
	if _, e = h.S.ConfirmManualRefund(h.ctx, b, f.ID, *f.EvidenceHash); e != nil {
		t.Fatal(e)
	}
	if h.count("SELECT count(*) FROM outbox WHERE topic='order.manual_refunds.v1'") != 1 {
		t.Fatal("manual refund event duplicated")
	}
	var event json.RawMessage
	if e = h.S.DB.QueryRow(h.ctx, "SELECT payload FROM outbox WHERE topic='order.manual_refunds.v1'").Scan(&event); e != nil {
		t.Fatal(e)
	}
	mvpEvidence(t, "manual-refund-event", event)
	view, e := h.S.PaymentReview(h.ctx, b)
	if e != nil {
		t.Fatal(e)
	}
	mvpEvidence(t, "payment-review", view)
	if h.order(o.ID).Payment != "CONFIRMED" || h.order(o.ID).Stock != "COMMITTED" {
		t.Fatal("refunding extra receipt changed original sale")
	}
	if h.mockCount("SELECT count(*) FROM provider_refunds") != 0 {
		t.Fatal("manual workflow called automatic provider refund")
	}
	var original string
	if e = h.S.DB.QueryRow(h.ctx, "SELECT id FROM receipts WHERE transaction_id='original'").Scan(&original); e != nil {
		t.Fatal(e)
	}
	in.ReceiptID = original
	_, e = h.S.ApproveManualRefund(h.ctx, a, uuid.NewString(), in)
	requireCode(t, e, "ORDER_REFUND_NOT_SAFE")
}

func TestMVPPaidCancellationReviewAndManualRefund(t *testing.T) {
	h := newMVP(t)
	o := h.place("VIETQR")
	receiveMVP(t, h, mvpNotification(o, "paid-to-cancel", o.Final))
	h.jobs()
	o = h.order(o.ID)
	request, e := h.S.Cancel(h.ctx, h.Principal, o.ID, uuid.NewString(), o.Version, "Customer changed mind", true)
	if e != nil {
		t.Fatal(e)
	}
	id := request["request_id"].(string)
	manager := domain.Principal{ID: uuid.NewString(), Role: "ADMIN"}
	result, e := h.S.ReviewCancellation(h.ctx, manager, id, true, "Approved before packing")
	if e != nil || result["status"] != "APPROVED" {
		t.Fatalf("review %v %v", result, e)
	}
	if _, e = h.S.ReviewCancellation(h.ctx, manager, id, false, "Changed decision"); e == nil {
		t.Fatal("decision overwritten")
	}
	h.jobs()
	o = h.order(o.ID)
	if o.Status != "CANCELLED_BY_ADMIN" || o.Stock != "RELEASED" || o.Ready {
		t.Fatalf("cancellation %+v", o)
	}
	if h.mockCount("SELECT physical FROM stock WHERE sku='MX-GION-500G'") != 100 {
		t.Fatal("committed stock not reversed")
	}
	var receiptID string
	if e = h.S.DB.QueryRow(h.ctx, "SELECT id FROM receipts WHERE transaction_id='paid-to-cancel'").Scan(&receiptID); e != nil {
		t.Fatal(e)
	}
	in := app.ManualRefundInput{ReceiptID: receiptID, Amount: o.Final, Reason: "Approved cancelled order", Payee: app.RefundPayee{Bank: "TEST", Account: "12345678", Name: "Synthetic", Verification: "case-verified"}}
	f, e := h.S.ApproveManualRefund(h.ctx, manager, uuid.NewString(), in)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = h.S.ClaimManualRefund(h.ctx, manager, f.ID); e != nil {
		t.Fatal(e)
	}
	f, e = h.S.SubmitManualRefundEvidence(h.ctx, manager, f.ID, app.RefundEvidence{Bank: "TEST", Account: "source", TransactionID: "refund-transfer", Amount: o.Final, PaidAt: time.Now().UTC(), Statement: "verified-statement"})
	if e != nil {
		t.Fatal(e)
	}
	verifier := domain.Principal{ID: uuid.NewString(), Role: "ADMIN"}
	if _, e = h.S.ConfirmManualRefund(h.ctx, verifier, f.ID, *f.EvidenceHash); e != nil {
		t.Fatal(e)
	}
	if h.order(o.ID).Payment != "REFUNDED" {
		t.Fatal("allocated refund not reflected in payment")
	}
}
