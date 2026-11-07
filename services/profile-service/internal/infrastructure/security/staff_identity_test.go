package security

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/omamx/profile-service/internal/config"
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
			json.NewEncoder(w).Encode(map[string]any{"success": true, "data": map[string]any{"principalId": id, "idpSubject": subject, "principalType": "EMPLOYEE", "provisioningStatus": "ACTIVE", "permissionGrants": []any{map[string]any{"permissionCode": "EMPLOYEE_PROFILE_VIEW", "scopeType": "GLOBAL"}}}})
			return
		}
		w.WriteHeader(503)
	}))
	defer server.Close()
	verifier, err := NewOIDCIdentity(&config.Config{AuthMode: "keycloak", OIDCIssuer: server.URL, OIDCAudience: "omama-api", OIDCJWKSURL: server.URL + "/certs", IdentityURL: server.URL, OIDCAllowLocalHTTP: true})
	if err != nil {
		t.Fatal(err)
	}
	sign := func(edit func(jwt.MapClaims)) string {
		claims := jwt.MapClaims{"iss": server.URL, "aud": "omama-api", "sub": subject, "user_id": id, "typ": "Bearer", "iat": time.Now().Add(-time.Minute).Unix(), "exp": time.Now().Add(time.Hour).Unix(), "principal_type": "EMPLOYEE", "principal_status": "ACTIVE", "authz_version": 1,
			"permission_grants": []any{
				map[string]any{"permissionCode": "EMPLOYEE_PROFILE_VIEW", "scopeType": "GLOBAL"},
				map[string]any{"permissionCode": "EMPLOYEE_PROFILE_SENSITIVE_VIEW", "scopeType": "DEPARTMENT", "scopeId": "B"}},
			"realm_access": map[string]any{"roles": []string{"ADMIN", "CUSTOMER"}}, "permissions": []string{"EMPLOYEE_PROFILE_SENSITIVE_VIEW"}}
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

	// New tokens reflect revocation. Previously issued tokens retain their signed
	// snapshot until expiry instead of reading the current /users/me permissions.
	empty := sign(func(c jwt.MapClaims) { c["permission_grants"] = []any{} })
	p, err = verifier.Authenticate(context.Background(), empty)
	if err != nil || len(p.Grants) != 0 {
		t.Fatal("role or flat permissions bypassed empty grants", p, err)
	}
	p, err = verifier.Authenticate(context.Background(), original)
	if err != nil || len(p.Grants) != 2 {
		t.Fatal("existing JWT snapshot changed", p, err)
	}
	for name, edit := range map[string]func(jwt.MapClaims){
		"missing version":     func(c jwt.MapClaims) { delete(c, "authz_version") },
		"unknown version":     func(c jwt.MapClaims) { c["authz_version"] = 2 },
		"missing status":      func(c jwt.MapClaims) { delete(c, "principal_status") },
		"inactive":            func(c jwt.MapClaims) { c["principal_status"] = "SUSPENDED" },
		"role is not persona": func(c jwt.MapClaims) { c["principal_type"] = "ADMIN" },
		"service credential":  func(c jwt.MapClaims) { c["service"] = "order-service" },
		"missing grants":      func(c jwt.MapClaims) { delete(c, "permission_grants") },
		"invalid global scope": func(c jwt.MapClaims) {
			c["permission_grants"] = []any{map[string]any{"permissionCode": "EMPLOYEE_PROFILE_VIEW", "scopeType": "GLOBAL", "scopeId": "A"}}
		},
		"unknown scope": func(c jwt.MapClaims) {
			c["permission_grants"] = []any{map[string]any{"permissionCode": "EMPLOYEE_PROFILE_VIEW", "scopeType": "UNKNOWN", "scopeId": "A"}}
		},
		"misspelled expiry": func(c jwt.MapClaims) {
			c["permission_grants"] = []any{map[string]any{"permissionCode": "EMPLOYEE_PROFILE_VIEW", "scopeType": "GLOBAL", "valid_until": time.Now().Add(-time.Hour).Format(time.RFC3339)}}
		},
		"invalid interval": func(c jwt.MapClaims) {
			c["permission_grants"] = []any{map[string]any{"permissionCode": "EMPLOYEE_PROFILE_VIEW", "scopeType": "GLOBAL", "validFrom": time.Now().Add(time.Hour).Format(time.RFC3339), "validUntil": time.Now().Format(time.RFC3339)}}
		},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := verifier.Authenticate(context.Background(), sign(edit)); err == nil {
				t.Fatal("unsafe claims accepted")
			}
		})
	}
	timed := sign(func(c jwt.MapClaims) {
		c["permission_grants"] = []any{
			map[string]any{"permissionCode": "EMPLOYEE_PROFILE_VIEW", "scopeType": "GLOBAL", "validUntil": time.Now().Add(-time.Minute).Format(time.RFC3339)},
			map[string]any{"permissionCode": "EMPLOYEE_PROFILE_VIEW", "scopeType": "GLOBAL", "validFrom": time.Now().Add(time.Minute).Format(time.RFC3339)}}
	})
	p, err = verifier.Authenticate(context.Background(), timed)
	if err != nil || len(p.Grants) != 0 {
		t.Fatal("inactive assignment became effective", p, err)
	}
	if identityCalls.Load() != 0 {
		t.Fatal("malformed staff token fell back to Identity")
	}
	legacyEmployee.Store(true)
	legacy := sign(func(c jwt.MapClaims) {
		delete(c, "principal_type")
		delete(c, "principal_status")
		delete(c, "authz_version")
		delete(c, "permission_grants")
	})
	if _, err := verifier.Authenticate(context.Background(), legacy); err == nil {
		t.Fatal("legacy employee obtained staff access through /users/me")
	}
}
