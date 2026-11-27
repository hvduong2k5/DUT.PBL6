package grpctransport

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/application"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/domain"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/testutil"
	traceabilityv1 "github.com/hvduong2k5/DUT.PBL6/services/traceability-service/pkg/proto/traceability/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const orderToken = "synthetic-order-service-credential-32-bytes"

type resolutionStore func(context.Context, []domain.LinkBinding) ([]domain.ResolutionProfile, error)

func (f resolutionStore) ResolveProfiles(ctx context.Context, b []domain.LinkBinding) ([]domain.ResolutionProfile, error) {
	return f(ctx, b)
}
func rpcClient(t *testing.T, store resolutionStore, base string) traceabilityv1.BatchLinkResolverClient {
	t.Helper()
	connection := testutil.GRPC(t, func(options ...grpc.ServerOption) (*grpc.Server, error) {
		return NewServer(&application.Resolver{Store: store, Clock: time.Now, PublicWebURL: base}, orderToken, options...)
	})
	return traceabilityv1.NewBatchLinkResolverClient(connection)
}
func validBinding() *traceabilityv1.TraceabilityBatchBinding {
	return &traceabilityv1.TraceabilityBatchBinding{SourceNamespace: "inventory", BatchId: "33333333-3333-4333-8333-333333333333", SkuCode: "SKU-1"}
}
func TestResolverTLSMetadataAndBounds(t *testing.T) {
	var calls atomic.Int32
	client := rpcClient(t, func(_ context.Context, b []domain.LinkBinding) ([]domain.ResolutionProfile, error) {
		calls.Add(1)
		return make([]domain.ResolutionProfile, len(b)), nil
	}, "https://shop.example.test")
	request := &traceabilityv1.ResolveBatchLinksRequest{Items: []*traceabilityv1.TraceabilityBatchBinding{validBinding()}}
	if _, err := client.ResolveBatchLinks(context.Background(), request); status.Code(err) != codes.Unauthenticated || calls.Load() != 0 {
		t.Fatal("anonymous query reached storage", err)
	}
	duplicate := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+orderToken, "authorization", "Bearer "+orderToken))
	if _, err := client.ResolveBatchLinks(duplicate, request); status.Code(err) != codes.Unauthenticated {
		t.Fatal("duplicate credentials accepted", err)
	}
	ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer "+orderToken, "x-request-id", "rpc-fixture-001")
	var headers metadata.MD
	reply, err := client.ResolveBatchLinks(ctx, request, grpc.Header(&headers))
	if err != nil || reply.GetTotal() != 1 || reply.GetItems()[0].GetStatus() != traceabilityv1.BatchLinkStatus_BATCH_LINK_STATUS_NOT_AVAILABLE || reply.Items[0].QrUrl != nil || len(headers.Get("x-request-id")) != 1 || headers.Get("x-request-id")[0] != "rpc-fixture-001" {
		t.Fatal("invalid response/presence/request correlation", err)
	}
	badContext := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer "+orderToken, "x-request-id", strings.Repeat("x", 129))
	headers = nil
	if _, err = client.ResolveBatchLinks(badContext, request, grpc.Header(&headers)); err != nil || len(headers.Get("x-request-id")) != 1 || headers.Get("x-request-id")[0] == strings.Repeat("x", 129) {
		t.Fatal("invalid request ID not replaced", err)
	}
	request.Items[0].SourceNamespace = "fixture"
	if _, err = client.ResolveBatchLinks(ctx, request); status.Code(err) != codes.InvalidArgument {
		t.Fatal("fixture entered integration resolver", err)
	}
}
func TestResolverResponseCapAndDeadline(t *testing.T) {
	store := resolutionStore(func(_ context.Context, b []domain.LinkBinding) ([]domain.ResolutionProfile, error) {
		rows := make([]domain.ResolutionProfile, len(b))
		for i, binding := range b {
			rows[i].Profile = &domain.Profile{Binding: domain.Binding{SourceNamespace: binding.SourceNamespace, BatchID: binding.BatchID, SKUCode: binding.SKUCode}, EverPublished: true, Status: "SUSPENDED", QRCode: "abcdefghijklmnopqrstuv"}
		}
		return rows, nil
	})
	client := rpcClient(t, store, "https://shop.example.test/"+strings.Repeat("x", 8192))
	ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer "+orderToken)
	many := &traceabilityv1.ResolveBatchLinksRequest{Items: make([]*traceabilityv1.TraceabilityBatchBinding, 100)}
	for i := range many.Items {
		many.Items[i] = validBinding()
	}
	if _, err := client.ResolveBatchLinks(ctx, many); status.Code(err) != codes.ResourceExhausted {
		t.Fatal("response byte cap not enforced", err)
	}
	blocked := rpcClient(t, func(ctx context.Context, _ []domain.LinkBinding) ([]domain.ResolutionProfile, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}, "https://shop.example.test")
	deadline, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()
	_, err := blocked.ResolveBatchLinks(deadline, &traceabilityv1.ResolveBatchLinksRequest{Items: []*traceabilityv1.TraceabilityBatchBinding{validBinding()}})
	if status.Code(err) != codes.DeadlineExceeded && status.Code(err) != codes.Unavailable {
		t.Fatal("caller deadline ignored", err)
	}
}
