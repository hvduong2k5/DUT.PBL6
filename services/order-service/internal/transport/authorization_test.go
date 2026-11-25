package transport

import (
	"net/http/httptest"
	"testing"

	"github.com/omamx/order-service/internal/app"
	"github.com/omamx/order-service/internal/domain"
)

func TestEmployeeRouteAllowlist(t *testing.T) {
	for _, test := range []struct {
		method, path string
		allow        bool
	}{
		{"GET", "/api/v1/admin/orders", true},
		{"GET", "/api/v1/admin/orders/queue", true},
		{"GET", "/api/v1/admin/orders/id", true},
		{"GET", "/api/v1/admin/orders/id/future-route", false},
		{"GET", "/api/v1/orders/id", true},
		{"GET", "/api/v1/orders/id/tracking", true},
		{"POST", "/api/v1/orders/id/cancellation-requests", true},
		{"POST", "/api/v1/admin/orders/id/hold", true},
		{"GET", "/api/v1/cart", false},
		{"POST", "/api/v1/checkout", false},
		{"GET", "/api/v1/orders", false},
		{"GET", "/api/v1/checkout-operations/id", false},
		{"GET", "/api/v1/orders/id/future-customer-only-route", false},
		{"POST", "/api/v1/orders/id/cancel", false},
		{"GET", "/api/v1/admin/orders-export", false},
	} {
		s := &app.Service{C: app.Config{AuthMode: "keycloak"}}
		req := httptest.NewRequest(test.method, test.path, nil)
		if got := authorizeGuestRequest(s, domain.Principal{Kind: domain.Employee, IAM: true}, req) == nil; got != test.allow {
			t.Errorf("principal gate: %s %s = %v", test.method, test.path, got)
		}
		if got := staffOrderRoute(test.method, test.path); got != test.allow {
			t.Errorf("%s %s = %v", test.method, test.path, got)
		}
	}
}
