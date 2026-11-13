package transport

import (
	"context"
	"crypto/rsa"
	"crypto/subtle"
	"github.com/golang-jwt/jwt/v5"
	d "github.com/omamx/fulfillment-service/internal/domain"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode"
)

const jwtClockSkew = 30 * time.Second

type Claims struct {
	UserID           string   `json:"user_id"`
	Service          string   `json:"service"`
	PermissionGrants []string `json:"permission_grants"`
	jwt.RegisteredClaims
}
type Auth struct {
	Key            *rsa.PublicKey
	Issuer         string
	StaffAudience  string
	InternalTokens map[string]string
}

func (a Auth) Verify(header, audience string) (d.Principal, error) {
	if len(header) > 8192 {
		return d.Principal{}, d.Fail("UNAUTHENTICATED", 401)
	}
	if audience == "fulfillment-internal" && len(a.InternalTokens) > 0 {
		for service, token := range a.InternalTokens {
			if len(token) >= 32 && subtle.ConstantTimeCompare([]byte(header), []byte("Bearer "+token)) == 1 {
				return d.Principal{Service: service}, nil
			}
		}
		return d.Principal{}, d.Fail("UNAUTHENTICATED", 401)
	}
	if a.Key == nil || a.Issuer == "" || !strings.HasPrefix(header, "Bearer ") {
		return d.Principal{}, d.Fail("UNAUTHENTICATED", 401)
	}
	c := &Claims{}
	token, e := jwt.ParseWithClaims(strings.TrimPrefix(header, "Bearer "), c, func(token *jwt.Token) (any, error) { return a.Key, nil }, jwt.WithValidMethods([]string{"RS256"}), jwt.WithIssuer(a.Issuer), jwt.WithAudience(audience), jwt.WithExpirationRequired(), jwt.WithIssuedAt(), jwt.WithLeeway(jwtClockSkew))
	if e != nil || !token.Valid || c.Subject == "" {
		return d.Principal{}, d.Fail("UNAUTHENTICATED", 401)
	}
	if audience != "fulfillment-internal" {
		if c.Service != "" || !d.UUID(c.UserID) {
			return d.Principal{}, d.Fail("UNAUTHENTICATED", 401)
		}
		return a.staff(c)
	}
	if c.Service != "fulfillment-service" && c.Service != "order-service" && c.Service != "inventory-service" && c.Service != "shipping-service" {
		return d.Principal{}, d.Fail("FORBIDDEN", 403)
	}
	return d.Principal{ID: c.UserID, Service: c.Service}, nil
}

var staffPermissionCode = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_.:-]{0,159}$`)

func authorizationValue(value string) bool {
	if value == "" || len(value) > 160 || strings.TrimSpace(value) != value {
		return false
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}

// Staff authorization comes from the token's permission_grants claim —
// "PERM#SCOPE_TYPE" for GLOBAL grants, "PERM#SCOPE_TYPE#SCOPE_ID" otherwise —
// projected onto the token by Identity's permission sync, the same contract
// Profile and Order consume. A token that carries the claim (even as an
// empty list) is a staff token and is authorized from this signed snapshot
// alone; a token without it is not staff and is rejected here. There is no
// per-request /users/me lookup anymore. Only FULFILLMENT_* grants take
// effect, and their scopes collapse into the principal's Global/Warehouses
// sets exactly as the previous ADMIN role/scope mapping did.
func (a Auth) staff(c *Claims) (d.Principal, error) {
	if c.PermissionGrants == nil {
		return d.Principal{}, d.Fail("EMPLOYEE_REQUIRED", 403)
	}
	if len(c.PermissionGrants) > 128 {
		return d.Principal{}, d.Fail("UNAUTHENTICATED", 401)
	}
	actor := d.Principal{ID: c.UserID}
	seenPermissions := map[string]bool{}
	seenWarehouses := map[string]bool{}
	for _, raw := range c.PermissionGrants {
		parts := strings.Split(raw, "#")
		if len(parts) < 2 || len(parts) > 3 || !staffPermissionCode.MatchString(parts[0]) {
			return d.Principal{}, d.Fail("UNAUTHENTICATED", 401)
		}
		code, scopeType := parts[0], parts[1]
		var scopeID string
		switch scopeType {
		case "GLOBAL":
			if len(parts) != 2 {
				return d.Principal{}, d.Fail("UNAUTHENTICATED", 401)
			}
		case "WAREHOUSE", "DEPARTMENT":
			if len(parts) != 3 || !authorizationValue(parts[2]) {
				return d.Principal{}, d.Fail("UNAUTHENTICATED", 401)
			}
			scopeID = parts[2]
		default:
			return d.Principal{}, d.Fail("UNAUTHENTICATED", 401)
		}
		if !strings.HasPrefix(code, "FULFILLMENT_") {
			continue
		}
		if !seenPermissions[code] {
			seenPermissions[code] = true
			actor.Permissions = append(actor.Permissions, code)
		}
		switch scopeType {
		case "GLOBAL":
			actor.Global = true
		case "WAREHOUSE":
			if !seenWarehouses[scopeID] {
				seenWarehouses[scopeID] = true
				actor.Warehouses = append(actor.Warehouses, scopeID)
			}
		}
	}
	if len(actor.Permissions) == 0 {
		return d.Principal{}, d.Fail("FULFILLMENT_ADMIN_REQUIRED", 403)
	}
	return actor, nil
}

type principalKey struct{}

func principal(ctx context.Context) d.Principal {
	v, _ := ctx.Value(principalKey{}).(d.Principal)
	return v
}
func (a Auth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		audience := a.StaffAudience
		if audience == "" {
			audience = "fulfillment-api"
		}
		p, e := a.Verify(r.Header.Get("Authorization"), audience)
		if e != nil {
			writeError(w, r, e)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), principalKey{}, p)))
	})
}
