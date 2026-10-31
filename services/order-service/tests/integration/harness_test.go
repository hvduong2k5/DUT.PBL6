package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/order-service/internal/app"
	"github.com/omamx/order-service/internal/domain"
	inventory "github.com/omamx/order-service/internal/gen/inventory/v1"
	order "github.com/omamx/order-service/internal/gen/order/v1"
	"github.com/omamx/order-service/internal/simulator"
	"github.com/omamx/order-service/internal/transport"
	"github.com/omamx/order-service/migrations"
	"google.golang.org/grpc"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

const internalToken = "order-integration-internal-token"
const jwtSecret = "order-integration-jwt-secret-32bytes"
const callbackSecret = "order-integration-callback-secret-32bytes"
const encryptionKey = "AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8="

type harness struct {
	MockGRPC  string
	t         *testing.T
	ctx       context.Context
	S         *app.Service
	Sim       *simulator.Simulator
	API, Mock *httptest.Server
	Config    app.Config
	Principal domain.Principal
	Token     string
	GRPC      string
	running   bool
	cancel    context.CancelFunc
}

func scopedDB(t *testing.T, dsn string) (string, func()) {
	t.Helper()
	db, e := pgxpool.New(context.Background(), dsn)
	if e != nil {
		t.Fatal(e)
	}
	schema := "order_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	name := pgx.Identifier{schema}.Sanitize()
	if _, e = db.Exec(context.Background(), "CREATE SCHEMA "+name); e != nil {
		t.Fatal(e)
	}
	u, e := url.Parse(dsn)
	if e != nil {
		t.Fatal(e)
	}
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()
	return u.String(), func() {
		_, e := db.Exec(context.Background(), "DROP SCHEMA "+name+" CASCADE")
		if e != nil {
			t.Error(e)
		}
		db.Close()
	}
}
func newHarness(t *testing.T) *harness {
	t.Helper()
	dsn := os.Getenv("ORDER_TEST_DATABASE")
	simDSN := os.Getenv("ORDER_TEST_SIM_DATABASE")
	if dsn == "" || simDSN == "" {
		t.Skip("NOT_RUN: real PostgreSQL/Redis/Kafka integration environment not configured")
	}
	mainDSN, cleanMain := scopedDB(t, dsn)
	mockDSN, cleanMock := scopedDB(t, simDSN)
	h := &harness{t: t, ctx: context.Background()}
	var e error
	h.Sim, e = simulator.New(h.ctx, mockDSN, internalToken, jwtSecret, callbackSecret, "SYNTHETIC-RECEIVER")
	if e != nil {
		t.Fatal(e)
	}
	h.Mock = httptest.NewServer(h.Sim.Handler())
	h.Config = app.Config{DB: mainDSN, Redis: os.Getenv("ORDER_TEST_REDIS"), RedisDB: 1, Dependencies: h.Mock.URL, InternalToken: internalToken, JWTSecret: jwtSecret, CallbackSecret: callbackSecret, EncryptionKey: encryptionKey, Receiver: "SYNTHETIC-RECEIVER", Kafka: os.Getenv("ORDER_TEST_KAFKA"), ConsumerGroup: "order-integration-" + uuid.NewString(), Mode: "sandbox", Poll: 10 * time.Millisecond, AutoComplete: true}
	h.S, e = app.New(h.ctx, h.Config)
	if e != nil {
		t.Fatal(e)
	}
	if e = migrations.Apply(h.ctx, h.S.DB); e != nil {
		t.Fatal(e)
	}
	h.API = httptest.NewServer(transport.Handler(h.S))
	h.Sim.OrderURL = h.API.URL
	listener, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	rpc := grpc.NewServer()
	order.RegisterOrderServiceServer(rpc, &transport.RPC{S: h.S})
	h.GRPC = listener.Addr().String()
	go rpc.Serve(listener)
	mockListener, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	mockRPC := grpc.NewServer()
	inventory.RegisterInventoryServiceServer(mockRPC, &simulator.InventoryRPC{S: h.Sim})
	h.MockGRPC = mockListener.Addr().String()
	go mockRPC.Serve(mockListener)
	t.Cleanup(mockRPC.Stop)
	h.Principal = domain.Principal{ID: uuid.NewString(), Role: "CUSTOMER"}
	h.Token = h.token(h.Principal)
	t.Cleanup(func() {
		if h.cancel != nil {
			h.cancel()
		}
		h.API.Close()
		rpc.Stop()
		h.S.Close()
		h.Mock.Close()
		h.Sim.DB.Close()
		cleanMain()
		cleanMock()
	})
	return h
}
func (h *harness) token(p domain.Principal) string {
	h.t.Helper()
	var result struct {
		Token string `json:"token"`
	}
	code := h.request(h.Mock.URL, "POST", "/test/token", "", map[string]any{"user_id": p.ID, "role": p.Role, "warehouse": p.Warehouse}, &result)
	if code != 200 {
		h.t.Fatalf("fixture token status %d", code)
	}
	return result.Token
}
func (h *harness) request(base, method, path, token string, in, out any, headers ...string) int {
	h.t.Helper()
	var body io.Reader
	if in != nil {
		b, e := json.Marshal(in)
		if e != nil {
			h.t.Fatal(e)
		}
		body = bytes.NewReader(b)
	}
	r, e := http.NewRequest(method, base+path, body)
	if e != nil {
		h.t.Fatal(e)
	}
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-Internal-Token", internalToken)
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	for i := 0; i+1 < len(headers); i += 2 {
		r.Header.Set(headers[i], headers[i+1])
	}
	resp, e := http.DefaultClient.Do(r)
	if e != nil {
		h.t.Fatal(e)
	}
	defer resp.Body.Close()
	if out != nil {
		if e = json.NewDecoder(resp.Body).Decode(out); e != nil {
			h.t.Fatalf("decode HTTP %d: %v", resp.StatusCode, e)
		}
	}
	return resp.StatusCode
}
func (h *harness) quote(method string) domain.Quote {
	h.t.Helper()
	cart, e := h.S.Cart(h.ctx, h.Principal)
	if e != nil {
		h.t.Fatal(e)
	}
	cart, e = h.S.MutateCart(h.ctx, h.Principal, "MX-GION-500G", 2, cart.Revision, false)
	if e != nil {
		h.t.Fatal(e)
	}
	q, e := h.S.CreateQuote(h.ctx, h.Principal, domain.QuoteInput{Revision: cart.Revision, Method: method, Address: &domain.Address{Name: "Synthetic customer", Phone: "+84905123456", Street: "Test only", Ward: "HUE-01", Province: "75"}})
	if e != nil {
		h.t.Fatal(e)
	}
	return q
}
func (h *harness) accept(q domain.Quote, key string) (domain.Operation, int, error) {
	return h.S.Accept(h.ctx, h.Principal, key, domain.CheckoutInput{QuoteID: q.ID, Revision: q.Revision, Method: q.Method})
}
func (h *harness) driveOperation(id string) domain.Operation {
	h.t.Helper()
	deadline := time.Now().Add(12 * time.Second)
	for time.Now().Before(deadline) {
		if _, e := h.S.WorkPlacement(h.ctx); e != nil {
			h.t.Fatal(e)
		}
		o, e := h.S.Operation(h.ctx, id, h.Principal)
		if e != nil {
			h.t.Fatal(e)
		}
		if o.Status == "SUCCEEDED" || o.Status == "FAILED" || o.Status == "MANUAL_REVIEW" {
			return o
		}
		time.Sleep(20 * time.Millisecond)
	}
	h.t.Fatal("operation did not settle")
	return domain.Operation{}
}
func (h *harness) place(method string) domain.Order {
	h.t.Helper()
	q := h.quote(method)
	op, code, e := h.accept(q, uuid.NewString())
	if e != nil || code != 202 {
		h.t.Fatalf("accept status %d error %v", code, e)
	}
	op = h.driveOperation(op.ID)
	if op.Status != "SUCCEEDED" || op.OrderID == nil {
		h.t.Fatalf("placement did not succeed: %+v", op)
	}
	o, e := h.S.GetOrder(h.ctx, *op.OrderID, h.Principal, true)
	if e != nil {
		h.t.Fatal(e)
	}
	return o
}
func (h *harness) jobs() {
	h.t.Helper()
	deadline := time.Now().Add(12 * time.Second)
	for time.Now().Before(deadline) {
		if _, e := h.S.WorkJob(h.ctx); e != nil {
			h.t.Fatal(e)
		}
		var n int
		if e := h.S.DB.QueryRow(h.ctx, "SELECT count(*) FROM jobs WHERE status IN ('PENDING','RETRY')").Scan(&n); e != nil {
			h.t.Fatal(e)
		}
		if n == 0 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	h.t.Fatal("jobs did not settle")
}
func (h *harness) pay(o domain.Order, amount int64) domain.Receipt {
	h.t.Helper()
	var r domain.Receipt
	if code := h.request(h.Mock.URL, "POST", "/test/pay", "", map[string]any{"reference": o.Reference, "amount": amount, "currency": "VND"}, &r); code != 200 {
		h.t.Fatalf("payment fixture status %d", code)
	}
	return r
}
func (h *harness) paid() domain.Order {
	o := h.place("VIETQR")
	r := h.pay(o, o.Final)
	if e := h.S.RecordReceipt(h.ctx, r); e != nil {
		h.t.Fatal(e)
	}
	h.jobs()
	current, e := h.S.GetOrder(h.ctx, o.ID, h.Principal, true)
	if e != nil {
		h.t.Fatal(e)
	}
	if current.Status != "PAID" || current.Stock != "COMMITTED" || !current.Ready {
		h.t.Fatalf("paid guard failed: %+v", current)
	}
	return current
}
func (h *harness) count(query string, args ...any) int {
	h.t.Helper()
	var n int
	if e := h.S.DB.QueryRow(h.ctx, query, args...).Scan(&n); e != nil {
		h.t.Fatal(e)
	}
	return n
}
func (h *harness) mockCount(query string, args ...any) int {
	h.t.Helper()
	var n int
	if e := h.Sim.DB.QueryRow(h.ctx, query, args...).Scan(&n); e != nil {
		h.t.Fatal(e)
	}
	return n
}
func (h *harness) source(o domain.Order, kind string) domain.Event {
	h.t.Helper()
	var event domain.Event
	if code := h.request(h.Mock.URL, "POST", "/test/event", "", domain.Event{OrderID: o.ID, Type: kind}, &event); code != 200 {
		h.t.Fatalf("source fixture status %d", code)
	}
	return event
}
func (h *harness) apply(event domain.Event) string {
	h.t.Helper()
	state, e := h.S.ProcessEvent(h.ctx, event)
	if e != nil {
		h.t.Fatal(e)
	}
	return state
}
func (h *harness) acceptTask(o domain.Order) {
	h.t.Helper()
	code := h.request(h.Mock.URL, "POST", "/fulfillment/accept", "", map[string]any{"order_id": o.ID, "generation": o.Generation}, nil)
	if code != 200 {
		h.t.Fatalf("task acceptance status %d", code)
	}
}
func (h *harness) order(id string) domain.Order {
	h.t.Helper()
	o, e := h.S.GetOrder(h.ctx, id, h.Principal, true)
	if e != nil {
		h.t.Fatal(e)
	}
	return o
}
func (h *harness) approval(o domain.Order, receipt string, amount int64) domain.Approval {
	h.t.Helper()
	a := domain.Approval{ID: uuid.NewString(), OrderID: o.ID, ReceiptID: receipt, Amount: amount, Type: "REFUND", State: "APPROVED", Actor: "SALES_MANAGER"}
	var result domain.Approval
	if code := h.request(h.Mock.URL, "POST", "/test/approval", "", a, &result); code != 200 {
		h.t.Fatal("approval fixture failed")
	}
	return result
}
func (h *harness) receiptID(order string) string {
	h.t.Helper()
	var id string
	if e := h.S.DB.QueryRow(h.ctx, "SELECT receipt_id FROM allocations WHERE order_id=$1", order).Scan(&id); e != nil {
		h.t.Fatal(e)
	}
	return id
}
func (h *harness) exec(query string, args ...any) {
	h.t.Helper()
	if _, e := h.S.DB.Exec(h.ctx, query, args...); e != nil {
		h.t.Fatal(e)
	}
}
func (h *harness) mockExec(query string, args ...any) {
	h.t.Helper()
	if _, e := h.Sim.DB.Exec(h.ctx, query, args...); e != nil {
		h.t.Fatal(e)
	}
}
func requireCode(t *testing.T, e error, want string) {
	t.Helper()
	var d *domain.Error
	if !errors.As(e, &d) || d.Code != want {
		t.Fatalf("want %s got %v", want, e)
	}
}
