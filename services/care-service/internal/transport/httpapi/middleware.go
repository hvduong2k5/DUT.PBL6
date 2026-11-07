package httpapi

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/omamx/care-service/internal/requestcontext"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/rs/zerolog/log"
)

var (
	httpRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "care_http_requests_total", Help: "Total Care Service HTTP requests.",
	}, []string{"method", "route", "status"})
	httpDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "care_http_request_duration_seconds", Help: "Care Service HTTP request duration.",
	}, []string{"method", "route"})
	OutboxLag = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "care_outbox_pending_events", Help: "Unpublished Care Service outbox events.",
	})
)

func RequestID(ctx context.Context) string {
	return requestcontext.RequestID(ctx)
}

func CorrelationID(ctx context.Context) string {
	return requestcontext.CorrelationID(ctx)
}

func RequestContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if !validTrackingID(requestID) {
			requestID = uuid.NewString()
		}
		correlationID := r.Header.Get("X-Correlation-ID")
		if !validTrackingID(correlationID) {
			correlationID = uuid.NewString()
		}
		w.Header().Set("X-Request-ID", requestID)
		w.Header().Set("X-Correlation-ID", correlationID)
		next.ServeHTTP(w, r.WithContext(requestcontext.WithIDs(r.Context(), requestID, correlationID)))
	})
}

func validTrackingID(value string) bool {
	if len(value) < 1 || len(value) > 100 {
		return false
	}
	for _, character := range value {
		if (character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') || character == '-' || character == '_' ||
			character == '.' || character == ':' {
			continue
		}
		return false
	}
	return true
}

type responseRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *responseRecorder) WriteHeader(status int) {
	if r.status != 0 {
		return
	}
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *responseRecorder) Write(payload []byte) (int, error) {
	if r.status == 0 {
		r.WriteHeader(http.StatusOK)
	}
	written, err := r.ResponseWriter.Write(payload)
	r.bytes += written
	return written, err
}

func ObserveRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &responseRecorder{ResponseWriter: w}
		next.ServeHTTP(recorder, r)
		status := recorder.status
		if status == 0 {
			status = http.StatusOK
		}
		route := chi.RouteContext(r.Context()).RoutePattern()
		if route == "" {
			route = "unmatched"
		}
		duration := time.Since(start)
		httpRequests.WithLabelValues(r.Method, route, strconv.Itoa(status)).Inc()
		httpDuration.WithLabelValues(r.Method, route).Observe(duration.Seconds())
		log.Info().
			Str("request_id", RequestID(r.Context())).
			Str("correlation_id", CorrelationID(r.Context())).
			Str("method", r.Method).
			Str("route", route).
			Int("status", status).
			Int("bytes", recorder.bytes).
			Dur("duration", duration).
			Msg("HTTP request handled")
	})
}
