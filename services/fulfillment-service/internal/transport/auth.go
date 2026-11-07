package transport

import (
	"context"
	"crypto/rsa"
	"crypto/subtle"
	"encoding/json"
	"github.com/golang-jwt/jwt/v5"
	d "github.com/omamx/fulfillment-service/internal/domain"
	"io"
	"net/http"
	"strings"
	"time"
)

type Claims struct {
	UserID      string   `json:"user_id"`
	Service     string   `json:"service"`
	Permissions []string `json:"permissions"`
	Warehouses  []string `json:"warehouse_ids"`
	jwt.RegisteredClaims
}
type Auth struct {
	Key                        *rsa.PublicKey
	Issuer                     string
	IdentityURL, StaffAudience string
	InternalTokens             map[string]string
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
	token, e := jwt.ParseWithClaims(strings.TrimPrefix(header, "Bearer "), c, func(token *jwt.Token) (any, error) { return a.Key, nil }, jwt.WithValidMethods([]string{"RS256"}), jwt.WithIssuer(a.Issuer), jwt.WithAudience(audience), jwt.WithExpirationRequired(), jwt.WithIssuedAt(), jwt.WithLeeway(5*time.Second))
	if e != nil || !token.Valid || c.Subject == "" {
		return d.Principal{}, d.Fail("UNAUTHENTICATED", 401)
	}
	if audience != "fulfillment-internal" {
		if c.Service != "" || !d.UUID(c.UserID) {
			return d.Principal{}, d.Fail("UNAUTHENTICATED", 401)
		}
		if a.IdentityURL != "" {
			return a.employee(header, c)
		}
	} else {
		if c.Service != "fulfillment-service" && c.Service != "order-service" && c.Service != "inventory-service" && c.Service != "shipping-service" {
			return d.Principal{}, d.Fail("FORBIDDEN", 403)
		}
	}
	return d.Principal{ID: c.UserID, Service: c.Service, Permissions: c.Permissions, Warehouses: c.Warehouses}, nil
}

// Reuse Identity's verified persona and existing ADMIN role/scope. Never invent
// permissions/warehouse JWT claims for the owner's current tokens.
func (a Auth) employee(header string, c *Claims) (d.Principal, error) {
	request, e := http.NewRequest("GET", strings.TrimRight(a.IdentityURL, "/")+"/api/v1/users/me", nil)
	if e != nil {
		return d.Principal{}, d.Fail("IDENTITY_UNAVAILABLE", 503)
	}
	request.Header.Set("Authorization", header)
	client := &http.Client{Timeout: 2 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, e := client.Do(request)
	if e != nil {
		return d.Principal{}, d.Fail("IDENTITY_UNAVAILABLE", 503)
	}
	defer response.Body.Close()
	if response.StatusCode >= 500 {
		return d.Principal{}, d.Fail("IDENTITY_UNAVAILABLE", 503)
	}
	if response.StatusCode != 200 {
		return d.Principal{}, d.Fail("EMPLOYEE_REQUIRED", 403)
	}
	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			ID      string `json:"principalId"`
			Subject string `json:"idpSubject"`
			Type    string `json:"principalType"`
			Status  string `json:"provisioningStatus"`
			Roles   []struct {
				Code  string `json:"roleCode"`
				Scope string `json:"scopeType"`
				ID    string `json:"scopeId"`
			} `json:"roles"`
		} `json:"data"`
	}
	raw, e := io.ReadAll(io.LimitReader(response.Body, (1<<20)+1))
	if e != nil || len(raw) > 1<<20 || json.Unmarshal(raw, &envelope) != nil || !envelope.Success {
		return d.Principal{}, d.Fail("IDENTITY_UNAVAILABLE", 503)
	}
	p := envelope.Data
	if p.ID != c.UserID || p.Subject != c.Subject || p.Type != "EMPLOYEE" || p.Status != "ACTIVE" {
		return d.Principal{}, d.Fail("EMPLOYEE_REQUIRED", 403)
	}
	actor := d.Principal{ID: p.ID}
	for _, role := range p.Roles {
		if role.Code != "ADMIN" {
			continue
		}
		if role.Scope == "GLOBAL" {
			actor.Global = true
		}
		if role.Scope == "WAREHOUSE" && role.ID != "" {
			actor.Warehouses = append(actor.Warehouses, role.ID)
		}
	}
	if !actor.Global && len(actor.Warehouses) == 0 {
		return d.Principal{}, d.Fail("FULFILLMENT_ADMIN_REQUIRED", 403)
	}
	// Service-local checks remain owned by Fulfillment. These are an ADMIN MVP
	// mapping, not new JWT authorities or approved fine-grained IAM permissions.
	actor.Permissions = []string{"fulfillment.task.read", "fulfillment.task.read_history", "fulfillment.task.assign_self", "fulfillment.task.work", "fulfillment.task.seal", "fulfillment.task.block", "fulfillment.task.resolve_block", "fulfillment.media.upload", "fulfillment.media.read_metadata", "fulfillment.media.view"}
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
