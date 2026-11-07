package middleware

import (
	"context"
	"github.com/omamx/profile-service/internal/domain"
	"github.com/omamx/profile-service/internal/infrastructure/security"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStaffPermissionRequiresVerifiedMatchingGrant(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		scope := StaffScope(r.Context(), "EMPLOYEE_PROFILE_VIEW")
		if !scope.Allows("A") || scope.Allows("B") {
			t.Error("scope hopping")
		}
		w.WriteHeader(204)
	})
	verified := security.Principal{Role: "EMPLOYEE", Grants: []domain.PermissionGrant{
		{PermissionCode: "EMPLOYEE_PROFILE_VIEW", ScopeType: "DEPARTMENT", ScopeID: "A"},
		{PermissionCode: "EMPLOYEE_PROFILE_SENSITIVE_VIEW", ScopeType: "DEPARTMENT", ScopeID: "B"},
	}}
	for _, p := range []security.Principal{verified, {Role: "ADMIN"}, {Role: "EMPLOYEE"}, {Role: "CUSTOMER", Grants: verified.Grants}} {
		r := httptest.NewRequest("GET", "/", nil)
		r = r.WithContext(context.WithValue(r.Context(), PrincipalKey, p))
		r.Header.Set("X-User-Role", "ADMIN")
		w := httptest.NewRecorder()
		RequirePermission("EMPLOYEE_PROFILE_VIEW")(next).ServeHTTP(w, r)
		expected := 403
		if p.Role == "EMPLOYEE" && len(p.Grants) > 0 {
			expected = 204
		}
		if w.Code != expected {
			t.Fatalf("%+v: %d", p, w.Code)
		}
	}
	r := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	r.Header.Set("X-Internal-Token", "internal")
	r.Header.Set("X-User-ID", "d0000000-0000-0000-0000-000000000001")
	r.Header.Set("X-User-Role", "ADMIN")
	AuthContextWithToken("internal")(RequirePermission("EMPLOYEE_PROFILE_VIEW")(next)).ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal("internal forwarding bypassed IAM", w.Code)
	}
}
