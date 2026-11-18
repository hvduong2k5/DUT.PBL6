package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync/atomic"
	"testing"
	"time"
)

func TestShutdownDrainsAndClosesResourcesOnce(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var closed atomic.Int32
	done := make(chan error, 1)
	go func() {
		done <- serve(ctx, server, func() error { return server.Serve(listener) }, func() { closed.Add(1) }, time.Second)
	}()
	response, err := http.Get("http://" + listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	cancel()
	select {
	case err = <-done:
		if err != nil || closed.Load() != 1 {
			t.Fatalf("shutdown: closed=%d %v", closed.Load(), err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("shutdown did not finish")
	}
}
func TestCleanupHangHasBoundedFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	release := make(chan struct{})
	defer close(release)
	started := time.Now()
	err := serve(ctx, &http.Server{}, func() error { return http.ErrServerClosed }, func() { <-release }, 50*time.Millisecond)
	if !errors.Is(err, context.DeadlineExceeded) || time.Since(started) > time.Second {
		t.Fatal("cleanup ignored overall deadline", err)
	}
}
func TestInFlightRequestIsCancelledOnDrainDeadline(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	entered := make(chan struct{})
	cancelled := make(chan struct{})
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { close(entered); <-r.Context().Done(); close(cancelled) })}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- serve(ctx, server, func() error { return server.Serve(listener) }, func() {}, 50*time.Millisecond)
	}()
	go func() {
		response, err := http.Get("http://" + listener.Addr().String())
		if err == nil {
			response.Body.Close()
		}
	}()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("request did not enter")
	}
	cancel()
	select {
	case err = <-done:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatal("drain timeout hidden", err)
		}
	case <-time.After(time.Second):
		t.Fatal("drain hung")
	}
	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("request context not cancelled")
	}
}
