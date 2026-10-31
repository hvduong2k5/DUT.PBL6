package integration

import (
	"context"
	"github.com/google/uuid"
	"github.com/omamx/order-service/internal/app"
	"github.com/omamx/order-service/internal/domain"
	"sync"
	"testing"
	"time"
)

func TestV10PlacementAcceptanceAndRestartRecovery(t *testing.T) {
	h := newHarness(t)
	q := h.quote("VIETQR")
	key := uuid.NewString()
	op, code, e := h.accept(q, key)
	if e != nil || code != 202 || op.OrderID != nil {
		t.Fatalf("accept %d %v %+v", code, e, op)
	}
	if h.count("SELECT count(*) FROM orders") != 0 {
		t.Fatal("Order exists before placement")
	}
	if _, e = h.S.Redis.FlushDB(h.ctx).Result(); e != nil {
		t.Fatal(e)
	}
	h.S.Close()
	h.S, e = app.New(h.ctx, h.Config)
	if e != nil {
		t.Fatal(e)
	}
	op = h.driveOperation(op.ID)
	if op.Status != "SUCCEEDED" || op.OrderID == nil {
		t.Fatalf("failed recovery %+v", op)
	}
	again, code, e := h.accept(q, key)
	if e != nil || code != 201 || again.ID != op.ID {
		t.Fatalf("replay %d %v", code, e)
	}
	if h.count("SELECT count(*) FROM orders") != 1 || h.mockCount("SELECT count(*) FROM reservations") != 1 {
		t.Fatal("duplicate business effect")
	}
}
func TestV10ConcurrentAcceptanceAndRetention(t *testing.T) {
	h := newHarness(t)
	q := h.quote("VIETQR")
	key := uuid.NewString()
	var wg sync.WaitGroup
	ids := make(chan string, 100)
	fail := make(chan error, 100)
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			o, code, e := h.accept(q, key)
			if e != nil {
				fail <- e
				return
			}
			if code != 202 {
				fail <- domain.E("UNEXPECTED_STATUS", code)
				return
			}
			ids <- o.ID
		}()
	}
	wg.Wait()
	close(ids)
	close(fail)
	for e := range fail {
		t.Error(e)
	}
	unique := map[string]bool{}
	for id := range ids {
		unique[id] = true
	}
	if len(unique) != 1 || h.count("SELECT count(*) FROM checkout_operations") != 1 {
		t.Fatal("dedup did not serialize acceptance")
	}
	var id string
	for id = range unique {
	}
	op := h.driveOperation(id)
	if op.Status != "SUCCEEDED" {
		t.Fatalf("operation %+v", op)
	}
	_, _, e := h.S.Accept(h.ctx, h.Principal, key, domain.CheckoutInput{QuoteID: q.ID, Revision: q.Revision + 1, Method: q.Method})
	requireCode(t, e, "IDEMPOTENCY_CONFLICT")
	h.exec("UPDATE idempotency_records SET response_expires_at=clock_timestamp()-interval '8 days',tombstone_min_until=clock_timestamp()-interval '40 days'")
	old, code, e := h.accept(q, key)
	requireCode(t, e, "IDEMPOTENCY_RESPONSE_EXPIRED")
	if code != 410 || old.ID != id || h.count("SELECT count(*) FROM orders") != 1 {
		t.Fatal("expired key created new effect")
	}
}
func TestV10FailedValidationReplayHasNoOrder(t *testing.T) {
	h := newHarness(t)
	q := h.quote("VIETQR")
	key := uuid.NewString()
	h.mockExec("UPDATE catalog SET price=price+100 WHERE sku='MX-GION-500G'")
	op, _, e := h.accept(q, key)
	if e != nil {
		t.Fatal(e)
	}
	op = h.driveOperation(op.ID)
	if op.Status != "FAILED" || op.OrderID != nil || op.ErrorCode != "PRICE_CHANGED" {
		t.Fatalf("invalid failure resource %+v", op)
	}
	again, code, e := h.accept(q, key)
	if e != nil || code != 409 || again.ID != op.ID || again.Status != "FAILED" {
		t.Fatalf("failure replay %d %v", code, e)
	}
	if h.count("SELECT count(*) FROM orders") != 0 || h.mockCount("SELECT count(*) FROM reservations") != 0 {
		t.Fatal("validation failure acquired resources")
	}
}
func TestV10LostReserveACKAndTombstone(t *testing.T) {
	h := newHarness(t)
	if e := h.Sim.SetFault(h.ctx, "reserve", "lost_ack", 1); e != nil {
		t.Fatal(e)
	}
	o := h.place("VIETQR")
	if h.mockCount("SELECT count(*) FROM reservations") != 1 || h.mockCount("SELECT available FROM stock WHERE sku='MX-GION-500G'") != 98 {
		t.Fatal("lost ACK deducted twice")
	}
	var released domain.Reservation
	e := h.S.Call(h.ctx, "POST", "/inventory/release", map[string]any{"order_id": uuid.NewString(), "operation_key": "tombstone-release-key"}, &released)
	if e != nil {
		t.Fatal(e)
	}
	var late domain.Reservation
	e = h.S.Call(h.ctx, "POST", "/inventory/reserve", map[string]any{"order_id": released.OrderID, "operation_key": "tombstone-reserve-key", "items": o.Snapshot.Items, "ttl_seconds": 900}, &late)
	if e != nil || late.State != "RELEASED" {
		t.Fatalf("late reserve not fenced %v %+v", e, late)
	}
	if h.mockCount("SELECT available FROM stock WHERE sku='MX-GION-500G'") != 98 {
		t.Fatal("late reserve changed stock")
	}
}
func TestV10QuoteExpiryEligibilityAndOwnership(t *testing.T) {
	h := newHarness(t)
	q := h.quote("VIETQR")
	var other = domain.Principal{ID: uuid.NewString(), Role: "CUSTOMER"}
	_, _, e := h.S.Accept(h.ctx, other, uuid.NewString(), domain.CheckoutInput{QuoteID: q.ID, Revision: q.Revision, Method: q.Method})
	requireCode(t, e, "QUOTE_NOT_FOUND")
	op, _, e := h.accept(q, uuid.NewString())
	if e != nil {
		t.Fatal(e)
	}
	var b []byte
	if e = h.S.DB.QueryRow(h.ctx, "SELECT input_cipher FROM checkout_operations WHERE id=$1", op.ID).Scan(&b); e != nil {
		t.Fatal(e)
	}
	var stored domain.StoredQuote
	if e = h.S.Open(b, &stored); e != nil {
		t.Fatal(e)
	}
	stored.Quote.ExpiresAt = op.AcceptedAt.Add(time.Microsecond)
	b, e = h.S.Seal(stored)
	if e != nil {
		t.Fatal(e)
	}
	h.exec("UPDATE checkout_operations SET input_cipher=$2 WHERE id=$1", op.ID, b)
	if h.driveOperation(op.ID).Status != "SUCCEEDED" {
		t.Fatal("worker incorrectly reused preview expiry")
	}
	_, e = h.S.Operation(h.ctx, op.ID, other)
	if e == nil {
		t.Fatal("operation visible to wrong owner")
	}
	stored.Quote.ExpiresAt = time.Now().Add(-time.Minute)
	b, e = h.S.Seal(stored)
	if e != nil {
		t.Fatal(e)
	}
	if e = h.S.Redis.Set(h.ctx, "quote:"+q.ID, b, time.Minute).Err(); e != nil {
		t.Fatal(e)
	}
	_, _, e = h.accept(q, uuid.NewString())
	requireCode(t, e, "QUOTE_EXPIRED")
}
func TestV10RegisteredIdentityGuestAndHTTPResource(t *testing.T) {
	h := newHarness(t)
	q := h.quote("VIETQR")
	if q.CustomerID == h.Principal.ID || q.CustomerID == "" {
		t.Fatal("user ID used as customer ID")
	}
	var op domain.Operation
	code := h.request(h.API.URL, "POST", "/api/v1/checkout", h.Token, domain.CheckoutInput{QuoteID: q.ID, Revision: q.Revision, Method: q.Method}, &op, "X-Idempotency-Key", uuid.NewString())
	if code != 202 || op.OrderID != nil {
		t.Fatal("acceptance endpoint contract")
	}
	var status domain.Operation
	if h.request(h.API.URL, "GET", "/api/v1/checkout-operations/"+op.ID, h.Token, nil, &status) != 200 {
		t.Fatal("Location does not identify existing resource")
	}
	op = h.driveOperation(op.ID)
	o := h.order(*op.OrderID)
	if o.CustomerID == nil || *o.CustomerID == h.Principal.ID {
		t.Fatal("bad customer mapping")
	}
	if h.request(h.API.URL, "GET", "/api/v1/orders/"+o.ID, "", nil, nil) != 401 {
		t.Fatal("untrusted headers granted access")
	}
	guest := domain.Principal{ID: uuid.NewString(), Role: "GUEST"}
	h.Principal = guest
	h.Token = h.token(guest)
	g := h.place("COD")
	if g.CustomerID != nil || g.Payment != "UNPAID" {
		t.Fatal("Guest/COD data incorrect")
	}
}
func TestV10InsufficientStockCompensation(t *testing.T) {
	h := newHarness(t)
	h.mockExec("UPDATE stock SET physical=1,available=1 WHERE sku='MX-GION-500G'")
	q := h.quote("VIETQR")
	op, _, e := h.accept(q, uuid.NewString())
	if e != nil {
		t.Fatal(e)
	}
	op = h.driveOperation(op.ID)
	if op.Status != "FAILED" || op.OrderID != nil {
		t.Fatalf("unsafe result %+v", op)
	}
	if h.mockCount("SELECT available FROM stock WHERE sku='MX-GION-500G'") != 1 {
		t.Fatal("partial stock effect")
	}
}
func TestV10ServerRejectedQuoteAndInput(t *testing.T) {
	h := newHarness(t)
	var result any
	code := h.request(h.API.URL, "POST", "/api/v1/checkout", h.Token, map[string]any{"quote_id": uuid.NewString(), "cart_revision": 0, "payment_method": "VIETQR", "final_amount": 1}, &result, "X-Idempotency-Key", uuid.NewString())
	if code != 400 {
		t.Fatalf("unknown client price accepted %d", code)
	}
	if h.count("SELECT count(*) FROM checkout_operations") != 0 {
		t.Fatal("invalid request durable accepted")
	}
	if _, e := h.S.Redis.Ping(context.Background()).Result(); e != nil {
		t.Fatal(e)
	}
}
