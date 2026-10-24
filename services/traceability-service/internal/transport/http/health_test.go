package httptransport

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestReadinessAndLiveness(t *testing.T) {
	for _, tt := range []struct {
		name  string
		ready Readiness
		want  int
	}{
		{"missing readiness", nil, 503},
		{"database failure", func(context.Context) error { return errors.New("postgres://secret@internal") }, 503},
		{"ready", func(ctx context.Context) error {
			deadline, ok := ctx.Deadline()
			if !ok || time.Until(deadline) > 2*time.Second {
				t.Error("missing bounded deadline")
			}
			return nil
		}, 200},
		{"panic", func(context.Context) error { panic("private database details") }, 500},
	} {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHealthHandler(tt.ready, slog.New(slog.NewTextHandler(io.Discard, nil)))
			for path, want := range map[string]int{"/livez": 200, "/readyz": tt.want, "/api/v1/admin/trace/records": 404} {
				r := httptest.NewRequest("GET", path, nil)
				r.Header.Set("X-Request-Id", "untrusted-request-id")
				w := httptest.NewRecorder()
				h.ServeHTTP(w, r)
				if w.Code != want {
					t.Fatalf("%s: got %d want %d", path, w.Code, want)
				}
				if id := w.Header().Get("X-Request-Id"); len(id) != 32 || id == "untrusted-request-id" {
					t.Fatal("unsafe request ID")
				}
				if strings.Contains(w.Body.String(), "secret") || strings.Contains(w.Body.String(), "private") {
					t.Fatal("leaked readiness internals")
				}
				if w.Header().Get("Cache-Control") != "no-store" {
					t.Fatal("missing no-store")
				}
			}
		})
	}
}
