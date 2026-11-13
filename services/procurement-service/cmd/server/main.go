package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/procurement-service/internal/application"
	"github.com/omamx/procurement-service/internal/config"
	kafkainfra "github.com/omamx/procurement-service/internal/infrastructure/kafka"
	postgresrepo "github.com/omamx/procurement-service/internal/repository/postgres"
	"github.com/omamx/procurement-service/internal/transport/httpapi"
	"github.com/omamx/procurement-service/internal/worker"
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
	var outboxDone chan struct{}
	if cfg.KafkaBrokers == "" {
		logger.Warn("KAFKA_BROKERS is empty; goods-received events remain durable in the outbox")
	} else {
		producer, producerErr := kafkainfra.NewProducer(strings.Split(cfg.KafkaBrokers, ","))
		if producerErr != nil {
			logger.Error("configure Kafka producer", "error", producerErr)
			os.Exit(1)
		}
		defer func() {
			if err := producer.Close(); err != nil {
				logger.Error("close Kafka producer", "error", err)
			}
		}()
		outboxWorker := worker.NewGoodsReceivedOutboxWorker(repository, producer, logger)
		outboxDone = make(chan struct{})
		go func() {
			defer close(outboxDone)
			outboxWorker.Run(rootCtx)
		}()
		logger.Info("goods-received outbox publisher started", "topic", "procurement.events.v1")
	}
	handler, err := httpapi.NewRouter(service, httpapi.IdentityConfig{
		Issuer: cfg.OIDCIssuer, Audience: cfg.OIDCAudience, JWKSURL: cfg.OIDCJWKSURL,
		AllowLocalHTTP: cfg.OIDCAllowLocalHTTP,
	}, repository.Ping)
	if err != nil {
		logger.Error("configure Identity JWT verification", "error", err)
		os.Exit(1)
	}
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
		if outboxDone != nil {
			select {
			case <-outboxDone:
			case <-ctx.Done():
				logger.Warn("timed out waiting for goods-received outbox publisher shutdown")
			}
		}
	case err := <-serverErr:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("HTTP server stopped", "error", err)
			os.Exit(1)
		}
	}
}
