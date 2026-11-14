package main

import (
	"context"
	"errors"
	"github.com/hvduong2k5/DUT.PBL6/services/content-service/internal/config"
	"github.com/hvduong2k5/DUT.PBL6/services/content-service/internal/infrastructure/postgres"
	"github.com/hvduong2k5/DUT.PBL6/services/content-service/internal/repository"
	transport "github.com/hvduong2k5/DUT.PBL6/services/content-service/internal/transport/http"
	"github.com/hvduong2k5/DUT.PBL6/services/content-service/internal/transport/http/handlers"
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
	s := usecase.New(repository.New(pool))
	h := handlers.New(s, log)
	// Fail closed: wire a real verified-identity adapter here when its contract is available.
	r := transport.NewRouter(transport.RouterConfig{Handler: h, Logger: log, Ready: func(ctx context.Context) error { return postgres.Ready(ctx, pool) }})
	srv := &http.Server{Addr: ":" + c.HTTPPort, Handler: r, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 << 10}
	listener, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		log.Error().Msg("HTTP listen failed")
		return err
	}
	done := make(chan error, 1)
	go func() { done <- srv.Serve(listener) }()
	log.Info().Str("port", c.HTTPPort).Msg("HTTP server listening; admin identity integration unavailable")
	select {
	case err = <-done:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	case <-ctx.Done():
	}
	drain, cancelDrain := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelDrain()
	if err = srv.Shutdown(drain); err != nil {
		_ = srv.Close()
		return err
	}
	log.Info().Msg("graceful shutdown complete")
	return nil
}
