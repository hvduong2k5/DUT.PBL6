package httptransport

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/domain"
	"log/slog"
	"net/http"
	"time"
)

type Readiness func(context.Context) error
type HTTPMetrics interface {
	http.Handler
	ObserveHTTP(string, int, time.Duration)
}
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
		w.ResponseWriter.WriteHeader(status)
	}
}
func (w *statusWriter) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(200)
	}
	return w.ResponseWriter.Write(data)
}
func (w *statusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

// NewHealthHandler does not equate a live process with a ready schema or adapter.
func NewHealthHandler(ready Readiness, logger *slog.Logger) http.Handler {
	return withMiddleware(healthMux(ready), logger)
}

func healthMux(ready Readiness) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /livez", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "live"})
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if ready == nil || ready(ctx) != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not_ready"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	})
	return mux
}

func withMiddleware(next http.Handler, logger *slog.Logger, metrics ...HTTPMetrics) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		started := time.Now()
		tracked := &statusWriter{ResponseWriter: w}
		w = tracked
		defer func() {
			status := tracked.status
			if status == 0 {
				status = 200
			}
			if len(metrics) > 0 && metrics[0] != nil {
				metrics[0].ObserveHTTP(r.Pattern, status, time.Since(started))
			}
			logger.Info("http request", "route", r.Pattern, "status", status, "duration_ms", time.Since(started).Milliseconds(), "request_id", w.Header().Get("X-Request-Id"))
		}()
		var random [16]byte
		if _, err := rand.Read(random[:]); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error_code": "INTERNAL_ERROR", "user_message": "Unable to process request", "request_id": "unavailable"})
			return
		}
		requestID := hex.EncodeToString(random[:])
		w.Header().Set("X-Request-Id", requestID)
		trace, err := domain.NewTraceParent()
		if parents := r.Header.Values("Traceparent"); len(parents) == 1 && domain.ValidTraceParent(parents[0]) {
			trace, err = domain.ChildTraceParent(parents[0])
		}
		if err != nil {
			writeJSON(w, 500, map[string]string{"error_code": "INTERNAL_ERROR", "user_message": "Unable to process request", "request_id": requestID})
			return
		}
		w.Header().Set("Traceparent", trace)
		defer func() {
			if recover() != nil {
				logger.Error("request panic recovered", "request_id", requestID)
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error_code": "INTERNAL_ERROR", "user_message": "Unable to process request", "request_id": requestID})
			}
		}()
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
