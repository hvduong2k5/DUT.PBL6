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
	"testing"
	"time"
)

func TestStaffRequiresAuthoritativeIdentityGrants(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	id, subject := uuid.NewString(), uuid.NewString()
	persona, state, boundSubject, online := "EMPLOYEE", "ACTIVE", subject, true
	grants := []map[string]string{{"permissionCode": "ORDER_VIEW", "scopeType": "WAREHOUSE", "scopeId": "A"}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/certs" {
			json.NewEncoder(w).Encode(map[string]any{"keys": []any{map[string]any{"kty": "RSA", "kid": "key", "alg": "RS256", "use": "sig", "n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()), "e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes())}}})
			return
		}
		if !online {
			w.WriteHeader(503)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"success": true, "data": map[string]any{"principalId": id, "idpSubject": boundSubject, "principalType": persona, "provisioningStatus": state, "permissionGrants": grants, "permissions": []string{"ADMIN", "ORDER_SENSITIVE_VIEW"}}})
	}))
	defer server.Close()
	v, err := NewCustomerIdentity(Config{Mode: "sandbox", AuthMode: "keycloak", OIDCIssuer: server.URL, OIDCAudience: "omama-api", OIDCJWKSURL: server.URL + "/certs", IdentityURL: server.URL, OIDCAllowLocalHTTP: true, Profile: server.URL, ProfileToken: "test-internal-profile-token"})
	if err != nil {
		t.Fatal(err)
	}
	claims := jwt.MapClaims{"iss": server.URL, "aud": "omama-api", "sub": subject, "user_id": id, "typ": "Bearer", "iat": time.Now().Add(-time.Minute).Unix(), "exp": time.Now().Add(time.Hour).Unix(), "realm_access": map[string]any{"roles": []string{"ADMIN", "CUSTOMER"}}, "permissions": []string{"ORDER_SENSITIVE_VIEW"}}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = "key"
	raw, err := token.SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	authenticate := func() {
		t.Helper()
		p, e := v.Authenticate(context.Background(), "Bearer "+raw)
		if e != nil || p.Role != "EMPLOYEE" || len(p.Grants) != 1 {
			t.Fatalf("authoritative staff grants: %+v %v", p, e)
		}
	}
	authenticate()
	svc := &Service{C: Config{AuthMode: "keycloak"}, Identity: v}
	if _, err := svc.PaymentStaff("Bearer " + raw); err == nil {
		t.Fatal("Order IAM opened finance protocol")
	}
	// Neither the union permissions nor realm ADMIN becomes an effective grant.
	grants = nil
	p, err := v.Authenticate(context.Background(), "Bearer "+raw)
	if err != nil || len(p.Grants) != 0 {
		t.Fatalf("trusted flat permissions: %+v %v", p, err)
	}
	grants = []map[string]string{{"permissionCode": "ORDER_VIEW", "scopeType": "WAREHOUSE", "scopeId": "A"}}
	for _, test := range []string{"suspended", "subject mismatch", "outage", "unknown persona"} {
		t.Run(test, func(t *testing.T) {
			switch test {
			case "suspended":
				state = "SUSPENDED"
			case "subject mismatch":
				boundSubject = uuid.NewString()
			case "outage":
				online = false
			case "unknown persona":
				persona = "SERVICE"
			}
			if _, err := v.Authenticate(context.Background(), "Bearer "+raw); err == nil {
				t.Fatal("unsafe identity accepted")
			}
			state, boundSubject, online, persona = "ACTIVE", subject, true, "EMPLOYEE"
		})
	}
	authenticate()
}
