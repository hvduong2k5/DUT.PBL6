package app

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/omamx/order-service/internal/domain"
)

func rsaFixture(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	k, e := rsa.GenerateKey(rand.Reader, 2048)
	if e != nil {
		t.Fatal(e)
	}
	return k
}
func jwk(k *rsa.PrivateKey, kid string) map[string]any {
	return map[string]any{"kty": "RSA", "use": "sig", "alg": "RS256", "kid": kid, "n": base64.RawURLEncoding.EncodeToString(k.N.Bytes()), "e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(k.E)).Bytes())}
}
func signedCustomer(t *testing.T, k *rsa.PrivateKey, kid, issuer, user string, edit func(jwt.MapClaims)) string {
	t.Helper()
	c := jwt.MapClaims{"iss": issuer, "aud": []string{"account", "order-api"}, "sub": uuid.NewString(), "user_id": user, "typ": "Bearer", "iat": time.Now().Add(-time.Minute).Unix(), "exp": time.Now().Add(time.Hour).Unix(), "realm_access": map[string]any{"roles": []string{"CUSTOMER"}}}
	if edit != nil {
		edit(c)
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, c)
	token.Header["kid"] = kid
	s, e := token.SignedString(k)
	if e != nil {
		t.Fatal(e)
	}
	return "Bearer " + s
}
func identityFixture(t *testing.T, server *httptest.Server) *Service {
	t.Helper()
	c := Config{Mode: "sandbox", AuthMode: "keycloak", OIDCIssuer: server.URL + "/realms/test", OIDCAudience: "order-api", OIDCJWKSURL: server.URL + "/certs", OIDCAllowLocalHTTP: true, Profile: server.URL, ProfileToken: "test-internal-profile-token"}
	v, e := NewCustomerIdentity(c)
	if e != nil {
		t.Fatal(e)
	}
	return &Service{C: c, Identity: v, Client: server.Client()}
}

func TestKeycloakValidationAndCustomerOwnership(t *testing.T) {
	k := rsaFixture(t)
	other := rsaFixture(t)
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		json.NewEncoder(w).Encode(map[string]any{"keys": []any{jwk(k, "key-1")}})
	}))
	defer server.Close()
	s := identityFixture(t, server)
	a, b := uuid.NewString(), uuid.NewString()
	token := signedCustomer(t, k, "key-1", s.C.OIDCIssuer, a, nil)
	p, e := s.Authenticate(token)
	if e != nil || p.ID != a || p.CustomerID != "" || p.Role != "CUSTOMER" {
		t.Fatalf("principal mapping: %+v %v", p, e)
	}
	order := domain.Order{Scope: p.Scope()}
	pb, e := s.Authenticate(signedCustomer(t, k, "key-1", s.C.OIDCIssuer, b, nil))
	if e != nil {
		t.Fatal(e)
	}
	if !CanRead(p, order) || CanRead(pb, order) {
		t.Fatal("cross-customer order ownership bypass")
	}
	cases := map[string]func(jwt.MapClaims){
		"wrong issuer":       func(c jwt.MapClaims) { c["iss"] = "https://attacker.invalid" },
		"wrong audience":     func(c jwt.MapClaims) { c["aud"] = "omama-api" },
		"expired":            func(c jwt.MapClaims) { c["exp"] = time.Now().Add(-time.Minute).Unix() },
		"missing expiry":     func(c jwt.MapClaims) { delete(c, "exp") },
		"future nbf":         func(c jwt.MapClaims) { c["nbf"] = time.Now().Add(time.Hour).Unix() },
		"future issued":      func(c jwt.MapClaims) { c["iat"] = time.Now().Add(time.Hour).Unix() },
		"missing issued":     func(c jwt.MapClaims) { delete(c, "iat") },
		"missing subject":    func(c jwt.MapClaims) { delete(c, "sub") },
		"missing principal":  func(c jwt.MapClaims) { delete(c, "user_id") },
		"invalid principal":  func(c jwt.MapClaims) { c["user_id"] = "not-a-uuid" },
		"nil principal":      func(c jwt.MapClaims) { c["user_id"] = uuid.Nil.String() },
		"id token":           func(c jwt.MapClaims) { c["typ"] = "ID" },
		"staff not customer": func(c jwt.MapClaims) { c["realm_access"] = map[string]any{"roles": []string{"EMPLOYEE", "ADMIN"}} },
		"self declared role": func(c jwt.MapClaims) { delete(c, "realm_access"); c["role"] = "CUSTOMER" },
	}
	for name, edit := range cases {
		t.Run(name, func(t *testing.T) {
			if _, e := s.Authenticate(signedCustomer(t, k, "key-1", s.C.OIDCIssuer, a, edit)); e == nil {
				t.Fatal("accepted invalid token")
			}
		})
	}
	for name, value := range map[string]string{
		"missing": "", "malformed": "Bearer garbage", "no bearer": token[7:],
		"wrong signature":  signedCustomer(t, other, "key-1", s.C.OIDCIssuer, a, nil),
		"unknown kid":      signedCustomer(t, k, "missing", s.C.OIDCIssuer, a, nil),
		"sandbox fallback": "Bearer " + SignToken(s.C.JWTSecret, Claims{Subject: a, Role: "CUSTOMER", Issuer: "order-sandbox", Audience: "order-api", Expires: time.Now().Add(time.Hour).Unix()}),
	} {
		t.Run(name, func(t *testing.T) {
			if _, e := s.Authenticate(value); e == nil {
				t.Fatal("accepted invalid credential")
			}
		})
	}
	if requests.Load() != 1 {
		t.Fatalf("JWKS cache/throttle: %d calls", requests.Load())
	}
	// A string audience is valid too; a role/permissions claim cannot elevate the customer.
	p, e = s.Authenticate(signedCustomer(t, k, "key-1", s.C.OIDCIssuer, a, func(c jwt.MapClaims) { c["aud"] = "order-api"; c["role"] = "ADMIN" }))
	if e != nil || p.Staff() {
		t.Fatal("customer privilege mapping failed", e)
	}
}

func TestKeycloakRotationAndUnavailableJWKS(t *testing.T) {
	k1, k2 := rsaFixture(t), rsaFixture(t)
	var stage atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if stage.Load() == 2 {
			http.Error(w, "offline", 503)
			return
		}
		key := jwk(k1, "old")
		if stage.Load() == 1 {
			key = jwk(k2, "new")
		}
		json.NewEncoder(w).Encode(map[string]any{"keys": []any{key}})
	}))
	defer server.Close()
	s := identityFixture(t, server)
	user := uuid.NewString()
	old := signedCustomer(t, k1, "old", s.C.OIDCIssuer, user, nil)
	if _, e := s.Authenticate(old); e != nil {
		t.Fatal(e)
	}
	stage.Store(1)
	s.Identity.nextFetch = time.Time{}
	if _, e := s.Authenticate(signedCustomer(t, k2, "new", s.C.OIDCIssuer, user, nil)); e != nil {
		t.Fatal("rotation", e)
	}
	if _, e := s.Authenticate(old); e == nil {
		t.Fatal("removed key still accepted")
	}
	stage.Store(2)
	s.Identity.expires = time.Time{}
	s.Identity.nextFetch = time.Time{}
	if _, e := s.Authenticate(signedCustomer(t, k2, "new", s.C.OIDCIssuer, user, nil)); e == nil {
		t.Fatal("expired cache accepted during outage")
	}
}

func TestKeycloakProfileResolution(t *testing.T) {
	principal, customer := uuid.NewString(), uuid.NewString()
	var mismatch atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/profile" || r.Header.Get("X-User-ID") != principal || r.Header.Get("X-Internal-Token") != "test-internal-profile-token" {
			http.Error(w, "unauthorized", 401)
			return
		}
		user := principal
		if mismatch.Load() {
			user = uuid.NewString()
		}
		json.NewEncoder(w).Encode(map[string]string{"id": customer, "user_id": user})
	}))
	defer server.Close()
	s := identityFixture(t, server)
	p, e := s.Resolve(context.Background(), domain.Principal{ID: principal, Role: "CUSTOMER"})
	if e != nil || p.CustomerID != customer {
		t.Fatal("profile mapping", e)
	}
	mismatch.Store(true)
	if _, e = s.Resolve(context.Background(), domain.Principal{ID: principal, Role: "CUSTOMER"}); e == nil {
		t.Fatal("accepted wrong profile owner")
	}
	s.C.Profile = ""
	if _, e = s.Resolve(context.Background(), domain.Principal{ID: principal, Role: "CUSTOMER"}); e == nil {
		t.Fatal("fell back to simulator")
	}
}

func TestKeycloakConfiguration(t *testing.T) {
	c := Config{Mode: "sandbox", AuthMode: "keycloak", OIDCIssuer: "https://identity.example/realms/omama", OIDCAudience: "order-api", Profile: "http://profile:8003", ProfileToken: "test-internal-profile-token"}
	if _, e := NewCustomerIdentity(c); e != nil {
		t.Fatal(e)
	}
	for _, edit := range []func(*Config){
		func(c *Config) { c.AuthMode = "typo" }, func(c *Config) { c.OIDCAudience = "" }, func(c *Config) { c.Profile = "" },
		func(c *Config) { c.AuthMode = "" },
		func(c *Config) { c.OIDCIssuer = "http://identity.example/realms/omama"; c.OIDCAllowLocalHTTP = true },
		func(c *Config) { c.OIDCJWKSURL = "https://user:password@identity.example/certs" },
		func(c *Config) { c.OIDCJWKSURL = "https://identity.example/certs?url=elsewhere" },
	} {
		bad := c
		edit(&bad)
		if _, e := NewCustomerIdentity(bad); e == nil {
			t.Fatalf("accepted bad configuration %+v", bad)
		}
	}
}
