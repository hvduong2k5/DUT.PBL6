package httpapi

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/care-service/internal/domain"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type RouterConfig struct {
	InternalToken string
	Pool          *pgxpool.Pool
	Storage       domain.ObjectStorage
	MediaHandler  *MediaHandler
	TicketHandler *TicketHandler
}

func NewRouter(cfg RouterConfig) http.Handler {
	router := chi.NewRouter()
	router.Use(chimiddleware.RealIP)
	router.Use(chimiddleware.Timeout(15 * time.Second))
	router.Use(chimiddleware.Recoverer)
	router.Use(RequestContext)
	router.Use(ObserveRequests)
	router.Use(func(next http.Handler) http.Handler {
		return otelhttp.NewHandler(next, "care.http")
	})

	router.Get("/health/live", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "alive"})
	})
	router.Get("/health/ready", readiness(cfg.Pool, cfg.Storage))
	router.Handle("/metrics", promhttp.Handler())

	router.Route("/api/v1", func(api chi.Router) {
		api.Use(TrustedGatewayAuth(cfg.InternalToken))
		api.Route("/media", func(media chi.Router) {
			media.Post("/upload-sessions", cfg.MediaHandler.CreateUploadSession)
			media.Post("/upload-sessions/{fileID}/complete", cfg.MediaHandler.Complete)
			media.Get("/{fileID}/access", cfg.MediaHandler.Access)
			media.Delete("/{fileID}", cfg.MediaHandler.Delete)
		})
		api.Post("/tickets", cfg.TicketHandler.Create)
		api.Get("/tickets", cfg.TicketHandler.List)
		api.Get("/tickets/{ticketID}", cfg.TicketHandler.Get)
		api.Post("/tickets/{ticketID}/messages", cfg.TicketHandler.AddMessage)
	})

	router.NotFound(func(w http.ResponseWriter, r *http.Request) {
		writeError(w, r, http.StatusNotFound, "ROUTE_NOT_FOUND", "The requested route was not found", nil)
	})
	router.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		writeError(w, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "The HTTP method is not allowed for this route", nil)
	})

	return router
}

func readiness(pool *pgxpool.Pool, storage domain.ObjectStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), time.Second)
		defer cancel()
		databaseReady := pool.Ping(ctx) == nil
		storageReady := storage.Health(ctx) == nil
		if !databaseReady || !storageReady {
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{
				"status": "not_ready", "database": databaseReady, "object_storage": storageReady,
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"status": "ready", "database": true, "object_storage": true,
		})
	}
}
