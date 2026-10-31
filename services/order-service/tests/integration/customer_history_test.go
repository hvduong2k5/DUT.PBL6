package integration

import (
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omamx/order-service/internal/app"
	"github.com/omamx/order-service/internal/domain"
)

func TestCustomerHistoryCursorAndMinimalTracking(t *testing.T) {
	h := newHarness(t)
	orders := []domain.Order{h.place("COD"), h.place("COD"), h.place("COD")}
	// Equal timestamps exercise the UUID tie-break instead of relying on clock precision.
	h.exec("UPDATE orders SET created_at='2026-10-01T00:00:00Z'")
	type page struct {
		Items []domain.Order `json:"items"`
		Next  string         `json:"next_cursor"`
	}
	var first, last page
	if code := h.request(h.API.URL, "GET", "/api/v1/orders?limit=2", h.Token, nil, &first); code != 200 || len(first.Items) != 2 || first.Next == "" {
		t.Fatalf("first page: %d %+v", code, first)
	}
	if code := h.request(h.API.URL, "GET", "/api/v1/orders?limit=2&cursor="+url.QueryEscape(first.Next), h.Token, nil, &last); code != 200 || len(last.Items) != 1 || last.Next != "" {
		t.Fatalf("last page: %d %+v", code, last)
	}
	seen := map[string]bool{}
	for _, o := range append(first.Items, last.Items...) {
		if seen[o.ID] {
			t.Fatal("duplicate across cursor pages")
		}
		seen[o.ID] = true
	}
	for _, o := range orders {
		if !seen[o.ID] {
			t.Fatal("missing order", o.ID)
		}
	}
	var exact page
	if code := h.request(h.API.URL, "GET", "/api/v1/orders?limit=3", h.Token, nil, &exact); code != 200 || len(exact.Items) != 3 || exact.Next != "" {
		t.Fatal("exact-size terminal page", code, exact)
	}
	for _, query := range []string{"limit=0", "limit=101", "limit=garbage", "limit=1&limit=2", "q=ORD", "page=2", "cursor=bad", "status=CANCELLED_BY_USER&cursor=" + url.QueryEscape(first.Next)} {
		if code := h.request(h.API.URL, "GET", "/api/v1/orders?"+query, h.Token, nil, nil); code != 400 {
			t.Fatalf("query %s returned %d", query, code)
		}
	}
	other := app.SignToken(jwtSecret, app.Claims{Subject: uuid.NewString(), Role: "CUSTOMER", Issuer: "order-sandbox", Audience: "order-api", Expires: time.Now().Add(time.Hour).Unix()})
	if code := h.request(h.API.URL, "GET", "/api/v1/orders?cursor="+url.QueryEscape(first.Next), other, nil, nil); code != 400 {
		t.Fatal("cross-customer cursor accepted", code)
	}
	var empty page
	if code := h.request(h.API.URL, "GET", "/api/v1/orders", other, nil, &empty); code != 200 || len(empty.Items) != 0 || empty.Next != "" {
		t.Fatal("customer isolation", code, empty)
	}
	var tracking struct {
		Shipment *string `json:"shipment_id"`
		Status   string  `json:"order_status"`
	}
	path := "/api/v1/orders/" + orders[0].ID + "/tracking"
	if code := h.request(h.API.URL, "GET", path, h.Token, nil, &tracking); code != 200 || tracking.Shipment != nil || tracking.Status != "CONFIRMED_COD" {
		t.Fatal("not-shipped projection", code, tracking)
	}
	shipment := uuid.NewString()
	h.exec("UPDATE orders SET shipment_id=$2 WHERE id=$1", orders[0].ID, shipment)
	if code := h.request(h.API.URL, "GET", path, h.Token, nil, &tracking); code != 200 || tracking.Shipment == nil || *tracking.Shipment != shipment {
		t.Fatal("shipment reference", code, tracking)
	}
	if code := h.request(h.API.URL, "GET", path, other, nil, nil); code != 404 {
		t.Fatal("tracking ownership leaked", code)
	}
}
