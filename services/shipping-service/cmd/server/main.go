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
	"github.com/omamx/shipping-service/internal/config"
	shippingv1 "github.com/omamx/shipping-service/internal/gen/shipping/v1"
	shippingv2 "github.com/omamx/shipping-service/internal/gen/shipping/v2"
	shippingkafka "github.com/omamx/shipping-service/internal/infrastructure/kafka"
	"github.com/omamx/shipping-service/internal/infrastructure/ordergrpc"
	postgresrepo "github.com/omamx/shipping-service/internal/repository/postgres"
	"github.com/omamx/shipping-service/internal/transport/grpcapi"
	"github.com/omamx/shipping-service/internal/transport/httpapi"
	"github.com/omamx/shipping-service/internal/worker"
	"github.com/omamx/shipping-service/migrations"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
)

func main() {
	logger := zerolog.New(os.Stdout).With().Timestamp().Str("service", "shipping-service").Logger()
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal().Err(err).Msg("load configuration")
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
	registry := carrier.NewRegistry(carrier.NewFake())
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
	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(grpcapi.AuthUnaryInterceptor(cfg.InternalToken)))
	shippingv1.RegisterShippingServiceServer(grpcServer, grpcapi.NewServer(service))
	if cfg.V2Enabled {
		shippingv2.RegisterShippingServiceServer(grpcServer, grpcapi.NewV2Server(service))
		logger.Info().Msg("Shipping gRPC v2 enabled alongside v1")
	} else {
		logger.Info().Msg("Shipping gRPC v2 disabled by SHIPPING_V2_ENABLED")
	}

	var packingConsumers []*worker.PackingConsumer
	if cfg.OrderGRPCEndpoint != "" {
		orderClient, clientErr := ordergrpc.New(cfg.OrderGRPCEndpoint, cfg.OrderInternalToken)
		if clientErr != nil {
			logger.Fatal().Err(clientErr).Msg("create Order service client")
		}
		defer func() {
			if closeErr := orderClient.Close(); closeErr != nil {
				logger.Error().Err(closeErr).Msg("close Order service client")
			}
		}()
		packingV1Consumer := worker.NewPackingConsumer(
			cfg.KafkaBrokers, cfg.FulfillmentTopic, cfg.FulfillmentGroup,
			service, orderClient, cfg.DefaultCarrier, cfg.DefaultService, logger,
		)
		packingConsumers = append(packingConsumers, packingV1Consumer)
		go packingV1Consumer.Run(rootCtx)
		if cfg.PackingV2Enabled {
			packingV2Consumer := worker.NewPackingV2Consumer(
				cfg.KafkaBrokers, cfg.FulfillmentV2Topic, cfg.FulfillmentV2Group,
				service, orderClient, cfg.DefaultCarrier, cfg.DefaultService, logger,
			)
			packingConsumers = append(packingConsumers, packingV2Consumer)
			go packingV2Consumer.Run(rootCtx)
			logger.Info().Str("topic", cfg.FulfillmentV2Topic).Msg("PackingCompleted v2 consumer enabled alongside v1")
		} else {
			logger.Info().Msg("PackingCompleted v2 consumer disabled by SHIPPING_PACKING_V2_ENABLED")
		}
		defer func() {
			for _, consumer := range packingConsumers {
				if closeErr := consumer.Close(); closeErr != nil {
					logger.Error().Err(closeErr).Msg("close PackingCompleted consumer")
				}
			}
		}()
	} else {
		logger.Warn().Msg("PackingCompleted consumer disabled: SHIPPING_ORDER_GRPC_ENDPOINT is not configured")
	}

	handler := httpapi.NewRouter(service, cfg.InternalToken, cfg.FakeWebhookSecret, logger)
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
