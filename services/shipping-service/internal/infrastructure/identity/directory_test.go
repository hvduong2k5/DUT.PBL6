package identity

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/omamx/shipping-service/internal/application"
)

func TestDirectoryUsesExistingIdentityContractsAndFailsClosed(t *testing.T) {
	id := uuid.New()
	subject := "keycloak-subject"
	for _, tc := range []struct {
		name   string
		status int
		body   string
		want   error
	}{
		{"active", 200, `{"success":true,"data":{"principalId":"` + id.String() + `","idpSubject":"keycloak-subject","principalType":"EMPLOYEE","provisioningStatus":"ACTIVE"}}`, nil},
		{"suspended", 200, `{"success":true,"data":{"principalId":"` + id.String() + `","idpSubject":"keycloak-subject","principalType":"EMPLOYEE","provisioningStatus":"SUSPENDED"}}`, application.ErrDriverIneligible},
		{"customer", 200, `{"success":true,"data":{"principalId":"` + id.String() + `","idpSubject":"keycloak-subject","principalType":"CUSTOMER","provisioningStatus":"ACTIVE"}}`, application.ErrDriverIneligible},
		{"mismatched ID", 200, `{"success":true,"data":{"principalId":"` + uuid.NewString() + `","principalType":"EMPLOYEE","provisioningStatus":"ACTIVE"}}`, application.ErrEmployeeDirectoryUnavailable},
		{"empty data", 200, `{"success":true}`, application.ErrEmployeeDirectoryUnavailable},
		{"failed envelope", 200, `{"success":false}`, application.ErrEmployeeDirectoryUnavailable},
		{"malformed", 200, `not json`, application.ErrEmployeeDirectoryUnavailable},
		{"too large", 200, strings.Repeat("x", (1<<20)+1), application.ErrEmployeeDirectoryUnavailable},
		{"unauthorized", 401, `{}`, application.ErrEmployeeUnauthenticated},
		{"directory grant missing", 403, `{}`, application.ErrEmployeeForbidden},
		{"missing old producer", 400, `{}`, application.ErrDriverIneligible},
		{"missing", 404, `{}`, application.ErrDriverIneligible},
		{"upstream down", 503, `{}`, application.ErrEmployeeDirectoryUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/v1/admin/users/"+id.String() || r.Method != "GET" || r.Header.Get("Authorization") != "Bearer verified-staff" || r.Header.Get("X-Actor-Principal-Id") != "" {
					t.Error("Identity request did not use existing bearer contract")
				}
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()
			d, err := NewDirectory(server.URL, true)
			if err != nil {
				t.Fatal(err)
			}
			if err := d.ValidateDriver(context.Background(), "Bearer verified-staff", id); !errors.Is(err, tc.want) {
				t.Fatalf("got %v want %v", err, tc.want)
			}
		})
	}
	for _, tc := range []struct {
		typ, status, subject string
		want                 error
	}{
		{"EMPLOYEE", "ACTIVE", subject, nil}, {"EMPLOYEE", "SUSPENDED", subject, application.ErrEmployeeForbidden}, {"CUSTOMER", "ACTIVE", subject, application.ErrEmployeeForbidden}, {"EMPLOYEE", "ACTIVE", "wrong-subject", application.ErrEmployeeDirectoryUnavailable},
	} {
		t.Run("actor-"+tc.typ+tc.status+tc.subject, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/v1/users/me" {
					t.Error("wrong actor lookup")
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": map[string]any{"principalId": id.String(), "idpSubject": tc.subject, "principalType": tc.typ, "provisioningStatus": tc.status}})
			}))
			defer server.Close()
			d, _ := NewDirectory(server.URL, true)
			if err := d.ValidateActor(context.Background(), "Bearer staff", id, subject); !errors.Is(err, tc.want) {
				t.Fatalf("actor: %v want %v", err, tc.want)
			}
		})
	}
}

func TestDirectoryDoesNotFollowRedirectsOrAcceptUnsafeOrigins(t *testing.T) {
	targetCalled := false
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { targetCalled = true }))
	defer target.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, 302) }))
	defer source.Close()
	d, _ := NewDirectory(source.URL, true)
	if err := d.ValidateDriver(context.Background(), "Bearer private-credential", uuid.New()); !errors.Is(err, application.ErrEmployeeDirectoryUnavailable) || targetCalled {
		t.Fatal("redirect exposed bearer")
	}
	for _, url := range []string{"http://identity.internal:8080", "https://user:password@identity.test", "https://identity.test/path", "https://identity.test?query=1", "https://identity.test#fragment", "file:///tmp/identity"} {
		if _, err := NewDirectory(url, true); err == nil {
			t.Fatalf("unsafe origin: %s", url)
		}
	}
	if _, err := NewDirectory(source.URL, false); err == nil {
		t.Fatal("production accepted HTTP")
	}
	closed := httptest.NewServer(http.NotFoundHandler())
	origin := closed.URL
	closed.Close()
	offline, _ := NewDirectory(origin, true)
	if err := offline.ValidateDriver(context.Background(), "Bearer staff", uuid.New()); !errors.Is(err, application.ErrEmployeeDirectoryUnavailable) {
		t.Fatalf("network failure: %v", err)
	}
}
