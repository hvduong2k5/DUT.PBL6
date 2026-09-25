package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/profile-service/internal/config"
	"github.com/omamx/profile-service/internal/infrastructure/cache"
	"github.com/omamx/profile-service/internal/infrastructure/kafka"
	"github.com/omamx/profile-service/internal/infrastructure/security"
	"github.com/omamx/profile-service/internal/repository"
	transportGRPC "github.com/omamx/profile-service/internal/transport/grpc"
	transportHTTP "github.com/omamx/profile-service/internal/transport/http"
	"github.com/omamx/profile-service/internal/transport/http/middleware"
	"github.com/omamx/profile-service/internal/usecase"
	"github.com/omamx/profile-service/internal/worker"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// 1. Initialize Structured Logging
	zerolog.TimeFieldFormat = time.RFC3339Nano
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: "15:04:05"}).With().Caller().Logger()
	log.Info().Msg("Starting MS-15 Profile Service (Mè Xửng O Mạ OCOP)...")

	// 2. Load Configuration
	cfg := config.LoadConfig()
	log.Info().
		Str("http_port", cfg.HTTPPort).
		Str("grpc_port", cfg.GRPCPort).
		Str("db_url", cfg.DatabaseURL).
		Str("redis_addr", cfg.RedisAddr).
		Msg("Configuration loaded successfully")

	rootCtx, rootCancel := context.WithCancel(context.Background())
	defer rootCancel()

	// 3. Connect PostgreSQL 16 Pool
	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to parse database configuration")
	}
	poolConfig.MaxConns = int32(cfg.DBMaxConns)
	poolConfig.MinConns = int32(cfg.DBMinConns)
	poolConfig.MaxConnIdleTime = time.Duration(cfg.DBMaxIdleSec) * time.Second

	dbPool, err := pgxpool.NewWithConfig(rootCtx, poolConfig)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed connecting to PostgreSQL pool")
	}
	defer dbPool.Close()

	if err := dbPool.Ping(rootCtx); err != nil {
		log.Warn().Err(err).Msg("PostgreSQL ping failed on startup (will retry in background)")
	} else {
		log.Info().Msg("PostgreSQL 16 connection pool initialized successfully")
	}

	// 4. Connect Redis 7 Cluster / Single Instance
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	defer redisClient.Close()

	if err := redisClient.Ping(rootCtx).Err(); err != nil {
		log.Warn().Err(err).Msg("Redis ping failed on startup (will retry in background)")
	} else {
		log.Info().Msg("Redis 7 connection established")
	}

	// 5. Initialize Infrastructure Clients
	var vaultClient security.VaultKMSClient
	if cfg.UseVaultMockDev {
		log.Info().Msg("Using In-Memory Mock Vault KMS for Development / Testing Mode")
		vaultClient = security.NewMockVaultClient()
	} else {
		log.Info().Str("vault_addr", cfg.VaultAddr).Msg("Connecting to HashiCorp Vault Transit Engine")
		vaultClient = security.NewHTTPVaultClient(cfg.VaultAddr, cfg.VaultToken)
	}

	envelopeEncryptor := security.NewEnvelopeEncryptor(vaultClient, cfg.VaultKeyName, cfg.DEKCacheTTL)
	kafkaProducer := kafka.NewKafkaProducer(cfg.KafkaBrokers)
	defer kafkaProducer.Close()

	// 6. Initialize Dual-Layer Cache & Start Cross-Pod Pub/Sub Subscriber
	podID, _ := os.Hostname()
	dualLayerCache, err := cache.NewDualLayerCache(podID, cfg.L1CacheCap, redisClient)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed initializing Dual-Layer Cache")
	}
	defer dualLayerCache.Close()

	if err := dualLayerCache.StartSubscriber(rootCtx); err != nil {
		log.Warn().Err(err).Msg("DualLayerCache Pub/Sub subscriber warning (will retry on redis live)")
	} else {
		log.Info().Str("channel", cache.DefaultInvalidationChannel).Msg("Redis Pub/Sub cache invalidation subscriber started")
	}

	// 7. Initialize Repositories
	custRepo := repository.NewCustomerRepository(dbPool)
	addrRepo := repository.NewAddressRepository(dbPool)
	empRepo := repository.NewEmployeeRepository(dbPool)
	fuzzyMatcher := usecase.NewAddressFuzzyMatcher()

	// 8. Initialize Usecases
	custUsecase := usecase.NewCustomerUsecase(custRepo, dualLayerCache, dbPool)
	addrUsecase := usecase.NewAddressUsecase(addrRepo, fuzzyMatcher, dualLayerCache, dbPool)
	empUsecase := usecase.NewEmployeeUsecase(empRepo, envelopeEncryptor)

	// 9. Initialize & Start Background Workers
	outboxPublisher := worker.NewOutboxPublisher(dbPool, kafkaProducer, cfg.OutboxBatchSize, cfg.OutboxPollInterval)
	outboxPublisher.SetLagMetricCallback(func(count float64) {
		middleware.OutboxLagGauge.Set(count)
	})
	outboxPublisher.Start(rootCtx)
	log.Info().Msg("Transactional Outbox Publisher worker started (SELECT FOR UPDATE SKIP LOCKED)")

	complianceChecker := worker.NewComplianceChecker(dbPool, empRepo, 24*time.Hour)
	complianceChecker.Start(rootCtx)
	log.Info().Msg("VSATTP Food Safety Compliance Checker cron worker started")

	// 10. Start HTTP REST Server (Chi Router) on Port 8080
	httpRouter := transportHTTP.NewRouter(transportHTTP.RouterConfig{
		ProfileHandler:  transportHTTP.NewProfileHandler(custUsecase),
		AddressHandler:  transportHTTP.NewAddressHandler(addrUsecase),
		EmployeeHandler: transportHTTP.NewEmployeeHandler(empUsecase),
		Pool:            dbPool,
		RedisClient:     redisClient,
	})

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.HTTPPort),
		Handler:      httpRouter,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info().Str("port", cfg.HTTPPort).Msg("HTTP REST Server listening")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("HTTP Server crashed")
		}
	}()

	// 11. Start gRPC Server on Port 50051
	grpcProfileServer := transportGRPC.NewProfileGRPCServer(custUsecase, addrUsecase)
	grpcServer, err := transportGRPC.NewServer(cfg.GRPCPort, grpcProfileServer)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed creating gRPC Server")
	}

	go func() {
		if err := grpcServer.Start(); err != nil {
			log.Fatal().Err(err).Msg("gRPC Server crashed")
		}
	}()

	// 12. Graceful Shutdown Signal Trap
	shutdownSig := make(chan os.Signal, 1)
	signal.Notify(shutdownSig, syscall.SIGINT, syscall.SIGTERM)

	sigReceived := <-shutdownSig
	log.Info().Str("signal", sigReceived.String()).Msg("Graceful shutdown signal received. Draining in-flight requests...")

	// 15-second grace period for in-flight requests to complete
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelShutdown()

	// Stop background workers
	outboxPublisher.Stop()
	complianceChecker.Stop()

	// Graceful stop HTTP & gRPC
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("HTTP Server forced shutdown error")
	} else {
		log.Info().Msg("HTTP Server drained and closed cleanly")
	}

	grpcServer.Stop()

	log.Info().Msg("MS-15 Profile Service shutdown completed successfully. Bye!")
}
