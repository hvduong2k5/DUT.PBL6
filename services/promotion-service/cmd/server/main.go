package main

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/promotion-service/internal/config"
	"github.com/omamx/promotion-service/internal/repository/postgres"
	"github.com/omamx/promotion-service/internal/transport/grpcapi"
	"github.com/omamx/promotion-service/internal/transport/httpapi"
	"github.com/omamx/promotion-service/internal/worker"
	"github.com/omamx/promotion-service/migrations"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
)

func main() {
	zerolog.TimeFieldFormat = time.RFC3339Nano
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: "15:04:05"}).With().Caller().Logger()
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("invalid configuration")
	}

	rootCtx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	pool, err := pgxpool.New(rootCtx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("create PostgreSQL pool")
	}
	defer pool.Close()

	dbCtx, dbCancel := context.WithTimeout(rootCtx, cfg.DBTimeout)
	if err := pool.Ping(dbCtx); err != nil {
		dbCancel()
		log.Fatal().Err(err).Msg("connect to PostgreSQL")
	}
	dbCancel()
	if applied, err := migrations.Apply(rootCtx, pool); err != nil {
		log.Fatal().Err(err).Msg("apply database migrations")
	} else if len(applied) > 0 {
		log.Info().Strs("migrations", applied).Msg("database migrations applied")
	}
	voucherRepository := postgres.NewVoucherRepository(pool)
	adminCouponRepository := postgres.NewAdminCouponRepository(pool)
	loyaltyRepository := postgres.NewLoyaltyRepository(pool)
	startCouponLifecycle(rootCtx, adminCouponRepository)
	startVoucherHoldExpiry(rootCtx, voucherRepository)
	startLoyaltyHoldExpiry(rootCtx, loyaltyRepository)
	var eventWorkers *worker.EventWorkers
	if cfg.OrderEventsEnabled {
		eventWorkers = worker.NewEventWorkers(cfg.KafkaBrokers, cfg.OrderEventsTopic, cfg.OrderConsumerGroup, cfg.PromotionEventsTopic, cfg.OrderEventsDLQTopic, voucherRepository)
		eventWorkers.Run(rootCtx)
		log.Info().Str("input_topic", cfg.OrderEventsTopic).Str("output_topic", cfg.PromotionEventsTopic).Msg("Order v3 event integration enabled")
	}

	router := chi.NewRouter()
	router.Get("/health/live", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	staffAuth := httpapi.StaffJWTAuth{Issuer: cfg.StaffJWTIssuer, Audience: cfg.StaffJWTAudience}
	if cfg.StaffJWTPublicKeyPath != "" {
		publicKeyPEM, err := os.ReadFile(cfg.StaffJWTPublicKeyPath)
		if err != nil {
			log.Fatal().Err(err).Msg("read Promotion staff JWT public key")
		}
		staffAuth.Key, err = parseRSAPublicKey(publicKeyPEM)
		if err != nil {
			log.Fatal().Err(err).Msg("parse Promotion staff JWT public key")
		}
	}
	httpapi.RegisterAdminCouponRoutes(router, adminCouponRepository, staffAuth)
	httpapi.RegisterPublicPromotionRoutes(router, adminCouponRepository)
	httpapi.RegisterLoyaltyRoutes(router, loyaltyRepository, cfg.InternalToken, cfg.LoyaltyOrderToken, cfg.LoyaltyRefundToken, cfg.CustomerProfileURL)
	router.Get("/health/ready", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), cfg.DBTimeout)
		defer cancel()
		if err := pool.Ping(ctx); err != nil {
			http.Error(w, "database unavailable", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	server := &http.Server{Addr: cfg.HTTPAddress, Handler: router, ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second}
	grpcListener, err := net.Listen("tcp", cfg.GRPCAddress)
	if err != nil {
		log.Fatal().Err(err).Str("address", cfg.GRPCAddress).Msg("listen for gRPC")
	}
	grpcServer := grpcapi.NewGRPCServer(voucherRepository, cfg.InternalToken)
	go func() {
		log.Info().Str("address", cfg.GRPCAddress).Msg("Promotion gRPC server listening")
		if err := grpcapi.Serve(grpcListener, grpcServer); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			log.Error().Err(err).Msg("gRPC server stopped unexpectedly")
			cancel()
		}
	}()
	go func() {
		log.Info().Str("address", cfg.HTTPAddress).Msg("Promotion HTTP server listening")
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error().Err(err).Msg("HTTP server stopped unexpectedly")
			cancel()
		}
	}()
	<-rootCtx.Done()
	if eventWorkers != nil {
		if err := eventWorkers.Close(); err != nil {
			log.Warn().Err(err).Msg("close Promotion Kafka workers")
		}
	}
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownWait)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("HTTP server shutdown failed")
	}
	grpcStopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(grpcStopped)
	}()
	select {
	case <-grpcStopped:
	case <-shutdownCtx.Done():
		grpcServer.Stop()
	}
}

func parseRSAPublicKey(publicKeyPEM []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(publicKeyPEM)
	if block == nil {
		return nil, errors.New("public key file is not PEM encoded")
	}
	if key, err := x509.ParsePKIXPublicKey(block.Bytes); err == nil {
		if rsaKey, ok := key.(*rsa.PublicKey); ok {
			return rsaKey, nil
		}
		return nil, errors.New("public key is not RSA")
	}
	if key, err := x509.ParsePKCS1PublicKey(block.Bytes); err == nil {
		return key, nil
	}
	if certificate, err := x509.ParseCertificate(block.Bytes); err == nil {
		if key, ok := certificate.PublicKey.(*rsa.PublicKey); ok {
			return key, nil
		}
	}
	return nil, errors.New("public key PEM does not contain an RSA public key")
}

func startCouponLifecycle(ctx context.Context, repository *postgres.AdminCouponRepository) {
	if err := repository.AdvanceLifecycle(ctx); err != nil {
		log.Warn().Err(err).Msg("initial coupon lifecycle advancement failed")
	}
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := repository.AdvanceLifecycle(ctx); err != nil {
					log.Warn().Err(err).Msg("coupon lifecycle advancement failed")
				}
			}
		}
	}()
}

func startVoucherHoldExpiry(ctx context.Context, repository *postgres.VoucherRepository) {
	advance := func() {
		workCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		expired, err := repository.ExpireDueHolds(workCtx)
		if err != nil {
			log.Warn().Err(err).Msg("voucher hold expiry sweep failed")
			return
		}
		if expired > 0 {
			log.Info().Int64("count", expired).Msg("voucher holds expired")
		}
	}
	advance()
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				advance()
			}
		}
	}()
}

func startLoyaltyHoldExpiry(ctx context.Context, repository *postgres.LoyaltyRepository) {
	advance := func() {
		workCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		expired, err := repository.ExpireDueRedemptionHolds(workCtx, 100)
		if err != nil {
			log.Warn().Err(err).Msg("loyalty redemption hold expiry sweep failed")
			return
		}
		if expired > 0 {
			log.Info().Int64("count", expired).Msg("loyalty redemption holds expired")
		}
	}
	advance()
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				advance()
			}
		}
	}()
}
