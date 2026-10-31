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

func TestProfileSignedIdentity(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	var jwksCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jwksCalls.Add(1)
		json.NewEncoder(w).Encode(map[string]any{"keys": []any{map[string]any{"kty": "RSA", "kid": "key", "alg": "RS256", "use": "sig", "n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()), "e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes())}}})
	}))
	defer server.Close()
	cfg := &config.Config{AuthMode: "keycloak", OIDCIssuer: server.URL, OIDCAudience: "omama-api", OIDCJWKSURL: server.URL, OIDCAllowLocalHTTP: true}
	verifier, err := NewOIDCIdentity(cfg)
	if err != nil {
		t.Fatal(err)
	}
	principal := uuid.NewString()
	signed := func(edit func(jwt.MapClaims)) string {
		claims := jwt.MapClaims{"iss": server.URL, "aud": []string{"omama-api", "order-api"}, "sub": uuid.NewString(), "user_id": principal, "typ": "Bearer", "iat": time.Now().Add(-time.Minute).Unix(), "exp": time.Now().Add(time.Minute).Unix(), "realm_access": map[string]any{"roles": []string{"CUSTOMER"}}}
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
	p, err := verifier.Authenticate(context.Background(), signed(nil))
	if err != nil || p.ID != principal || p.Role != "CUSTOMER" {
		t.Fatal(p, err)
	}
	for name, edit := range map[string]func(jwt.MapClaims){
		"wrong audience": func(c jwt.MapClaims) { c["aud"] = "other-api" },
		"wrong issuer":   func(c jwt.MapClaims) { c["iss"] = "https://other.invalid" },
		"expired":        func(c jwt.MapClaims) { c["exp"] = time.Now().Add(-time.Minute).Unix() },
		"no expiry":      func(c jwt.MapClaims) { delete(c, "exp") },
		"no principal":   func(c jwt.MapClaims) { delete(c, "user_id") },
		"nil principal":  func(c jwt.MapClaims) { c["user_id"] = uuid.Nil.String() },
		"id token":       func(c jwt.MapClaims) { c["typ"] = "ID" },
		"unscoped staff": func(c jwt.MapClaims) { c["realm_access"] = map[string]any{"roles": []string{"ADMIN", "EMPLOYEE"}} },
	} {
		t.Run(name, func(t *testing.T) {
			if _, e := verifier.Authenticate(context.Background(), signed(edit)); e == nil {
				t.Fatal("invalid identity accepted")
			}
		})
	}
	if jwksCalls.Load() != 1 {
		t.Fatal("JWKS cache not reused", jwksCalls.Load())
	}
	cfg.OIDCAllowLocalHTTP = false
	if _, err = NewOIDCIdentity(cfg); err == nil {
		t.Fatal("plaintext accepted without explicit TEST flag")
	}
	cfg.AuthMode = "internal"
	if _, err = NewOIDCIdentity(cfg); err == nil {
		t.Fatal("misconfigured OIDC silently downgraded")
	}
}
