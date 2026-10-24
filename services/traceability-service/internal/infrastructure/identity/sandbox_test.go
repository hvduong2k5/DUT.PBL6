package identity

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/domain"
)

func signed(key ed25519.PrivateKey, header, payload any) string {
	h, _ := json.Marshal(header)
	p, _ := json.Marshal(payload)
	return signBytes(key, h, p)
}
func signBytes(key ed25519.PrivateKey, header, payload []byte) string {
	data := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload)
	return data + "." + base64.RawURLEncoding.EncodeToString(ed25519.Sign(key, []byte(data)))
}

func TestSandboxIdentity(t *testing.T) {
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)
	verifier, err := NewSandboxVerifier("sandbox", map[string]ed25519.PublicKey{"test": public}, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	header := map[string]any{"alg": "EdDSA", "typ": "JWT", "kid": "test"}
	valid := func() map[string]any {
		return map[string]any{"iss": SandboxIssuer, "aud": SandboxAudience, "sub": "ABCDEFAB-2222-4222-8222-222222222222", "iat": now.Unix(), "exp": now.Add(10 * time.Minute).Unix(), "permissions": []string{domain.Create, domain.ReadInternal}}
	}
	p, err := verifier.Verify(context.Background(), signed(private, header, valid()))
	if err != nil || p.ID != "abcdefab-2222-4222-8222-222222222222" || !p.Can(domain.Create) || p.Can(domain.Approve) {
		t.Fatalf("principal: %+v %v", p, err)
	}
	cases := map[string]func(map[string]any){
		"wrong issuer":         func(c map[string]any) { c["iss"] = "production" },
		"wrong audience":       func(c map[string]any) { c["aud"] = "order-service" },
		"expired":              func(c map[string]any) { c["exp"] = now.Unix() },
		"future iat":           func(c map[string]any) { c["iat"] = now.Add(time.Second).Unix() },
		"future nbf":           func(c map[string]any) { c["nbf"] = now.Add(time.Second).Unix() },
		"long lifetime":        func(c map[string]any) { c["exp"] = now.Add(time.Hour).Unix() },
		"missing iat":          func(c map[string]any) { delete(c, "iat") },
		"missing expiry":       func(c map[string]any) { delete(c, "exp") },
		"nil subject":          func(c map[string]any) { c["sub"] = "00000000-0000-0000-0000-000000000000" },
		"role bypass":          func(c map[string]any) { c["role"] = "ADMIN" },
		"unknown permission":   func(c map[string]any) { c["permissions"] = []string{"ADMIN"} },
		"duplicate permission": func(c map[string]any) { c["permissions"] = []string{domain.Create, domain.Create} },
		"multiple audiences":   func(c map[string]any) { c["aud"] = []string{SandboxAudience, "other"} },
		"fraction expiry":      func(c map[string]any) { c["exp"] = float64(now.Unix()) + 0.5 },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			claims := valid()
			mutate(claims)
			if _, err := verifier.Verify(context.Background(), signed(private, header, claims)); !errors.Is(err, domain.ErrUnauthenticated) {
				t.Fatalf("unsafe token accepted: %v", err)
			}
		})
	}
	claims := valid()
	claims["aud"] = []string{SandboxAudience}
	if _, err = verifier.Verify(context.Background(), signed(private, header, claims)); err != nil {
		t.Fatal("single audience array rejected", err)
	}
	for _, h := range []map[string]any{{"alg": "none", "typ": "JWT", "kid": "test"}, {"alg": "HS256", "typ": "JWT", "kid": "test"}, {"alg": "EdDSA", "typ": "JWT", "kid": "unknown"}, {"alg": "EdDSA", "typ": "JWT", "kid": "test", "jku": "https://attacker.test/keys"}} {
		if _, err = verifier.Verify(context.Background(), signed(private, h, valid())); !errors.Is(err, domain.ErrUnauthenticated) {
			t.Fatal("unsafe header accepted")
		}
	}
	_, other, _ := ed25519.GenerateKey(rand.Reader)
	if _, err = verifier.Verify(context.Background(), signed(other, header, valid())); !errors.Is(err, domain.ErrUnauthenticated) {
		t.Fatal("wrong signature accepted")
	}
	h, _ := json.Marshal(header)
	if _, err = verifier.Verify(context.Background(), signBytes(private, h, []byte(`{"iss":"wrong","ISS":"`+SandboxIssuer+`"}`))); !errors.Is(err, domain.ErrUnauthenticated) {
		t.Fatal("duplicate claims accepted")
	}
	for _, raw := range []string{"", "not-a-jwt", "a.b.c", "...."} {
		if _, err = verifier.Verify(context.Background(), raw); !errors.Is(err, domain.ErrUnauthenticated) {
			t.Fatal("malformed token accepted")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err = verifier.Verify(ctx, signed(private, header, valid())); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestSandboxCannotBecomeIntegration(t *testing.T) {
	public, _, _ := ed25519.GenerateKey(rand.Reader)
	for _, mode := range []string{"", "integration", "production"} {
		if _, err := NewSandboxVerifier(mode, map[string]ed25519.PublicKey{"test": public}, time.Now); err == nil {
			t.Fatal("sandbox verifier activated outside sandbox")
		}
	}
	if _, err := NewSandboxVerifier("sandbox", nil, time.Now); err == nil {
		t.Fatal("missing keys accepted")
	}
	if _, err := NewSandboxVerifier("sandbox", map[string]ed25519.PublicKey{"test": {1}}, time.Now); err == nil {
		t.Fatal("invalid key accepted")
	}
	var verifier *SandboxVerifier
	if _, err := verifier.Verify(context.Background(), "x"); !errors.Is(err, domain.ErrUnauthenticated) {
		t.Fatal(err)
	}
}
