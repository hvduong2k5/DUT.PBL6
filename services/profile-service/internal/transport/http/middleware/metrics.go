package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "profile_http_requests_total",
			Help: "Total count of HTTP requests processed by profile-service",
		},
		[]string{"method", "path", "status"},
	)

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "profile_http_request_duration_seconds",
			Help:    "Histogram of response latency for HTTP requests in seconds",
			Buckets: []float64{0.001, 0.003, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5},
		},
		[]string{"method", "path"},
	)

	CacheOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "profile_cache_operations_total",
			Help: "Total count of L1 and L2 cache operations",
		},
		[]string{"layer", "operation", "status"}, // layer: l1_ram, l2_redis; status: hit, miss, error
	)

	OutboxLagGauge = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "profile_outbox_lag_count",
			Help: "Count of unpublished events remaining in the transactional outbox table",
		},
	)
)

// PrometheusMiddleware intercepts HTTP requests to record request count and latency histograms.
func PrometheusMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		interceptor := &responseWriterInterceptor{ResponseWriter: w}

		next.ServeHTTP(interceptor, r)

		duration := time.Since(start).Seconds()
		statusCode := interceptor.statusCode
		if statusCode == 0 {
			statusCode = http.StatusOK
		}

		path := r.URL.Path
		// Normalize UUIDs or dynamic path params if needed
		HTTPRequestsTotal.WithLabelValues(r.Method, path, strconv.Itoa(statusCode)).Inc()
		HTTPRequestDuration.WithLabelValues(r.Method, path).Observe(duration)
	})
}
