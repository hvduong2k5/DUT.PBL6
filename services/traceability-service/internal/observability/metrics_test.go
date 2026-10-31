package observability

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/domain"
)

type statsFunc func(context.Context) (domain.OutboxStats, error)

func (f statsFunc) Stats(ctx context.Context) (domain.OutboxStats, error) { return f(ctx) }
func TestMetricsAndSanitizedFailure(t *testing.T) {
	queue := statsFunc(func(context.Context) (domain.OutboxStats, error) {
		return domain.OutboxStats{Pending: 2, Failed: 1, OldestPendingSeconds: 3}, nil
	})
	metrics := New(queue, true)
	metrics.ObserveHTTP("GET /api/v1/trace/{qr_code}", 200, 10*time.Millisecond)
	metrics.ObserveHTTP("", 404, time.Millisecond)
	metrics.Relay(true, nil)
	metrics.Relay(false, errors.New("private broker error"))
	metrics.Expiry(2, nil)
	metrics.Expiry(0, errors.New("private worker error"))
	metrics.Dependency(nil)
	metrics.Dependency(errors.New("private source error"))
	response := httptest.NewRecorder()
	metrics.ServeHTTP(response, httptest.NewRequest("GET", "/metrics", nil))
	for _, expected := range []string{"traceability_outbox_pending 2", "traceability_outbox_failed 1", "traceability_relay_delivered_total 1", "traceability_expiry_observations_total 2", `route="unmatched"`} {
		if !strings.Contains(response.Body.String(), expected) {
			t.Fatalf("missing %s", expected)
		}
	}
	if strings.Contains(response.Body.String(), "private") {
		t.Fatal("metrics leaked error contents")
	}
	failing := New(statsFunc(func(context.Context) (domain.OutboxStats, error) {
		return domain.OutboxStats{}, errors.New("postgres://credential@private")
	}), false)
	response = httptest.NewRecorder()
	failing.ServeHTTP(response, httptest.NewRequest("GET", "/metrics", nil))
	if response.Code != 503 || strings.Contains(response.Body.String(), "credential") {
		t.Fatal("metric failure leaked credential")
	}
}
