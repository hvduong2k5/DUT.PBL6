package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/config"
)

func run() error {
	cfg, err := config.Load(os.Getenv)
	if err != nil {
		return err
	}
	if cfg.Mode != "sandbox" {
		return errors.New("integration adapters are not installed at this implementation checkpoint")
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	handler, closeResources, err := assemble(ctx, cfg, os.Getenv, logger)
	if err != nil {
		return err
	}
	server := &http.Server{Addr: cfg.ListenAddress, Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 32 * 1024}
	logger.Info("traceability listener starting", "address", cfg.ListenAddress, "mode", cfg.Mode, "checkpoint", "P6-workers")
	return serve(ctx, server, server.ListenAndServe, closeResources, cfg.ShutdownTimeout)
}

func main() {
	if err := run(); err != nil {
		slog.Error("traceability stopped", "error", err)
		os.Exit(1)
	}
}
