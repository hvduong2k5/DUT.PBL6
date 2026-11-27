// Package identity provides an explicitly sandbox-only signed identity adapter.
// It is not the unresolved shared Identity/Gateway integration contract.
package identity

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/domain"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/jsonutil"
)

const SandboxIssuer = "urn:omama:traceability:sandbox"
const SandboxAudience = "traceability-service:sandbox"
const MaxTokenLifetime = 15 * time.Minute

type SandboxVerifier struct {
	keys map[string]ed25519.PublicKey
	now  func() time.Time
}

func NewSandboxVerifier(mode string, keys map[string]ed25519.PublicKey, clock func() time.Time) (*SandboxVerifier, error) {
	if mode != "sandbox" || len(keys) == 0 || clock == nil {
		return nil, errors.New("sandbox verifier requires explicit mode, keys and clock")
	}
	copied := make(map[string]ed25519.PublicKey, len(keys))
	for id, key := range keys {
		if id == "" || len(id) > 100 || len(key) != ed25519.PublicKeySize {
			return nil, errors.New("invalid sandbox verification key")
		}
		copied[id] = append(ed25519.PublicKey(nil), key...)
	}
	return &SandboxVerifier{keys: copied, now: clock}, nil
}

func (v *SandboxVerifier) Verify(ctx context.Context, token string) (domain.Principal, error) {
	fail := func() (domain.Principal, error) { return domain.Principal{}, domain.ErrUnauthenticated }
	if ctx.Err() != nil {
		return domain.Principal{}, ctx.Err()
	}
	if v == nil || len(token) > 8192 {
		return fail()
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return fail()
	}
	header, err := base64.RawURLEncoding.Strict().DecodeString(parts[0])
	if err != nil || len(header) > 1024 {
		return fail()
	}
	var h struct {
		Alg string `json:"alg"`
		Typ string `json:"typ"`
		KID string `json:"kid"`
	}
	if jsonutil.Decode(header, &h) != nil || h.Alg != "EdDSA" || h.Typ != "JWT" {
		return fail()
	}
	key, exists := v.keys[h.KID]
	if !exists {
		return fail()
	}
	signature, err := base64.RawURLEncoding.Strict().DecodeString(parts[2])
	if err != nil || !ed25519.Verify(key, []byte(parts[0]+"."+parts[1]), signature) {
		return fail()
	}
	payload, err := base64.RawURLEncoding.Strict().DecodeString(parts[1])
	if err != nil || len(payload) > 6144 {
		return fail()
	}
	var claims struct {
		Issuer      string          `json:"iss"`
		Subject     string          `json:"sub"`
		Audience    json.RawMessage `json:"aud"`
		Expires     int64           `json:"exp"`
		Issued      int64           `json:"iat"`
		NotBefore   int64           `json:"nbf"`
		Permissions []string        `json:"permissions"`
	}
	if jsonutil.Decode(payload, &claims) != nil || claims.Issuer != SandboxIssuer || !domain.ValidUUID(claims.Subject) {
		return fail()
	}
	var audience string
	if json.Unmarshal(claims.Audience, &audience) != nil {
		var audiences []string
		if json.Unmarshal(claims.Audience, &audiences) != nil || len(audiences) != 1 {
			return fail()
		}
		audience = audiences[0]
	}
	if audience != SandboxAudience {
		return fail()
	}
	now := v.now().Unix()
	if claims.Issued <= 0 || claims.Issued > now || claims.Expires <= now || claims.Expires <= claims.Issued || claims.Expires-claims.Issued > int64(MaxTokenLifetime/time.Second) || claims.NotBefore > now {
		return fail()
	}
	seen := map[string]bool{}
	for _, permission := range claims.Permissions {
		if !domain.KnownPermission(permission) || seen[permission] {
			return fail()
		}
		seen[permission] = true
	}
	return domain.Principal{ID: strings.ToLower(claims.Subject), Permissions: append([]string(nil), claims.Permissions...)}, nil
}
