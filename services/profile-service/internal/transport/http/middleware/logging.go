package middleware

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type responseWriterInterceptor struct {
	http.ResponseWriter
	statusCode int
	bytesCount int
}

func (w *responseWriterInterceptor) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *responseWriterInterceptor) Write(b []byte) (int, error) {
	if w.statusCode == 0 {
		w.statusCode = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(b)
	w.bytesCount += n
	return n, err
}

// RequestLogger logs HTTP requests as structured JSON with execution duration and trace correlation.
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = uuid.New().String()
		}
		w.Header().Set("X-Request-ID", reqID)

		start := time.Now()
		interceptor := &responseWriterInterceptor{ResponseWriter: w}

		next.ServeHTTP(interceptor, r)

		duration := time.Since(start)

		log.Info().
			Str("request_id", reqID).
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("remote_ip", r.RemoteAddr).
			Int("status", interceptor.statusCode).
			Int("bytes", interceptor.bytesCount).
			Dur("duration_ms", duration).
			Msg("HTTP request handled")
	})
}
