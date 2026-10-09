package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/care-service/internal/config"
	carekafka "github.com/omamx/care-service/internal/infrastructure/kafka"
	"github.com/omamx/care-service/internal/infrastructure/scanner"
	"github.com/omamx/care-service/internal/infrastructure/storage"
	"github.com/omamx/care-service/internal/infrastructure/telemetry"
	"github.com/omamx/care-service/internal/media"
	postgresrepo "github.com/omamx/care-service/internal/repository/postgres"
	"github.com/omamx/care-service/internal/tickets"
	"github.com/omamx/care-service/internal/transport/httpapi"
	"github.com/omamx/care-service/internal/worker"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	zerolog.TimeFieldFormat = time.RFC3339Nano
	log.Logger = zerolog.New(os.Stdout).With().Timestamp().Str("service", "care-service").Logger()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("invalid configuration")
	}
	if level, parseErr := zerolog.ParseLevel(cfg.LogLevel); parseErr == nil {
		log.Logger = log.Logger.Level(level)
	}
	rootCtx, cancelRoot := context.WithCancel(context.Background())
	defer cancelRoot()

	shutdownTelemetry, err := telemetry.Init(rootCtx, telemetry.Config{
		Endpoint: cfg.OTLPEndpoint, Insecure: cfg.OTLPInsecure, ServiceName: cfg.OTelServiceName,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("initialize telemetry")
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdownTelemetry(ctx); err != nil {
			log.Error().Err(err).Msg("shutdown telemetry")
		}
	}()

	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("parse database configuration")
	}
	poolConfig.MaxConns = cfg.DBMaxConns
	poolConfig.MinConns = cfg.DBMinConns
	poolConfig.MaxConnIdleTime = cfg.DBMaxConnIdle
	pool, err := pgxpool.NewWithConfig(rootCtx, poolConfig)
	if err != nil {
		log.Fatal().Err(err).Msg("open PostgreSQL pool")
	}
	defer pool.Close()
	if err := pool.Ping(rootCtx); err != nil {
		log.Fatal().Err(err).Msg("PostgreSQL is required at startup")
	}

	objectStorage, err := storage.NewS3(storage.S3Config{
		Endpoint: cfg.S3Endpoint, AccessKey: cfg.S3AccessKey, SecretKey: cfg.S3SecretKey,
		Bucket: cfg.S3Bucket, Region: cfg.S3Region, UseSSL: cfg.S3UseSSL,
		AutoCreateBucket: cfg.S3AutoCreateBucket,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("create object storage client")
	}
	if err := objectStorage.EnsureBucket(rootCtx); err != nil {
		log.Fatal().Err(err).Msg("ensure private media bucket")
	}

	mediaRepository := postgresrepo.NewMediaRepository(pool)
	ticketRepository := postgresrepo.NewTicketRepository(pool, cfg.KafkaCareTopic)
	mediaService := media.NewService(mediaRepository, objectStorage, scanner.Noop{}, media.Config{
		Bucket: cfg.S3Bucket, UploadTTL: cfg.MediaUploadURLTTL,
		AccessTTL: cfg.MediaAccessURLTTL, OrphanTTL: cfg.MediaOrphanTTL,
	})
	ticketService := tickets.NewService(ticketRepository, nil)

	producer := carekafka.NewProducer(cfg.KafkaBrokers)
	defer producer.Close()
	outbox := worker.NewOutbox(pool, producer, cfg.OutboxPollInterval, cfg.OutboxBatchSize)
	outbox.SetLagCallback(httpapi.OutboxLag.Set)
	outbox.Start(rootCtx)

	handler := httpapi.NewRouter(httpapi.RouterConfig{
		InternalToken: cfg.InternalToken, Pool: pool, Storage: objectStorage,
		MediaHandler:  httpapi.NewMediaHandler(mediaService),
		TicketHandler: httpapi.NewTicketHandler(ticketService),
	})
	server := &http.Server{
		Addr:              fmt.Sprintf(":%s", cfg.HTTPPort),
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      20 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverError := make(chan error, 1)
	go func() {
		log.Info().Str("port", cfg.HTTPPort).Msg("Care Service HTTP server listening")
		serverError <- server.ListenAndServe()
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	select {
	case signal := <-signals:
		log.Info().Str("signal", signal.String()).Msg("shutdown requested")
	case err := <-serverError:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Error().Err(err).Msg("HTTP server stopped unexpectedly")
		}
	}

	cancelRoot()
	outbox.Stop()
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelShutdown()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("HTTP server graceful shutdown failed")
	}
}
