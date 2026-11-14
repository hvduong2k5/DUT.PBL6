package main

import (
	"context"
	"github.com/omamx/order-service/internal/app"
	order "github.com/omamx/order-service/internal/gen/order/v1"
	packing "github.com/omamx/order-service/internal/gen/packing/v1"
	"github.com/omamx/order-service/internal/transport"
	"github.com/omamx/order-service/migrations"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	cfg, e := app.LoadConfig()
	if e != nil {
		slog.Error("configuration", "error", e)
		os.Exit(1)
	}
	s, e := app.New(ctx, cfg)
	if e != nil {
		slog.Error("database/configuration unavailable")
		os.Exit(1)
	}
	defer s.Close()
	if e = migrations.Apply(ctx, s.DB); e != nil {
		slog.Error("migration failed", "error", e)
		os.Exit(1)
	}
	if e = s.EnableSePayFromEnv(ctx); e != nil {
		slog.Error("SePay startup configuration invalid")
		os.Exit(1)
	}
	if eCore := s.EnableCoreCheckout(); eCore != nil {
		slog.Error("target checkout configuration invalid")
		os.Exit(1)
	}
	listener, e := net.Listen("tcp", cfg.GRPC)
	if e != nil {
		slog.Error("gRPC listen failed")
		os.Exit(1)
	}
	options := []grpc.ServerOption{grpc.MaxRecvMsgSize(64 << 10)}
	certificate, key := os.Getenv("ORDER_GRPC_TLS_CERT"), os.Getenv("ORDER_GRPC_TLS_KEY")
	if cfg.ShippingOwnsPackingHandoff &&
		(s.Fulfillment == nil || certificate == "" || key == "" || len(strings.TrimSpace(os.Getenv("ORDER_SHIPPING_CALLER_TOKEN"))) < 32) {
		slog.Error("Shipping packing handoff requires Fulfillment, Order gRPC TLS and a dedicated Shipping credential")
		os.Exit(1)
	}
	if certificate != "" || key != "" {
		creds, e := credentials.NewServerTLSFromFile(certificate, key)
		if e != nil {
			slog.Error("invalid Order gRPC TLS configuration")
			os.Exit(1)
		}
		options = append(options, grpc.Creds(creds))
	} else if s.Fulfillment != nil && cfg.Mode != "sandbox" {
		slog.Error("Fulfillment integration requires Order gRPC TLS outside sandbox")
		os.Exit(1)
	}
	rpc := grpc.NewServer(options...)
	orderRPC := &transport.RPC{S: s}
	order.RegisterOrderServiceServer(rpc, orderRPC)
	packing.RegisterOrderPackingInputsCandidateServer(rpc, &transport.PackingRPC{S: orderRPC})
	go func() {
		if e := rpc.Serve(listener); e != nil {
			cancel()
		}
	}()
	httpServer := &http.Server{Addr: cfg.HTTP, Handler: transport.Handler(s), ReadHeaderTimeout: 3 * time.Second, ReadTimeout: 5 * time.Second, WriteTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second}
	s.Start(ctx)
	go func() {
		if e := httpServer.ListenAndServe(); e != nil && e != http.ErrServerClosed {
			cancel()
		}
	}()
	<-ctx.Done()
	shutdown, stop := context.WithTimeout(context.Background(), 5*time.Second)
	defer stop()
	_ = httpServer.Shutdown(shutdown)
	done := make(chan struct{})
	go func() { rpc.GracefulStop(); close(done) }()
	select {
	case <-done:
	case <-shutdown.Done():
		rpc.Stop()
	}
}
