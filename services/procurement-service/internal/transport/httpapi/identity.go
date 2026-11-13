package httpapi

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
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

	"github.com/google/uuid"
)

type IdentityConfig struct {
	Issuer, Audience, JWKSURL string
	AllowLocalHTTP            bool
}

type authConfig struct {
	issuer, audience, jwksURL string
}

type employeeClaims struct {
	Issuer        string          `json:"iss"`
	Subject       string          `json:"sub"`
	Audience      json.RawMessage `json:"aud"`
	ExpiresAt     int64           `json:"exp"`
	IssuedAt      int64           `json:"iat"`
	NotBefore     int64           `json:"nbf,omitempty"`
	UserID        string          `json:"user_id"`
	Type          string          `json:"typ"`
	PrincipalType string          `json:"principal_type"`
	Grants        []string        `json:"permission_grants"`
}

type permissionGrant struct {
	code, scopeType, scopeID string
}

type employeePrincipal struct {
	id     string
	grants []permissionGrant
}

func (p employeePrincipal) allows(code, scopeType, scopeID string) bool {
	for _, grant := range p.grants {
		if grant.code != code {
			continue
		}
		if grant.scopeType == "GLOBAL" && scopeType == "GLOBAL" {
			return true
		}
		if grant.scopeType == scopeType && grant.scopeID != "" && grant.scopeID == scopeID {
			return true
		}
	}
	return false
}

type jwtVerifier struct {
	config    authConfig
	client    *http.Client
	mu        sync.Mutex
	keys      map[string]*rsa.PublicKey
	until     time.Time
	lastFetch time.Time
}

func newJWTVerifier(config IdentityConfig) (*jwtVerifier, error) {
	if config.Issuer == "" || config.Audience == "" {
		return nil, errors.New("OIDC issuer and audience are required")
	}
	if config.JWKSURL == "" {
		config.JWKSURL = strings.TrimRight(config.Issuer, "/") + "/protocol/openid-connect/certs"
	}
	for _, raw := range []string{config.Issuer, config.JWKSURL} {
		u, err := url.Parse(raw)
		if err != nil || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
			return nil, errors.New("invalid OIDC URL")
		}
		ip := net.ParseIP(u.Hostname())
		local := u.Hostname() == "localhost" || ip != nil && ip.IsLoopback()
		if u.Scheme != "https" && !(u.Scheme == "http" && local && config.AllowLocalHTTP) {
			return nil, errors.New("OIDC URLs require HTTPS; loopback HTTP is only allowed when explicitly enabled")
		}
	}
	return &jwtVerifier{config: authConfig{issuer: config.Issuer, audience: config.Audience, jwksURL: config.JWKSURL}, client: &http.Client{Timeout: 3 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}, keys: map[string]*rsa.PublicKey{}}, nil
}

func (v *jwtVerifier) authenticate(ctx context.Context, header string) (employeePrincipal, int) {
	unauthorized := employeePrincipal{}
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || len(parts[1]) > 8192 {
		return unauthorized, http.StatusUnauthorized
	}
	compact := strings.Split(parts[1], ".")
	if len(compact) != 3 {
		return unauthorized, http.StatusUnauthorized
	}
	headerJSON, err := base64.RawURLEncoding.DecodeString(compact[0])
	if err != nil {
		return unauthorized, http.StatusUnauthorized
	}
	var tokenHeader struct {
		Algorithm string `json:"alg"`
		KeyID     string `json:"kid"`
	}
	if json.Unmarshal(headerJSON, &tokenHeader) != nil || tokenHeader.Algorithm != "RS256" || tokenHeader.KeyID == "" || len(tokenHeader.KeyID) > 256 {
		return unauthorized, http.StatusUnauthorized
	}
	key, err := v.key(ctx, tokenHeader.KeyID)
	if err != nil {
		return unauthorized, http.StatusUnauthorized
	}
	signature, err := base64.RawURLEncoding.DecodeString(compact[2])
	if err != nil {
		return unauthorized, http.StatusUnauthorized
	}
	digest := sha256.Sum256([]byte(compact[0] + "." + compact[1]))
	if rsa.VerifyPKCS1v15(key, crypto.SHA256, digest[:], signature) != nil {
		return unauthorized, http.StatusUnauthorized
	}
	claimsJSON, err := base64.RawURLEncoding.DecodeString(compact[1])
	if err != nil {
		return unauthorized, http.StatusUnauthorized
	}
	claims := employeeClaims{}
	if json.Unmarshal(claimsJSON, &claims) != nil || claims.Subject == "" || claims.Type != "Bearer" || claims.Issuer != v.config.issuer || claims.ExpiresAt == 0 || claims.IssuedAt == 0 || !hasAudience(claims.Audience, v.config.audience) {
		return unauthorized, http.StatusUnauthorized
	}
	now := time.Now().Unix()
	if claims.ExpiresAt < now-5 || claims.IssuedAt > now+5 || claims.NotBefore != 0 && claims.NotBefore > now+5 {
		return unauthorized, http.StatusUnauthorized
	}
	if claims.PrincipalType == "CUSTOMER" {
		return unauthorized, http.StatusForbidden
	}
	if claims.PrincipalType != "EMPLOYEE" {
		return unauthorized, http.StatusUnauthorized
	}
	id, err := uuid.Parse(claims.UserID)
	if err != nil || id == uuid.Nil || claims.Grants == nil || len(claims.Grants) > 128 {
		return unauthorized, http.StatusUnauthorized
	}
	principal := employeePrincipal{id: id.String(), grants: make([]permissionGrant, 0, len(claims.Grants))}
	for _, raw := range claims.Grants {
		parts := strings.Split(raw, "#")
		if len(parts) < 2 || len(parts) > 3 || strings.TrimSpace(parts[0]) != parts[0] || parts[0] == "" {
			return unauthorized, http.StatusUnauthorized
		}
		grant := permissionGrant{code: parts[0], scopeType: parts[1]}
		switch grant.scopeType {
		case "GLOBAL":
			if len(parts) != 2 {
				return unauthorized, http.StatusUnauthorized
			}
		case "WAREHOUSE":
			if len(parts) != 3 || strings.TrimSpace(parts[2]) == "" || strings.TrimSpace(parts[2]) != parts[2] {
				return unauthorized, http.StatusUnauthorized
			}
			grant.scopeID = parts[2]
		default:
			return unauthorized, http.StatusUnauthorized
		}
		principal.grants = append(principal.grants, grant)
	}
	return principal, http.StatusOK
}

func hasAudience(raw json.RawMessage, expected string) bool {
	var single string
	if json.Unmarshal(raw, &single) == nil {
		return single == expected
	}
	var multiple []string
	if json.Unmarshal(raw, &multiple) != nil {
		return false
	}
	for _, audience := range multiple {
		if audience == expected {
			return true
		}
	}
	return false
}

func (v *jwtVerifier) key(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if time.Now().Before(v.until) {
		if key := v.keys[kid]; key != nil {
			return key, nil
		}
	}
	if time.Since(v.lastFetch) < 10*time.Second {
		return nil, errors.New("JWKS refresh is rate limited")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, v.config.jwksURL, nil)
	if err != nil {
		return nil, err
	}
	response, err := v.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, errors.New("JWKS endpoint returned a non-success status")
	}
	var document struct {
		Keys []struct {
			KID string `json:"kid"`
			KTY string `json:"kty"`
			ALG string `json:"alg"`
			USE string `json:"use"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&document); err != nil {
		return nil, err
	}
	keys := make(map[string]*rsa.PublicKey, len(document.Keys))
	for _, item := range document.Keys {
		if item.KTY != "RSA" || item.KID == "" || item.ALG != "RS256" && item.ALG != "" || item.USE != "sig" && item.USE != "" {
			continue
		}
		modulus, modErr := base64.RawURLEncoding.DecodeString(item.N)
		exponent, expErr := base64.RawURLEncoding.DecodeString(item.E)
		if modErr != nil || expErr != nil || len(modulus) == 0 || len(exponent) == 0 || len(exponent) > 4 {
			continue
		}
		eBig := new(big.Int).SetBytes(exponent)
		if !eBig.IsInt64() || new(big.Int).SetBytes(modulus).BitLen() < 2048 {
			continue
		}
		e := int(eBig.Int64())
		if e < 3 || e%2 == 0 {
			continue
		}
		keys[item.KID] = &rsa.PublicKey{N: new(big.Int).SetBytes(modulus), E: e}
	}
	v.keys = keys
	v.until = time.Now().Add(5 * time.Minute)
	v.lastFetch = time.Now()
	if key := keys[kid]; key != nil {
		return key, nil
	}
	return nil, errors.New("JWT signing key is unknown")
}
