package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestAdminCouponAuthorization(t *testing.T) {
	router := chi.NewRouter()
	RegisterAdminCouponRoutes(router, nil, "internal-secret")
	tests := []struct {
		name, token, user, role string
		want                    int
	}{
		{name: "missing token", user: "employee-1", role: "SALES_MANAGER", want: http.StatusUnauthorized},
		{name: "wrong token", token: "wrong", user: "employee-1", role: "SALES_MANAGER", want: http.StatusUnauthorized},
		{name: "missing actor", token: "internal-secret", role: "SALES_MANAGER", want: http.StatusUnauthorized},
		{name: "insufficient role", token: "internal-secret", user: "employee-1", role: "ADMIN", want: http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/coupons", nil)
			request.Header.Set("X-Internal-Token", tt.token)
			request.Header.Set("X-User-ID", tt.user)
			request.Header.Set("X-User-Role", tt.role)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != tt.want {
				t.Fatalf("status = %d, want %d; body=%s", response.Code, tt.want, response.Body.String())
			}
		})
	}
}

func TestLoyaltyInternalRoutesRequireConfiguredToken(t *testing.T) {
	router := chi.NewRouter()
	RegisterLoyaltyRoutes(router, nil, "gateway-token", "order-token", "refund-token")
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

func TestAdminPaginationRejectsInvalidInputBeforeRepositoryAccess(t *testing.T) {
	router := chi.NewRouter()
	RegisterAdminCouponRoutes(router, nil, "internal-secret")
	for _, path := range []string{"/api/v1/admin/coupons?limit=0", "/api/v1/admin/coupons?limit=101", "/api/v1/admin/coupons?limit=nope", "/api/v1/admin/coupons?offset=-1", "/api/v1/admin/coupons?offset=nope"} {
		t.Run(path, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, path, nil)
			request.Header.Set("X-Internal-Token", "internal-secret")
			request.Header.Set("X-User-ID", "employee-1")
			request.Header.Set("X-User-Role", "SALES_MANAGER")
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
	for _, body := range []string{
		`{"code":"SAVE10","unknown":true}`,
		`{"code":"SAVE10"} {"extra":true}`,
		`not-json`,
	} {
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
