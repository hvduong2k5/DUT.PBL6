package httpapi

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestJWTStaffAuthenticatorPermissionAndAudiencePolicy(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	const issuer = "http://127.0.0.1/realms/omama"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON := func(value any) { _ = json.NewEncoder(w).Encode(value) }
		writeJSON(map[string]any{"keys": []any{map[string]any{
			"kty": "RSA", "kid": "test-key", "use": "sig", "alg": "RS256",
			"n": base64.RawURLEncoding.EncodeToString(privateKey.N.Bytes()),
			"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(privateKey.E)).Bytes()),
		}}})
	}))
	defer server.Close()

	auth, err := NewJWTStaffAuthenticator(issuer, "omama-api", server.URL)
	if err != nil {
		t.Fatal(err)
	}
	baseClaims := map[string]any{
		"iss": issuer, "sub": "keycloak-subject", "aud": []string{"order-api", "omama-api"},
		"exp": time.Now().Add(time.Minute).Unix(), "iat": time.Now().Add(-time.Second).Unix(),
		"typ": "Bearer", "principal_type": "EMPLOYEE", "user_id": "a0000000-0000-0000-0000-000000000001",
		"permission_grants": []string{"SHIPMENT_VIEW#GLOBAL", "ORDER_VIEW#GLOBAL"},
	}
	if principal, err := auth.Authenticate(context.Background(), "Bearer "+signStaffToken(t, privateKey, baseClaims)); err != nil || principal.UserID != baseClaims["user_id"] {
		t.Fatalf("valid staff token rejected: principal=%+v err=%v", principal, err)
	}

	tests := []struct {
		name      string
		mutate    func(map[string]any)
		wantError error
	}{
		{name: "wrong audience", mutate: func(c map[string]any) { c["aud"] = "order-api" }, wantError: errStaffUnauthenticated},
		{name: "wrong issuer", mutate: func(c map[string]any) { c["iss"] = "https://other.example/realm" }, wantError: errStaffUnauthenticated},
		{name: "expired", mutate: func(c map[string]any) { c["exp"] = time.Now().Add(-time.Minute).Unix() }, wantError: errStaffUnauthenticated},
		{name: "customer principal", mutate: func(c map[string]any) { c["principal_type"] = "CUSTOMER" }, wantError: errStaffUnauthenticated},
		{name: "order permission only", mutate: func(c map[string]any) { c["permission_grants"] = []string{"ORDER_VIEW#GLOBAL"} }, wantError: errStaffForbidden},
		{name: "warehouse-only shipment permission", mutate: func(c map[string]any) { c["permission_grants"] = []string{"SHIPMENT_VIEW#WAREHOUSE#WH-HCM-01"} }, wantError: errStaffForbidden},
		{name: "malformed grant", mutate: func(c map[string]any) { c["permission_grants"] = []string{"SHIPMENT_VIEW#ASSIGNED_ONLY"} }, wantError: errStaffUnauthenticated},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			claims := cloneClaims(baseClaims)
			test.mutate(claims)
			_, err := auth.Authenticate(context.Background(), "Bearer "+signStaffToken(t, privateKey, claims))
			if err != test.wantError {
				t.Fatalf("error = %v, want %v", err, test.wantError)
			}
		})
	}

	if _, err := auth.Authenticate(context.Background(), "Bearer internal-service-token"); err != errStaffUnauthenticated {
		t.Fatalf("internal token must not authenticate employee admin routes: %v", err)
	}
}

func signStaffToken(t *testing.T, key *rsa.PrivateKey, claims map[string]any) string {
	t.Helper()
	header, err := json.Marshal(map[string]string{"alg": "RS256", "kid": "test-key", "typ": "JWT"})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	input := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload)
	digest := sha256Sum([]byte(input))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest)
	if err != nil {
		t.Fatal(err)
	}
	return input + "." + base64.RawURLEncoding.EncodeToString(signature)
}

func cloneClaims(source map[string]any) map[string]any {
	copy := make(map[string]any, len(source))
	for key, value := range source {
		copy[key] = value
	}
	return copy
}

func sha256Sum(value []byte) []byte {
	digest := sha256.Sum256(value)
	return digest[:]
}
