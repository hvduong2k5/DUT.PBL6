package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCustomerProfileResolverRequiresBearerAndConfiguration(t *testing.T) {
	tests := []struct {
		name     string
		resolver *customerProfileResolver
		headers  http.Header
		wantKind customerProfileErrorKind
	}{
		{name: "missing bearer", resolver: newCustomerProfileResolver("http://profile.test/api/v1/profile"), headers: make(http.Header), wantKind: customerProfileAuthError},
		{name: "duplicate bearer", resolver: newCustomerProfileResolver("http://profile.test/api/v1/profile"), headers: http.Header{"Authorization": {"Bearer one", "Bearer two"}}, wantKind: customerProfileAuthError},
		{name: "wrong scheme", resolver: newCustomerProfileResolver("http://profile.test/api/v1/profile"), headers: http.Header{"Authorization": {"Basic token"}}, wantKind: customerProfileAuthError},
		{name: "not configured", resolver: nil, headers: http.Header{"Authorization": {"Bearer token"}}, wantKind: customerProfileUnavailableError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.resolver.Resolve(context.Background(), tt.headers)
			profileErr, ok := err.(customerProfileError)
			if !ok || profileErr.kind != tt.wantKind {
				t.Fatalf("error = %#v, want kind %d", err, tt.wantKind)
			}
		})
	}
}

func TestCustomerProfileResolverUsesProfileVerifiedIdentityAndID(t *testing.T) {
	const customerID = "a0000000-0000-4000-8000-000000000001"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/profile" {
			t.Errorf("unexpected profile request: %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer identity-issued-token" {
			t.Errorf("Authorization = %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get("X-User-ID") != "" || r.Header.Get("X-Customer-ID") != "" {
			t.Errorf("untrusted identity headers were forwarded: %#v", r.Header)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"` + customerID + `","status":"ACTIVE"}`))
	}))
	defer server.Close()

	resolver := newCustomerProfileResolver(server.URL + "/api/v1/profile")
	got, err := resolver.Resolve(context.Background(), http.Header{"Authorization": {"Bearer identity-issued-token"}})
	if err != nil {
		t.Fatal(err)
	}
	if got != customerID {
		t.Fatalf("resolved customer ID = %q, want %q", got, customerID)
	}
}

func TestCustomerProfileResolverFailsClosedForInactiveOrUntrustedProfile(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantKind   customerProfileErrorKind
	}{
		{name: "inactive customer", statusCode: http.StatusOK, body: `{"id":"a0000000-0000-4000-8000-000000000001","status":"PENDING_VERIFICATION"}`, wantKind: customerProfileForbiddenError},
		{name: "profile denies customer", statusCode: http.StatusForbidden, body: `{"code":"CUSTOMER_REQUIRED"}`, wantKind: customerProfileForbiddenError},
		{name: "invalid profile ID", statusCode: http.StatusOK, body: `{"id":"client-supplied","status":"ACTIVE"}`, wantKind: customerProfileUnavailableError},
		{name: "profile unavailable", statusCode: http.StatusServiceUnavailable, body: `unavailable`, wantKind: customerProfileUnavailableError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()
			resolver := newCustomerProfileResolver(server.URL)
			_, err := resolver.Resolve(context.Background(), http.Header{"Authorization": {"Bearer token"}})
			profileErr, ok := err.(customerProfileError)
			if !ok || profileErr.kind != tt.wantKind {
				t.Fatalf("error = %#v, want kind %d", err, tt.wantKind)
			}
		})
	}
}

func TestCustomerProfileHTTPErrorMatchesContractAuthStates(t *testing.T) {
	for _, tt := range []struct {
		kind   customerProfileErrorKind
		status int
		code   string
	}{
		{kind: customerProfileAuthError, status: http.StatusUnauthorized, code: "UNAUTHENTICATED"},
		{kind: customerProfileForbiddenError, status: http.StatusForbidden, code: "CUSTOMER_FORBIDDEN"},
		{kind: customerProfileUnavailableError, status: http.StatusServiceUnavailable, code: "CUSTOMER_PROFILE_UNAVAILABLE"},
	} {
		status, code, _ := customerProfileHTTPError(customerProfileError{kind: tt.kind})
		if status != tt.status || code != tt.code {
			t.Errorf("mapping for kind %d = (%d, %q), want (%d, %q)", tt.kind, status, code, tt.status, tt.code)
		}
	}
}
