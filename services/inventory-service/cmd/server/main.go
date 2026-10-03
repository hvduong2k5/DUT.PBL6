package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"dut-pbl6/inventory-service/internal/application/port"
	"dut-pbl6/inventory-service/internal/application/usecase"
	"dut-pbl6/inventory-service/internal/domain/entity"
	kafkainfra "dut-pbl6/inventory-service/internal/infrastructure/kafka"
	"dut-pbl6/inventory-service/internal/infrastructure/postgres"
	redisinfra "dut-pbl6/inventory-service/internal/infrastructure/redis"
	"dut-pbl6/inventory-service/internal/infrastructure/worker"
	presentationgrpc "dut-pbl6/inventory-service/internal/presentation/grpc"
	presentationkafka "dut-pbl6/inventory-service/internal/presentation/kafka"
	inventoryv1 "dut-pbl6/inventory-service/pkg/proto/inventory/v1"
	"runtime/debug"

	_ "github.com/lib/pq"
	goredis "github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return defaultVal
}

// recoveryUnaryServerInterceptor intercepts RPC calls and catches any runtime panics,
// logging the error and stack trace and returning an Internal error to prevent process crashes.
func recoveryUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp any, err error) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[gRPC Panic Recovery] Panic recovered in method %s: %v\nStack trace:\n%s",
					info.FullMethod, r, debug.Stack())
				err = status.Errorf(codes.Internal, "internal server error")
			}
		}()
		return handler(ctx, req)
	}
}

// recoveryStreamServerInterceptor intercepts streaming RPC calls and catches any runtime panics,
// logging the error and stack trace and returning an Internal error.
func recoveryStreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(
		srv any,
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) (err error) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[gRPC Stream Panic Recovery] Panic recovered in stream method %s: %v\nStack trace:\n%s",
					info.FullMethod, r, debug.Stack())
				err = status.Errorf(codes.Internal, "internal server error")
			}
		}()
		return handler(srv, ss)
	}
}

func main() {
	portStr := getEnv("PORT", "8001")
	dbURL := getEnv("DB_URL", "postgres://postgres:postgrespassword@localhost:5432/om_inventory_db?sslmode=disable")
	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	redisPassword := getEnv("REDIS_PASSWORD", "redispassword")
	kafkaBrokersStr := getEnv("KAFKA_BROKERS", "") // e.g. "localhost:9092"

	log.Printf("[InventoryService] Starting gRPC service on port %s...", portStr)

	// 1. Initialize PostgreSQL Connection Pool
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("[InventoryService] Failed to open database connection: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	// 2. Initialize Redis Client & Distributed Lock Adapter
	rdb := goredis.NewClient(&goredis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
	})
	defer rdb.Close()

	redisAdapter := redisinfra.NewGoRedisAdapter(rdb)
	lockService := redisinfra.NewRedlockService(redisAdapter)

	// 3. Initialize Repositories and Transaction Manager
	repo := postgres.NewPostgresInventoryRepository(db)
	outboxRepo := postgres.NewPostgresOutboxRepository(db)
	idempRepo := postgres.NewPostgresIdempotencyRepository(db)
	txManager := postgres.NewPostgresTxManager(db)

	// 4. Initialize Kafka Event Publisher (Producer)
	var eventPublisher port.EventPublisher
	if kafkaBrokersStr != "" {
		brokers := strings.Split(kafkaBrokersStr, ",")
		log.Printf("[InventoryService] Initializing Kafka Producer with brokers: %v", brokers)
		eventPublisher = kafkainfra.NewKafkaProducer(brokers)
	} else {
		log.Printf("[InventoryService] KAFKA_BROKERS not set; using LogEventPublisher fallback.")
		eventPublisher = kafkainfra.NewLogEventPublisher()
	}
	defer eventPublisher.Close()

	// 5. Wire Application Use Cases
	reserveStockUC := usecase.NewReserveStockUseCase(repo, txManager, lockService, outboxRepo)
	releaseReservationUC := usecase.NewReleaseReservationUseCase(repo, txManager, lockService, outboxRepo)
	commitStockUC := usecase.NewCommitStockDeductionUseCase(repo, txManager, lockService, outboxRepo)
	getStockLevelUC := usecase.NewGetStockLevelUseCase(repo)
	getBatchFEFOUC := usecase.NewGetBatchFEFODetailsUseCase(repo)

	// 6. Launch Background Workers
	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()

	// 6.1 Transactional Outbox Publisher Worker
	outboxWorker := worker.NewOutboxPublisherWorker(outboxRepo, eventPublisher)
	if err := outboxWorker.Start(workerCtx); err != nil {
		log.Printf("[InventoryService] Warning: Failed to start OutboxPublisherWorker: %v", err)
	} else {
		log.Printf("[InventoryService] OutboxPublisherWorker started.")
	}

	// 6.2 TTL Reservation Cleanup Worker (every 60s)
	cleanupWorker := worker.NewTTLReservationCleanupWorker(repo, releaseReservationUC)
	if err := cleanupWorker.Start(workerCtx); err != nil {
		log.Printf("[InventoryService] Warning: Failed to start TTLReservationCleanupWorker: %v", err)
	} else {
		log.Printf("[InventoryService] TTLReservationCleanupWorker started.")
	}

	// 6.3 Expiry Check Worker (every 24h, threshold 45 days)
	expiryWorker := worker.NewExpiryCheckWorker(repo, txManager, outboxRepo)
	if err := expiryWorker.Start(workerCtx); err != nil {
		log.Printf("[InventoryService] Warning: Failed to start ExpiryCheckWorker: %v", err)
	} else {
		log.Printf("[InventoryService] ExpiryCheckWorker started.")
	}

	// 6.4 Kafka Consumer (OrderPaid event listener) if Kafka brokers configured
	var consumerListener *presentationkafka.KafkaConsumerListener
	if kafkaBrokersStr != "" {
		brokers := strings.Split(kafkaBrokersStr, ",")
		orderPaidHandler := presentationkafka.NewOrderPaidHandler(commitStockUC, idempRepo)
		reader := presentationkafka.NewSegmentioReaderAdapter(brokers, entity.TopicOrderEvents, "inventory-service-order-paid-group")
		consumerListener = presentationkafka.NewKafkaConsumerListener(reader, orderPaidHandler)
		if err := consumerListener.Start(workerCtx); err != nil {
			log.Printf("[InventoryService] Warning: Failed to start KafkaConsumerListener: %v", err)
		} else {
			log.Printf("[InventoryService] KafkaConsumerListener started on topic '%s'.", entity.TopicOrderEvents)
		}
	}

	// 7. Initialize gRPC Presentation Handler & Register Service
	grpcHandler := presentationgrpc.NewInventoryGRPCHandler(
		reserveStockUC,
		releaseReservationUC,
		getStockLevelUC,
		getBatchFEFOUC,
	)

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(recoveryUnaryServerInterceptor()),
		grpc.ChainStreamInterceptor(recoveryStreamServerInterceptor()),
	)
	inventoryv1.RegisterInventoryServiceServer(grpcServer, grpcHandler)
	reflection.Register(grpcServer)

	// 8. Bind TCP Listener
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", portStr))
	if err != nil {
		log.Fatalf("[InventoryService] Failed to listen on port %s: %v", portStr, err)
	}

	// 9. Start gRPC Server in background goroutine
	go func() {
		log.Printf("[InventoryService] gRPC server listening at %s", lis.Addr().String())
		if err := grpcServer.Serve(lis); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			log.Fatalf("[InventoryService] gRPC server encountered fatal error: %v", err)
		}
	}()

	// 10. Graceful Shutdown on SIGINT/SIGTERM
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	sig := <-stopChan
	log.Printf("[InventoryService] Received signal (%v), initiating graceful shutdown...", sig)

	// Stop workers
	workerCancel()
	_ = outboxWorker.Stop(3 * time.Second)
	_ = cleanupWorker.Stop(3 * time.Second)
	_ = expiryWorker.Stop(3 * time.Second)
	if consumerListener != nil {
		_ = consumerListener.Stop(3 * time.Second)
	}

	// Stop gRPC server
	shutdownDone := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(shutdownDone)
	}()

	select {
	case <-shutdownDone:
		log.Println("[InventoryService] gRPC server stopped cleanly.")
	case <-time.After(10 * time.Second):
		log.Println("[InventoryService] Graceful shutdown timed out, force stopping server.")
		grpcServer.Stop()
	}

	log.Println("[InventoryService] Service shutdown complete.")
}
