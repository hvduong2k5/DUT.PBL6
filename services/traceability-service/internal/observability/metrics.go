// Package observability exposes bounded-cardinality, secret-free sandbox metrics.
package observability

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/domain"
)

type queueStats interface {
	Stats(context.Context) (domain.OutboxStats, error)
}
type Metrics struct {
	queue                                                             queueStats
	relayEnabled                                                      bool
	mu                                                                sync.Mutex
	requests                                                          map[string]uint64
	latencyCount                                                      uint64
	latencySum                                                        float64
	relayDelivered, relayFailures, expiryObservations, expiryFailures atomic.Uint64
	dependencySuccess, dependencyFailures                             atomic.Uint64
}

func New(queue queueStats, relayEnabled bool) *Metrics {
	return &Metrics{queue: queue, relayEnabled: relayEnabled, requests: map[string]uint64{}}
}
func (m *Metrics) ObserveHTTP(route string, status int, duration time.Duration) {
	if route == "" {
		route = "unmatched"
	}
	key := "route=" + strconv.Quote(route) + ",status=" + strconv.Quote(strconv.Itoa(status))
	m.mu.Lock()
	m.requests[key]++
	m.latencyCount++
	m.latencySum += duration.Seconds()
	m.mu.Unlock()
}
func (m *Metrics) Relay(processed bool, err error) {
	if err != nil {
		m.relayFailures.Add(1)
	} else if processed {
		m.relayDelivered.Add(1)
	}
}
func (m *Metrics) Expiry(count int, err error) {
	if count > 0 {
		m.expiryObservations.Add(uint64(count))
	}
	if err != nil {
		m.expiryFailures.Add(1)
	}
}
func (m *Metrics) Dependency(err error) {
	if err == nil {
		m.dependencySuccess.Add(1)
	} else {
		m.dependencyFailures.Add(1)
	}
}
func (m *Metrics) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	stats, err := m.queue.Stats(ctx)
	if err != nil {
		http.Error(w, "metrics unavailable", 503)
		return
	}
	var text strings.Builder
	m.mu.Lock()
	keys := make([]string, 0, len(m.requests))
	for key := range m.requests {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	text.WriteString("# TYPE traceability_http_requests_total counter\n")
	for _, key := range keys {
		fmt.Fprintf(&text, "traceability_http_requests_total{%s} %d\n", key, m.requests[key])
	}
	fmt.Fprintf(&text, "# TYPE traceability_http_duration_seconds summary\ntraceability_http_duration_seconds_count %d\ntraceability_http_duration_seconds_sum %g\n", m.latencyCount, m.latencySum)
	m.mu.Unlock()
	enabled := 0
	if m.relayEnabled {
		enabled = 1
	}
	fmt.Fprintf(&text, "# TYPE traceability_relay_enabled gauge\ntraceability_relay_enabled %d\n", enabled)
	fmt.Fprintf(&text, "# TYPE traceability_outbox_pending gauge\ntraceability_outbox_pending %d\n# TYPE traceability_outbox_failed gauge\ntraceability_outbox_failed %d\n# TYPE traceability_outbox_oldest_pending_seconds gauge\ntraceability_outbox_oldest_pending_seconds %g\n", stats.Pending, stats.Failed, stats.OldestPendingSeconds)
	for _, counter := range []struct {
		name  string
		value uint64
	}{{"relay_delivered", m.relayDelivered.Load()}, {"relay_errors", m.relayFailures.Load()}, {"expiry_observations", m.expiryObservations.Load()}, {"expiry_errors", m.expiryFailures.Load()}, {"dependency_success", m.dependencySuccess.Load()}, {"dependency_errors", m.dependencyFailures.Load()}} {
		fmt.Fprintf(&text, "# TYPE traceability_%s_total counter\ntraceability_%s_total %d\n", counter.name, counter.name, counter.value)
	}
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	_, _ = w.Write([]byte(text.String()))
}

type Source interface {
	Snapshot(context.Context, domain.Binding) (domain.DependencySnapshot, error)
}
type Dependency struct {
	Next    Source
	Metrics *Metrics
}

func (d Dependency) Snapshot(ctx context.Context, b domain.Binding) (domain.DependencySnapshot, error) {
	snapshot, err := d.Next.Snapshot(ctx, b)
	d.Metrics.Dependency(err)
	return snapshot, err
}
