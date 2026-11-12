package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/shipping-service/internal/application"
	"github.com/omamx/shipping-service/internal/carrier"
	ghncarrier "github.com/omamx/shipping-service/internal/carrier/ghn"
	"github.com/omamx/shipping-service/internal/config"
	shippingv1 "github.com/omamx/shipping-service/internal/gen/shipping/v1"
	shippingv2 "github.com/omamx/shipping-service/internal/gen/shipping/v2"
	shippingkafka "github.com/omamx/shipping-service/internal/infrastructure/kafka"
	postgresrepo "github.com/omamx/shipping-service/internal/repository/postgres"
	"github.com/omamx/shipping-service/internal/transport/grpcapi"
	"github.com/omamx/shipping-service/internal/transport/httpapi"
	"github.com/omamx/shipping-service/internal/worker"
	"github.com/omamx/shipping-service/migrations"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func main() {
	logger := zerolog.New(os.Stdout).With().Timestamp().Str("service", "shipping-service").Logger()
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal().Err(err).Msg("load configuration")
	}
	staffAuth, err := httpapi.NewJWTStaffAuthenticator(cfg.OIDCIssuer, cfg.OIDCAudience, cfg.OIDCJWKSURL)
	if err != nil {
		logger.Fatal().Err(err).Msg("configure employee JWT authentication")
	}

	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		logger.Fatal().Err(err).Msg("parse database configuration")
	}
	poolConfig.ConnConfig.ConnectTimeout = cfg.DatabaseTimeout
	pool, err := pgxpool.NewWithConfig(rootCtx, poolConfig)
	if err != nil {
		logger.Fatal().Err(err).Msg("create database pool")
	}
	defer pool.Close()
	if err := pool.Ping(rootCtx); err != nil {
		logger.Fatal().Err(err).Msg("connect to database")
	}
	if applied, err := migrations.Apply(rootCtx, pool); err != nil {
		logger.Fatal().Err(err).Msg("apply database migrations")
	} else if len(applied) > 0 {
		logger.Info().Strs("migrations", applied).Msg("database migrations applied")
	}

	repository := postgresrepo.New(pool)
	providers := []carrier.Carrier{carrier.NewFake()}
	if cfg.GHNEnabled {
		ghnProvider, providerErr := ghncarrier.New(ghncarrier.Config{
			BaseURL: cfg.GHNBaseURL, Token: cfg.GHNToken, ShopID: cfg.GHNShopID,
			FromDistrictID: cfg.GHNFromDistrictID, FromWardCode: cfg.GHNFromWardCode,
			Timeout: cfg.GHNRequestTimeout, ReconcileTimeout: cfg.GHNRecoveryTimeout, QuoteTTL: cfg.GHNQuoteTTL,
		})
		if providerErr != nil {
			logger.Fatal().Err(providerErr).Msg("create GHN carrier")
		}
		providers = append(providers, ghnProvider)
		logger.Info().Int64("shop_id", cfg.GHNShopID).Msg("GHN carrier enabled")
	} else {
		logger.Info().Msg("GHN carrier disabled by SHIPPING_GHN_ENABLED")
	}
	registry := carrier.NewRegistry(providers...)
	service := application.NewService(repository, registry, repository)
	producer := shippingkafka.NewProducer(cfg.KafkaBrokers)
	defer func() {
		if err := producer.Close(); err != nil {
			logger.Error().Err(err).Msg("close Kafka producer")
		}
	}()
	outbox := worker.NewOutbox(pool, producer, cfg.OutboxPollInterval, logger)
	go outbox.Run(rootCtx)

	grpcListener, err := net.Listen("tcp", cfg.GRPCAddress)
	if err != nil {
		logger.Fatal().Err(err).Msg("listen for gRPC")
	}
	grpcOptions := []grpc.ServerOption{grpc.UnaryInterceptor(grpcapi.AuthUnaryInterceptor(cfg.InternalToken))}
	certFile, keyFile := cfg.GRPCTLSCertFile, cfg.GRPCTLSKeyFile
	if certFile != "" || keyFile != "" {
		transport, err := credentials.NewServerTLSFromFile(certFile, keyFile)
		if err != nil {
			logger.Fatal().Err(err).Msg("load Shipping gRPC TLS identity")
		}
		grpcOptions = append(grpcOptions, grpc.Creds(transport))
	}
	grpcServer := grpc.NewServer(grpcOptions...)
	shippingv1.RegisterShippingServiceServer(grpcServer, grpcapi.NewServer(service))
	if cfg.V2Enabled {
		shippingv2.RegisterShippingServiceServer(grpcServer, grpcapi.NewV2Server(service))
		logger.Info().Msg("Shipping gRPC v2 enabled alongside v1")
	} else {
		logger.Info().Msg("Shipping gRPC v2 disabled by SHIPPING_V2_ENABLED")
	}

	handler := httpapi.NewRouter(service, cfg.InternalToken, cfg.FakeWebhookSecret, cfg.GHNWebhookSecret, logger, staffAuth)
	server := &http.Server{
		Addr: cfg.HTTPAddress, Handler: handler,
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second,
		WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second,
	}

	serverErrors := make(chan error, 2)
	go func() {
		logger.Info().Str("address", cfg.HTTPAddress).Msg("shipping HTTP server started")
		serverErrors <- server.ListenAndServe()
	}()
	go func() {
		logger.Info().Str("address", cfg.GRPCAddress).Msg("shipping gRPC server started")
		serverErrors <- grpcServer.Serve(grpcListener)
	}()

	select {
	case <-rootCtx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error().Err(err).Msg("graceful shutdown failed")
		}
		grpcServer.GracefulStop()
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) && !errors.Is(err, grpc.ErrServerStopped) {
			logger.Fatal().Err(err).Msg("server stopped unexpectedly")
		}
	}
}
