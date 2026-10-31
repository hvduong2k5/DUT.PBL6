package security

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/omamx/profile-service/internal/config"
)

type OIDCIdentity struct {
	issuer, audience, jwksURL string
	client                    *http.Client
	mu                        sync.Mutex
	keys                      map[string]*rsa.PublicKey
	expires, nextFetch        time.Time
}

// NewOIDCIdentity opts into the same signed Customer token profile as Order.
// Internal callers retain their separate credential; invalid bearer never falls back.
func NewOIDCIdentity(c *config.Config) (*OIDCIdentity, error) {
	if c.AuthMode == "" || c.AuthMode == "internal" {
		if c.OIDCIssuer != "" || c.OIDCAudience != "" || c.OIDCJWKSURL != "" || c.OIDCAllowLocalHTTP {
			return nil, errors.New("OIDC config requires PROFILE_AUTH_MODE=keycloak")
		}
		return nil, nil
	}
	if c.AuthMode != "keycloak" || c.OIDCIssuer == "" || c.OIDCAudience == "" {
		return nil, errors.New("Profile requires keycloak mode, issuer and audience")
	}
	jwks := c.OIDCJWKSURL
	if jwks == "" {
		jwks = strings.TrimRight(c.OIDCIssuer, "/") + "/protocol/openid-connect/certs"
	}
	for _, raw := range []string{c.OIDCIssuer, jwks} {
		u, err := url.Parse(raw)
		if err != nil || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
			return nil, errors.New("invalid configured OIDC URL")
		}
		ip := net.ParseIP(u.Hostname())
		local := u.Hostname() == "localhost" || ip != nil && ip.IsLoopback()
		if u.Scheme != "https" && !(u.Scheme == "http" && local && c.OIDCAllowLocalHTTP) {
			return nil, errors.New("OIDC requires HTTPS; loopback HTTP is explicit TEST only")
		}
	}
	return &OIDCIdentity{issuer: c.OIDCIssuer, audience: c.OIDCAudience, jwksURL: jwks, client: &http.Client{Timeout: 2 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}

type Principal struct {
	ID   string
	Role string
}

var ErrUnauthorized = errors.New("UNAUTHENTICATED")

type customerClaims struct {
	jwt.RegisteredClaims
	UserID      string `json:"user_id"`
	Type        string `json:"typ"`
	RealmAccess struct {
		Roles []string `json:"roles"`
	} `json:"realm_access"`
}

func (v *OIDCIdentity) Authenticate(ctx context.Context, bearer string) (Principal, error) {
	if v == nil {
		return Principal{}, ErrUnauthorized
	}
	denied := ErrUnauthorized
	parts := strings.Fields(bearer)
	if len(bearer) > 8192 || len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return Principal{}, denied
	}
	var claims customerClaims
	token, err := jwt.ParseWithClaims(parts[1], &claims, func(t *jwt.Token) (any, error) {
		kid, ok := t.Header["kid"].(string)
		if !ok || kid == "" || len(kid) > 256 {
			return nil, errors.New("missing key ID")
		}
		return v.key(ctx, kid)
	}, jwt.WithValidMethods([]string{"RS256"}), jwt.WithIssuer(v.issuer), jwt.WithAudience(v.audience), jwt.WithExpirationRequired(), jwt.WithIssuedAt())
	if err != nil || !token.Valid || claims.Subject == "" || claims.Type != "Bearer" || claims.IssuedAt == nil {
		return Principal{}, denied
	}
	id, err := uuid.Parse(claims.UserID)
	if err != nil || id == uuid.Nil {
		return Principal{}, denied
	}
	for _, role := range claims.RealmAccess.Roles {
		if role == "CUSTOMER" {
			// sub belongs to Keycloak; user_id is Identity's immutable principal UUID.
			// CustomerID is deliberately left empty until Profile.Resolve proves it.
			return Principal{ID: id.String(), Role: "CUSTOMER"}, nil
		}
	}
	return Principal{}, denied
}

func (v *OIDCIdentity) key(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	now := time.Now()
	if k := v.keys[kid]; k != nil && now.Before(v.expires) {
		return k, nil
	}
	// Bound attacker-driven unknown-kid fetches and outage retries. New keys may
	// take up to 15 seconds to become visible; callers retry rather than bypass.
	if now.Before(v.nextFetch) {
		return nil, errors.New("JWKS refresh throttled")
	}
	v.nextFetch = now.Add(15 * time.Second)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.jwksURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := v.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("JWKS unavailable")
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, (1<<20)+1))
	if err != nil || len(body) > 1<<20 {
		return nil, errors.New("invalid JWKS response")
	}
	var doc struct {
		Keys []struct {
			Kty    string   `json:"kty"`
			Kid    string   `json:"kid"`
			Use    string   `json:"use"`
			Alg    string   `json:"alg"`
			N      string   `json:"n"`
			E      string   `json:"e"`
			KeyOps []string `json:"key_ops"`
		} `json:"keys"`
	}
	if json.Unmarshal(body, &doc) != nil || len(doc.Keys) > 100 {
		return nil, errors.New("invalid JWKS")
	}
	keys := map[string]*rsa.PublicKey{}
	for _, j := range doc.Keys {
		if j.Kty != "RSA" || j.Kid == "" || j.Use != "" && j.Use != "sig" || j.Alg != "" && j.Alg != "RS256" {
			continue
		}
		if len(j.KeyOps) > 0 {
			verify := false
			for _, op := range j.KeyOps {
				if op == "verify" {
					verify = true
				}
			}
			if !verify {
				continue
			}
		}
		n, en := base64.RawURLEncoding.DecodeString(j.N)
		e, ee := base64.RawURLEncoding.DecodeString(j.E)
		if en != nil || ee != nil || len(e) == 0 || len(e) > 4 {
			return nil, errors.New("invalid RSA key")
		}
		modulus := new(big.Int).SetBytes(n)
		exponent := new(big.Int).SetBytes(e).Int64()
		if modulus.BitLen() < 2048 || modulus.BitLen() > 8192 || exponent < 3 || exponent > 2147483647 || exponent%2 == 0 {
			return nil, errors.New("unsafe RSA key")
		}
		if keys[j.Kid] != nil {
			return nil, errors.New("ambiguous key ID")
		}
		keys[j.Kid] = &rsa.PublicKey{N: modulus, E: int(exponent)}
	}
	if len(keys) == 0 {
		return nil, errors.New("no usable signing keys")
	}
	v.keys = keys
	v.expires = now.Add(5 * time.Minute)
	if k := keys[kid]; k != nil {
		return k, nil
	}
	return nil, errors.New("unknown signing key")
}
