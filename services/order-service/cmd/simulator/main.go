package main

import (
	"context"
	inventory "github.com/omamx/order-service/internal/gen/inventory/v1"
	"github.com/omamx/order-service/internal/simulator"
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
	if os.Getenv("APP_MODE") != "sandbox" {
		slog.Error("simulator requires APP_MODE=sandbox")
		os.Exit(1)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	s, e := simulator.New(ctx, os.Getenv("SIMULATOR_DATABASE_URL"), os.Getenv("ORDER_INTERNAL_TOKEN"), os.Getenv("ORDER_JWT_SECRET"), os.Getenv("CALLBACK_SECRET"), os.Getenv("PAYMENT_RECEIVER"))
	if e != nil {
		slog.Error("simulator storage unavailable")
		os.Exit(1)
	}
	defer s.DB.Close()
	s.OrderURL = os.Getenv("ORDER_URL")
	listener, e := net.Listen("tcp", ":9100")
	if e != nil {
		slog.Error("simulator gRPC listen failed")
		os.Exit(1)
	}
	rpc := grpc.NewServer(grpc.MaxRecvMsgSize(64 << 10))
	inventory.RegisterInventoryServiceServer(rpc, &simulator.InventoryRPC{S: s})
	go rpc.Serve(listener)
	defer rpc.Stop()
	srv := &http.Server{Addr: ":8100", Handler: s.Handler(), ReadHeaderTimeout: 3 * time.Second, ReadTimeout: 5 * time.Second, WriteTimeout: 5 * time.Second}
	go func() {
		if e := srv.ListenAndServe(); e != nil && e != http.ErrServerClosed {
			cancel()
		}
	}()
	<-ctx.Done()
	shutdown, stop := context.WithTimeout(context.Background(), 5*time.Second)
	defer stop()
	_ = srv.Shutdown(shutdown)
}
