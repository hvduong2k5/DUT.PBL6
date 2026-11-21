package main

import (
	"context"
	"errors"
	"github.com/hvduong2k5/DUT.PBL6/services/content-service/internal/config"
	"github.com/hvduong2k5/DUT.PBL6/services/content-service/internal/infrastructure/postgres"
	"github.com/hvduong2k5/DUT.PBL6/services/content-service/internal/repository"
	transport "github.com/hvduong2k5/DUT.PBL6/services/content-service/internal/transport/http"
	"github.com/hvduong2k5/DUT.PBL6/services/content-service/internal/transport/http/handlers"
	"github.com/hvduong2k5/DUT.PBL6/services/content-service/internal/transport/http/middleware"
	"github.com/hvduong2k5/DUT.PBL6/services/content-service/internal/usecase"
	"github.com/rs/zerolog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	if err := run(); err != nil {
		log := zerolog.New(os.Stderr)
		log.Error().Msg("content service stopped with an error")
		os.Exit(1)
	}
}
func run() error {
	c, err := config.LoadConfig()
	if err != nil {
		log := zerolog.New(os.Stderr)
		log.Error().Err(err).Msg("configuration validation failed")
		return err
	}
	log := zerolog.New(os.Stdout).Level(c.LogLevel).With().Timestamp().Str("service", "content-service").Logger()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	pool, err := postgres.Open(ctx, c)
	if err != nil {
		log.Error().Msg("invalid database configuration")
		return err
	}
	defer pool.Close()
	probe, cancel := context.WithTimeout(ctx, 3*time.Second)
	if err = postgres.Ready(probe, pool); err != nil {
		log.Warn().Msg("database or migrations unavailable; readiness will fail until restored")
	}
	cancel()
	repo := repository.New(pool)
	h := handlers.New(usecase.New(repo), log, c.MaxBodyBytes)
	verifier := staffVerifier(c, log)
	r := transport.NewRouter(transport.RouterConfig{
		Handler: h, Logger: log, Verifier: verifier, Idempotency: repo,
		Ready:          func(ctx context.Context) error { return postgres.Ready(ctx, pool) },
		RequestTimeout: c.RequestTimeout, ReadinessTimeout: c.ReadinessTimeout, MaxBodyBytes: c.MaxBodyBytes,
	})
	srv := &http.Server{Addr: ":" + c.HTTPPort, Handler: r, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: c.RequestTimeout, WriteTimeout: c.RequestTimeout + 5*time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 << 10}
	listener, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		log.Error().Msg("HTTP listen failed")
		return err
	}
	done := make(chan error, 1)
	go func() { done <- srv.Serve(listener) }()
	log.Info().Str("port", c.HTTPPort).Bool("admin_enabled", verifier != nil).Msg("HTTP server listening")
	select {
	case err = <-done:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	case <-ctx.Done():
	}
	drain, cancelDrain := context.WithTimeout(context.Background(), c.ShutdownTimeout)
	defer cancelDrain()
	if err = srv.Shutdown(drain); err != nil {
		_ = srv.Close()
		return err
	}
	log.Info().Msg("graceful shutdown complete")
	return nil
}

// staffVerifier returns nil (admin API answers 401) unless the issuer and public
// key are configured and valid; a misconfiguration never opens the admin API.
func staffVerifier(c config.Config, log zerolog.Logger) middleware.Verifier {
	if c.JWTIssuer == "" || c.JWTPublicKeyFile == "" {
		log.Warn().Msg("CONTENT_JWT_ISSUER or CONTENT_JWT_PUBLIC_KEY not set; admin API stays closed (401)")
		return nil
	}
	pem, err := os.ReadFile(c.JWTPublicKeyFile)
	if err != nil {
		log.Error().Msg("cannot read CONTENT_JWT_PUBLIC_KEY; admin API stays closed (401)")
		return nil
	}
	v, err := middleware.NewJWTVerifier(pem, c.JWTIssuer, c.StaffAudience)
	if err != nil {
		log.Error().Err(err).Msg("invalid staff token configuration; admin API stays closed (401)")
		return nil
	}
	return v
}
