package http

import (
	"context"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/hvduong2k5/DUT.PBL6/services/content-service/internal/domain"
	"github.com/hvduong2k5/DUT.PBL6/services/content-service/internal/transport/http/handlers"
	"github.com/hvduong2k5/DUT.PBL6/services/content-service/internal/transport/http/middleware"
	"github.com/rs/zerolog"
	"net/http"
	"runtime/debug"
	"time"
)

type RouterConfig struct {
	Handler     *handlers.Handler
	Verifier    middleware.Verifier
	Idempotency domain.IdempotencyStore
	Ready       func(context.Context) error
	Logger      zerolog.Logger

	// Zero values fall back to the defaults below.
	RequestTimeout   time.Duration
	ReadinessTimeout time.Duration
	MaxBodyBytes     int64
}

func NewRouter(c RouterConfig) http.Handler {
	if c.RequestTimeout <= 0 {
		c.RequestTimeout = 10 * time.Second
	}
	if c.ReadinessTimeout <= 0 {
		c.ReadinessTimeout = time.Second
	}
	if c.MaxBodyBytes <= 0 {
		c.MaxBodyBytes = handlers.DefaultMaxBody
	}
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			start := time.Now()
			rid := uuid.NewString()
			req = req.WithContext(middleware.WithRequestID(req.Context(), rid))
			w.Header().Set("X-Request-ID", rid)
			w.Header().Set("X-Content-Type-Options", "nosniff")
			// Admin and error responses are never cached; published reads opt in.
			w.Header().Set("Cache-Control", "no-store")
			wrapped := chiMiddleware.NewWrapResponseWriter(w, req.ProtoMajor)
			defer func() {
				if p := recover(); p != nil {
					if p == http.ErrAbortHandler {
						panic(p)
					}
					c.Logger.Error().Str("request_id", rid).Interface("panic", p).Bytes("stack", debug.Stack()).Msg("request panic")
					if wrapped.Status() == 0 {
						handlers.Error(wrapped, req, 500, "INTERNAL_ERROR", "internal server error")
					} else {
						c.Logger.Error().Str("request_id", rid).Int("status", wrapped.Status()).Msg("panic after the response had started")
					}
				}
				status := wrapped.Status()
				if status == 0 {
					status = 200
				}
				c.Logger.Info().Str("request_id", rid).Str("method", req.Method).Int("status", status).Dur("duration", time.Since(start)).Msg("HTTP request handled")
			}()
			ctx, cancel := context.WithTimeout(req.Context(), c.RequestTimeout)
			defer cancel()
			next.ServeHTTP(wrapped, req.WithContext(ctx))
		})
	})
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		handlers.Error(w, r, 404, "NOT_FOUND", "route not found")
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		handlers.Error(w, r, 405, "METHOD_NOT_ALLOWED", "method not allowed")
	})
	r.Get("/livez", func(w http.ResponseWriter, r *http.Request) {
		handlers.JSON(w, 200, map[string]string{"status": "alive"})
	})
	health := func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), c.ReadinessTimeout)
		defer cancel()
		if c.Ready == nil || c.Ready(ctx) != nil {
			handlers.JSON(w, 503, map[string]any{"status": "not_ready", "database": false, "code": "NOT_READY", "message": "service is not ready", "requestId": middleware.RequestID(r.Context())})
			return
		}
		handlers.JSON(w, 200, map[string]any{"status": "ready", "database": true})
	}
	r.Get("/readyz", health)
	r.Get("/healthz", health)
	r.Get("/api/v1/articles", c.Handler.List(true))
	r.Get("/api/v1/articles/{slug}", c.Handler.PublicDetail)
	r.Route("/api/v1/admin/articles", func(a chi.Router) {
		a.Use(middleware.Authenticate(c.Verifier, c.Handler.DomainError))
		a.Use(middleware.Idempotency(c.Idempotency, c.MaxBodyBytes, handlers.Error))
		a.Get("/", c.Handler.List(false))
		a.Post("/", c.Handler.Create)
		a.Get("/{id}", c.Handler.Get)
		a.Patch("/{id}", c.Handler.Update)
		for _, action := range []string{"submit", "approve", "reject", "publish", "hide", "restore"} {
			a.Post("/{id}/"+action, c.Handler.Action(action))
		}
	})
	return r
}
