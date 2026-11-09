package transport

import (
	"context"
	"crypto/rsa"
	"github.com/golang-jwt/jwt/v5"
	d "github.com/omamx/fulfillment-service/internal/domain"
	"net/http"
	"strings"
)

type Claims struct {
	UserID      string   `json:"user_id"`
	Service     string   `json:"service"`
	Permissions []string `json:"permissions"`
	Warehouses  []string `json:"warehouse_ids"`
	jwt.RegisteredClaims
}
type Auth struct {
	Key    *rsa.PublicKey
	Issuer string
}

func (a Auth) Verify(header, audience string) (d.Principal, error) {
	if a.Key == nil || a.Issuer == "" || !strings.HasPrefix(header, "Bearer ") {
		return d.Principal{}, d.Fail("UNAUTHENTICATED", 401)
	}
	c := &Claims{}
	token, e := jwt.ParseWithClaims(strings.TrimPrefix(header, "Bearer "), c, func(token *jwt.Token) (any, error) { return a.Key, nil }, jwt.WithValidMethods([]string{"RS256"}), jwt.WithIssuer(a.Issuer), jwt.WithAudience(audience), jwt.WithExpirationRequired(), jwt.WithIssuedAt())
	if e != nil || !token.Valid || c.Subject == "" {
		return d.Principal{}, d.Fail("UNAUTHENTICATED", 401)
	}
	if audience == "fulfillment-api" {
		if c.Service != "" || !d.UUID(c.UserID) {
			return d.Principal{}, d.Fail("UNAUTHENTICATED", 401)
		}
	} else {
		if c.Service != "fulfillment-service" && c.Service != "order-service" && c.Service != "inventory-service" && c.Service != "shipping-service" {
			return d.Principal{}, d.Fail("FORBIDDEN", 403)
		}
	}
	return d.Principal{ID: c.UserID, Service: c.Service, Permissions: c.Permissions, Warehouses: c.Warehouses}, nil
}

type principalKey struct{}

func principal(ctx context.Context) d.Principal {
	v, _ := ctx.Value(principalKey{}).(d.Principal)
	return v
}
func (a Auth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, e := a.Verify(r.Header.Get("Authorization"), "fulfillment-api")
		if e != nil {
			writeError(w, r, e)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), principalKey{}, p)))
	})
}
