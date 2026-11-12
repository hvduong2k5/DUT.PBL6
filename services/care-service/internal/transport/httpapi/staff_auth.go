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
	"unicode"

	"github.com/google/uuid"
)

var (
	errStaffUnauthenticated = errors.New("staff JWT is not valid")
	errStaffForbidden       = errors.New("staff JWT lacks Care permission")
)

type StaffAuthenticator interface {
	Authenticate(context.Context, string) (Actor, error)
}

type staffClaims struct {
	Issuer           string          `json:"iss"`
	Subject          string          `json:"sub"`
	Audience         json.RawMessage `json:"aud"`
	ExpiresAt        json.Number     `json:"exp"`
	IssuedAt         json.Number     `json:"iat"`
	NotBefore        json.Number     `json:"nbf"`
	UserID           string          `json:"user_id"`
	Type             string          `json:"typ"`
	PrincipalType    string          `json:"principal_type"`
	PermissionGrants []string        `json:"permission_grants"`
	Service          string          `json:"service"`
}

type jwtHeader struct {
	Algorithm string `json:"alg"`
	KeyID     string `json:"kid"`
}

type jwtStaffAuthenticator struct {
	issuer   string
	audience string
	jwksURL  string
	client   *http.Client

	mu        sync.Mutex
	keys      map[string]*rsa.PublicKey
	expires   time.Time
	nextFetch time.Time
}

func NewJWTStaffAuthenticator(issuer, audience, jwksURL string) (StaffAuthenticator, error) {
	issuerURL, err := url.Parse(issuer)
	if err != nil || issuerURL.Hostname() == "" || issuerURL.User != nil || issuerURL.RawQuery != "" || issuerURL.Fragment != "" || !secureOIDCURL(issuerURL) {
		return nil, errors.New("Care OIDC requires a valid HTTPS issuer (loopback HTTP is allowed for local tests)")
	}
	if strings.TrimSpace(audience) == "" {
		return nil, errors.New("Care OIDC audience is required")
	}
	if jwksURL == "" {
		jwksURL = strings.TrimRight(issuer, "/") + "/protocol/openid-connect/certs"
	}
	jwks, err := url.Parse(jwksURL)
	if err != nil || jwks.Hostname() == "" || jwks.User != nil || jwks.RawQuery != "" || jwks.Fragment != "" || !secureOIDCURL(jwks) {
		return nil, errors.New("invalid Care OIDC JWKS URL")
	}
	return &jwtStaffAuthenticator{
		issuer: issuer, audience: audience, jwksURL: jwksURL,
		client: &http.Client{Timeout: 3 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
	}, nil
}

func secureOIDCURL(value *url.URL) bool {
	if value.Scheme == "https" {
		return true
	}
	ip := net.ParseIP(value.Hostname())
	return value.Scheme == "http" && (value.Hostname() == "localhost" || (ip != nil && ip.IsLoopback()))
}

func (a *jwtStaffAuthenticator) Authenticate(ctx context.Context, authorization string) (Actor, error) {
	parts := strings.Fields(authorization)
	if len(authorization) > 8192 || len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return Actor{}, errStaffUnauthenticated
	}
	jwtParts := strings.Split(parts[1], ".")
	if len(jwtParts) != 3 {
		return Actor{}, errStaffUnauthenticated
	}
	headerBytes, headerErr := base64.RawURLEncoding.DecodeString(jwtParts[0])
	claimsBytes, claimsErr := base64.RawURLEncoding.DecodeString(jwtParts[1])
	signature, signatureErr := base64.RawURLEncoding.DecodeString(jwtParts[2])
	var header jwtHeader
	if headerErr != nil || claimsErr != nil || signatureErr != nil || json.Unmarshal(headerBytes, &header) != nil || header.Algorithm != "RS256" || header.KeyID == "" || len(header.KeyID) > 256 {
		return Actor{}, errStaffUnauthenticated
	}
	key, err := a.key(ctx, header.KeyID)
	if err != nil {
		return Actor{}, errStaffUnauthenticated
	}
	signed := []byte(jwtParts[0] + "." + jwtParts[1])
	digest := sha256.Sum256(signed)
	if rsa.VerifyPKCS1v15(key, crypto.SHA256, digest[:], signature) != nil {
		return Actor{}, errStaffUnauthenticated
	}
	decoder := json.NewDecoder(strings.NewReader(string(claimsBytes)))
	decoder.UseNumber()
	var claims staffClaims
	if decoder.Decode(&claims) != nil || !validCareStaffClaims(claims, a.issuer, a.audience, time.Now()) {
		return Actor{}, errStaffUnauthenticated
	}
	userID, err := uuid.Parse(claims.UserID)
	if err != nil || userID == uuid.Nil {
		return Actor{}, errStaffUnauthenticated
	}
	if claims.PermissionGrants == nil || len(claims.PermissionGrants) > 128 {
		return Actor{}, errStaffForbidden
	}
	permissions := make(map[string]struct{})
	if err := validateCarePermissionGrants(claims.PermissionGrants, permissions); err != nil {
		return Actor{}, errStaffUnauthenticated
	}
	return Actor{ID: userID, PrincipalType: "EMPLOYEE", Permissions: permissions}, nil
}

func validCareStaffClaims(claims staffClaims, issuer, audience string, now time.Time) bool {
	if claims.Issuer != issuer || claims.Subject == "" || claims.Type != "Bearer" || claims.PrincipalType != "EMPLOYEE" || claims.Service != "" || !hasAudience(claims.Audience, audience) {
		return false
	}
	expiresAt, err := claims.ExpiresAt.Int64()
	if err != nil || now.After(time.Unix(expiresAt, 0).Add(5*time.Second)) {
		return false
	}
	issuedAt, err := claims.IssuedAt.Int64()
	if err != nil || time.Unix(issuedAt, 0).After(now.Add(5*time.Second)) {
		return false
	}
	if claims.NotBefore != "" {
		notBefore, err := claims.NotBefore.Int64()
		if err != nil || time.Unix(notBefore, 0).After(now.Add(5*time.Second)) {
			return false
		}
	}
	return true
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
	for _, value := range multiple {
		if value == expected {
			return true
		}
	}
	return false
}

func validateCarePermissionGrants(grants []string, permissions map[string]struct{}) error {
	for _, grant := range grants {
		parts := strings.Split(grant, "#")
		if len(parts) < 2 || len(parts) > 3 || !validCarePermissionCode(parts[0]) {
			return errStaffUnauthenticated
		}
		global := false
		switch parts[1] {
		case "GLOBAL":
			if len(parts) != 2 {
				return errStaffUnauthenticated
			}
			global = true
		case "WAREHOUSE", "DEPARTMENT":
			if len(parts) != 3 || !validCareScopeID(parts[2]) {
				return errStaffUnauthenticated
			}
		default:
			return errStaffUnauthenticated
		}
		// Current Care ticket endpoints have no warehouse/department ACL. Until
		// that policy is defined, only an explicit GLOBAL grant can authorize them.
		if global {
			permission := canonicalCarePermission(parts[0])
			if isCarePermission(permission) {
				permissions[permission] = struct{}{}
			}
		}
	}
	return nil
}

func isCarePermission(permission string) bool {
	switch permission {
	case "TICKET_QUEUE_VIEW", "TICKET_CLAIM", "TICKET_NOTE_WRITE", "TICKET_REPLY", "TICKET_TRANSITION",
		"RETURN_CASE_VIEW", "RETURN_CASE_REVIEW", "RETURN_CASE_DECIDE", "RETURN_CASE_REFUND_WITHOUT_RETURN":
		return true
	default:
		return false
	}
}

func validCarePermissionCode(value string) bool {
	if len(value) == 0 || len(value) > 160 || value[0] < 'A' || value[0] > 'Z' {
		return false
	}
	for _, r := range value {
		if !(r >= 'A' && r <= 'Z') && !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9') && !strings.ContainsRune("_.:-", r) {
			return false
		}
	}
	return true
}

func validCareScopeID(value string) bool {
	if value == "" || len(value) > 160 || strings.TrimSpace(value) != value {
		return false
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}

func (a *jwtStaffAuthenticator) key(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	now := time.Now()
	if key := a.keys[kid]; key != nil && now.Before(a.expires) {
		return key, nil
	}
	if now.Before(a.nextFetch) {
		return nil, errors.New("JWKS refresh throttled")
	}
	a.nextFetch = now.Add(15 * time.Second)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.jwksURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := a.client.Do(req)
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
	var document struct {
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
	if json.Unmarshal(body, &document) != nil || len(document.Keys) > 100 {
		return nil, errors.New("invalid JWKS")
	}
	keys := make(map[string]*rsa.PublicKey)
	for _, item := range document.Keys {
		if item.Kty != "RSA" || item.Kid == "" || (item.Use != "" && item.Use != "sig") || (item.Alg != "" && item.Alg != "RS256") {
			continue
		}
		if len(item.KeyOps) > 0 && !contains(item.KeyOps, "verify") {
			continue
		}
		modulusBytes, modulusErr := base64.RawURLEncoding.DecodeString(item.N)
		exponentBytes, exponentErr := base64.RawURLEncoding.DecodeString(item.E)
		if modulusErr != nil || exponentErr != nil || len(exponentBytes) == 0 || len(exponentBytes) > 4 {
			return nil, errors.New("invalid RSA JWK")
		}
		modulus := new(big.Int).SetBytes(modulusBytes)
		exponent := new(big.Int).SetBytes(exponentBytes).Int64()
		if modulus.BitLen() < 2048 || modulus.BitLen() > 8192 || exponent < 3 || exponent > 2147483647 || exponent%2 == 0 || keys[item.Kid] != nil {
			return nil, errors.New("unsafe or ambiguous RSA JWK")
		}
		keys[item.Kid] = &rsa.PublicKey{N: modulus, E: int(exponent)}
	}
	if len(keys) == 0 {
		return nil, errors.New("no usable signing keys")
	}
	a.keys = keys
	a.expires = now.Add(5 * time.Minute)
	if key := keys[kid]; key != nil {
		return key, nil
	}
	return nil, errors.New("unknown JWT key ID")
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
