package integration

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omamx/order-service/internal/domain"
	"github.com/omamx/order-service/internal/persistence"
	"github.com/omamx/order-service/internal/sepay"
)

func enableKeycloakGuest(t *testing.T, h *harness) {
	t.Helper()
	h.S.C.AuthMode = "keycloak"
	h.S.C.GuestSessionSecret = strings.Repeat("g", 32)
	h.S.C.GuestSessionIssuanceEnabled = true
	h.S.C.GuestCartMutationEnabled = true
	h.S.C.GuestCODCheckoutEnabled = true
	key, _ := base64.StdEncoding.DecodeString(encryptionKey)
	cipher, e := persistence.NewCipher("core-v1", map[string][]byte{"core-v1": key})
	if e != nil {
		t.Fatal(e)
	}
	store, e := persistence.NewStore(h.S.DB, cipher, []byte(callbackSecret))
	if e != nil {
		t.Fatal(e)
	}
	h.S.GuestStore = &persistence.Payment{Store: store}
	h.Principal = domain.Principal{ID: uuid.NewString(), Role: "GUEST", Kind: domain.Guest}
	now := time.Now()
	h.Token, e = h.S.SignGuestSession(h.Principal.ID, now, now.Add(time.Hour))
	if e != nil {
		t.Fatal(e)
	}
	if e = h.S.GuestStore.CreateGuestSession(h.ctx, h.Principal.ID, h.Token, now, now.Add(time.Hour)); e != nil {
		t.Fatal(e)
	}
}

func TestGuestCODKeycloakHTTPAndPendingLimit(t *testing.T) {
	h := newHarness(t)
	enableKeycloakGuest(t, h)
	var cart domain.Cart
	if code := h.request(h.API.URL, "POST", "/api/v1/cart/items", h.Token, map[string]any{"sku_code": "MX-GION-500G", "quantity": 2, "revision": 0}, &cart); code != 200 {
		t.Fatal("anonymous guest cart mutation", code)
	}
	var firstQuote domain.Quote
	var firstKey string
	for i := 0; i < 3; i++ {
		var q domain.Quote
		input := domain.QuoteInput{Revision: cart.Revision, Method: "COD", Address: &domain.Address{Name: "Guest COD", Phone: "0905123456", Street: "Test only", Ward: "HUE-01", Province: "75"}}
		if code := h.request(h.API.URL, "POST", "/api/v1/checkout/quote", h.Token, input, &q, "Accept", "application/vnd.omama.customer-mvp+json"); code != 200 {
			t.Fatal("guest inline quote", code)
		}
		key := uuid.NewString()
		if i == 0 {
			firstQuote = q
			firstKey = key
		}
		var op domain.Operation
		code := h.request(h.API.URL, "POST", "/api/v1/checkout", h.Token, domain.CheckoutInput{QuoteID: q.ID, Revision: q.Revision, Method: "COD"}, &op, "Idempotency-Key", key)
		if code != 202 {
			t.Fatal("guest COD acceptance", code)
		}
		op = h.driveOperation(op.ID)
		if op.Status != "SUCCEEDED" || op.OrderID == nil {
			t.Fatal(op)
		}
		order := h.order(*op.OrderID)
		if order.Status != "CONFIRMED_COD" || order.CustomerID != nil || order.Code == "" {
			t.Fatal("guest persisted COD", order)
		}
		if code = h.request(h.API.URL, "GET", "/api/v1/orders/"+order.ID, h.Token, nil, nil); code != 200 {
			t.Fatal("guest own order", code)
		}
	}
	q := h.quote("COD")
	if _, code, e := h.accept(q, uuid.NewString()); code != 429 || e == nil {
		t.Fatal("fourth unpaid guest order admitted", code, e)
	}
	if h.count("SELECT count(*) FROM orders WHERE status='CONFIRMED_COD'") != 3 {
		t.Fatal("pending limit changed persisted count")
	}
	if _, code, e := h.accept(firstQuote, firstKey); code != 201 || e != nil {
		t.Fatal("quota denied persisted idempotent replay", code, e)
	}
}

func TestGuestVietQRGrantSettlementAndReplay(t *testing.T) {
	h, statement, receiver := connectedHarness(t)
	enableKeycloakGuest(t, h)
	h.S.C.GuestVietQRCheckoutEnabled = true
	_, order := httpCoreOrder(t, h, "VIETQR")
	if code := h.request(h.API.URL, "GET", "/api/v1/orders/"+order.ID+"/payment", h.Token, nil, nil); code != 403 {
		t.Fatal("session used as payment grant", code)
	}
	var grant persistence.GuestGrant
	if code := h.request(h.API.URL, "POST", "/api/v1/orders/"+order.ID+"/guest-payment-access", h.Token, nil, &grant, "Idempotency-Key", uuid.NewString()); code != 201 {
		t.Fatal("guest grant mint", code)
	}
	var instructions persistence.Instructions
	untilConnected(t, func() bool {
		return h.request(h.API.URL, "GET", "/api/v1/orders/"+order.ID+"/payment-instructions", grant.Token, nil, &instructions) == 200
	})
	if code := h.request(h.API.URL, "GET", "/api/v1/orders/"+uuid.NewString()+"/payment", grant.Token, nil, nil); code != 401 {
		t.Fatal("cross-order grant", code)
	}
	row := sepay.Transaction{ID: uuid.NewString(), BankAccountID: receiver.BankID, Account: receiver.Account, Direction: "in", In: instructions.Amount, Date: time.Now().UTC().Truncate(time.Second).Format(time.RFC3339), Content: instructions.Code, Reference: uuid.NewString()}
	sendConnectedWebhook(t, h, row)
	if h.count("SELECT count(*) FROM receipts") != 0 {
		t.Fatal("webhook booked unverified money")
	}
	statement.add(row)
	sendConnectedWebhook(t, h, row)
	untilConnected(t, func() bool {
		var status persistence.PaymentStatus
		return h.request(h.API.URL, "GET", "/api/v1/orders/"+order.ID+"/payment", grant.Token, nil, &status) == 200 && status.OrderStatus == "PAID"
	})
	sendConnectedWebhook(t, h, row)
	if h.count("SELECT count(*) FROM allocations") != 1 || h.count("SELECT count(*) FROM receipts") != 1 || h.count("SELECT count(*) FROM order_milestones WHERE kind='FIRST_PAID'") != 1 {
		t.Fatal("guest settlement duplicated")
	}
}
