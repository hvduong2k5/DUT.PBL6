package transport

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/omamx/order-service/internal/app"
	"github.com/omamx/order-service/internal/domain"
	"github.com/redis/go-redis/v9"
)

// Real Order HTTP routing and JWT verification with loopback JWKS and a Redis
// protocol test server; this is not evidence of a live Keycloak deployment.
func TestKeycloakCartHTTPIsolation(t *testing.T) {
	k, e := rsa.GenerateKey(rand.Reader, 2048)
	if e != nil {
		t.Fatal(e)
	}
	jwks := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"keys": []any{map[string]any{"kty": "RSA", "kid": "test", "alg": "RS256", "n": base64.RawURLEncoding.EncodeToString(k.N.Bytes()), "e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(k.E)).Bytes())}}})
	}))
	defer jwks.Close()
	c := app.Config{Mode: "sandbox", AuthMode: "keycloak", OIDCIssuer: jwks.URL, OIDCAudience: "order-api", OIDCJWKSURL: jwks.URL, OIDCAllowLocalHTTP: true, Profile: "http://profile.test", ProfileToken: "test-profile-internal-token"}
	v, e := app.NewCustomerIdentity(c)
	if e != nil {
		t.Fatal(e)
	}
	rdb := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: rdb.Addr()})
	defer client.Close()
	s := &app.Service{C: c, Identity: v, Redis: client}
	a, b := uuid.NewString(), uuid.NewString()
	for id, sku := range map[string]string{a: "A-private-sku", b: "B-private-sku"} {
		body, _ := json.Marshal(domain.Cart{Revision: 1, Items: []domain.Item{{SKU: sku}}})
		rdb.Set("cart:"+(domain.Principal{ID: id, Role: "CUSTOMER"}).Scope(), string(body))
	}
	token := func(id string) string {
		v := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{"iss": jwks.URL, "aud": "order-api", "sub": uuid.NewString(), "user_id": id, "typ": "Bearer", "iat": time.Now().Add(-time.Minute).Unix(), "exp": time.Now().Add(time.Hour).Unix(), "realm_access": map[string]any{"roles": []string{"CUSTOMER"}}})
		v.Header["kid"] = "test"
		signed, err := v.SignedString(k)
		if err != nil {
			t.Fatal(err)
		}
		return "Bearer " + signed
	}
	h := Handler(s)
	for _, tc := range []struct {
		name, auth, sku string
		status          int
	}{
		{"customer A", token(a), "A-private-sku", 200},
		{"customer B with forged A headers", token(b), "B-private-sku", 200},
		{"missing bearer", "", "", 401},
		{"invalid bearer", "Bearer invalid", "", 401},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/api/v1/cart", nil)
			r.Header.Set("Authorization", tc.auth)
			r.Header.Set("X-User-ID", a)
			r.Header.Set("X-User-Role", "ADMIN")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != tc.status {
				t.Fatalf("status %d: %s", w.Code, w.Body.String())
			}
			if w.Code == 200 {
				var cart domain.Cart
				if json.Unmarshal(w.Body.Bytes(), &cart) != nil || len(cart.Items) != 1 || cart.Items[0].SKU != tc.sku {
					t.Fatalf("wrong customer cart: %s", w.Body.String())
				}
			}
		})
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("POST", "/api/v1/guest-sessions", nil))
	if w.Code != 403 {
		t.Fatalf("guest credentials issued in keycloak mode: %d", w.Code)
	}
}
