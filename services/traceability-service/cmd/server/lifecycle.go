package main

import (
	"context"
	"errors"
	"net/http"
	"time"
)

// One deadline bounds HTTP draining AND resource cleanup. A timed-out cleanup
// returns a failure to main, which exits; it must not hide a shutdown hang.
func serve(ctx context.Context, server *http.Server, start func() error, closeResources func(), timeout time.Duration) error {
	result := make(chan error, 1)
	go func() { result <- start() }()
	var serverError error
	select {
	case serverError = <-result:
	case <-ctx.Done():
	}
	shutdown, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	var drainError error
	if ctx.Err() != nil {
		drainError = server.Shutdown(shutdown)
		if drainError != nil {
			_ = server.Close()
		}
	}
	cleanup := make(chan struct{})
	go func() { closeResources(); close(cleanup) }()
	var cleanupError error
	select {
	case <-cleanup:
	case <-shutdown.Done():
		cleanupError = shutdown.Err()
	}
	if errors.Is(serverError, http.ErrServerClosed) {
		serverError = nil
	}
	return errors.Join(serverError, drainError, cleanupError)
}
