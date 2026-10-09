package middleware

import (
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIdentityCannotBeSpoofed(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })
	for _, token := range []string{"", "wrong", "valid"} {
		request := httptest.NewRequest("GET", "/api/v1/profile?customer_id=c0000000-0000-0000-0000-000000000001", nil)
		request.Header.Set("X-User-ID", "d0000000-0000-0000-0000-000000000001")
		request.Header.Set("X-Internal-Token", token)
		response := httptest.NewRecorder()
		AuthContextWithToken("valid")(next).ServeHTTP(response, request)
		expected := 401
		if token == "valid" {
			expected = 204
		}
		require.Equal(t, expected, response.Code)
	}
	response := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/", nil)
	request.Header.Set("X-Internal-Token", "valid")
	AuthContextWithToken("valid")(next).ServeHTTP(response, request)
	require.Equal(t, 401, response.Code)
}
