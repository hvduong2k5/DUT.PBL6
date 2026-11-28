package traceabilityprovider

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	trace "github.com/omamx/order-service/internal/gen/traceability/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
)

type resolverFixture struct {
	trace.UnimplementedBatchLinkResolverServer
	active, maximum, calls atomic.Int32
	mode                   string
}

func (f *resolverFixture) ResolveBatchLinks(ctx context.Context, request *trace.ResolveBatchLinksRequest) (*trace.ResolveBatchLinksResponse, error) {
	f.calls.Add(1)
	md, _ := metadata.FromIncomingContext(ctx)
	if len(md.Get("authorization")) != 1 || md.Get("authorization")[0] != "Bearer "+strings.Repeat("s", 40) {
		return nil, status.Error(codes.Unauthenticated, "wrong caller")
	}
	deadline, ok := ctx.Deadline()
	if !ok || time.Until(deadline) > 1500*time.Millisecond || len(request.Items) > 100 {
		return nil, status.Error(codes.InvalidArgument, "invalid budget")
	}
	n := f.active.Add(1)
	defer f.active.Add(-1)
	for old := f.maximum.Load(); n > old && !f.maximum.CompareAndSwap(old, n); old = f.maximum.Load() {
	}
	if f.mode == "timeout" {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	time.Sleep(5 * time.Millisecond)
	if f.mode == "partial" && request.Items[0].SkuCode == "SKU-100" {
		return nil, status.Error(codes.Unavailable, "fixture offline")
	}
	response := &trace.ResolveBatchLinksResponse{Total: int32(len(request.Items))}
	for _, binding := range request.Items {
		url := "https://shop.example.test/trace/abcdefghijklmnopqrstuv"
		response.Items = append(response.Items, &trace.ResolvedBatchLink{Binding: proto.Clone(binding).(*trace.TraceabilityBatchBinding), Status: trace.BatchLinkStatus_BATCH_LINK_STATUS_AVAILABLE, QrUrl: &url})
	}
	if f.mode == "echo" {
		response.Items[0].Binding.SkuCode = "OTHER"
	}
	if f.mode == "presence" {
		response.Items[0].Status = trace.BatchLinkStatus_BATCH_LINK_STATUS_NOT_AVAILABLE
	}
	if f.mode == "url" {
		*response.Items[0].QrUrl = "https://attacker.test/trace/token"
	}
	return response, nil
}
func fixture(t *testing.T, mode string) (*Adapter, *resolverFixture) {
	t.Helper()
	listener := bufconn.Listen(1 << 20)
	server := grpc.NewServer()
	provider := &resolverFixture{mode: mode}
	trace.RegisterBatchLinkResolverServer(server, provider)
	go server.Serve(listener)
	t.Cleanup(server.Stop)
	conn, err := grpc.NewClient("passthrough:///trace-fixture", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	return &Adapter{Client: trace.NewBatchLinkResolverClient(conn), Token: strings.Repeat("s", 40), PublicOrigin: "https://shop.example.test"}, provider
}
func TestResolverChunksAndPartialFailure(t *testing.T) {
	adapter, server := fixture(t, "partial")
	bindings := make([]Binding, 250)
	for i := range bindings {
		bindings[i] = Binding{Batch: "11111111-1111-4111-8111-111111111111", SKU: fmt.Sprintf("SKU-%d", i)}
	}
	links := adapter.Resolve(context.Background(), bindings)
	if server.calls.Load() != 3 || server.maximum.Load() > 2 {
		t.Fatal("unbounded calls", server.calls.Load(), server.maximum.Load())
	}
	for i, link := range links {
		expected := "AVAILABLE"
		if i >= 100 && i < 200 {
			expected = "UNAVAILABLE"
		}
		if link.Status != expected || expected == "UNAVAILABLE" && link.URL != nil {
			t.Fatal(i, link)
		}
	}
}
func TestResolverRejectsEchoPresenceAndURLMismatch(t *testing.T) {
	for _, mode := range []string{"echo", "presence", "url"} {
		t.Run(mode, func(t *testing.T) {
			adapter, _ := fixture(t, mode)
			links := adapter.Resolve(context.Background(), []Binding{{Batch: "11111111-1111-4111-8111-111111111111", SKU: "SKU-1"}})
			if links[0].Status != "UNAVAILABLE" || links[0].URL != nil {
				t.Fatal(links)
			}
		})
	}
}
func TestResolverHonorsOuterCancellation(t *testing.T) {
	adapter, _ := fixture(t, "timeout")
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	links := adapter.Resolve(ctx, []Binding{{Batch: "11111111-1111-4111-8111-111111111111", SKU: "SKU-1"}})
	if links[0].Status != "UNAVAILABLE" {
		t.Fatal(links)
	}
}
