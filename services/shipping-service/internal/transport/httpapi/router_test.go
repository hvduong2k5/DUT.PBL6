package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
)

func TestGHNWebhookIsHiddenWhenDisabled(t *testing.T) {
	handler := NewRouter(nil, "internal-token", "fake-secret", "", zerolog.Nop())
	request := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/logistics/ghn", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", response.Code)
	}
}

func TestGHNWebhookRejectsInvalidSecretBeforeProcessing(t *testing.T) {
	handler := NewRouter(nil, "internal-token", "fake-secret", "expected-secret", zerolog.Nop())
	request := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/logistics/ghn", nil)
	request.Header.Set("X-GHN-Webhook-Secret", "wrong-secret")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", response.Code)
	}
}
