package middleware

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/omamx/profile-service/internal/infrastructure/security"
	"net/http"
	"net/http/httptest"
	"testing"
)

type verifiedBearer struct {
	id  string
	err error
}

func (v verifiedBearer) Authenticate(context.Context, string) (security.Principal, error) {
	return security.Principal{ID: v.id, Role: "CUSTOMER"}, v.err
}

func TestBearerCannotUseForwardedIdentityOrFallBack(t *testing.T) {
	owner, attacker := uuid.NewString(), uuid.NewString()
	for _, bad := range []bool{false, true} {
		var err error
		if bad {
			err = errors.New("invalid bearer")
		}
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, _ := GetUserIDFromContext(r.Context())
			if id.String() != owner {
				t.Error("browser header selected principal")
			}
			w.WriteHeader(204)
		})
		handler := AuthContextWithIdentity("dedicated-internal-credential", verifiedBearer{owner, err})(next)
		r := httptest.NewRequest("GET", "/", nil)
		r.Header.Set("Authorization", "Bearer value")
		r.Header.Set("X-User-ID", attacker)
		r.Header.Set("X-User-Role", "ADMIN")
		r.Header.Set("X-Internal-Token", "dedicated-internal-credential")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		expected := 204
		if bad {
			expected = 401
		}
		if w.Code != expected {
			t.Fatal("bearer fallback", w.Code)
		}
	}
	var disabled *security.OIDCIdentity
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", "Bearer invalid")
	AuthContextWithIdentity("secret", disabled)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("disabled identity accepted bearer") })).ServeHTTP(w, r)
	if w.Code != 401 {
		t.Fatal(w.Code)
	}
}
