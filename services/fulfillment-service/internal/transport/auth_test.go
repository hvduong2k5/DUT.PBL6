package transport

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"github.com/golang-jwt/jwt/v5"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestIdentityPersonaScopeAndRevocation(t *testing.T) {
	key, e := rsa.GenerateKey(rand.Reader, 2048)
	if e != nil {
		t.Fatal(e)
	}
	id := "e5f9ea69-bdc3-4ee7-9076-96c327335410"
	claims := Claims{UserID: id, Permissions: []string{"fulfillment.task.seal"}, Warehouses: []string{"forged"}, RegisteredClaims: jwt.RegisteredClaims{Issuer: "https://issuer.example", Subject: "verified-sub", Audience: jwt.ClaimStrings{"omama-api"}, IssuedAt: jwt.NewNumericDate(time.Now().Add(-time.Minute)), ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute))}}
	token, e := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(key)
	if e != nil {
		t.Fatal(e)
	}
	for _, tc := range []struct {
		name, persona, state, subject, role, scope, warehouse string
		allowed, global                                       bool
	}{
		{"employee-global", "EMPLOYEE", "ACTIVE", "verified-sub", "ADMIN", "GLOBAL", "", true, true},
		{"employee-warehouse", "EMPLOYEE", "ACTIVE", "verified-sub", "ADMIN", "WAREHOUSE", "w1", true, false},
		{"customer-even-with-admin", "CUSTOMER", "ACTIVE", "verified-sub", "ADMIN", "GLOBAL", "", false, false},
		{"suspended", "EMPLOYEE", "SUSPENDED", "verified-sub", "ADMIN", "GLOBAL", "", false, false},
		{"subject-mismatch", "EMPLOYEE", "ACTIVE", "other-sub", "ADMIN", "GLOBAL", "", false, false},
		{"no-admin-cannot-use-forged-jwt-permissions", "EMPLOYEE", "ACTIVE", "verified-sub", "OTHER", "GLOBAL", "", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/v1/users/me" || r.Header.Get("Authorization") != "Bearer "+token {
					t.Error("wrong Identity request")
				}
				json.NewEncoder(w).Encode(map[string]any{"success": true, "data": map[string]any{"principalId": id, "idpSubject": tc.subject, "principalType": tc.persona, "provisioningStatus": tc.state, "roles": []any{map[string]any{"roleCode": tc.role, "scopeType": tc.scope, "scopeId": tc.warehouse}}}})
			}))
			defer server.Close()
			auth := Auth{Key: &key.PublicKey, Issuer: claims.Issuer, IdentityURL: server.URL}
			p, e := auth.Verify("Bearer "+token, "omama-api")
			if (e == nil) != tc.allowed {
				t.Fatalf("allowed=%v err=%v", tc.allowed, e)
			}
			if tc.allowed && (p.Global != tc.global || !p.Can("fulfillment.task.seal", "w1") || p.Can("fulfillment.task.assign_other", "w1") || (!tc.global && p.Can("fulfillment.task.seal", "other"))) {
				t.Fatal("incorrect service-owned scope mapping")
			}
			if tc.allowed {
				server.Close()
				if _, e = auth.Verify("Bearer "+token, "omama-api"); e == nil {
					t.Fatal("Identity outage allowed cached staff")
				}
			}
		})
	}
}
func TestInternalReadCredentialIsSeparateFromStaff(t *testing.T) {
	auth := Auth{InternalTokens: map[string]string{"order-service": strings.Repeat("o", 40)}}
	p, e := auth.Verify("Bearer "+strings.Repeat("o", 40), "fulfillment-internal")
	if e != nil || p.Service != "order-service" || p.Can("fulfillment.task.seal", "w1") {
		t.Fatal("internal read became staff")
	}
	for _, header := range []string{"", "Bearer wrong", "Bearer " + strings.Repeat("i", 40)} {
		if _, e = auth.Verify(header, "fulfillment-internal"); e == nil {
			t.Fatal("wrong peer allowed")
		}
	}
	if _, e = auth.Verify("Bearer "+strings.Repeat("o", 40), "omama-api"); e == nil {
		t.Fatal("internal credential entered staff API")
	}
}
