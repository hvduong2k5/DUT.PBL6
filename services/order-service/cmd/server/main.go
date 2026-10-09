package main

import (
	"context"
	"github.com/omamx/order-service/internal/app"
	order "github.com/omamx/order-service/internal/gen/order/v1"
	"github.com/omamx/order-service/internal/transport"
	"github.com/omamx/order-service/migrations"
	"google.golang.org/grpc"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
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
	listener, e := net.Listen("tcp", cfg.GRPC)
	if e != nil {
		slog.Error("gRPC listen failed")
		os.Exit(1)
	}
	rpc := grpc.NewServer(grpc.MaxRecvMsgSize(64 << 10))
	order.RegisterOrderServiceServer(rpc, &transport.RPC{S: s})
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
