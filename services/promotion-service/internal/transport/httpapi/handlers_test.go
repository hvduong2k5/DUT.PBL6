package httpapi

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

const testStaffIssuer = "https://identity.example/realms/omama"

func TestAdminCouponAuthorization(t *testing.T) {
	key, viewToken := staffAuthFixture(t, "EMPLOYEE", []string{"PROMOTION_VIEW#GLOBAL"}, time.Now().Add(time.Hour))
	auth := StaffJWTAuth{Key: &key.PublicKey, Issuer: testStaffIssuer, Audience: "omama-api"}
	router := chi.NewRouter()
	RegisterAdminCouponRoutes(router, nil, auth)
	tests := []struct {
		name, bearer, role string
		want               int
	}{
		{name: "missing bearer", role: "SALES_MANAGER", want: http.StatusUnauthorized},
		{name: "internal token and role do not bypass JWT", bearer: "internal-secret", role: "SALES_MANAGER", want: http.StatusUnauthorized},
		{name: "role alone does not authorize", bearer: viewToken, role: "ADMIN", want: http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/coupons", nil)
			if tt.bearer != "" {
				request.Header.Set("Authorization", "Bearer "+tt.bearer)
			}
			request.Header.Set("X-User-Role", tt.role)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != tt.want {
				t.Fatalf("status = %d, want %d; body=%s", response.Code, tt.want, response.Body.String())
			}
		})
	}

	manageKey, manageToken := staffAuthFixture(t, "EMPLOYEE", []string{"PROMOTION_MANAGE#GLOBAL"}, time.Now().Add(time.Hour))
	manageRouter := chi.NewRouter()
	RegisterAdminCouponRoutes(manageRouter, nil, StaffJWTAuth{Key: &manageKey.PublicKey, Issuer: testStaffIssuer, Audience: "omama-api"})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/coupons", strings.NewReader(`not-json`))
	request.Header.Set("Authorization", "Bearer "+manageToken)
	response := httptest.NewRecorder()
	manageRouter.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("manage grant should reach request validation: status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestStaffJWTRequiresValidPrincipalPermissionAndGlobalScope(t *testing.T) {
	validKey, validToken := staffAuthFixture(t, "EMPLOYEE", []string{"PROMOTION_VIEW#GLOBAL"}, time.Now().Add(time.Hour))
	validAuth := StaffJWTAuth{Key: &validKey.PublicKey, Issuer: testStaffIssuer, Audience: "omama-api"}
	tests := []struct {
		name  string
		auth  StaffJWTAuth
		token string
		want  int
	}{
		{name: "missing verifier configuration", auth: StaffJWTAuth{Audience: "omama-api"}, token: validToken, want: http.StatusServiceUnavailable},
		{name: "customer principal", auth: validAuth, token: mustStaffToken(t, validKey, testStaffIssuer, "CUSTOMER", []string{"PROMOTION_VIEW#GLOBAL"}, time.Now().Add(time.Hour)), want: http.StatusForbidden},
		{name: "permission not granted", auth: validAuth, token: mustStaffToken(t, validKey, testStaffIssuer, "EMPLOYEE", []string{"PROMOTION_MANAGE#GLOBAL"}, time.Now().Add(time.Hour)), want: http.StatusForbidden},
		{name: "only warehouse scope", auth: validAuth, token: mustStaffToken(t, validKey, testStaffIssuer, "EMPLOYEE", []string{"PROMOTION_VIEW#WAREHOUSE#WH-1"}, time.Now().Add(time.Hour)), want: http.StatusForbidden},
		{name: "malformed grant", auth: validAuth, token: mustStaffToken(t, validKey, testStaffIssuer, "EMPLOYEE", []string{"PROMOTION_VIEW"}, time.Now().Add(time.Hour)), want: http.StatusUnauthorized},
		{name: "expired token", auth: validAuth, token: mustStaffToken(t, validKey, testStaffIssuer, "EMPLOYEE", []string{"PROMOTION_VIEW#GLOBAL"}, time.Now().Add(-time.Hour)), want: http.StatusUnauthorized},
		{name: "wrong issuer", auth: validAuth, token: signedStaffToken(t, validKey, "https://other.example/realm", "omama-api", "EMPLOYEE", []string{"PROMOTION_VIEW#GLOBAL"}, time.Now().Add(time.Hour)), want: http.StatusUnauthorized},
		{name: "wrong audience", auth: validAuth, token: signedStaffToken(t, validKey, testStaffIssuer, "promotion-api", "EMPLOYEE", []string{"PROMOTION_VIEW#GLOBAL"}, time.Now().Add(time.Hour)), want: http.StatusUnauthorized},
		{name: "Identity audience array", auth: validAuth, token: signedStaffTokenWithAudience(t, validKey, testStaffIssuer, []string{"order-api", "omama-api"}, "EMPLOYEE", []string{"PROMOTION_VIEW#GLOBAL"}, time.Now().Add(time.Hour)), want: http.StatusNoContent},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := tt.auth.Require(PermissionPromotionView)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				actor := staffActorFromRequest(r)
				w.Header().Set("X-Test-Actor", actor.ID)
				w.WriteHeader(http.StatusNoContent)
			}))
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			request.Header.Set("Authorization", "Bearer "+tt.token)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != tt.want {
				t.Fatalf("status=%d want=%d body=%s", response.Code, tt.want, response.Body.String())
			}
		})
	}

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Authorization", "Bearer "+validToken)
	request.Header.Set("X-User-ID", "spoofed-actor")
	request.Header.Set("X-User-Role", "ADMIN")
	response := httptest.NewRecorder()
	validAuth.Require(PermissionPromotionView)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actor := staffActorFromRequest(r)
		if actor.ID != "a0000000-0000-4000-8000-000000000001" || actor.Role != "EMPLOYEE" {
			t.Errorf("unexpected actor: %+v", actor)
		}
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("valid global grant status=%d want=%d", response.Code, http.StatusNoContent)
	}
}

func TestLoyaltyInternalRoutesRequireConfiguredToken(t *testing.T) {
	router := chi.NewRouter()
	RegisterLoyaltyRoutes(router, nil, "gateway-token", "order-token", "refund-token", "")
	for _, tt := range []struct {
		name, token string
		want        int
	}{
		{name: "no token", want: http.StatusUnauthorized},
		{name: "untrusted token", token: "random", want: http.StatusUnauthorized},
	} {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/internal/v1/loyalty/policy/active", nil)
			request.Header.Set("X-Internal-Token", tt.token)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != tt.want {
				t.Fatalf("status=%d want=%d", response.Code, tt.want)
			}
		})
	}
}

func TestCustomerLoyaltyPointsRequiresProfileVerifiedBearer(t *testing.T) {
	router := chi.NewRouter()
	RegisterLoyaltyRoutes(router, nil, "gateway-token", "order-token", "refund-token", "")
	request := httptest.NewRequest(http.MethodGet, "/api/v1/loyalty/points?customer_id=somebody-else", nil)
	request.Header.Set("X-Customer-ID", "somebody-else")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("missing bearer status=%d want=%d body=%s", response.Code, http.StatusUnauthorized, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodGet, "/api/v1/loyalty/points?customer_id=somebody-else", nil)
	request.Header.Set("Authorization", "Bearer customer-token")
	request.Header.Set("X-Customer-ID", "somebody-else")
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("unconfigured resolver status=%d want=%d body=%s", response.Code, http.StatusServiceUnavailable, response.Body.String())
	}
}

func TestAdminPaginationRejectsInvalidInputBeforeRepositoryAccess(t *testing.T) {
	key, token := staffAuthFixture(t, "EMPLOYEE", []string{"PROMOTION_VIEW#GLOBAL"}, time.Now().Add(time.Hour))
	router := chi.NewRouter()
	RegisterAdminCouponRoutes(router, nil, StaffJWTAuth{Key: &key.PublicKey, Issuer: testStaffIssuer, Audience: "omama-api"})
	for _, path := range []string{"/api/v1/admin/coupons?limit=0", "/api/v1/admin/coupons?limit=101", "/api/v1/admin/coupons?limit=nope", "/api/v1/admin/coupons?offset=-1", "/api/v1/admin/coupons?offset=nope"} {
		t.Run(path, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, path, nil)
			request.Header.Set("Authorization", "Bearer "+token)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("status=%d want=%d body=%s", response.Code, http.StatusBadRequest, response.Body.String())
			}
		})
	}
}

func TestPublicPromotionPaginationRejectsInvalidInput(t *testing.T) {
	router := chi.NewRouter()
	RegisterPublicPromotionRoutes(router, nil)
	for _, path := range []string{"/api/v1/promotions/active?limit=0", "/api/v1/promotions/active?limit=101", "/api/v1/promotions/active?limit=nope", "/api/v1/promotions/active?offset=-1", "/api/v1/promotions/active?offset=nope"} {
		t.Run(path, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, path, nil)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("status=%d want=%d body=%s", response.Code, http.StatusBadRequest, response.Body.String())
			}
		})
	}
}

func TestDecodeCouponRequestRejectsUnknownAndTrailingJSON(t *testing.T) {
	for _, body := range []string{`{"code":"SAVE10","unknown":true}`, `{"code":"SAVE10"} {"extra":true}`, `not-json`} {
		t.Run(body, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
			response := httptest.NewRecorder()
			if _, ok := decodeCouponRequest(response, request); ok {
				t.Fatal("decodeCouponRequest accepted invalid body")
			}
			if response.Code != http.StatusBadRequest {
				t.Fatalf("status=%d", response.Code)
			}
		})
	}
}

func TestDecodeLoyaltyCommandRequiresIdempotencyKeyAndStrictJSON(t *testing.T) {
	tests := []struct {
		name, key, body string
		want            bool
	}{
		{name: "missing key", body: `{"customer_id":"customer-1"}`},
		{name: "valid command", key: "loyalty-command-key-1", body: `{"customer_id":"customer-1"}`, want: true},
		{name: "unknown property", key: "loyalty-command-key-1", body: `{"customer_id":"customer-1","unexpected":1}`},
		{name: "trailing JSON", key: "loyalty-command-key-1", body: `{"customer_id":"customer-1"}{}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			request.Header.Set("Idempotency-Key", tt.key)
			response := httptest.NewRecorder()
			var payload struct {
				CustomerID string `json:"customer_id"`
			}
			got := decodeLoyaltyCommand(response, request, &payload)
			if got != tt.want {
				t.Fatalf("accepted=%v want=%v status=%d", got, tt.want, response.Code)
			}
		})
	}
}

func staffAuthFixture(t *testing.T, principal string, grants []string, expires time.Time) (*rsa.PrivateKey, string) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate signing key: %v", err)
	}
	return key, signedStaffToken(t, key, testStaffIssuer, "omama-api", principal, grants, expires)
}

func mustStaffToken(t *testing.T, key *rsa.PrivateKey, issuer, principal string, grants []string, expires time.Time) string {
	t.Helper()
	return signedStaffToken(t, key, issuer, "omama-api", principal, grants, expires)
}

func signedStaffToken(t *testing.T, key *rsa.PrivateKey, issuer, audience, principal string, grants []string, expires time.Time) string {
	return signedStaffTokenWithAudience(t, key, issuer, audience, principal, grants, expires)
}

func signedStaffTokenWithAudience(t *testing.T, key *rsa.PrivateKey, issuer string, audience any, principal string, grants []string, expires time.Time) string {
	t.Helper()
	now := time.Now().Add(-time.Minute).Unix()
	header, err := json.Marshal(map[string]string{"alg": "RS256", "typ": "JWT"})
	if err != nil {
		t.Fatalf("marshal test JWT header: %v", err)
	}
	claims, err := json.Marshal(map[string]any{"iss": issuer, "aud": audience, "sub": "identity-subject-1", "user_id": "a0000000-0000-4000-8000-000000000001",
		"principal_type": principal, "permission_grants": grants, "iat": now, "exp": expires.Unix()})
	if err != nil {
		t.Fatalf("marshal test JWT claims: %v", err)
	}
	encodedHeader := base64.RawURLEncoding.EncodeToString(header)
	encodedClaims := base64.RawURLEncoding.EncodeToString(claims)
	message := encodedHeader + "." + encodedClaims
	digest := sha256.Sum256([]byte(message))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatalf("sign test JWT: %v", err)
	}
	return message + "." + base64.RawURLEncoding.EncodeToString(signature)
}
