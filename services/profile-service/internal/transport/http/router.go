package http

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/profile-service/internal/transport/http/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
)

type RouterConfig struct {
	ProfileHandler  *ProfileHandler
	AddressHandler  *AddressHandler
	EmployeeHandler *EmployeeHandler
	Pool            *pgxpool.Pool
	RedisClient     *redis.Client
}

func NewRouter(cfg RouterConfig) http.Handler {
	r := chi.NewRouter()

	// Global Middlewares
	r.Use(chiMiddleware.RealIP)
	r.Use(middleware.RequestLogger)
	r.Use(chiMiddleware.Recoverer)
	r.Use(middleware.PrometheusMiddleware)

	// CORS Config
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-Request-ID", "X-User-ID", "X-User-Role"},
		ExposedHeaders:   []string{"Link", "X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Operational & Observability Endpoints
	r.Get("/metrics", promhttp.Handler().ServeHTTP)
	r.Get("/healthz", healthCheck(cfg.Pool, cfg.RedisClient))
	r.Get("/livez", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status": "alive"}`))
	})
	r.Get("/readyz", readinessCheck(cfg.Pool, cfg.RedisClient))

	// API Routes
	r.Route("/api/v1", func(api chi.Router) {
		api.Use(middleware.AuthContext)

		// Profile Routes
		api.Route("/profile", func(p chi.Router) {
			p.Get("/", cfg.ProfileHandler.GetProfile)
			p.Put("/", cfg.ProfileHandler.UpdateProfile)

			// Addresses Sub-routes
			p.Route("/addresses", func(a chi.Router) {
				a.Get("/", cfg.AddressHandler.ListAddresses)
				a.Post("/", cfg.AddressHandler.CreateAddress)
				a.Post("/validate", cfg.AddressHandler.ValidateAddress)
				a.Put("/{id}/default", cfg.AddressHandler.SwitchDefaultAddress)
				a.Delete("/{id}", cfg.AddressHandler.DeleteAddress)
			})
		})

		// Admin & HR Routes
		api.Route("/admin", func(adm chi.Router) {
			adm.Use(middleware.RequireRole("ADMIN", "HR_MANAGER"))
			adm.Route("/employees", func(e chi.Router) {
				e.Get("/", cfg.EmployeeHandler.ListEmployees)
				e.Post("/", cfg.EmployeeHandler.CreateEmployee)
				e.Get("/{id}", cfg.EmployeeHandler.GetEmployeeDetail)
			})
		})
	})

	return r
}

func healthCheck(pool *pgxpool.Pool, redisClient *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		dbErr := pool.Ping(ctx)
		redisErr := redisClient.Ping(ctx).Err()

		if dbErr != nil || redisErr != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{
				"status": "unhealthy",
				"db":     dbErr == nil,
				"redis":  redisErr == nil,
			})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"status": "healthy",
			"db":     true,
			"redis":  true,
		})
	}
}

func readinessCheck(pool *pgxpool.Pool, redisClient *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 500*time.Millisecond)
		defer cancel()

		if err := pool.Ping(ctx); err != nil {
			writeJSONError(w, http.StatusServiceUnavailable, "database not ready")
			return
		}
		if err := redisClient.Ping(ctx).Err(); err != nil {
			writeJSONError(w, http.StatusServiceUnavailable, "redis not ready")
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	}
}
