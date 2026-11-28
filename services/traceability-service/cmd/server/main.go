package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/config"
	grpctransport "github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/transport/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func run() error {
	cfg, err := config.Load(os.Getenv)
	if err != nil {
		return err
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	handler, resolver, closeResources, err := assembleApplication(ctx, cfg, os.Getenv, logger)
	if err != nil {
		return err
	}
	server := &http.Server{Addr: cfg.ListenAddress, Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 32 * 1024}
	listener, err := net.Listen("tcp", cfg.ListenAddress)
	if err != nil {
		closeResources()
		return errors.New("cannot bind configured HTTP listener")
	}
	defer listener.Close()
	if resolver != nil {
		options := []grpc.ServerOption{}
		if cfg.Mode == "integration" || cfg.GRPCCertFile != "" || cfg.GRPCKeyFile != "" {
			tls, err := credentials.NewServerTLSFromFile(cfg.GRPCCertFile, cfg.GRPCKeyFile)
			if err != nil {
				closeResources()
				return errors.New("invalid resolver TLS certificate configuration")
			}
			options = append(options, grpc.Creds(tls))
		}
		rpc, err := grpctransport.NewServer(resolver, cfg.OrderResolverToken, options...)
		if err != nil {
			closeResources()
			return err
		}
		rpcListener, err := net.Listen("tcp", cfg.GRPCAddress)
		if err != nil {
			rpc.Stop()
			closeResources()
			return errors.New("cannot bind configured internal gRPC listener")
		}
		defer rpcListener.Close()
		logger.Info("traceability listeners starting", "http_address", cfg.ListenAddress, "grpc_address", cfg.GRPCAddress, "mode", cfg.Mode)
		return serveBoth(ctx, server, listener, rpc, rpcListener, closeResources, cfg.ShutdownTimeout)
	}
	logger.Info("traceability listener starting", "address", cfg.ListenAddress, "mode", cfg.Mode)
	return serve(ctx, server, func() error { return server.Serve(listener) }, closeResources, cfg.ShutdownTimeout)
}

func main() {
	if err := run(); err != nil {
		slog.Error("traceability stopped", "error", err)
		os.Exit(1)
	}
}
