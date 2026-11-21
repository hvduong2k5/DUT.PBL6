package app

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/omamx/order-service/internal/domain"
	"github.com/omamx/order-service/internal/persistence"
)

const GuestSessionIssuer = "order-service-guest-session"
const GuestSessionAudience = "order-storefront"
const GuestSessionType = "guest-session+jwt"

func (s *Service) GuestSessions() *persistence.Payment {
	if s.GuestStore != nil {
		return s.GuestStore
	}
	return s.Payment
}

func (s *Service) GuestPaymentMethodAllowed(p domain.Principal, method string) error {
	if s.C.AuthMode != "keycloak" || p.PrincipalKind() != domain.Guest {
		return nil
	}
	if method == "COD" && s.C.GuestCODCheckoutEnabled || method == "VIETQR" && s.C.GuestVietQRCheckoutEnabled {
		return nil
	}
	return domain.E("GUEST_CHECKOUT_DISABLED", 403)
}

func IsGuestPaymentGrantCandidate(bearer string) bool {
	if len(bearer) > 8192 {
		return false
	}
	parts := strings.Split(strings.TrimPrefix(bearer, "Bearer "), ".")
	if len(parts) != 3 {
		return false
	}
	raw, _ := base64.RawURLEncoding.DecodeString(parts[1])
	var c struct {
		Issuer   string `json:"iss"`
		Audience string `json:"aud"`
	}
	if json.Unmarshal(raw, &c) != nil {
		return false
	}
	return c.Issuer == "order-service-guest" || c.Audience == "order-payment"
}

func isGuestSessionCandidate(bearer string) bool {
	if len(bearer) > 8192 {
		return false
	}
	parts := strings.Split(strings.TrimPrefix(bearer, "Bearer "), ".")
	if len(parts) != 3 {
		return false
	}
	h, _ := base64.RawURLEncoding.DecodeString(parts[0])
	b, _ := base64.RawURLEncoding.DecodeString(parts[1])
	var header struct {
		Type string `json:"typ"`
	}
	var claims struct {
		Issuer   string          `json:"iss"`
		Audience json.RawMessage `json:"aud"`
	}
	_ = json.Unmarshal(h, &header)
	_ = json.Unmarshal(b, &claims)
	return header.Type == GuestSessionType || claims.Issuer == GuestSessionIssuer || strings.Contains(string(claims.Audience), GuestSessionAudience)
}

type guestSessionClaims struct {
	jwt.RegisteredClaims
	Role  string `json:"role"`
	Scope string `json:"scope"`
}

func (s *Service) SignGuestSession(id string, issued, expires time.Time) (string, error) {
	if len(s.C.GuestSessionSecret) < 32 || s.C.GuestSessionSecret == s.C.JWTSecret {
		return "", domain.E("GUEST_SESSION_UNAVAILABLE", 503)
	}
	c := guestSessionClaims{RegisteredClaims: jwt.RegisteredClaims{Issuer: GuestSessionIssuer, Audience: jwt.ClaimStrings{GuestSessionAudience}, Subject: id, IssuedAt: jwt.NewNumericDate(issued), ExpiresAt: jwt.NewNumericDate(expires)}, Role: "GUEST", Scope: "storefront:guest"}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	t.Header["typ"] = GuestSessionType
	return t.SignedString([]byte(s.C.GuestSessionSecret))
}

// checkActive=false is reserved for merge replay: signature and expiry are
// still mandatory; only a customer-bound persisted merge result may be read.
func (s *Service) AuthenticateGuestSession(ctx context.Context, bearer string, checkActive bool) (domain.Principal, error) {
	denied := domain.E("UNAUTHENTICATED", 401)
	if len(bearer) > 8192 || !strings.HasPrefix(bearer, "Bearer ") || len(s.C.GuestSessionSecret) < 32 || s.C.GuestSessionSecret == s.C.JWTSecret {
		return domain.Principal{}, denied
	}
	var c guestSessionClaims
	t, e := jwt.ParseWithClaims(strings.TrimPrefix(bearer, "Bearer "), &c, func(t *jwt.Token) (any, error) {
		if t.Header["typ"] != GuestSessionType {
			return nil, denied
		}
		return []byte(s.C.GuestSessionSecret), nil
	}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithIssuer(GuestSessionIssuer), jwt.WithAudience(GuestSessionAudience), jwt.WithExpirationRequired(), jwt.WithIssuedAt())
	if e != nil || !t.Valid || c.IssuedAt == nil || c.ExpiresAt == nil || c.ExpiresAt.Sub(c.IssuedAt.Time) > 24*time.Hour || c.Role != "GUEST" || c.Scope != "storefront:guest" {
		return domain.Principal{}, denied
	}
	if id, e := uuid.Parse(c.Subject); e != nil || id == uuid.Nil {
		return domain.Principal{}, denied
	}
	if checkActive {
		if s.GuestSessions() == nil {
			return domain.Principal{}, denied
		}
		if e = s.GuestSessions().CheckGuestSession(ctx, c.Subject); e != nil {
			return domain.Principal{}, denied
		}
	}
	return domain.Principal{ID: c.Subject, Role: "GUEST", Kind: domain.Guest}, nil
}

// Default deny: adding a shared route does not automatically authorize guests.
// Payment grants are verified separately, only on the two order-bound reads.
func GuestRouteAllowed(method, path string) bool {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	if path == "/api/v1/cart" {
		return method == "GET"
	}
	if path == "/api/v1/cart/items" {
		return method == "POST"
	}
	if len(segments) == 5 && strings.Join(segments[:4], "/") == "api/v1/cart/items" {
		return method == "PUT" || method == "DELETE"
	}
	if path == "/api/v1/checkout/quote" || path == "/api/v1/checkout" {
		return method == "POST"
	}
	if len(segments) == 4 && strings.Join(segments[:3], "/") == "api/v1/checkout-operations" {
		return method == "GET"
	}
	if path == "/api/v1/guest-sessions/current" {
		return method == "DELETE"
	}
	if len(segments) >= 4 && strings.Join(segments[:3], "/") == "api/v1/orders" {
		if len(segments) == 4 {
			_, e := uuid.Parse(segments[3])
			return method == "GET" && e == nil
		}
		if len(segments) == 5 {
			_, e := uuid.Parse(segments[3])
			if e != nil {
				return false
			}
			return segments[4] == "tracking" && method == "GET" || segments[4] == "guest-payment-access" && (method == "POST" || method == "DELETE")
		}
	}
	return false
}

func AuthorizeRoute(p domain.Principal, method, path string) error {
	switch p.PrincipalKind() {
	case domain.Guest:
		if GuestRouteAllowed(method, path) {
			return nil
		}
	case domain.GuestPaymentGrant:
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if method == "GET" && len(parts) == 5 && strings.Join(parts[:3], "/") == "api/v1/orders" && (parts[4] == "payment" || parts[4] == "payment-instructions") {
			return nil
		}
	case domain.Customer:
		if GuestRouteAllowed(method, path) && !strings.HasPrefix(path, "/api/v1/guest-sessions/") && !strings.HasSuffix(path, "/guest-payment-access") {
			return nil
		}
		if path == "/api/v1/orders" && method == "GET" || path == "/api/v1/cart/merge" && method == "POST" {
			return nil
		}
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) == 5 && strings.Join(parts[:3], "/") == "api/v1/orders" {
			if method == "POST" && (parts[4] == "cancel" || parts[4] == "cancellation-requests") {
				return nil
			}
			if method == "GET" && (parts[4] == "payment" || parts[4] == "payment-instructions") {
				return nil
			}
		}
	case domain.Employee:
		if method == "GET" && (path == "/api/v1/admin/orders" || path == "/api/v1/admin/orders/queue") {
			return nil
		}
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if method == "POST" && len(parts) == 6 && strings.Join(parts[:4], "/") == "api/v1/admin/orders" && parts[5] == "hold" {
			return nil
		}
		if method == "GET" && len(parts) >= 4 && strings.Join(parts[:3], "/") == "api/v1/orders" && (len(parts) == 4 || len(parts) == 5 && parts[4] == "tracking") {
			return nil
		}
	}
	return domain.E("PERMISSION_DENIED", 403)
}
