package sepay

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMoneyPrecision(t *testing.T) {
	for _, value := range []string{"200000", "200000.0", "200000.00"} {
		n, e := VND(Scalar(value))
		if e != nil || n != 200000 {
			t.Fatalf("%s: %d %v", value, n, e)
		}
	}
	for _, value := range []string{"200000.01", "1e5", "-1", "0", "1000000000001", "200000.000", " 200000", "NaN"} {
		if _, e := VND(Scalar(value)); e == nil {
			t.Errorf("accepted invalid money %q", value)
		}
	}
}
func TestNormalizeUsesTransactionNotOrderAmount(t *testing.T) {
	o := Order{OrderID: "REMOTE-1", Invoice: "OMAMA-1", Currency: "VND", Amount: "200000.00", Status: "CANCELLED"}
	tx := Transaction{ID: "transaction-resource-1", Method: "BANK_TRANSFER", Type: "PAYMENT", Status: "APPROVED", Amount: "199999.00", Currency: "VND", Date: "2026-10-15 10:30:00"}
	p, e := Normalize(o, tx)
	if e != nil || p.Amount != 199999 || p.TransactionID != "transaction-resource-1" || p.PaidAt.Format(time.RFC3339) != "2026-10-15T03:30:00Z" {
		t.Fatalf("lost actual/late money: %+v %v", p, e)
	}
	tx.Method = "CARD"
	if _, e = Normalize(o, tx); e == nil {
		t.Fatal("card was interpreted as bank transfer")
	}
}
func TestCheckoutOrderedAndSecretNeverReturned(t *testing.T) {
	c := Config{Environment: "sandbox", MerchantID: "TEST-MERCHANT", SecretKey: "synthetic-merchant-secret", IPNSecret: "synthetic-ipn-secret", ReturnURL: "https://shop.example/payment"}
	if e := c.Validate(); e != nil {
		t.Fatal(e)
	}
	form := c.Checkout("attempt-1", "INV-1", 200000, time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC))
	names := []string{"merchant", "operation", "payment_method", "order_invoice_number", "order_amount", "currency", "order_description", "success_url", "error_url", "cancel_url", "signature"}
	if len(form.Fields) != len(names) {
		t.Fatal("wrong field count")
	}
	for i, n := range names {
		if form.Fields[i].Name != n {
			t.Fatalf("field %d: %s", i, form.Fields[i].Name)
		}
		if form.Fields[i].Value == c.SecretKey {
			t.Fatal("secret exposed")
		}
	}
	// Cross-language HMAC fixture using the SDK's documented wire string.
	if form.Fields[6].Value != "Thanh toan don hang INV-1" {
		t.Fatal("missing signed description required by the live sandbox")
	}
	if form.Fields[10].Value != "ON113IxiwCCSmfi1NSeYFuttQtiyVuqNKeVFI3xkjJI=" {
		t.Fatalf("unexpected signature %s", form.Fields[10].Value)
	}
	if !c.VerifyIPN(c.IPNSecret) || c.VerifyIPN("") || c.VerifyIPN(c.SecretKey) {
		t.Fatal("IPN auth confused with another credential")
	}
	c.ReturnURL = "http://localhost:3000/payment"
	if c.Validate() != nil {
		t.Fatal("local sandbox return page rejected")
	}
	c.ReturnURL = "http://shop.example/payment"
	if c.Validate() == nil {
		t.Fatal("nonlocal insecure return page accepted")
	}
	c.Environment = "production"
	c.ReturnURL = "https://shop.example/payment"
	if c.Validate() == nil {
		t.Fatal("unverified production enabled")
	}
}
func TestGatewayQueryEvidence(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "merchant" || pass != "secret" {
			t.Error("missing provider authentication")
		}
		calls++
		if r.URL.Path == "/order" {
			if r.URL.Query().Get("sort") != "created_at:asc" || r.URL.Query().Get("page") != "1" || r.URL.Query().Get("order_status") != "" {
				t.Error("query excluded cancelled orders")
			}
			fmt.Fprint(w, `{"data":[{"id":12,"order_id":"PROVIDER-1","order_invoice_number":"INV-1"}],"meta":{"current_page":1,"page_count":1,"has_more":false}}`)
		} else {
			if r.URL.Path != "/order/detail/INV-1" {
				t.Error("detail lookup must use merchant invoice")
			}
			fmt.Fprint(w, `{"data":{"id":"12","order_id":"WRONG-RESOURCE"}}`)
		}
	}))
	defer server.Close()
	c := New(Config{MerchantID: "merchant", SecretKey: "secret"})
	c.BaseURL = server.URL
	c.HTTP = server.Client()
	orders, last, e := c.Orders(context.Background(), 1)
	if e != nil || last != 1 || len(orders) != 1 || orders[0].ID != "12" {
		t.Fatalf("query: %+v %d %v", orders, last, e)
	}
	if _, e = c.Detail(context.Background(), "INV-1", "PROVIDER-1"); e == nil {
		t.Fatal("accepted wrong provider resource")
	}
	if calls != 2 {
		t.Fatal("unexpected query count")
	}
	var tx Transaction
	if e = json.Unmarshal([]byte(`{"id":true}`), &tx); e == nil {
		t.Fatal("boolean accepted as provider identity")
	}
}
