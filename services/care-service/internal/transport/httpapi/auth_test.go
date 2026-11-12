package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestTrustedGatewayAuth(t *testing.T) {
	called := false
	handler := TrustedGatewayAuth("server-secret")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actor, ok := AuthenticatedActor(r.Context())
		require.True(t, ok)
		require.NotEqual(t, uuid.Nil, actor.ID)
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/tickets", nil)
	request.Header.Set("X-Internal-Token", "server-secret")
	request.Header.Set("X-User-ID", uuid.NewString())
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	require.True(t, called)
	require.Equal(t, http.StatusNoContent, response.Code)
}

func TestTrustedGatewayAuthRejectsUntrustedHeaders(t *testing.T) {
	handler := TrustedGatewayAuth("server-secret")(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("untrusted request reached protected handler")
	}))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/tickets", nil)
	request.Header.Set("X-User-ID", uuid.NewString())
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	require.Equal(t, http.StatusUnauthorized, response.Code)
}

func TestTrustedGatewayAuthKeepsLegacyCarePermissionAliases(t *testing.T) {
	handler := TrustedGatewayAuth("server-secret")(
		RequirePermission("TICKET_QUEUE_VIEW")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		})),
	)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/support/tickets", nil)
	request.Header.Set("X-Internal-Token", "server-secret")
	request.Header.Set("X-User-ID", uuid.NewString())
	request.Header.Set("X-Permissions", "care.tickets.read")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	require.Equal(t, http.StatusNoContent, response.Code)
}

func TestInvalidBearerDoesNotFallBackToGatewayHeaders(t *testing.T) {
	handler := GatewayOrStaffJWTAuth("server-secret", nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/support/tickets", nil)
	request.Header.Set("Authorization", "Bearer invalid")
	request.Header.Set("X-Internal-Token", "server-secret")
	request.Header.Set("X-User-ID", uuid.NewString())
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	require.Equal(t, http.StatusUnauthorized, response.Code)
}
