package integration

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"github.com/google/uuid"
	"github.com/omamx/order-service/internal/domain"
	"github.com/omamx/order-service/internal/persistence"
	"github.com/omamx/order-service/internal/sepay"
	"github.com/segmentio/kafka-go"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

type redirectTransport struct{ target *url.URL }

func (t redirectTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	copy := r.Clone(r.Context())
	u := *r.URL
	u.Scheme = t.target.Scheme
	u.Host = t.target.Host
	copy.URL = &u
	return http.DefaultTransport.RoundTrip(copy)
}

type statementServer struct {
	mu    sync.Mutex
	rows  []sepay.Transaction
	calls int
}

func (s *statementServer) handler(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if r.URL.Path != "/v2/transactions" || r.Header.Get("Authorization") != "Bearer "+strings.Repeat("a", 64) {
		w.WriteHeader(401)
		return
	}
	rows := s.rows
	if rows == nil {
		rows = []sepay.Transaction{}
	}
	last := 1
	if len(rows) == 0 {
		last = 0
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": rows, "meta": map[string]any{"pagination": map[string]any{"total": len(rows), "current_page": 1, "last_page": last, "per_page": 100, "has_more": false}}})
}
func (s *statementServer) add(row sepay.Transaction) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rows = append(s.rows, row)
}
func connectedHarness(t *testing.T) (*harness, *statementServer, sepay.Receiver) {
	t.Helper()
	h := newHarness(t)
	t.Cleanup(func() {
		if !t.Failed() {
			return
		}
		for _, query := range []string{
			"SELECT status||':'||stage||':'||COALESCE(error_code,'') FROM checkout_operations",
			"SELECT status||':'||kind||':'||COALESCE(error_code,'') FROM jobs",
			"SELECT status||':'||stock_status FROM orders",
			"SELECT source||':'||status||':'||source_version FROM inbox",
		} {
			rows, e := h.S.DB.Query(h.ctx, query)
			if e != nil {
				t.Log("diagnostic query unavailable")
				continue
			}
			for rows.Next() {
				var v string
				if rows.Scan(&v) == nil {
					t.Log("CONNECTED_DIAGNOSTIC", v)
				}
			}
			rows.Close()
		}
	})
	folder := t.TempDir()
	receiver := sepay.Receiver{BankID: uuid.NewString(), BankCode: "VCB", BankName: "Vietcombank", Account: "0000000000", Holder: "SYNTHETIC ONLY", Mode: "BANK_ACCOUNT", Currency: "VND"}
	raw, _ := json.Marshal(receiver)
	for name, data := range map[string][]byte{"token": []byte(strings.Repeat("a", 64)), "secret": []byte(strings.Repeat("s", 32)), "receiver": raw} {
		if e := os.WriteFile(filepath.Join(folder, name), data, 0600); e != nil {
			t.Fatal(e)
		}
	}
	env := map[string]string{"ORDER_CORE_ENABLED": "true", "ORDER_MERCHANT_ID": "11111111-1111-4111-8111-111111111111", "SEPAY_ENABLED": "true", "SEPAY_API_TOKEN_FILE": filepath.Join(folder, "token"), "SEPAY_WEBHOOK_SECRET_FILE": filepath.Join(folder, "secret"), "SEPAY_RECEIVER_FILE": filepath.Join(folder, "receiver"), "SEPAY_BOOTSTRAP_AT": time.Now().Add(-time.Minute).UTC().Format(time.RFC3339), "SEPAY_CONNECTION_ID": uuid.NewString(), "SEPAY_RECEIVER_ID": uuid.NewString(), "SEPAY_EGRESS_SCOPE": uuid.NewString(), "SEPAY_MERCHANT_IDENTITY": "synthetic-connected-merchant", "SEPAY_INVENTORY_GRPC": h.MockGRPC}
	for key, value := range env {
		t.Setenv(key, value)
	}
	grantKey, e := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if e != nil {
		t.Fatal(e)
	}
	der, e := x509.MarshalPKCS8PrivateKey(grantKey)
	if e != nil {
		t.Fatal(e)
	}
	keyFile := filepath.Join(folder, "guest-key.pem")
	if e = os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), 0600); e != nil {
		t.Fatal(e)
	}
	t.Setenv("SEPAY_GUEST_SIGNING_KEY_FILE", keyFile)
	t.Setenv("SEPAY_GUEST_SIGNING_KEY_ID", "connected-test-v1")
	if e := h.S.EnableSePayFromEnv(h.ctx); e != nil {
		t.Fatal(e)
	}
	if e := h.S.EnableCoreCheckout(); e != nil {
		t.Fatal(e)
	}
	statement := &statementServer{}
	provider := httptest.NewServer(http.HandlerFunc(statement.handler))
	t.Cleanup(provider.Close)
	target, _ := url.Parse(provider.URL)
	reader, e := sepay.NewSandboxClientWithTransport(strings.Repeat("a", 64), redirectTransport{target})
	if e != nil {
		t.Fatal(e)
	}
	h.S.PaymentReader = reader
	// Isolate source history from other harnesses sharing the same Kafka broker.
	h.S.C.SourceTopic = "order.connected.sources." + uuid.NewString()
	client := &kafka.Client{Addr: kafka.TCP(strings.Split(h.Config.Kafka, ",")...), Timeout: 10 * time.Second}
	created, e := client.CreateTopics(h.ctx, &kafka.CreateTopicsRequest{Topics: []kafka.TopicConfig{{Topic: h.S.C.SourceTopic, NumPartitions: 1, ReplicationFactor: 1}}})
	if e != nil {
		t.Fatal("source topic creation", e)
	}
	if e = created.Errors[h.S.C.SourceTopic]; e != nil {
		t.Fatal("source topic creation", e)
	}
	t.Cleanup(func() {
		_, _ = client.DeleteTopics(context.Background(), &kafka.DeleteTopicsRequest{Topics: []string{h.S.C.SourceTopic}})
	})
	ctx, cancel := context.WithCancel(h.ctx)
	h.cancel = cancel
	h.S.Start(ctx)
	return h, statement, receiver
}
func untilConnected(t *testing.T, fn func() bool) {
	t.Helper()
	deadline := time.Now().Add(25 * time.Second)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatal("connected runtime did not reach expected durable state")
}
func httpCoreOrder(t *testing.T, h *harness, method string) (domain.Quote, domain.Order) {
	t.Helper()
	var cart domain.Cart
	if code := h.request(h.API.URL, "POST", "/api/v1/cart/items", h.Token, map[string]any{"sku_code": "MX-GION-500G", "quantity": 2, "revision": 0}, &cart); code != 200 {
		t.Fatal("cart", code)
	}
	var q domain.Quote
	var preview map[string]json.RawMessage
	if code := h.request(h.API.URL, "POST", "/api/v1/checkout/quote", h.Token, domain.QuoteInput{Revision: cart.Revision, Method: method, Address: &domain.Address{Name: "Synthetic customer", Phone: "+84905123456", Street: "Test only", Ward: "HUE-01", Province: "75"}}, &preview); code != 200 {
		t.Fatal("quote", code, preview)
	}
	raw, _ := json.Marshal(preview)
	t.Log("CONNECTED_CONTRACT Quote " + string(raw))
	var fields struct {
		domain.Quote
		Items json.RawMessage `json:"items"`
	}
	if e := json.Unmarshal(raw, &fields); e != nil {
		t.Fatal(e)
	}
	q = fields.Quote
	key := uuid.NewString()
	input := domain.CheckoutInput{QuoteID: q.ID, Revision: q.Revision, Method: method}
	var op domain.Operation
	if code := h.request(h.API.URL, "POST", "/api/v1/checkout", h.Token, input, &op, "X-Idempotency-Key", key); code != 202 {
		t.Fatal("accept", code, op)
	}
	var replay domain.Operation
	if code := h.request(h.API.URL, "POST", "/api/v1/checkout", h.Token, input, &replay, "X-Idempotency-Key", key); (code != 202 && code != 201) || replay.ID != op.ID {
		t.Fatal("replay", code, replay, op)
	}
	untilConnected(t, func() bool {
		if _, e := h.S.CoreCheckout.Work(h.ctx); e != nil {
			t.Fatal("target checkout executor", e)
		}
		h.request(h.API.URL, "GET", "/api/v1/checkout-operations/"+op.ID, h.Token, nil, &op)
		if op.Status == "FAILED" {
			t.Fatalf("operation failed: %+v", op)
		}
		return op.Status == "SUCCEEDED" && op.OrderID != nil
	})
	var order domain.Order
	if code := h.request(h.API.URL, "GET", "/api/v1/orders/"+*op.OrderID, h.Token, nil, &order); code != 200 {
		t.Fatal("get target order", code)
	}
	captureConnected(t, h, "CheckoutOperation", "/api/v1/checkout-operations/"+op.ID, h.Token)
	return q, order
}
func sendConnectedWebhook(t *testing.T, h *harness, row sepay.Transaction) {
	t.Helper()
	at, _ := sepay.BankTime(row.Date)
	raw, _ := json.Marshal(sepay.Webhook{ID: 9007199254740993, Gateway: "VCB", Date: at.In(time.FixedZone("VN", 7*3600)).Format("2006-01-02 15:04:05"), Account: row.Account, Content: row.Content, Direction: "in", Amount: row.In, Reference: row.Reference})
	stamp := strconv.FormatInt(time.Now().Unix(), 10)
	mac := hmac.New(sha256.New, []byte(strings.Repeat("s", 32)))
	mac.Write([]byte(stamp + "."))
	mac.Write(raw)
	request, _ := http.NewRequest("POST", h.API.URL+"/api/v1/payments/sepay/webhook", bytes.NewReader(raw))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-SePay-Timestamp", stamp)
	request.Header.Set("X-SePay-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	response, e := http.DefaultClient.Do(request)
	if e != nil {
		t.Fatal(e)
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		body, _ := io.ReadAll(response.Body)
		t.Fatal("webhook", response.StatusCode, string(body))
	}
}
func TestConnectedPublicCheckoutSePayPaidReadyKafka(t *testing.T) {
	h, statement, receiver := connectedHarness(t)
	_, order := httpCoreOrder(t, h, "VIETQR")
	var instruction persistence.Instructions
	untilConnected(t, func() bool {
		return h.request(h.API.URL, "GET", "/api/v1/orders/"+order.ID+"/payment-instructions", h.Token, nil, &instruction) == 200
	})
	row := sepay.Transaction{ID: uuid.NewString(), BankAccountID: receiver.BankID, Account: receiver.Account, Direction: "in", In: instruction.Amount, Date: time.Now().UTC().Truncate(time.Second).Format(time.RFC3339), Content: instruction.Code, Reference: uuid.NewString()}
	// Delivery first cannot book money; canonical API discovery is the authority.
	if e := h.Sim.SetFault(h.ctx, "core_finalize", "lost_ack", 1); e != nil {
		t.Fatal(e)
	}
	sendConnectedWebhook(t, h, row)
	if h.count("SELECT count(*) FROM receipts") != 0 {
		t.Fatal("webhook booked funds")
	}
	statement.add(row)
	sendConnectedWebhook(t, h, row)
	untilConnected(t, func() bool {
		h.request(h.API.URL, "GET", "/api/v1/orders/"+order.ID, h.Token, nil, &order)
		return order.Status == "PAID" && order.Ready
	})
	untilConnected(t, func() bool {
		return h.count("SELECT count(*) FROM outbox_events WHERE publish_status<>'PUBLISHED'") == 0
	})
	if h.count("SELECT count(*) FROM allocations") != 1 || h.count("SELECT count(*) FROM order_milestones WHERE kind='FIRST_PAID'") != 1 {
		t.Fatal("financial replay")
	}
	if h.mockCount("SELECT physical FROM stock WHERE sku='MX-GION-500G'") != 98 {
		t.Fatal("stock not committed exactly once")
	}
	untilConnected(t, func() bool {
		var cart domain.Cart
		h.request(h.API.URL, "GET", "/api/v1/cart", h.Token, nil, &cart)
		return len(cart.Items) == 0
	})
	// Read actual broker records, including the same immutable event ids/bytes.
	reader := kafka.NewReader(kafka.ReaderConfig{Brokers: strings.Split(h.Config.Kafka, ","), Topic: "order.events.v3", GroupID: "connected-" + uuid.NewString(), StartOffset: kafka.FirstOffset, MinBytes: 1, MaxBytes: 1 << 20})
	defer reader.Close()
	ctx, cancel := context.WithTimeout(h.ctx, 15*time.Second)
	defer cancel()
	seen := map[string]bool{}
	for len(seen) < 3 {
		message, e := reader.ReadMessage(ctx)
		if e != nil {
			t.Fatal("broker verification", e)
		}
		var envelope struct {
			ID      string `json:"id"`
			Subject string `json:"subject"`
			Type    string `json:"type"`
		}
		if json.Unmarshal(message.Value, &envelope) != nil {
			continue
		}
		if envelope.Subject == "order:"+order.ID {
			if string(message.Key) != order.ID {
				t.Fatal("partition key")
			}
			seen[envelope.Type] = true
		}
	}
	for _, kind := range []string{"vn.omama.order.placed.v3", "vn.omama.order.paid.v3", "vn.omama.order.fulfillment_ready.v3"} {
		if !seen[kind] {
			t.Fatal("missing event", kind)
		}
	}
	t.Logf("CONNECTED_EVIDENCE order=%s status=%s ready=%t allocation=1 stock=98 kafka_event_types=3", order.ID, order.Status, order.Ready)
	completeConnectedOrder(t, h, &order)
}

func sourceEvent(t *testing.T, h *harness, o domain.Order, kind, source, resource string, version int64, seal string) domain.Event {
	t.Helper()
	proof := persistence.SourceEvidence{Order: o.ID, Resource: resource, Source: source, Kind: kind, Version: version, Generation: o.Generation, Seal: seal}
	var canonical persistence.SourceEvidence
	if code := h.request(h.Mock.URL, "POST", "/test/core-source", "", proof, &canonical); code != 201 && code != 200 {
		t.Fatal("canonical source fixture", kind, code)
	}
	event := domain.Event{ID: uuid.NewString(), Source: source, OrderID: o.ID, ResourceID: resource, Version: version, Type: kind, Generation: o.Generation, Seal: seal}
	event.Signature = domain.EventSignature(event, internalToken)
	return event
}
func deliverEvent(t *testing.T, h *harness, event domain.Event) string {
	t.Helper()
	var out map[string]string
	if code := h.request(h.API.URL, "POST", "/internal/events", "", event, &out); code != 202 {
		t.Fatal("source delivery", event.Type, code, out)
	}
	return out["status"]
}
func completeConnectedOrder(t *testing.T, h *harness, order *domain.Order) {
	t.Helper()
	task := uuid.NewString()
	accepted := sourceEvent(t, h, *order, "PACKING_ACCEPTED", "fulfillment", task, 1, "")
	sealed := sourceEvent(t, h, *order, "PACKING_SEALED", "fulfillment", task, 2, "synthetic-seal")
	if deliverEvent(t, h, sealed) != "DEFERRED" {
		t.Fatal("out-of-order seal was not deferred")
	}
	if deliverEvent(t, h, accepted) != "APPLIED" {
		t.Fatal("packing acceptance")
	}
	if deliverEvent(t, h, accepted) != "APPLIED" {
		t.Fatal("event replay")
	}
	untilConnected(t, func() bool {
		h.request(h.API.URL, "GET", "/api/v1/orders/"+order.ID, h.Token, nil, order)
		return order.Status == "PACKED"
	})
	shipment := uuid.NewString()
	for _, step := range []struct {
		kind    string
		version int64
	}{{"SHIPMENT_CREATED", 1}, {"SHIPMENT_DISPATCHED", 2}, {"SHIPMENT_DELIVERED", 3}} {
		event := sourceEvent(t, h, *order, step.kind, "shipping", shipment, step.version, "")
		if step.kind == "SHIPMENT_DELIVERED" {
			raw, _ := json.Marshal(event)
			writer := &kafka.Writer{Addr: kafka.TCP(strings.Split(h.Config.Kafka, ",")...), Topic: h.S.C.SourceTopic, RequiredAcks: kafka.RequireAll, AllowAutoTopicCreation: true}
			ctx, cancel := context.WithTimeout(h.ctx, 15*time.Second)
			e := writer.WriteMessages(ctx, kafka.Message{Key: []byte(order.ID), Value: raw})
			cancel()
			_ = writer.Close()
			if e != nil {
				t.Fatal("source Kafka publish", e)
			}
			untilConnected(t, func() bool {
				h.request(h.API.URL, "GET", "/api/v1/orders/"+order.ID, h.Token, nil, order)
				return order.Status == "DELIVERED"
			})
		} else if deliverEvent(t, h, event) != "APPLIED" {
			t.Fatal("shipping transition", step.kind)
		}
	}
	h.request(h.API.URL, "GET", "/api/v1/orders/"+order.ID, h.Token, nil, order)
	if order.Status != "DELIVERED" {
		t.Fatal(order.Status)
	}
	// Accelerate the nonfinancial test deadline only. Canonical delivered_at remains
	// immutable and bank/payment expiry is never changed by this fixture.
	if _, e := h.S.DB.Exec(h.ctx, "UPDATE orders SET completion_due_at=clock_timestamp() WHERE id=$1", order.ID); e != nil {
		t.Fatal(e)
	}
	untilConnected(t, func() bool {
		h.request(h.API.URL, "GET", "/api/v1/orders/"+order.ID, h.Token, nil, order)
		return order.Status == "COMPLETED"
	})
	if h.count("SELECT count(*) FROM order_milestones WHERE kind='COMPLETED'") != 1 {
		t.Fatal("completion milestone")
	}
	if e := h.S.CompleteCoreOrders(h.ctx); e != nil {
		t.Fatal(e)
	}
	if h.count("SELECT count(*) FROM outbox_events WHERE event_type='vn.omama.order.completed.v3'") != 1 {
		t.Fatal("completion repeated")
	}
	t.Logf("CONNECTED_LIFECYCLE order=%s status=%s source_history=verified out_of_order=replayed completion=once", order.ID, order.Status)
	captureConnected(t, h, "OrderDetail", "/api/v1/orders/"+order.ID, h.Token)
	rows, e := h.S.DB.Query(h.ctx, "SELECT payload_bytes FROM outbox_events ORDER BY occurred_at,event_id")
	if e != nil {
		t.Fatal(e)
	}
	defer rows.Close()
	for rows.Next() {
		var raw []byte
		if e = rows.Scan(&raw); e != nil {
			t.Fatal(e)
		}
		t.Log("CONNECTED_EVENT " + string(raw))
	}
	if e = rows.Err(); e != nil {
		t.Fatal(e)
	}
}
func captureConnected(t *testing.T, h *harness, label, path, token string) {
	t.Helper()
	var body map[string]json.RawMessage
	if code := h.request(h.API.URL, "GET", path, token, nil, &body); code != 200 {
		t.Fatal("capture", label, code)
	}
	raw, _ := json.Marshal(body)
	t.Log("CONNECTED_CONTRACT " + label + " " + string(raw))
}
func TestConnectedCODPlacementReady(t *testing.T) {
	h, _, _ := connectedHarness(t)
	_, order := httpCoreOrder(t, h, "COD")
	untilConnected(t, func() bool {
		h.request(h.API.URL, "GET", "/api/v1/orders/"+order.ID, h.Token, nil, &order)
		return order.Status == "CONFIRMED_COD" && order.Ready
	})
	if h.count("SELECT count(*) FROM allocations") != 0 || h.count("SELECT count(*) FROM payment_intents") != 0 {
		t.Fatal("COD got payment intent/allocation")
	}
}
func TestConnectedCancelReleasesAndLateMoneyCannotReopen(t *testing.T) {
	h, statement, receiver := connectedHarness(t)
	_, order := httpCoreOrder(t, h, "VIETQR")
	var instruction persistence.Instructions
	untilConnected(t, func() bool {
		return h.request(h.API.URL, "GET", "/api/v1/orders/"+order.ID+"/payment-instructions", h.Token, nil, &instruction) == 200
	})
	h.request(h.API.URL, "GET", "/api/v1/orders/"+order.ID, h.Token, nil, &order)
	var result map[string]any
	if code := h.request(h.API.URL, "POST", "/api/v1/orders/"+order.ID+"/cancel", h.Token, map[string]any{"reason": "synthetic cancellation"}, &result, "X-Idempotency-Key", uuid.NewString(), "If-Match", strconv.FormatInt(order.Version, 10)); code != 202 && code != 200 {
		t.Fatal("cancel", code, result)
	}
	untilConnected(t, func() bool {
		h.request(h.API.URL, "GET", "/api/v1/orders/"+order.ID, h.Token, nil, &order)
		return order.Status == "CANCELLED_BY_USER" && order.Stock == "RELEASED"
	})
	row := sepay.Transaction{ID: uuid.NewString(), BankAccountID: receiver.BankID, Account: receiver.Account, Direction: "in", In: instruction.Amount, Date: time.Now().UTC().Truncate(time.Second).Format(time.RFC3339), Content: instruction.Code, Reference: uuid.NewString()}
	statement.add(row)
	sendConnectedWebhook(t, h, row)
	untilConnected(t, func() bool {
		return h.count("SELECT count(*) FROM reconciliation_cases WHERE reason='LATE_PAYMENT'") == 1
	})
	if h.count("SELECT count(*) FROM allocations") != 0 || h.count("SELECT count(*) FROM order_milestones WHERE kind='FIRST_PAID'") != 0 || h.mockCount("SELECT available FROM stock WHERE sku='MX-GION-500G'") != 100 {
		t.Fatal("cancel/late money invariant")
	}
}
func TestConnectedGuestGrantAndLogout(t *testing.T) {
	h, _, _ := connectedHarness(t)
	var session struct {
		ID    string `json:"session_id"`
		Token string `json:"access_token"`
	}
	if code := h.request(h.API.URL, "POST", "/api/v1/guest-sessions", "", nil, &session); code != 201 {
		t.Fatal(code)
	}
	h.Token = session.Token
	h.Principal = domain.Principal{ID: session.ID, Role: "GUEST"}
	_, order := httpCoreOrder(t, h, "VIETQR")
	if code := h.request(h.API.URL, "GET", "/api/v1/orders/"+order.ID+"/payment", h.Token, nil, nil); code != 401 {
		t.Fatal("generic Guest read payment", code)
	}
	var grant persistence.GuestGrant
	if code := h.request(h.API.URL, "POST", "/api/v1/orders/"+order.ID+"/guest-payment-access", h.Token, nil, &grant, "X-Idempotency-Key", uuid.NewString()); code != 201 {
		t.Fatal("mint", code)
	}
	if code := h.request(h.API.URL, "GET", "/api/v1/orders/"+order.ID+"/payment", grant.Token, nil, nil); code != 200 {
		t.Fatal("grant read", code)
	}
	if code := h.request(h.API.URL, "DELETE", "/api/v1/guest-sessions/current", h.Token, nil, nil); code != 204 {
		t.Fatal("logout", code)
	}
	if code := h.request(h.API.URL, "GET", "/api/v1/orders/"+order.ID+"/payment", grant.Token, nil, nil); code != 401 {
		t.Fatal("grant survived logout", code)
	}
	if code := h.request(h.API.URL, "GET", "/api/v1/orders/"+order.ID, h.Token, nil, nil); code != 401 {
		t.Fatal("generic session survived logout", code)
	}
}
func TestConnectedAdmissionHoldResumeFencesOldGeneration(t *testing.T) {
	h, _, _ := connectedHarness(t)
	_, order := httpCoreOrder(t, h, "COD")
	untilConnected(t, func() bool {
		h.request(h.API.URL, "GET", "/api/v1/orders/"+order.ID, h.Token, nil, &order)
		return order.Ready
	})
	generation := order.Generation
	manager := h.token(domain.Principal{ID: uuid.NewString(), Role: "ADMIN"})
	key := uuid.NewString()
	for _, active := range []bool{true, false} {
		h.request(h.API.URL, "GET", "/api/v1/orders/"+order.ID, h.Token, nil, &order)
		var result map[string]any
		if code := h.request(h.API.URL, "POST", "/api/v1/admin/orders/"+order.ID+"/hold", manager, map[string]any{"active": active}, &result, "X-Idempotency-Key", key, "If-Match", strconv.FormatInt(order.Version, 10)); code != 200 {
			t.Fatal("admission hold/resume", active, code, result)
		}
		if active {
			var ready struct {
				Authorized bool `json:"authorized"`
			}
			h.request(h.API.URL, "GET", "/internal/ready-authorizations/"+order.ID, "", nil, &ready)
			if ready.Authorized {
				t.Fatal("held order still authorized")
			}
		}
		key = uuid.NewString()
	}
	untilConnected(t, func() bool {
		h.request(h.API.URL, "GET", "/api/v1/orders/"+order.ID, h.Token, nil, &order)
		return order.Ready && order.Generation > generation
	})
	old := persistence.SourceEvidence{Order: order.ID, Resource: uuid.NewString(), Source: "fulfillment", Kind: "PACKING_ACCEPTED", Version: 1, Generation: generation}
	if code := h.request(h.Mock.URL, "POST", "/test/core-source", "", old, nil); code != 409 {
		t.Fatal("old generation admitted", code)
	}
	accepted := sourceEvent(t, h, order, "PACKING_ACCEPTED", "fulfillment", uuid.NewString(), 1, "")
	if deliverEvent(t, h, accepted) != "APPLIED" {
		t.Fatal("current generation not accepted")
	}
	h.request(h.API.URL, "GET", "/api/v1/orders/"+order.ID, h.Token, nil, &order)
	if code := h.request(h.API.URL, "POST", "/api/v1/admin/orders/"+order.ID+"/hold", manager, map[string]any{"active": true}, nil, "X-Idempotency-Key", uuid.NewString(), "If-Match", strconv.FormatInt(order.Version, 10)); code != 409 {
		t.Fatal("unsafe admission-only hold after task accepted", code)
	}
}
func TestConnectedUnpaidTimeoutAndBankTimeBeforeDeadline(t *testing.T) {
	h, statement, receiver := connectedHarness(t)
	if e := h.Sim.SetFault(h.ctx, "core_terms", "short_expiry", 1); e != nil {
		t.Fatal(e)
	}
	_, order := httpCoreOrder(t, h, "VIETQR")
	if order.ExpiresAt == nil {
		t.Fatal("missing deadline")
	}
	var code string
	if e := h.S.DB.QueryRow(h.ctx, "SELECT payment_code FROM payment_intents WHERE order_id=$1", order.ID).Scan(&code); e != nil {
		t.Fatal(e)
	}
	row := sepay.Transaction{ID: uuid.NewString(), BankAccountID: receiver.BankID, Account: receiver.Account, Direction: "in", In: order.Final, Date: time.Now().UTC().Truncate(time.Second).Format(time.RFC3339), Content: code, Reference: uuid.NewString()}
	timer := time.NewTimer(time.Until(*order.ExpiresAt) + 100*time.Millisecond)
	defer timer.Stop()
	<-timer.C
	untilConnected(t, func() bool {
		h.request(h.API.URL, "GET", "/api/v1/orders/"+order.ID, h.Token, nil, &order)
		return order.Status == "CANCELLED_TIMEOUT" && (order.Stock == "RELEASED" || order.Stock == "EXPIRED")
	})
	statement.add(row)
	sendConnectedWebhook(t, h, row)
	untilConnected(t, func() bool {
		return h.count("SELECT count(*) FROM reconciliation_cases WHERE reason='LATE_PAYMENT'") == 1
	})
	if h.count("SELECT count(*) FROM allocations") != 0 || h.count("SELECT count(*) FROM order_milestones WHERE kind='FIRST_PAID'") != 0 {
		t.Fatal("bank time backdated allocation")
	}
	t.Log("CONNECTED_TIMEOUT actual_deadline_elapsed=true late_bank_time_preserved=true allocation=0")
}
func TestConnectedChangedTermsFailBeforeStockAndFreshQuoteCanRetry(t *testing.T) {
	h, _, _ := connectedHarness(t)
	q := h.quote("VIETQR")
	if _, e := h.Sim.DB.Exec(h.ctx, "UPDATE catalog SET price=price+1,version=version+1 WHERE sku='MX-GION-500G'"); e != nil {
		t.Fatal(e)
	}
	input := domain.CheckoutInput{QuoteID: q.ID, Revision: q.Revision, Method: q.Method}
	key := uuid.NewString()
	var op domain.Operation
	if code := h.request(h.API.URL, "POST", "/api/v1/checkout", h.Token, input, &op, "X-Idempotency-Key", key); code != 202 {
		t.Fatal(code)
	}
	untilConnected(t, func() bool {
		var current domain.Operation
		h.request(h.API.URL, "GET", "/api/v1/checkout-operations/"+op.ID, h.Token, nil, &current)
		op = current
		return op.Status == "FAILED"
	})
	if op.ErrorCode != "TERMS_CHANGED" || h.count("SELECT count(*) FROM orders") != 0 || h.mockCount("SELECT count(*) FROM core_inventory") != 0 || h.count("SELECT count(*) FROM checkout_submission_guards WHERE state='RELEASED'") != 1 {
		t.Fatal("invalid quote made stock effect", op)
	}
	var replay domain.Operation
	if code := h.request(h.API.URL, "POST", "/api/v1/checkout", h.Token, input, &replay, "X-Idempotency-Key", key); code != 409 || replay.ID != op.ID {
		t.Fatal("failed acceptance replay", code, replay)
	}
	fresh, e := h.S.CreateQuote(h.ctx, h.Principal, domain.QuoteInput{Revision: q.Revision, Method: q.Method, Address: &q.Address})
	if e != nil {
		t.Fatal(e)
	}
	next, code, e := h.S.Accept(h.ctx, h.Principal, uuid.NewString(), domain.CheckoutInput{QuoteID: fresh.ID, Revision: fresh.Revision, Method: fresh.Method})
	if e != nil || code != 202 {
		t.Fatal(next, code, e)
	}
	untilConnected(t, func() bool {
		current, e := h.S.Operation(h.ctx, next.ID, h.Principal)
		if e != nil {
			t.Fatal(e)
		}
		return current.Status == "SUCCEEDED"
	})
	if h.count("SELECT count(*) FROM orders") != 1 {
		t.Fatal("fresh quote recovery")
	}
}
func TestConnectedCleanupPreservesCartEditsAfterAcceptance(t *testing.T) {
	h, _, _ := connectedHarness(t)
	q := h.quote("VIETQR")
	if e := h.Sim.SetFault(h.ctx, "core_revalidate", "unavailable", 1); e != nil {
		t.Fatal(e)
	}
	op, code, e := h.accept(q, uuid.NewString())
	if e != nil || code != 202 {
		t.Fatal(op, code, e)
	}
	cart, e := h.S.MutateCart(h.ctx, h.Principal, "MX-DEO-300G", 1, q.Revision, false)
	if e != nil {
		t.Fatal(e)
	}
	untilConnected(t, func() bool {
		current, e := h.S.Operation(h.ctx, op.ID, h.Principal)
		if e != nil {
			t.Fatal(e)
		}
		op = current
		return op.Status == "SUCCEEDED"
	})
	untilConnected(t, func() bool {
		var status string
		_ = h.S.DB.QueryRow(h.ctx, "SELECT COALESCE(payload->>'cleanup_status','') FROM jobs WHERE order_id=$1 AND kind='CART_CLEANUP'", op.OrderID).Scan(&status)
		return status == "SKIPPED_CHANGED"
	})
	current, e := h.S.Cart(h.ctx, h.Principal)
	if e != nil || current.Revision != cart.Revision || len(current.Items) != 2 {
		t.Fatal("cart edits lost", current, e)
	}
	var expected int64
	if e = h.S.DB.QueryRow(h.ctx, "SELECT amount FROM payment_intents WHERE order_id=$1", op.OrderID).Scan(&expected); e != nil || expected != q.Final {
		t.Fatal("accepted price changed with cart", expected, e)
	}
}
func TestConnectedParkedCheckoutRecoveryKeepsRemoteIdentity(t *testing.T) {
	h, _, _ := connectedHarness(t)
	q := h.quote("VIETQR")
	if e := h.Sim.SetFault(h.ctx, "core_revalidate", "unavailable", 100); e != nil {
		t.Fatal(e)
	}
	op, code, e := h.accept(q, uuid.NewString())
	if e != nil || code != 202 {
		t.Fatal(op, code, e)
	}
	untilConnected(t, func() bool {
		_, e := h.S.DB.Exec(h.ctx, "UPDATE jobs SET run_after=clock_timestamp() WHERE cohort='order-core-1' AND kind='CHECKOUT_VALIDATE' AND status='RETRY_WAIT'")
		if e != nil {
			t.Fatal(e)
		}
		current, e := h.S.Operation(h.ctx, op.ID, h.Principal)
		if e != nil {
			t.Fatal(e)
		}
		return current.Status == "MANUAL_REVIEW"
	})
	path := "/api/v1/admin/checkout-operations/" + op.ID + "/recover"
	if code := h.request(h.API.URL, "POST", path, h.Token, map[string]string{"reason": "dependency restored"}, nil, "X-Idempotency-Key", uuid.NewString()); code != 403 {
		t.Fatal("customer recovered operation", code)
	}
	if e := h.Sim.SetFault(h.ctx, "core_revalidate", "unavailable", 0); e != nil {
		t.Fatal(e)
	}
	admin := h.token(domain.Principal{ID: uuid.NewString(), Role: "ADMIN"})
	key := uuid.NewString()
	if code := h.request(h.API.URL, "POST", path, admin, map[string]string{"reason": "dependency restored"}, nil, "X-Idempotency-Key", key); code != 202 {
		t.Fatal("recover", code)
	}
	untilConnected(t, func() bool {
		current, e := h.S.Operation(h.ctx, op.ID, h.Principal)
		if e != nil {
			t.Fatal(e)
		}
		return current.Status == "SUCCEEDED"
	})
	if code := h.request(h.API.URL, "POST", path, admin, map[string]string{"reason": "dependency restored"}, nil, "X-Idempotency-Key", key); code != 202 {
		t.Fatal("recovery replay", code)
	}
	if h.count("SELECT count(*) FROM orders") != 1 || h.mockCount("SELECT count(*) FROM core_inventory") != 1 {
		t.Fatal("recovery created another identity")
	}
}
