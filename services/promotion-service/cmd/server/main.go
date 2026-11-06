package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/promotion-service/internal/config"
	"github.com/omamx/promotion-service/migrations"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	zerolog.TimeFieldFormat = time.RFC3339Nano
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: "15:04:05"}).With().Caller().Logger()
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("invalid configuration")
	}

	rootCtx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	pool, err := pgxpool.New(rootCtx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("create PostgreSQL pool")
	}
	defer pool.Close()

	dbCtx, dbCancel := context.WithTimeout(rootCtx, cfg.DBTimeout)
	if err := pool.Ping(dbCtx); err != nil {
		dbCancel()
		log.Fatal().Err(err).Msg("connect to PostgreSQL")
	}
	dbCancel()
	if applied, err := migrations.Apply(rootCtx, pool); err != nil {
		log.Fatal().Err(err).Msg("apply database migrations")
	} else if len(applied) > 0 {
		log.Info().Strs("migrations", applied).Msg("database migrations applied")
	}

	router := chi.NewRouter()
	router.Get("/health/live", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	router.Get("/health/ready", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), cfg.DBTimeout)
		defer cancel()
		if err := pool.Ping(ctx); err != nil {
			http.Error(w, "database unavailable", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	server := &http.Server{Addr: cfg.HTTPAddress, Handler: router, ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second}
	go func() {
		log.Info().Str("address", cfg.HTTPAddress).Msg("Promotion HTTP server listening")
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error().Err(err).Msg("HTTP server stopped unexpectedly")
			cancel()
		}
	}()
	<-rootCtx.Done()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownWait)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("HTTP server shutdown failed")
	}
}
