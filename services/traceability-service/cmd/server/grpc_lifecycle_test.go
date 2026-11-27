package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"testing"
	"time"

	traceabilityv1 "github.com/hvduong2k5/DUT.PBL6/services/traceability-service/pkg/proto/traceability/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type blockingRPC struct {
	traceabilityv1.UnimplementedBatchLinkResolverServer
	entered, cancelled chan struct{}
}

func (s blockingRPC) ResolveBatchLinks(ctx context.Context, _ *traceabilityv1.ResolveBatchLinksRequest) (*traceabilityv1.ResolveBatchLinksResponse, error) {
	close(s.entered)
	<-ctx.Done()
	close(s.cancelled)
	return nil, status.Error(codes.Canceled, "Cancelled")
}
func TestDualListenerShutdownCancelsStuckRPC(t *testing.T) {
	httpListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer httpListener.Close()
	rpcListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer rpcListener.Close()
	rpc := grpc.NewServer()
	block := blockingRPC{entered: make(chan struct{}), cancelled: make(chan struct{})}
	traceabilityv1.RegisterBatchLinkResolverServer(rpc, block)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	cleaned := make(chan struct{})
	go func() {
		done <- serveBoth(ctx, &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })}, httpListener, rpc, rpcListener, func() { close(cleaned) }, 100*time.Millisecond)
	}()
	conn, err := grpc.NewClient(rpcListener.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	go func() {
		_, _ = traceabilityv1.NewBatchLinkResolverClient(conn).ResolveBatchLinks(context.Background(), &traceabilityv1.ResolveBatchLinksRequest{})
	}()
	select {
	case <-block.entered:
	case <-time.After(time.Second):
		t.Fatal("RPC did not enter")
	}
	cancel()
	select {
	case err = <-done:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatal("drain timeout hidden", err)
		}
	case <-time.After(time.Second):
		t.Fatal("dual listener shutdown hung")
	}
	select {
	case <-block.cancelled:
	case <-time.After(time.Second):
		t.Fatal("stuck RPC was not cancelled")
	}
	select {
	case <-cleaned:
	case <-time.After(time.Second):
		t.Fatal("shared resources were not closed")
	}
}
