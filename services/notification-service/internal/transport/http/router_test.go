package httptransport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/omamx/notification-service/internal/adapters/identity"
	"github.com/omamx/notification-service/internal/domain"
	"github.com/omamx/notification-service/internal/usecase"
	"net/http/httptest"
	"strings"
	"testing"
)

type storeStub struct {
	calls     int
	err       error
	duplicate bool
}

func (s *storeStub) Accept(context.Context, string, string, domain.Command) (domain.Result, error) {
	s.calls++
	return domain.Result{Request: domain.Request{ID: "33333333-3333-4333-8333-333333333333", Status: "WAITING_CONTEXT"}, Duplicate: s.duplicate}, s.err
}
func (s *storeStub) Get(context.Context, string, string) (domain.Request, error) {
	s.calls++
	return domain.Request{}, domain.ErrNotFound
}
func (s *storeStub) Ping(context.Context) error { return s.err }
func command() domain.Command {
	return domain.Command{SourceEvent: domain.SourceEvent{ID: "11111111-1111-4111-8111-111111111111", Source: "order-service", Type: "vn.omama.order.placed.v3", AggregateID: "22222222-2222-4222-8222-222222222222", AggregateVersion: 2, Cohort: "order-core-1"}, Purpose: "ORDER_CONFIRMATION", Audience: "BUYER", RecipientRef: "order:22222222-2222-4222-8222-222222222222:buyer", Locale: "vi-VN", ContextRef: "opaque-context"}
}
func TestIntakeHTTP(t *testing.T) {
	token := strings.Repeat("x", 32)
	body, _ := json.Marshal(command())
	for _, tc := range []struct {
		name, body, auth, key string
		err                   error
		duplicate             bool
		want, calls           int
	}{
		{name: "accepted", body: string(body), auth: "Bearer " + token, key: "one", want: 202, calls: 1},
		{name: "duplicate", body: string(body), auth: "Bearer " + token, key: "one", duplicate: true, want: 200, calls: 1},
		{name: "unauthenticated", body: string(body), key: "one", want: 401},
		{name: "forbidden purpose", body: strings.Replace(string(body), "ORDER_CONFIRMATION", "OTP", 1), auth: "Bearer " + token, key: "one", want: 403},
		{name: "unknown field", body: strings.Replace(string(body), "opaque-context", "opaque-context", 1)[:len(body)-1] + `,"secret":"canary"}`, auth: "Bearer " + token, key: "one", want: 400},
		{name: "trailing JSON", body: string(body) + ` {}`, auth: "Bearer " + token, key: "one", want: 400},
		{name: "invalid key", body: string(body), auth: "Bearer " + token, key: "bad key", want: 422},
		{name: "invalid recipient", body: strings.Replace(string(body), ":buyer", ":other", 1), auth: "Bearer " + token, key: "one", want: 422},
		{name: "DB unavailable", body: string(body), auth: "Bearer " + token, key: "one", err: errors.New("private-db-secret"), want: 503, calls: 1},
		{name: "conflict", body: string(body), auth: "Bearer " + token, key: "one", err: domain.ErrConflict, want: 409, calls: 1},
		{name: "too large", body: `{"context_ref":"` + strings.Repeat("x", 65536) + `"}`, auth: "Bearer " + token, key: "one", want: 413},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &storeStub{err: tc.err, duplicate: tc.duplicate}
			api := API{Service: usecase.Service{Store: store}, Auth: identity.Local{Token: token}, Ready: store.Ping}
			req := httptest.NewRequest("POST", "/api/v1/internal/notification-requests", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			if tc.auth != "" {
				req.Header.Set("Authorization", tc.auth)
			}
			req.Header.Set("Idempotency-Key", tc.key)
			w := httptest.NewRecorder()
			api.Router().ServeHTTP(w, req)
			if w.Code != tc.want || store.calls != tc.calls {
				t.Fatalf("status %d want %d; calls %d want %d; body %s", w.Code, tc.want, store.calls, tc.calls, w.Body.String())
			}
			if strings.Contains(w.Body.String(), "private-db-secret") || strings.Contains(w.Body.String(), "canary") {
				t.Fatal("sensitive error leaked")
			}
		})
	}
}
func TestHealthAndForeignRequest(t *testing.T) {
	token := strings.Repeat("x", 32)
	store := &storeStub{err: errors.New("down")}
	api := API{Service: usecase.Service{Store: store}, Auth: identity.Local{Token: token}, Ready: store.Ping}
	for path, want := range map[string]int{"/health/live": 200, "/health/ready": 503, "/api/v1/internal/notification-requests/33333333-3333-4333-8333-333333333333": 404} {
		req := httptest.NewRequest("GET", path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		api.Router().ServeHTTP(w, req)
		if w.Code != want {
			t.Fatalf("%s: %d", path, w.Code)
		}
	}
}
