package app

import (
	"context"
	"github.com/google/uuid"
	"github.com/omamx/order-service/internal/domain"
	"strings"
	"testing"
	"time"
)

func TestGuestEndpointAuthorizationMatrix(t *testing.T) {
	id := uuid.NewString()
	for _, tc := range []struct {
		kind         domain.PrincipalKind
		method, path string
		allowed      bool
	}{
		{domain.Guest, "GET", "/api/v1/cart", true},
		{domain.Guest, "POST", "/api/v1/cart/items", true},
		{domain.Guest, "PUT", "/api/v1/cart/items/A", true},
		{domain.Guest, "DELETE", "/api/v1/cart/items/A", true},
		{domain.Guest, "POST", "/api/v1/checkout/quote", true},
		{domain.Guest, "POST", "/api/v1/checkout", true},
		{domain.Guest, "GET", "/api/v1/orders/" + id, true},
		{domain.Guest, "GET", "/api/v1/orders/" + id + "/tracking", true},
		{domain.Guest, "GET", "/api/v1/orders/" + id + "/traceability", true},
		{domain.Customer, "GET", "/api/v1/orders/" + id + "/traceability", true},
		{domain.GuestPaymentGrant, "GET", "/api/v1/orders/" + id + "/traceability", false},
		{domain.Employee, "GET", "/api/v1/orders/" + id + "/traceability", false},
		{domain.Guest, "DELETE", "/api/v1/guest-sessions/current", true},
		{domain.Guest, "POST", "/api/v1/orders/" + id + "/guest-payment-access", true},
		{domain.Guest, "GET", "/api/v1/orders", false},
		{domain.Guest, "GET", "/api/v1/admin/orders/" + id, false},
		{domain.Customer, "GET", "/api/v1/admin/orders/" + id, false},
		{domain.GuestPaymentGrant, "GET", "/api/v1/admin/orders/" + id, false},
		{domain.Employee, "GET", "/api/v1/admin/orders/" + id, true},
		{domain.Employee, "POST", "/api/v1/orders/" + id + "/cancellation-requests", true},
		{domain.Guest, "GET", "/api/v1/profile", false},
		{domain.Guest, "POST", "/api/v1/admin/orders/" + id + "/cancel", false},
		{domain.Guest, "GET", "/api/v1/orders/" + id + "/payment", false},
		{domain.Guest, "POST", "/api/v1/cart/merge", false},
		{domain.GuestPaymentGrant, "GET", "/api/v1/orders/" + id + "/payment", true},
		{domain.GuestPaymentGrant, "GET", "/api/v1/orders/" + id + "/payment-instructions", true},
		{domain.GuestPaymentGrant, "POST", "/api/v1/orders/" + id + "/guest-payment-access", false},
		{domain.GuestPaymentGrant, "GET", "/api/v1/cart", false},
		{domain.GuestPaymentGrant, "POST", "/api/v1/admin/payments/refunds/" + id + "/start", false},
	} {
		if got := AuthorizeRoute(domain.Principal{Kind: tc.kind}, tc.method, tc.path) == nil; got != tc.allowed {
			t.Errorf("%s %s %s allowed=%v", tc.kind, tc.method, tc.path, got)
		}
	}
}

func TestGuestJWTTrustAndIndependentFlags(t *testing.T) {
	s := &Service{C: Config{AuthMode: "keycloak", JWTSecret: strings.Repeat("s", 32), GuestSessionSecret: strings.Repeat("g", 32)}}
	now := time.Now()
	token, e := s.SignGuestSession(uuid.NewString(), now, now.Add(time.Hour))
	if e != nil {
		t.Fatal(e)
	}
	guest, e := s.AuthenticateGuestSession(context.Background(), "Bearer "+token, false)
	if e != nil || guest.PrincipalKind() != domain.Guest {
		t.Fatal(guest, e)
	}
	// RS256-only CustomerIdentity never accepts the valid HS256 guest token.
	identity := &CustomerIdentity{}
	if _, e = identity.Authenticate(context.Background(), "Bearer "+token); e == nil {
		t.Fatal("guest token cross-validated as customer")
	}
	for _, method := range []string{"COD", "VIETQR"} {
		if s.GuestPaymentMethodAllowed(guest, method) == nil {
			t.Fatal("flag-off checkout enabled", method)
		}
	}
	s.C.GuestCODCheckoutEnabled = true
	if s.GuestPaymentMethodAllowed(guest, "COD") != nil || s.GuestPaymentMethodAllowed(guest, "VIETQR") == nil {
		t.Fatal("guest method flags not independent")
	}
	s.C.AuthMode = "sandbox"
	if s.GuestPaymentMethodAllowed(guest, "VIETQR") != nil {
		t.Fatal("sandbox behavior changed")
	}
}
