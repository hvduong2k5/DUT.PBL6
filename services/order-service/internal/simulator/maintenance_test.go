package simulator

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// With no database the maintenance check must fail closed: checkout is
// treated as under maintenance and the provider paths answer 503.
func TestCatalogueBlockedFailsClosedWithoutDB(t *testing.T) {
	s := &Simulator{Token: "test-token"}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/address/validate", nil)
	if !s.catalogueBlocked(rec, req) {
		t.Fatal("request not blocked with an unreadable maintenance flag")
	}
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
}

// The controlled internal smoke path (runbook) bypasses the barrier only
// with the internal token in the bypass header.
func TestCatalogueBlockedBypassNeedsInternalToken(t *testing.T) {
	s := &Simulator{Token: "test-token"}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/address/validate", nil)
	req.Header.Set(catalogueBypassHeader, "test-token")
	if s.catalogueBlocked(rec, req) {
		t.Fatal("internal smoke bypass was blocked")
	}
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/address/validate", nil)
	req2.Header.Set(catalogueBypassHeader, "wrong-token")
	if !s.catalogueBlocked(rec2, req2) {
		t.Fatal("bypass with a wrong token was allowed")
	}
}
