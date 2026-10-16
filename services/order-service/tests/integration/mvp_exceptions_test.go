package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omamx/order-service/internal/app"
	"github.com/omamx/order-service/internal/domain"
	"github.com/omamx/order-service/internal/sepay"
)

func TestMVPReceiptExceptionMatrix(t *testing.T) {
	for _, name := range []string{"under", "over", "split", "unmatched_invoice", "wrong_currency", "wrong_method", "declined", "unknown_type", "provider_order_conflict"} {
		t.Run(name, func(t *testing.T) {
			h := newMVP(t)
			o := h.place("VIETQR")
			n := mvpNotification(o, uuid.NewString(), o.Final)
			wantReceipts, wantReview, wantCases := 1, 0, 1
			switch name {
			case "under":
				n.Transaction.Amount = sepay.Scalar(fmt.Sprint(o.Final - 1))
			case "over":
				n.Transaction.Amount = sepay.Scalar(fmt.Sprint(o.Final + 1))
			case "split":
				n.Transaction.Amount = sepay.Scalar(fmt.Sprint(o.Final / 2))
				receiveMVP(t, h, n)
				n.Transaction.ID = sepay.Scalar(uuid.NewString())
				wantReceipts, wantCases = 2, 2
			case "unmatched_invoice":
				n.Order.Invoice = "UNKNOWN-" + uuid.NewString()
			case "wrong_currency":
				n.Transaction.Currency = "USD"
				wantReceipts, wantReview, wantCases = 0, 1, 0
			case "wrong_method":
				n.Transaction.Method = "CARD"
				wantReceipts, wantReview, wantCases = 0, 1, 0
			case "declined":
				n.Transaction.Status = "DECLINED"
				wantReceipts, wantReview, wantCases = 0, 1, 0
			case "unknown_type":
				n.Type = "UNRECOGNIZED_PROVIDER_EVENT"
				wantReceipts, wantReview, wantCases = 0, 1, 0
			case "provider_order_conflict":
				first := n
				first.Transaction.Amount = sepay.Scalar(fmt.Sprint(o.Final - 1))
				receiveMVP(t, h, first)
				n.Order.OrderID = "DIFFERENT-REMOTE-ORDER"
				n.Transaction.ID = sepay.Scalar(uuid.NewString())
				wantReceipts, wantCases = 2, 2
			}
			receiveMVP(t, h, n)
			h.jobs()
			if h.count("SELECT count(*) FROM receipts") != wantReceipts || h.count("SELECT count(*) FROM allocations") != 0 || h.count("SELECT count(*) FROM reconciliation_cases") != wantCases || h.count("SELECT count(*) FROM sepay_notifications WHERE status='MANUAL_REVIEW'") != wantReview {
				t.Fatal("exception lost financial evidence or allocated invalid payment")
			}
			if got := h.order(o.ID); got.Status != "PENDING_PAYMENT" || got.Payment != "PENDING" || got.Ready {
				t.Fatalf("exception authorized fulfillment: %+v", got)
			}
			if h.mockCount("SELECT physical FROM stock WHERE sku='MX-GION-500G'") != 100 {
				t.Fatal("exception deducted physical stock")
			}
		})
	}
}

func TestMVPConcurrentIPNAndWorkers(t *testing.T) {
	h := newMVP(t)
	o := h.place("VIETQR")
	n := mvpNotification(o, "one-provider-transaction", o.Final)
	var wg sync.WaitGroup
	errs := make(chan error, 40)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			copy := n
			copy.Timestamp += int64(i) // distinct delivery payloads, one transaction
			body, _ := json.Marshal(copy)
			if e := h.S.ReceiveSePay(h.ctx, body); e != nil {
				errs <- e
			}
		}(i)
	}
	wg.Wait()
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 30; i++ {
				worked, e := h.S.WorkSePayNotification(h.ctx)
				if e != nil {
					errs <- e
					return
				}
				if !worked {
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Error(e)
	}
	h.jobs()
	if h.count("SELECT count(*) FROM receipts") != 1 || h.count("SELECT count(*) FROM allocations") != 1 || h.count("SELECT count(*) FROM sepay_notifications WHERE status='APPLIED'") != 20 || h.count("SELECT count(*) FROM sepay_notifications WHERE status='MANUAL_REVIEW'") != 0 {
		t.Fatal("parallel deliveries were duplicated or put into financial review")
	}
	if got := h.order(o.ID); got.Status != "PAID" || got.Stock != "COMMITTED" {
		t.Fatal("parallel callbacks failed to settle")
	}
	if h.mockCount("SELECT physical FROM stock WHERE sku='MX-GION-500G'") != 98 {
		t.Fatal("stock committed more than once")
	}
}

func TestMVPIPNHTTPValidationMatrix(t *testing.T) {
	h := newMVP(t)
	for _, tc := range []struct {
		name, key string
		body      any
		status    int
	}{
		{"missing_auth", "", map[string]any{"timestamp": 1, "notification_type": "ORDER_PAID"}, 401},
		{"wrong_auth", "wrong-key", map[string]any{"timestamp": 1, "notification_type": "ORDER_PAID"}, 401},
		{"empty_object", h.Config.SePay.IPNSecret, map[string]any{}, 400},
		{"zero_timestamp", h.Config.SePay.IPNSecret, map[string]any{"timestamp": 0, "notification_type": "ORDER_PAID"}, 400},
		{"missing_type", h.Config.SePay.IPNSecret, map[string]any{"timestamp": 1}, 400},
		{"array", h.Config.SePay.IPNSecret, []any{}, 400},
		{"string", h.Config.SePay.IPNSecret, "not an object", 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := h.request(h.API.URL, "POST", "/api/v1/payments/sepay/ipn", "", tc.body, nil, "X-Secret-Key", tc.key); got != tc.status {
				t.Fatalf("HTTP %d, want %d", got, tc.status)
			}
		})
	}
	if h.count("SELECT count(*) FROM sepay_notifications") != 0 {
		t.Fatal("invalid envelope persisted")
	}
}

func TestMVPCancellationRejectPermissionsAndReplay(t *testing.T) {
	h := newMVP(t)
	o := h.place("VIETQR")
	receiveMVP(t, h, mvpNotification(o, "paid-cancel-reject", o.Final))
	h.jobs()
	o = h.order(o.ID)
	result, e := h.S.Cancel(h.ctx, h.Principal, o.ID, uuid.NewString(), o.Version, "change mind", true)
	if e != nil {
		t.Fatal(e)
	}
	id := result["request_id"].(string)
	path := "/api/v1/admin/cancellation-requests/" + id + "/decision"
	decision := map[string]any{"approve": false, "reason": "Already scheduled"}
	for _, tc := range []struct {
		name, role, warehouse string
		status                int
	}{{"customer", "CUSTOMER", "HUE", 403}, {"wrong_warehouse", "SALES_MANAGER", "OTHER", 404}, {"packing_staff", "PACKING_STAFF", "HUE", 403}} {
		t.Run(tc.name, func(t *testing.T) {
			p := domain.Principal{ID: uuid.NewString(), Role: tc.role, Warehouse: tc.warehouse}
			if got := h.request(h.API.URL, "POST", path, h.token(p), decision, nil); got != tc.status {
				t.Fatalf("HTTP %d", got)
			}
		})
	}
	manager := domain.Principal{ID: uuid.NewString(), Role: "SALES_MANAGER", Warehouse: "HUE"}
	token := h.token(manager)
	if got := h.request(h.API.URL, "POST", path, token, map[string]string{"reason": "missing decision"}, nil); got != 422 {
		t.Fatalf("missing decision %d", got)
	}
	for i := 0; i < 2; i++ {
		if got := h.request(h.API.URL, "POST", path, token, decision, nil); got != 200 {
			t.Fatalf("reject/replay %d", got)
		}
	}
	if got := h.request(h.API.URL, "POST", path, token, map[string]any{"approve": true, "reason": "Changed"}, nil); got != 409 {
		t.Fatalf("changed decision %d", got)
	}
	h.jobs()
	current := h.order(o.ID)
	if current.Status != "PAID" || !current.Ready || current.Stock != "COMMITTED" {
		t.Fatal("rejection cancelled paid order")
	}
}

func TestMVPManualRefundHTTPGuards(t *testing.T) {
	h := newMVP(t)
	o := h.place("VIETQR")
	receiveMVP(t, h, mvpNotification(o, "original-sale", o.Final))
	h.jobs()
	receiveMVP(t, h, mvpNotification(o, "extra-to-refund", o.Final))
	var receipt string
	if e := h.S.DB.QueryRow(h.ctx, "SELECT id FROM receipts WHERE transaction_id='extra-to-refund'").Scan(&receipt); e != nil {
		t.Fatal(e)
	}
	a := domain.Principal{ID: uuid.NewString(), Role: "ADMIN"}
	at := h.token(a)
	in := app.ManualRefundInput{ReceiptID: receipt, Amount: o.Final, Reason: "extra payment", Payee: app.RefundPayee{Bank: "TEST", Account: "synthetic-account", Name: "Synthetic", Verification: "synthetic-case"}}
	path := "/api/v1/admin/manual-refunds"
	key := uuid.NewString()
	if got := h.request(h.API.URL, "POST", path, h.Token, in, nil, "X-Idempotency-Key", key); got != 403 {
		t.Fatalf("customer approve %d", got)
	}
	if got := h.request(h.API.URL, "POST", path, at, in, nil); got != 400 {
		t.Fatalf("missing key %d", got)
	}
	partial := in
	partial.Amount--
	if got := h.request(h.API.URL, "POST", path, at, partial, nil, "X-Idempotency-Key", uuid.NewString()); got != 422 {
		t.Fatalf("partial refund %d", got)
	}
	var f, again app.ManualRefund
	if got := h.request(h.API.URL, "POST", path, at, in, &f, "X-Idempotency-Key", key); got != 200 {
		t.Fatalf("approval %d", got)
	}
	if got := h.request(h.API.URL, "POST", path, at, in, &again, "X-Idempotency-Key", key); got != 200 || f.ID != again.ID {
		t.Fatal("approval replay duplicated")
	}
	changed := in
	changed.Reason = "different payload"
	if got := h.request(h.API.URL, "POST", path, at, changed, nil, "X-Idempotency-Key", key); got != 409 {
		t.Fatalf("approval conflict %d", got)
	}
	path += "/" + f.ID
	for _, tc := range []struct {
		role, warehouse string
		status          int
	}{{"CUSTOMER", "HUE", 403}, {"SALES_MANAGER", "OTHER", 404}} {
		p := domain.Principal{ID: uuid.NewString(), Role: tc.role, Warehouse: tc.warehouse}
		if got := h.request(h.API.URL, "GET", path, h.token(p), nil, nil); got != tc.status {
			t.Fatalf("refund detail scope %d", got)
		}
	}
	if got := h.request(h.API.URL, "POST", path+"/confirm", at, map[string]string{"evidence_hash": "unsubmitted"}, nil); got != 409 {
		t.Fatalf("confirm before evidence %d", got)
	}
	if got := h.request(h.API.URL, "POST", path+"/claim", at, nil, nil); got != 200 {
		t.Fatalf("claim %d", got)
	}
	evidence := app.RefundEvidence{Bank: "TEST", Account: "source", TransactionID: uuid.NewString(), Amount: o.Final - 1, PaidAt: time.Now().UTC(), Statement: "synthetic-proof"}
	if got := h.request(h.API.URL, "POST", path+"/evidence", at, evidence, nil); got != 422 {
		t.Fatalf("wrong evidence amount %d", got)
	}
	evidence.Amount = o.Final
	evidence.PaidAt = time.Now().Add(time.Hour)
	if got := h.request(h.API.URL, "POST", path+"/evidence", at, evidence, nil); got != 422 {
		t.Fatalf("future evidence %d", got)
	}
	evidence.PaidAt = time.Now().UTC()
	if got := h.request(h.API.URL, "POST", path+"/evidence", at, evidence, &f); got != 200 {
		t.Fatalf("valid evidence %d", got)
	}
	if got := h.request(h.API.URL, "POST", path+"/evidence", at, evidence, nil); got != 200 {
		t.Fatalf("evidence replay %d", got)
	}
	evidence.Statement = "overwritten-proof"
	if got := h.request(h.API.URL, "POST", path+"/evidence", at, evidence, nil); got != 409 {
		t.Fatalf("immutable evidence %d", got)
	}
	if got := h.request(h.API.URL, "POST", path+"/confirm", at, map[string]string{"evidence_hash": *f.EvidenceHash}, nil); got != 403 {
		t.Fatalf("self verification %d", got)
	}
	if h.count("SELECT count(*) FROM outbox WHERE topic='order.manual_refunds.v1'") != 0 {
		t.Fatal("unverified refund emitted settlement")
	}
}

func TestMVPReconcileFailureDoesNotAdvanceOrInventMoney(t *testing.T) {
	for _, name := range []string{"unavailable", "wrong_identity", "captured_without_transactions"} {
		t.Run(name, func(t *testing.T) {
			h := newMVP(t)
			o := h.place("VIETQR")
			n := mvpNotification(o, "query-exception", o.Final)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if name == "unavailable" {
					w.WriteHeader(503)
					return
				}
				if r.URL.Path == "/order" {
					json.NewEncoder(w).Encode(map[string]any{"data": []sepay.Order{n.Order}, "meta": map[string]any{"current_page": 1, "page_count": 3, "has_more": true}})
					return
				}
				detail := n.Order
				if name == "wrong_identity" {
					detail.Invoice = "wrong-invoice"
					detail.Transactions = []sepay.Transaction{n.Transaction}
				}
				json.NewEncoder(w).Encode(map[string]any{"data": detail})
			}))
			defer server.Close()
			h.S.SePay.BaseURL = server.URL
			h.S.SePay.HTTP = server.Client()
			e := h.S.ReconcileSePay(h.ctx)
			if name != "captured_without_transactions" && e == nil {
				t.Fatal("failed query accepted")
			}
			if name == "captured_without_transactions" && e != nil {
				t.Fatal(e)
			}
			drainMVP(t, h)
			if name == "captured_without_transactions" && h.count("SELECT count(*) FROM sepay_notifications WHERE status='MANUAL_REVIEW'") != 1 {
				t.Fatal("provider reports captured money without transaction proof, but no review was raised")
			}
			if h.count("SELECT count(*) FROM receipts") != 0 || h.order(o.ID).Payment != "PENDING" {
				t.Fatal("query invented payment")
			}
			if name != "captured_without_transactions" && h.count("SELECT page FROM sepay_scan") != 1 {
				t.Fatal("failed page advanced")
			}
			if name == "captured_without_transactions" {
				// A query warning may arrive before a delayed, valid provider IPN.
				// It must not itself prevent an on-time exact payment settling.
				receiveMVP(t, h, n)
				h.jobs()
				if h.order(o.ID).Status != "PAID" {
					t.Fatal("query warning blocked a later valid IPN")
				}
			}
		})
	}
}

func TestMVPDeliveredNeverAutoCompletes(t *testing.T) {
	h := newMVP(t)
	o := h.place("VIETQR")
	receiveMVP(t, h, mvpNotification(o, "delivery-paid", o.Final))
	h.jobs()
	o = h.order(o.ID)
	h.acceptTask(o)
	for _, kind := range []string{"PACKING_ACCEPTED", "PACKING_SEALED", "SHIPMENT_CREATED", "SHIPMENT_DISPATCHED", "SHIPMENT_DELIVERED"} {
		if got := h.apply(h.source(o, kind)); got != "APPLIED" {
			t.Fatalf("%s: %s", kind, got)
		}
	}
	if h.order(o.ID).Status != "DELIVERED" {
		t.Fatal("delivery chain did not complete")
	}
	h.exec("UPDATE orders SET completion_due_at=clock_timestamp()-interval '1 day' WHERE id=$1", o.ID)
	if e := h.S.CompleteOrders(h.ctx); e != nil {
		t.Fatal(e)
	}
	if h.order(o.ID).Status != "DELIVERED" {
		t.Fatal("MVP silently auto-completed")
	}
}

func TestMVPSlowReconciliationDoesNotBlockIPN(t *testing.T) {
	h := newMVP(t)
	o := h.place("VIETQR")
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		once.Do(func() { close(entered) })
		select {
		case <-release:
		case <-r.Context().Done():
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"data":[],"meta":{"current_page":1,"page_count":0,"has_more":false}}`)
	}))
	defer provider.Close()
	defer close(release)
	h.S.SePay.BaseURL = provider.URL
	h.S.SePay.HTTP = provider.Client()
	ctx, cancel := context.WithCancel(h.ctx)
	h.cancel = cancel
	defer cancel()
	h.S.Start(ctx)
	select {
	case <-entered:
	case <-time.After(4 * time.Second):
		t.Fatal("query worker did not start")
	}
	n := mvpNotification(o, "ipn-during-query", o.Final)
	if got := h.request(h.API.URL, "POST", "/api/v1/payments/sepay/ipn", "", n, nil, "X-Secret-Key", h.Config.SePay.IPNSecret); got != 200 {
		t.Fatalf("IPN %d", got)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if h.order(o.ID).Status == "PAID" {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("a blocked provider query prevented durable IPN processing")
}

func TestMVPStockExpiryAndPaymentTimeoutRace(t *testing.T) {
	for _, name := range []string{"remote_stock_expired", "local_timeout_race"} {
		t.Run(name, func(t *testing.T) {
			h := newMVP(t)
			o := h.place("VIETQR")
			n := mvpNotification(o, "paid-but-expired", o.Final)
			if name == "remote_stock_expired" {
				receiveMVP(t, h, n)
				h.mockExec("UPDATE reservations SET expires_at=clock_timestamp()-interval '1 second' WHERE order_id=$1", o.ID)
			} else {
				h.exec("UPDATE orders SET payment_expires_at=clock_timestamp()-interval '1 second' WHERE id=$1", o.ID)
				raw, _ := json.Marshal(n)
				if e := h.S.ReceiveSePay(h.ctx, raw); e != nil {
					t.Fatal(e)
				}
				errs := make(chan error, 2)
				go func() { _, e := h.S.WorkSePayNotification(h.ctx); errs <- e }()
				go func() { errs <- h.S.SweepTimeouts(h.ctx) }()
				for i := 0; i < 2; i++ {
					if e := <-errs; e != nil {
						t.Fatal(e)
					}
				}
			}
			h.jobs()
			got := h.order(o.ID)
			if got.Status != "CANCELLED_TIMEOUT" || got.Ready || h.count("SELECT count(*) FROM receipts") != 1 || h.count("SELECT count(*) FROM reconciliation_cases") != 1 {
				t.Fatalf("unsafe expired payment: %+v", got)
			}
			if h.mockCount("SELECT physical FROM stock WHERE sku='MX-GION-500G'") != 100 || h.mockCount("SELECT available FROM stock WHERE sku='MX-GION-500G'") != 100 {
				t.Fatal("expired stock was deducted or not released")
			}
			if h.count("SELECT count(*) FROM outbox WHERE event_type IN ('vn.omama.order.paid.v2','vn.omama.order.fulfillment_ready.v2')") != 0 {
				t.Fatal("expired payment authorized fulfillment")
			}
		})
	}
}
