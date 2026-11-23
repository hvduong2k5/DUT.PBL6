package dependency

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/domain"
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

func TestLiveSnapshotValidatesBothBindings(t *testing.T) {
	binding := domain.Binding{ProductID: "44444444-4444-4444-8444-444444444444", SKUCode: "SKU-01", SourceNamespace: "inventory", BatchID: "22222222-2222-4222-8222-222222222222"}
	catalog := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Internal-Token") != "catalog-token" ||
			r.URL.Path != "/api/v1/internal/traceability/products/"+binding.ProductID+"/skus/"+binding.SKUCode {
			t.Error("incorrect Catalog path or credential")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"product_id":"44444444-4444-4444-8444-444444444444","sku_code":"SKU-01","catalog_version":"v:2:3"}`))
	}))
	defer catalog.Close()
	for _, tc := range []struct {
		name string
		sku  string
		err  error
		want error
	}{
		{name: "matching", sku: "SKU-01"},
		{name: "mismatched SKU", sku: "SKU-02", want: domain.ErrConflict},
		{name: "unavailable", err: status.Error(codes.Unavailable, "offline"), want: domain.ErrUnavailable},
		{name: "missing batch", err: status.Error(codes.NotFound, "not found"), want: domain.ErrConflict},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := Live{CatalogURL: catalog.URL, CatalogToken: "catalog-token", InventoryToken: "inventory-token", HTTP: catalog.Client()}
			source.Inventory = batchLookup(func(ctx context.Context, req *traceabilityv1.BatchRequest, _ ...grpc.CallOption) (*traceabilityv1.BatchSnapshot, error) {
				md, _ := metadata.FromOutgoingContext(ctx)
				if req.GetBatchId() != binding.BatchID || req.GetSkuCode() != binding.SKUCode || len(md.Get("authorization")) != 1 || md.Get("authorization")[0] != "Bearer inventory-token" {
					t.Fatal("incorrect Inventory binding or credential")
				}
				return &traceabilityv1.BatchSnapshot{BatchId: binding.BatchID, SkuCode: tc.sku, BatchCode: "LOT-01", ProductionDate: "2026-11-13", ExpiryDate: "2027-01-01", BatchVersion: 4}, tc.err
			})
			snapshot, err := source.Snapshot(context.Background(), binding)
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
			if err == nil && (snapshot.CatalogVersion != "v:2:3" || snapshot.BatchVersion != "4" || snapshot.BestBefore.Format("2006-01-02T15:04:05Z") != "2027-01-01T17:00:00Z") {
				t.Fatalf("unexpected snapshot: %+v", snapshot)
			}
		})
	}
}
