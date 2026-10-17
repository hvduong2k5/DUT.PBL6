package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestRequestContextPreservesValidTrackingIDs(t *testing.T) {
	handler := RequestContext(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "request-123", RequestID(r.Context()))
		require.Equal(t, "checkout:flow-456", CorrelationID(r.Context()))
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("X-Request-ID", "request-123")
	request.Header.Set("X-Correlation-ID", "checkout:flow-456")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	require.Equal(t, "request-123", response.Header().Get("X-Request-ID"))
	require.Equal(t, "checkout:flow-456", response.Header().Get("X-Correlation-ID"))
}

func TestRequestContextReplacesMissingOrUnsafeTrackingIDs(t *testing.T) {
	handler := RequestContext(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, requestErr := uuid.Parse(RequestID(r.Context()))
		_, correlationErr := uuid.Parse(CorrelationID(r.Context()))
		require.NoError(t, requestErr)
		require.NoError(t, correlationErr)
		require.NotEqual(t, RequestID(r.Context()), CorrelationID(r.Context()))
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("X-Request-ID", "contains spaces")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	require.NotEqual(t, "contains spaces", response.Header().Get("X-Request-ID"))
	require.NotEmpty(t, response.Header().Get("X-Correlation-ID"))
}
