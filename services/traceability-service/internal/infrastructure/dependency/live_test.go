package dependency

import (
	"context"
	"errors"
	"testing"

	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/domain"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/testutil"
	catalogv1 "github.com/hvduong2k5/DUT.PBL6/services/traceability-service/pkg/proto/catalog/v1"
	traceabilityv1 "github.com/hvduong2k5/DUT.PBL6/services/traceability-service/pkg/proto/traceability/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type batchLookup func(context.Context, *traceabilityv1.BatchRequest, ...grpc.CallOption) (*traceabilityv1.BatchSnapshot, error)

func (f batchLookup) GetTraceabilityBatch(ctx context.Context, req *traceabilityv1.BatchRequest, options ...grpc.CallOption) (*traceabilityv1.BatchSnapshot, error) {
	return f(ctx, req, options...)
}

type catalogProvider struct {
	catalogv1.UnimplementedCatalogTraceabilitySourceServer
	lookup func(context.Context, *catalogv1.TraceabilitySkuBindingRequest) (*catalogv1.TraceabilitySkuBindingResponse, error)
}

func (s catalogProvider) GetSkuBinding(ctx context.Context, r *catalogv1.TraceabilitySkuBindingRequest) (*catalogv1.TraceabilitySkuBindingResponse, error) {
	return s.lookup(ctx, r)
}
func catalogClient(t *testing.T, lookup func(context.Context, *catalogv1.TraceabilitySkuBindingRequest) (*catalogv1.TraceabilitySkuBindingResponse, error)) catalogv1.CatalogTraceabilitySourceClient {
	t.Helper()
	connection := testutil.GRPC(t, func(options ...grpc.ServerOption) (*grpc.Server, error) {
		server := grpc.NewServer(options...)
		catalogv1.RegisterCatalogTraceabilitySourceServer(server, catalogProvider{lookup: lookup})
		return server, nil
	})
	return catalogv1.NewCatalogTraceabilitySourceClient(connection)
}

func TestLiveSnapshotValidatesBothBindings(t *testing.T) {
	binding := domain.Binding{ProductID: "44444444-4444-4444-8444-444444444444", SKUCode: "SKU-01", SourceNamespace: "inventory", BatchID: "22222222-2222-4222-8222-222222222222"}
	catalog := catalogClient(t, func(ctx context.Context, r *catalogv1.TraceabilitySkuBindingRequest) (*catalogv1.TraceabilitySkuBindingResponse, error) {
		md, _ := metadata.FromIncomingContext(ctx)
		if md.Get("authorization")[0] != "Bearer catalog-token" || r.GetSkuCode() != binding.SKUCode {
			t.Error("incorrect Catalog binding or credential")
		}
		return &catalogv1.TraceabilitySkuBindingResponse{ProductId: binding.ProductID, SkuCode: binding.SKUCode, CatalogVersion: "v:2:3"}, nil
	})
	for _, tc := range []struct {
		name, sku string
		err, want error
	}{
		{name: "matching", sku: "SKU-01"}, {name: "mismatched SKU", sku: "SKU-02", want: domain.ErrConflict},
		{name: "unavailable", err: status.Error(codes.Unavailable, "offline"), want: domain.ErrUnavailable},
		{name: "missing batch", err: status.Error(codes.NotFound, "missing"), want: domain.ErrConflict},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := Live{CatalogSource: catalog, CatalogToken: "catalog-token", InventoryToken: "inventory-token"}
			source.Inventory = batchLookup(func(ctx context.Context, r *traceabilityv1.BatchRequest, _ ...grpc.CallOption) (*traceabilityv1.BatchSnapshot, error) {
				md, _ := metadata.FromOutgoingContext(ctx)
				if r.GetBatchId() != binding.BatchID || r.GetSkuCode() != binding.SKUCode || len(md.Get("authorization")) != 1 || md.Get("authorization")[0] != "Bearer inventory-token" {
					t.Fatal("incorrect Inventory binding or credential")
				}
				return &traceabilityv1.BatchSnapshot{BatchId: binding.BatchID, SkuCode: tc.sku, BatchCode: "LOT-01", ProductionDate: "2026-11-13", ExpiryDate: "2027-01-01", BatchVersion: 4, WarehouseId: "77777777-7777-4777-8777-777777777777"}, tc.err
			})
			snapshot, err := source.Snapshot(context.Background(), binding)
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v want %v", err, tc.want)
			}
			if err == nil && (snapshot.CatalogVersion != "v:2:3" || snapshot.BatchVersion != "4" || snapshot.BestBefore.Format("2006-01-02T15:04:05Z") != "2027-01-01T17:00:00Z") {
				t.Fatal("incorrect source snapshot")
			}
		})
	}
}
