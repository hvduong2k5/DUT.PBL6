package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/omamx/shipping-service/internal/application"
	"github.com/omamx/shipping-service/internal/carrier"
	"github.com/rs/zerolog"
)

type forbiddenGatewayStaff struct{}

func (forbiddenGatewayStaff) Authenticate(context.Context, string) (StaffPrincipal, error) {
	return StaffPrincipal{}, errStaffForbidden
}

func TestPublicV2AliasesKeepEmployeeAuthorization(t *testing.T) {
	svc := application.NewService(newHTTPMemoryRepository(), carrier.NewRegistry(carrier.NewFake()))
	for _, prefix := range []string{"/api/v1/admin/shipping", "/api/v2/admin/shipping"} {
		for _, suffix := range []string{"/shipments", "/shipments/" + uuid.NewString()} {
			path := prefix + suffix
			for _, auth := range []string{"", "Bearer internal-token", "Bearer staff-jwt-test"} {
				r := httptest.NewRequest("GET", path, nil)
				r.Header.Set("Authorization", auth)
				w := httptest.NewRecorder()
				NewRouter(svc, "internal-token", "fake-secret", "", zerolog.Nop(), fixedStaffAuthenticator{}).ServeHTTP(w, r)
				want := http.StatusUnauthorized
				if auth == "Bearer staff-jwt-test" {
					want = http.StatusOK
					if suffix != "/shipments" {
						want = http.StatusNotFound
					}
				}
				if w.Code != want {
					t.Fatalf("%s %q: got %d want %d", path, auth, w.Code, want)
				}
				if w.Code == 200 && !strings.Contains(w.Body.String(), `"items":[]`) {
					t.Fatalf("invalid provider list: %s", w.Body)
				}
			}
			r := httptest.NewRequest("GET", path, nil)
			w := httptest.NewRecorder()
			NewRouter(svc, "internal-token", "fake-secret", "", zerolog.Nop(), forbiddenGatewayStaff{}).ServeHTTP(w, r)
			if w.Code != http.StatusForbidden {
				t.Fatalf("scoped permission denial lost on %s", path)
			}
		}
	}
}

func TestShippingGatewayRegexUsesPortableSubset(t *testing.T) {
	// Go's RE2, like Kong's Rust expression regex, rejects look-around. Python re
	// accepted the old pattern, so the earlier static routing test missed the defect.
	config, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "api-gateway", "config", "kong.yml"))
	if err != nil {
		t.Fatal(err)
	}
	patterns := regexp.MustCompile(`'(~[^']*/api/v2/admin/shipping[^']*)'`).FindAllStringSubmatch(string(config), -1)
	if len(patterns) != 2 {
		t.Fatalf("expected two exact Shipping routes, got %d", len(patterns))
	}
	for _, pattern := range patterns {
		if _, err := regexp.Compile("^" + strings.TrimPrefix(pattern[1], "~")); err != nil {
			t.Fatalf("unsupported gateway regex %s: %v", pattern[1], err)
		}
	}
	if _, err := regexp.Compile(`^/api/v2/admin/shipping(?=/shipments$)`); err == nil {
		t.Fatal("guard accepted the known-bad look-ahead")
	}
}
