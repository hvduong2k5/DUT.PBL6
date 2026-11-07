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

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/procurement-service/internal/application"
	"github.com/omamx/procurement-service/internal/config"
	postgresrepo "github.com/omamx/procurement-service/internal/repository/postgres"
	"github.com/omamx/procurement-service/internal/transport/httpapi"
	"github.com/omamx/procurement-service/migrations"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("service", "procurement-service")
	cfg, err := config.Load()
	if err != nil {
		logger.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		logger.Error("parse database configuration", "error", err)
		os.Exit(1)
	}
	poolCfg.ConnConfig.ConnectTimeout = cfg.DBTimeout
	pool, err := pgxpool.NewWithConfig(rootCtx, poolCfg)
	if err != nil {
		logger.Error("create database pool", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	if err := pool.Ping(rootCtx); err != nil {
		logger.Error("connect to database", "error", err)
		os.Exit(1)
	}
	if applied, err := migrations.Apply(rootCtx, pool); err != nil {
		logger.Error("apply database migrations", "error", err)
		os.Exit(1)
	} else if len(applied) > 0 {
		logger.Info("database migrations applied", "migrations", applied)
	}

	repository := postgresrepo.New(pool)
	service := application.NewService(repository)
	handler := httpapi.NewRouter(service, cfg.InternalToken, repository.Ping)
	server := &http.Server{
		Addr: cfg.HTTPAddress, Handler: handler,
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second,
		WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second,
	}
	serverErr := make(chan error, 1)
	go func() {
		logger.Info("HTTP server started", "address", cfg.HTTPAddress)
		serverErr <- server.ListenAndServe()
	}()

	select {
	case <-rootCtx.Done():
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			logger.Error("graceful shutdown failed", "error", err)
		}
	case err := <-serverErr:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("HTTP server stopped", "error", err)
			os.Exit(1)
		}
	}
}
