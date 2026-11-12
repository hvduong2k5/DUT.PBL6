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
	InternalToken      string
	StaffAuthenticator StaffAuthenticator
	RateLimitPerSecond int
	RateLimitBurst     int
	Pool               *pgxpool.Pool
	Storage            domain.ObjectStorage
	MediaHandler       *MediaHandler
	TicketHandler      *TicketHandler
	StaffTicketHandler *StaffTicketHandler
	ReturnHandler      *ReturnHandler
}

func NewRouter(cfg RouterConfig) http.Handler {
	router := chi.NewRouter()
	rateLimiter := NewActorRateLimiter(cfg.RateLimitPerSecond, cfg.RateLimitBurst)
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
		api.Group(func(customer chi.Router) {
			customer.Use(TrustedGatewayAuth(cfg.InternalToken), rateLimiter.Middleware)
			customer.Route("/media", func(media chi.Router) {
				media.Post("/upload-sessions", cfg.MediaHandler.CreateUploadSession)
				media.Post("/upload-sessions/{fileID}/complete", cfg.MediaHandler.Complete)
				media.Get("/{fileID}/access", cfg.MediaHandler.Access)
				media.Delete("/{fileID}", cfg.MediaHandler.Delete)
			})
			customer.Post("/tickets", cfg.TicketHandler.Create)
			customer.Get("/tickets", cfg.TicketHandler.List)
			customer.Get("/tickets/{ticketID}", cfg.TicketHandler.Get)
			customer.Post("/tickets/{ticketID}/messages", cfg.TicketHandler.AddMessage)
			customer.Post("/returns", cfg.ReturnHandler.Create)
			customer.Get("/returns", cfg.ReturnHandler.ListOwned)
			customer.Get("/returns/{caseID}", cfg.ReturnHandler.GetOwned)
		})
		api.Group(func(staff chi.Router) {
			staff.Use(GatewayOrStaffJWTAuth(cfg.InternalToken, cfg.StaffAuthenticator), rateLimiter.Middleware)
			staff.With(RequirePermission("RETURN_CASE_VIEW")).Get("/admin/returns/cases", cfg.ReturnHandler.ListAdmin)
			staff.With(RequirePermission("RETURN_CASE_VIEW")).Get("/admin/returns/cases/{caseID}", cfg.ReturnHandler.GetAdmin)
			staff.Post("/admin/returns/cases/{caseID}/actions", cfg.ReturnHandler.Action)
			staff.With(RequirePermission("TICKET_QUEUE_VIEW")).Get("/admin/support/tickets", cfg.StaffTicketHandler.List)
			staff.With(RequirePermission("TICKET_QUEUE_VIEW")).Get("/admin/support/tickets/{ticketID}", cfg.StaffTicketHandler.Get)
			staff.With(RequirePermission("TICKET_CLAIM")).Post("/admin/support/tickets/{ticketID}/claim", cfg.StaffTicketHandler.Claim)
			staff.With(RequirePermission("TICKET_NOTE_WRITE")).Post("/admin/support/tickets/{ticketID}/internal-notes", cfg.StaffTicketHandler.AddInternalNote)
			staff.With(RequirePermission("TICKET_REPLY")).Post("/admin/support/tickets/{ticketID}/messages", cfg.StaffTicketHandler.AddReply)
			staff.With(RequirePermission("TICKET_TRANSITION")).Post("/admin/support/tickets/{ticketID}/transitions", cfg.StaffTicketHandler.Transition)
		})
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
