package main

import (
	"context"
	"errors"
	"google.golang.org/grpc"
	"net"
	"net/http"
	"time"
)

// Both listeners stop before their shared database/workers. A single deadline
// bounds HTTP drain, gRPC drain and cleanup; forced gRPC Stop cancels stuck RPCs.
func serveBoth(ctx context.Context, httpServer *http.Server, httpListener net.Listener, rpc *grpc.Server, rpcListener net.Listener, cleanup func(), timeout time.Duration) error {
	results := make(chan error, 2)
	go func() { results <- httpServer.Serve(httpListener) }()
	go func() { results <- rpc.Serve(rpcListener) }()
	var first error
	select {
	case first = <-results:
	case <-ctx.Done():
	}
	shutdown, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	httpDone := make(chan error, 1)
	go func() {
		err := httpServer.Shutdown(shutdown)
		if err != nil {
			_ = httpServer.Close()
		}
		httpDone <- err
	}()
	rpcDone := make(chan struct{})
	go func() { rpc.GracefulStop(); close(rpcDone) }()
	var rpcError error
	select {
	case <-rpcDone:
	case <-shutdown.Done():
		rpc.Stop()
		rpcError = shutdown.Err()
	}
	httpError := <-httpDone
	closed := make(chan struct{})
	go func() { cleanup(); close(closed) }()
	var cleanupError error
	select {
	case <-closed:
	case <-shutdown.Done():
		cleanupError = shutdown.Err()
	}
	if errors.Is(first, http.ErrServerClosed) || errors.Is(first, grpc.ErrServerStopped) {
		first = nil
	}
	return errors.Join(first, httpError, rpcError, cleanupError)
}
