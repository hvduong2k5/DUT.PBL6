package app

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/omamx/order-service/internal/persistence"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// F-42: an IAM (Keycloak) staff token carrying payment grants must produce a
// payment actor bound to the configured SePay connection/receiver, with the
// internal payment.* codes mapped 1:1 from the IAM grants (fail closed).
func TestPaymentStaffIAMGrants(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	id, subject := uuid.NewString(), uuid.NewString()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/certs" {
			json.NewEncoder(w).Encode(map[string]any{"keys": []any{map[string]any{"kty": "RSA", "kid": "key", "alg": "RS256", "use": "sig", "n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()), "e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes())}}})
			return
		}
		w.WriteHeader(503)
	}))
	defer server.Close()
	verifier, err := NewCustomerIdentity(Config{Mode: "sandbox", AuthMode: "keycloak", OIDCIssuer: server.URL, OIDCAudience: "omama-api", OIDCJWKSURL: server.URL + "/certs", IdentityURL: server.URL, OIDCAllowLocalHTTP: true, Profile: server.URL, ProfileToken: "test-internal-profile-token"})
	if err != nil {
		t.Fatal(err)
	}
	sign := func(grants []string, authTime time.Time) string {
		claims := jwt.MapClaims{"iss": server.URL, "aud": "omama-api", "sub": subject, "user_id": id, "typ": "Bearer", "iat": time.Now().Add(-time.Minute).Unix(), "exp": time.Now().Add(time.Hour).Unix()}
		if grants != nil {
			claims["permission_grants"] = grants
		}
		if !authTime.IsZero() {
			claims["auth_time"] = authTime.Unix()
		}
		token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
		token.Header["kid"] = "key"
		raw, e := token.SignedString(key)
		if e != nil {
			t.Fatal(e)
		}
		return "Bearer " + raw
	}
	svc := &Service{C: Config{AuthMode: "keycloak"}, Identity: verifier, Payment: &persistence.Payment{Connection: "conn-test", Receiver: "recv-test"}}
	fresh := time.Now().Add(-5 * time.Minute)
	stale := time.Now().Add(-time.Hour)

	t.Run("read grant maps and binds the configured connection", func(t *testing.T) {
		actor, e := svc.PaymentStaff(sign([]string{"PAYMENT_RECONCILIATION_VIEW#GLOBAL"}, fresh))
		if e != nil {
			t.Fatal(e)
		}
		if actor.ID != id || actor.Connection != "conn-test" || len(actor.Receivers) != 1 || actor.Receivers[0] != "recv-test" {
			t.Fatalf("actor not bound as configured: %+v", actor)
		}
		if !actor.Permissions["payment.reconciliation.read"] || actor.Permissions["payment.reconciliation.manage"] || actor.Permissions["payment.refund.read"] {
			t.Fatalf("unexpected permission mapping: %v", actor.Permissions)
		}
	})
	t.Run("resolve grant with fresh auth_time maps manage", func(t *testing.T) {
		actor, e := svc.PaymentStaff(sign([]string{"PAYMENT_RECONCILIATION_VIEW#GLOBAL", "PAYMENT_CASE_RESOLVE#GLOBAL"}, fresh))
		if e != nil || !actor.Permissions["payment.reconciliation.manage"] {
			t.Fatalf("manage missing: %v %v", actor.Permissions, e)
		}
	})
	t.Run("resolve grant with stale auth_time keeps read only", func(t *testing.T) {
		actor, e := svc.PaymentStaff(sign([]string{"PAYMENT_RECONCILIATION_VIEW#GLOBAL", "PAYMENT_CASE_RESOLVE#GLOBAL"}, stale))
		if e != nil {
			t.Fatal(e)
		}
		if actor.Permissions["payment.reconciliation.manage"] {
			t.Fatal("stale auth_time must not map the write tier")
		}
	})
	t.Run("resolve grant without auth_time gets no write tier", func(t *testing.T) {
		if _, e := svc.PaymentStaff(sign([]string{"PAYMENT_CASE_RESOLVE#GLOBAL"}, time.Time{})); e == nil {
			t.Fatal("write-only grant without auth_time must be forbidden")
		}
	})
	t.Run("PAYMENT_VIEW alone never authorizes the API", func(t *testing.T) {
		if _, e := svc.PaymentStaff(sign([]string{"PAYMENT_VIEW#GLOBAL"}, fresh)); e == nil {
			t.Fatal("PAYMENT_VIEW is UI-only and must not open the payment API")
		}
	})
	t.Run("refund read needs the sensitive grant as well", func(t *testing.T) {
		if _, e := svc.PaymentStaff(sign([]string{"PAYMENT_REFUND_VIEW#GLOBAL"}, fresh)); e == nil {
			t.Fatal("refund read without PAYMENT_SENSITIVE_VIEW must be forbidden")
		}
		actor, e := svc.PaymentStaff(sign([]string{"PAYMENT_REFUND_VIEW#GLOBAL", "PAYMENT_SENSITIVE_VIEW#GLOBAL"}, fresh))
		if e != nil || !actor.Permissions["payment.refund.read"] {
			t.Fatalf("refund read missing: %v %v", actor.Permissions, e)
		}
	})
	t.Run("non-global payment grant does not map", func(t *testing.T) {
		if _, e := svc.PaymentStaff(sign([]string{"PAYMENT_RECONCILIATION_VIEW#WAREHOUSE#A"}, fresh)); e == nil {
			t.Fatal("warehouse-scoped payment grant must not map")
		}
	})
	t.Run("customer token without grants is forbidden", func(t *testing.T) {
		if _, e := svc.PaymentStaff(sign(nil, fresh)); e == nil {
			t.Fatal("token without permission_grants must not open the payment API")
		}
	})
	t.Run("unconfigured payment store is forbidden", func(t *testing.T) {
		bare := &Service{C: Config{AuthMode: "keycloak"}, Identity: verifier}
		if _, e := bare.PaymentStaff(sign([]string{"PAYMENT_RECONCILIATION_VIEW#GLOBAL"}, fresh)); e == nil {
			t.Fatal("missing payment configuration must fail closed")
		}
	})
}
