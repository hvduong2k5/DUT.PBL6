package http

import (
	"context"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/hvduong2k5/DUT.PBL6/services/content-service/internal/transport/http/handlers"
	"github.com/hvduong2k5/DUT.PBL6/services/content-service/internal/transport/http/middleware"
	"github.com/rs/zerolog"
	"net/http"
	"time"
)

type RouterConfig struct {
	Handler  *handlers.Handler
	Verifier middleware.Verifier
	Ready    func(context.Context) error
	Logger   zerolog.Logger
}

func NewRouter(c RouterConfig) http.Handler {
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			start := time.Now()
			rid := uuid.NewString()
			w.Header().Set("X-Request-ID", rid)
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("Cache-Control", "no-store")
			wrapped := chiMiddleware.NewWrapResponseWriter(w, req.ProtoMajor)
			defer func() {
				if recover() != nil {
					c.Logger.Error().Str("request_id", rid).Msg("request panic")
					if wrapped.Status() == 0 {
						handlers.Error(wrapped, 500, "INTERNAL_ERROR", "internal server error")
					}
				}
				status := wrapped.Status()
				if status == 0 {
					status = 200
				}
				c.Logger.Info().Str("request_id", rid).Str("method", req.Method).Int("status", status).Dur("duration", time.Since(start)).Msg("HTTP request handled")
			}()
			ctx, cancel := context.WithTimeout(req.Context(), 10*time.Second)
			defer cancel()
			next.ServeHTTP(wrapped, req.WithContext(ctx))
		})
	})
	r.NotFound(func(w http.ResponseWriter, r *http.Request) { handlers.Error(w, 404, "NOT_FOUND", "route not found") })
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		handlers.Error(w, 405, "METHOD_NOT_ALLOWED", "method not allowed")
	})
	r.Get("/livez", func(w http.ResponseWriter, r *http.Request) {
		handlers.JSON(w, 200, map[string]string{"status": "alive"})
	})
	health := func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), time.Second)
		defer cancel()
		if c.Ready == nil || c.Ready(ctx) != nil {
			handlers.JSON(w, 503, map[string]any{"status": "not_ready", "database": false})
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
		a.Get("/", c.Handler.List(false))
		a.Post("/", c.Handler.Create)
		a.Get("/{id}", c.Handler.Get)
		a.Patch("/{id}", c.Handler.Update)
		for _, action := range []string{"submit", "approve", "reject", "publish", "hide"} {
			a.Post("/{id}/"+action, c.Handler.Action(action))
		}
	})
	return r
}
