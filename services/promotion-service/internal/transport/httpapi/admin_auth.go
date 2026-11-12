package httpapi

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/omamx/promotion-service/internal/repository/postgres"
)

const (
	PermissionPromotionView   = "PROMOTION_VIEW"
	PermissionPromotionManage = "PROMOTION_MANAGE"
)

type StaffJWTAuth struct {
	Key      *rsa.PublicKey
	Issuer   string
	Audience string
}

type staffClaims struct {
	UserID           string          `json:"user_id"`
	PrincipalType    string          `json:"principal_type"`
	Service          string          `json:"service"`
	PermissionGrants []string        `json:"permission_grants"`
	Issuer           string          `json:"iss"`
	Subject          string          `json:"sub"`
	Audience         json.RawMessage `json:"aud"`
	IssuedAt         *int64          `json:"iat"`
	ExpiresAt        *int64          `json:"exp"`
	NotBefore        *int64          `json:"nbf"`
}

type jwtHeader struct {
	Algorithm string `json:"alg"`
}

var errInvalidStaffJWT = errors.New("invalid staff JWT")

type staffActorContextKey struct{}

var (
	staffUUIDPattern       = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-8][0-9a-fA-F]{3}-[89aAbB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)
	staffPermissionPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_.:-]{0,159}$`)
)

func (a StaffJWTAuth) Require(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if a.Key == nil || strings.TrimSpace(a.Issuer) == "" || strings.TrimSpace(a.Audience) == "" {
				writeError(w, http.StatusServiceUnavailable, "AUTH_UNAVAILABLE", "staff JWT verification is not configured")
				return
			}
			header := r.Header.Get("Authorization")
			if len(header) > 8192 {
				writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "a valid employee bearer token is required")
				return
			}
			parts := strings.Fields(header)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "a valid employee bearer token is required")
				return
			}
			claims, err := verifyStaffJWT(parts[1], a, time.Now())
			if err != nil || claims.Subject == "" || !staffUUIDPattern.MatchString(claims.UserID) || claims.Service != "" {
				writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "a valid employee bearer token is required")
				return
			}
			if claims.PrincipalType != "EMPLOYEE" || claims.PermissionGrants == nil {
				writeError(w, http.StatusForbidden, "EMPLOYEE_REQUIRED", "employee identity is required")
				return
			}
			if len(claims.PermissionGrants) > 128 {
				writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "employee authorization claims are invalid")
				return
			}
			allowed, validGrants := hasGlobalGrant(claims.PermissionGrants, permission)
			if !validGrants {
				writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "employee authorization claims are invalid")
				return
			}
			if !allowed {
				writeError(w, http.StatusForbidden, "FORBIDDEN", "required promotion permission and scope are missing")
				return
			}
			actor := postgres.CouponActor{ID: claims.UserID, Role: "EMPLOYEE"}
			ctx := context.WithValue(r.Context(), staffActorContextKey{}, actor)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func verifyStaffJWT(raw string, auth StaffJWTAuth, now time.Time) (*staffClaims, error) {
	parts := strings.Split(raw, ".")
	if len(parts) != 3 || len(raw) > 6144 {
		return nil, errInvalidStaffJWT
	}
	headerBytes, err := base64.RawURLEncoding.Strict().DecodeString(parts[0])
	if err != nil {
		return nil, err
	}
	var header jwtHeader
	if err := json.Unmarshal(headerBytes, &header); err != nil || header.Algorithm != "RS256" {
		return nil, errInvalidStaffJWT
	}
	signature, err := base64.RawURLEncoding.Strict().DecodeString(parts[2])
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if err := rsa.VerifyPKCS1v15(auth.Key, crypto.SHA256, digest[:], signature); err != nil {
		return nil, err
	}
	payload, err := base64.RawURLEncoding.Strict().DecodeString(parts[1])
	if err != nil {
		return nil, err
	}
	claims := &staffClaims{}
	if err := json.Unmarshal(payload, claims); err != nil {
		return nil, err
	}
	nowUnix := now.Unix()
	const leeway = int64(5)
	if claims.Issuer != auth.Issuer || claims.IssuedAt == nil || claims.ExpiresAt == nil || claims.Subject == "" ||
		!audienceContains(claims.Audience, auth.Audience) || *claims.ExpiresAt <= nowUnix-leeway ||
		*claims.IssuedAt > nowUnix+leeway || claims.NotBefore != nil && *claims.NotBefore > nowUnix+leeway {
		return nil, errInvalidStaffJWT
	}
	return claims, nil
}

func audienceContains(raw json.RawMessage, expected string) bool {
	var audience string
	if json.Unmarshal(raw, &audience) == nil {
		return audience == expected
	}
	var audiences []string
	if json.Unmarshal(raw, &audiences) != nil {
		return false
	}
	for _, audience := range audiences {
		if audience == expected {
			return true
		}
	}
	return false
}

func hasGlobalGrant(grants []string, required string) (bool, bool) {
	granted := false
	for _, raw := range grants {
		parts := strings.Split(raw, "#")
		if len(parts) < 2 || len(parts) > 3 || !staffPermissionPattern.MatchString(parts[0]) {
			return false, false
		}
		scope := parts[1]
		switch scope {
		case "GLOBAL":
			if len(parts) != 2 {
				return false, false
			}
		case "WAREHOUSE", "DEPARTMENT", "STORE":
			if len(parts) != 3 || !validScopeID(parts[2]) {
				return false, false
			}
		default:
			return false, false
		}
		if parts[0] == required && scope == "GLOBAL" {
			granted = true
		}
	}
	return granted, true
}

func validScopeID(value string) bool {
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

func staffActorFromRequest(r *http.Request) postgres.CouponActor {
	actor, _ := r.Context().Value(staffActorContextKey{}).(postgres.CouponActor)
	return actor
}
