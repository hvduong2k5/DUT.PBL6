package integration

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/order-service/internal/app"
	"github.com/omamx/order-service/internal/domain"
	"github.com/omamx/order-service/internal/persistence"
	"github.com/omamx/order-service/internal/sepay"
)

func promotionHarness(t *testing.T) (*harness, *statementServer, sepay.Receiver, *pgxpool.Pool, string) {
	t.Helper()
	target, dsn := os.Getenv("ORDER_TEST_PROMOTION_GRPC"), os.Getenv("PROMOTION_TEST_DATABASE_URL")
	if target == "" || dsn == "" {
		t.Skip("ORDER_TEST_PROMOTION_GRPC and PROMOTION_TEST_DATABASE_URL are required for the real Promotion peer test")
	}
	file := filepath.Join(t.TempDir(), "promotion-token")
	if e := os.WriteFile(file, []byte(os.Getenv("PROMOTION_CHECKOUT_TEST_TOKEN")), 0600); e != nil {
		t.Fatal(e)
	}
	t.Setenv("ORDER_PROMOTION_GRPC", target)
	t.Setenv("ORDER_PROMOTION_TOKEN_FILE", file)
	t.Setenv("ORDER_PROMOTION_ALLOW_LOCAL_INSECURE", "true")
	db, e := pgxpool.New(context.Background(), dsn)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(db.Close)
	code := "TEST_" + strings.ToUpper(strings.ReplaceAll(uuid.NewString(), "-", ""))
	if _, e = db.Exec(context.Background(), `INSERT INTO coupons(code,discount_type,discount_value,starts_at,ends_at,status) VALUES($1,'FIXED_AMOUNT',10000,clock_timestamp()-interval '1 minute',clock_timestamp()+interval '1 hour','ACTIVE')`, code); e != nil {
		t.Fatal(e)
	}
	h, statement, receiver := connectedHarness(t)
	return h, statement, receiver, db, code
}

func checkoutVoucher(t *testing.T, h *harness, code string) (domain.Quote, domain.Order) {
	t.Helper()
	var cart domain.Cart
	if status := h.request(h.API.URL, "POST", "/api/v1/cart/items", h.Token, map[string]any{"sku_code": "MX-GION-500G", "quantity": 2, "revision": 0}, &cart); status != 200 {
		t.Fatal("cart", status)
	}
	var q domain.Quote
	if status := h.request(h.API.URL, "POST", "/api/v1/checkout/quote", h.Token, domain.QuoteInput{Revision: cart.Revision, Method: "VIETQR", Voucher: code, Address: &domain.Address{Name: "Synthetic Customer", Phone: "+84905123456", Street: "Test only", Ward: "HUE-01", Province: "75"}}, &q, "Accept", app.CustomerMediaType); status != 200 {
		t.Fatal("voucher quote", status, q)
	}
	if q.MerchandiseDiscount != 10000 || q.Final != q.Subtotal+q.Shipping-10000 || q.Voucher != code {
		t.Fatal("unbound discount", q)
	}
	var op, replay domain.Operation
	key := uuid.NewString()
	in := domain.CheckoutInput{QuoteID: q.ID, Revision: q.Revision, Method: q.Method}
	if status := h.request(h.API.URL, "POST", "/api/v1/checkout", h.Token, in, &op, "Idempotency-Key", key); status != 202 {
		t.Fatal("accept", status, op)
	}
	if status := h.request(h.API.URL, "POST", "/api/v1/checkout", h.Token, in, &replay, "Idempotency-Key", key); (status != 202 && status != 201) || replay.ID != op.ID {
		t.Fatal("accept replay", status, replay)
	}
	untilConnected(t, func() bool {
		h.request(h.API.URL, "GET", "/api/v1/checkout-operations/"+op.ID, h.Token, nil, &op)
		if op.Status == "FAILED" || op.Status == "MANUAL_REVIEW" {
			t.Fatal("voucher checkout failed", op)
		}
		return op.Status == "SUCCEEDED"
	})
	var order domain.Order
	if status := h.request(h.API.URL, "GET", "/api/v1/orders/"+*op.OrderID, h.Token, nil, &order); status != 200 {
		t.Fatal("order", status)
	}
	if order.Final != q.Final {
		t.Fatal("order changed quoted total")
	}
	return q, order
}

func TestPromotionCheckoutPaidKafka(t *testing.T) {
	h, statement, receiver, promotion, code := promotionHarness(t)
	_, order := checkoutVoucher(t, h, code)
	var instructions persistence.Instructions
	untilConnected(t, func() bool {
		return h.request(h.API.URL, "GET", "/api/v1/orders/"+order.ID+"/payment-instructions", h.Token, nil, &instructions) == 200
	})
	if instructions.Amount != order.Final {
		t.Fatal("payment ignored discount")
	}
	row := sepay.Transaction{ID: uuid.NewString(), BankAccountID: receiver.BankID, Account: receiver.Account, Direction: "in", In: instructions.Amount, Date: time.Now().UTC().Truncate(time.Second).Format(time.RFC3339), Content: instructions.Code, Reference: uuid.NewString()}
	statement.add(row)
	sendConnectedWebhook(t, h, row)
	untilConnected(t, func() bool {
		h.request(h.API.URL, "GET", "/api/v1/orders/"+order.ID, h.Token, nil, &order)
		return order.Status == "PAID" && order.Ready
	})
	untilConnected(t, func() bool {
		var n int
		e := promotion.QueryRow(h.ctx, "SELECT count(*) FROM promotion_consumed_events WHERE aggregate_id=$1 AND event_type='vn.omama.order.paid.v3'", order.ID).Scan(&n)
		return e == nil && n == 1
	})
	var uses int
	if e := promotion.QueryRow(h.ctx, "SELECT count(*) FROM voucher_usage WHERE order_id=$1", order.ID).Scan(&uses); e != nil || uses != 1 {
		t.Fatal("voucher must be consumed once", uses, e)
	}
	if h.count("SELECT count(*) FROM order_voucher_finalizations WHERE order_id=$1", order.ID) != 1 {
		t.Fatal("missing durable finalization proof")
	}
	var raw []byte
	if e := h.S.DB.QueryRow(h.ctx, "SELECT payload_bytes FROM outbox_events WHERE order_id=$1 AND event_type='vn.omama.order.paid.v3'", order.ID).Scan(&raw); e != nil {
		t.Fatal(e)
	}
	// Inspect the JSON keys explicitly; the source fact must contain provider evidence.
	var body map[string]any
	_ = json.Unmarshal(raw, &body)
	if body["data"].(map[string]any)["voucher_finalization"] == nil {
		t.Fatal("paid event lost voucher evidence")
	}
}

func TestPromotionCheckoutUnpaidCancellation(t *testing.T) {
	h, _, _, promotion, code := promotionHarness(t)
	_, order := checkoutVoucher(t, h, code)
	_, e := h.S.Cancel(h.ctx, h.Principal, order.ID, uuid.NewString(), order.Version, "test cancellation", false)
	if e != nil {
		t.Fatal(e)
	}
	untilConnected(t, func() bool {
		var state string
		e := promotion.QueryRow(h.ctx, "SELECT state FROM voucher_holds WHERE order_id=$1", order.ID).Scan(&state)
		return e == nil && state == "RELEASED"
	})
	var uses int
	if e := promotion.QueryRow(h.ctx, "SELECT count(*) FROM voucher_usage WHERE order_id=$1", order.ID).Scan(&uses); e != nil || uses != 0 {
		t.Fatal("unpaid voucher was consumed", uses, e)
	}
	id := order.ID
	untilConnected(t, func() bool {
		var current domain.Order
		h.request(h.API.URL, "GET", "/api/v1/orders/"+id, h.Token, nil, &current)
		return current.Stock == "RELEASED" || current.Stock == "EXPIRED"
	})
}

func TestPromotionCheckoutRejectedFinalizationBlocksFulfillment(t *testing.T) {
	h, statement, receiver, promotion, code := promotionHarness(t)
	_, order := checkoutVoucher(t, h, code)
	// Simulate the authoritative provider expiring a hold before finalization.
	if _, e := promotion.Exec(h.ctx, "UPDATE voucher_holds SET state='EXPIRED',resource_version=resource_version+1 WHERE order_id=$1 AND state='HELD'", order.ID); e != nil {
		t.Fatal(e)
	}
	var instructions persistence.Instructions
	untilConnected(t, func() bool {
		return h.request(h.API.URL, "GET", "/api/v1/orders/"+order.ID+"/payment-instructions", h.Token, nil, &instructions) == 200
	})
	row := sepay.Transaction{ID: uuid.NewString(), BankAccountID: receiver.BankID, Account: receiver.Account, Direction: "in", In: instructions.Amount, Date: time.Now().UTC().Truncate(time.Second).Format(time.RFC3339), Content: instructions.Code, Reference: uuid.NewString()}
	statement.add(row)
	sendConnectedWebhook(t, h, row)
	untilConnected(t, func() bool {
		return h.count("SELECT count(*) FROM jobs WHERE order_id=$1 AND kind='PAYMENT_FINALIZE' AND status='PARKED' AND error_code='VOUCHER_FINALIZATION_REJECTED'", order.ID) == 1
	})
	if h.count("SELECT count(*) FROM outbox_events WHERE order_id=$1 AND event_type IN ('vn.omama.order.paid.v3','vn.omama.order.fulfillment_ready.v3')", order.ID) != 0 {
		t.Fatal("a rejected voucher authorized PAID or fulfillment")
	}
	if h.count("SELECT count(*) FROM order_voucher_finalizations WHERE order_id=$1", order.ID) != 0 {
		t.Fatal("rejected provider decision was recorded as finalized")
	}
}
