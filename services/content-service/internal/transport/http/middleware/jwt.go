package middleware

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/hvduong2k5/DUT.PBL6/services/content-service/internal/domain"
)

const (
	jwtClockSkew   = 30 * time.Second
	maxHeaderBytes = 8192
	maxGrants      = 128
)

var grantPermission = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_.:-]{0,159}$`)

type staffClaims struct {
	UserID           string   `json:"user_id"`
	Service          string   `json:"service"`
	PermissionGrants []string `json:"permission_grants"`
	jwt.RegisteredClaims
}

// JWTVerifier validates the Identity-issued staff access token (RS256, pinned
// issuer and audience, required exp/iat) and reads authority only from its
// signed permission_grants claim: "PERM#GLOBAL" or "PERM#SCOPE_TYPE#SCOPE_ID".
// ADMIN is a role of an EMPLOYEE and never bypasses a missing grant.
type JWTVerifier struct {
	key      *rsa.PublicKey
	issuer   string
	audience string
}

// NewJWTVerifier parses a PEM encoded RSA public key (PKIX, PKCS#1 or an X.509
// certificate).
func NewJWTVerifier(publicKeyPEM []byte, issuer, audience string) (*JWTVerifier, error) {
	if issuer == "" || audience == "" {
		return nil, errors.New("JWT issuer and audience are required")
	}
	block, _ := pem.Decode(publicKeyPEM)
	if block == nil {
		return nil, errors.New("JWT public key is not valid PEM")
	}
	var key *rsa.PublicKey
	if parsed, err := x509.ParsePKIXPublicKey(block.Bytes); err == nil {
		key, _ = parsed.(*rsa.PublicKey)
	} else if pkcs1, err := x509.ParsePKCS1PublicKey(block.Bytes); err == nil {
		key = pkcs1
	} else if cert, err := x509.ParseCertificate(block.Bytes); err == nil {
		key, _ = cert.PublicKey.(*rsa.PublicKey)
	}
	if key == nil {
		return nil, errors.New("JWT public key must be an RSA key")
	}
	return &JWTVerifier{key: key, issuer: issuer, audience: audience}, nil
}

func (v *JWTVerifier) Verify(r *http.Request) (domain.Identity, error) {
	header := r.Header.Get("Authorization")
	if len(header) > maxHeaderBytes || !strings.HasPrefix(header, "Bearer ") {
		return domain.Identity{}, domain.ErrUnauthenticated
	}
	c := &staffClaims{}
	token, err := jwt.ParseWithClaims(strings.TrimPrefix(header, "Bearer "), c,
		func(*jwt.Token) (any, error) { return v.key, nil },
		jwt.WithValidMethods([]string{"RS256"}), jwt.WithIssuer(v.issuer), jwt.WithAudience(v.audience),
		jwt.WithExpirationRequired(), jwt.WithIssuedAt(), jwt.WithLeeway(jwtClockSkew))
	if err != nil || !token.Valid || c.Subject == "" {
		return domain.Identity{}, domain.ErrUnauthenticated
	}
	id, err := uuid.Parse(c.UserID)
	// A service credential, or a token without the claim (customer), is valid
	// but is not an employee: 403, not 401.
	if c.Service != "" || c.PermissionGrants == nil {
		return domain.Identity{}, domain.ErrForbidden
	}
	if err != nil || id == uuid.Nil || len(c.PermissionGrants) > maxGrants {
		return domain.Identity{}, domain.ErrUnauthenticated
	}
	grants, err := parseGrants(c.PermissionGrants)
	if err != nil {
		return domain.Identity{}, domain.ErrUnauthenticated
	}
	return domain.Identity{ID: id, Grants: grants}, nil
}

func scopeValue(s string) bool {
	if s == "" || len(s) > 160 || strings.TrimSpace(s) != s {
		return false
	}
	for _, r := range s {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}

// parseGrants rejects malformed grants outright and keeps only CONTENT_*
// permissions, the only ones this service can act on.
func parseGrants(raw []string) ([]domain.Grant, error) {
	var out []domain.Grant
	for _, g := range raw {
		parts := strings.Split(g, "#")
		if len(parts) < 2 || len(parts) > 3 || !grantPermission.MatchString(parts[0]) {
			return nil, fmt.Errorf("malformed grant")
		}
		grant := domain.Grant{Permission: domain.Permission(parts[0]), ScopeType: parts[1]}
		if parts[1] == domain.ScopeGlobal {
			if len(parts) != 2 {
				return nil, fmt.Errorf("malformed grant")
			}
		} else {
			if len(parts) != 3 || !scopeValue(parts[2]) || !grantPermission.MatchString(parts[1]) {
				return nil, fmt.Errorf("malformed grant")
			}
			grant.ScopeID = parts[2]
		}
		if strings.HasPrefix(parts[0], "CONTENT_") {
			out = append(out, grant)
		}
	}
	return out, nil
}
