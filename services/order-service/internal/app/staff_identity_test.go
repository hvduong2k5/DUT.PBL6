package app

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestStaffUsesSignedJWTGrantsWithoutIdentityLookup(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	id, subject := uuid.NewString(), uuid.NewString()
	var identityCalls atomic.Int32
	var legacyEmployee atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/certs" {
			json.NewEncoder(w).Encode(map[string]any{"keys": []any{map[string]any{"kty": "RSA", "kid": "key", "alg": "RS256", "use": "sig", "n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()), "e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes())}}})
			return
		}
		identityCalls.Add(1)
		if legacyEmployee.Load() {
			json.NewEncoder(w).Encode(map[string]any{"success": true, "data": map[string]any{"principalId": id, "idpSubject": subject, "principalType": "EMPLOYEE", "provisioningStatus": "ACTIVE", "permissionGrants": []any{map[string]any{"permissionCode": "ORDER_VIEW", "scopeType": "GLOBAL"}}}})
			return
		}
		w.WriteHeader(503)
	}))
	defer server.Close()
	verifier, err := NewCustomerIdentity(Config{Mode: "sandbox", AuthMode: "keycloak", OIDCIssuer: server.URL, OIDCAudience: "omama-api", OIDCJWKSURL: server.URL + "/certs", IdentityURL: server.URL, OIDCAllowLocalHTTP: true, Profile: server.URL, ProfileToken: "test-internal-profile-token"})
	if err != nil {
		t.Fatal(err)
	}
	sign := func(edit func(jwt.MapClaims)) string {
		claims := jwt.MapClaims{"iss": server.URL, "aud": "omama-api", "sub": subject, "user_id": id, "typ": "Bearer", "iat": time.Now().Add(-time.Minute).Unix(), "exp": time.Now().Add(time.Hour).Unix(),
			"permission_grants": []string{"ORDER_VIEW#WAREHOUSE#A", "ORDER_SENSITIVE_VIEW#WAREHOUSE#B"},
			"realm_access":      map[string]any{"roles": []string{"ADMIN", "CUSTOMER"}}, "permissions": []string{"ORDER_SENSITIVE_VIEW"}}
		if edit != nil {
			edit(claims)
		}
		token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
		token.Header["kid"] = "key"
		raw, e := token.SignedString(key)
		if e != nil {
			t.Fatal(e)
		}
		return "Bearer " + raw
	}
	original := sign(nil)
	p, err := verifier.Authenticate(context.Background(), original)
	if err != nil || p.Role != "EMPLOYEE" || len(p.Grants) != 2 {
		t.Fatalf("staff token: %+v %v", p, err)
	}
	if identityCalls.Load() != 0 {
		t.Fatal("staff auth called Identity")
	}
	svc := &Service{C: Config{AuthMode: "keycloak"}, Identity: verifier}
	if _, err := svc.PaymentStaff(original); err == nil {
		t.Fatal("staff JWT opened Finance protocol")
	}
	// New tokens reflect revocation. Previously issued tokens retain their signed
	// snapshot until expiry instead of reading the current /users/me permissions.
	empty := sign(func(c jwt.MapClaims) { c["permission_grants"] = []string{} })
	p, err = verifier.Authenticate(context.Background(), empty)
	if err != nil || len(p.Grants) != 0 {
		t.Fatal("role or flat permissions bypassed empty grants", p, err)
	}
	p, err = verifier.Authenticate(context.Background(), original)
	if err != nil || len(p.Grants) != 2 {
		t.Fatal("existing JWT snapshot changed", p, err)
	}
	for name, edit := range map[string]func(jwt.MapClaims){
		"service credential": func(c jwt.MapClaims) { c["service"] = "order-service" },
		"grants as objects": func(c jwt.MapClaims) {
			c["permission_grants"] = []any{map[string]any{"permissionCode": "ORDER_VIEW", "scopeType": "GLOBAL"}}
		},
		"grant without scope": func(c jwt.MapClaims) {
			c["permission_grants"] = []string{"ORDER_VIEW"}
		},
		"invalid permission code": func(c jwt.MapClaims) {
			c["permission_grants"] = []string{"#GLOBAL"}
		},
		"invalid global scope": func(c jwt.MapClaims) {
			c["permission_grants"] = []string{"ORDER_VIEW#GLOBAL#A"}
		},
		"missing scope id": func(c jwt.MapClaims) {
			c["permission_grants"] = []string{"ORDER_VIEW#WAREHOUSE"}
		},
		"unknown scope": func(c jwt.MapClaims) {
			c["permission_grants"] = []string{"ORDER_VIEW#UNKNOWN#A"}
		},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := verifier.Authenticate(context.Background(), sign(edit)); err == nil {
				t.Fatal("unsafe claims accepted")
			}
		})
	}
	if identityCalls.Load() != 0 {
		t.Fatal("malformed staff token fell back to Identity")
	}
	legacyEmployee.Store(true)
	legacy := sign(func(c jwt.MapClaims) {
		delete(c, "permission_grants")
	})
	if _, err := verifier.Authenticate(context.Background(), legacy); err == nil {
		t.Fatal("legacy employee obtained staff access through /users/me")
	}
}
