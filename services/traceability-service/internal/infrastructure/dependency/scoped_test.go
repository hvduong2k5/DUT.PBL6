package dependency

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/domain"
	catalogv1 "github.com/hvduong2k5/DUT.PBL6/services/traceability-service/pkg/proto/catalog/v1"
	traceabilityv1 "github.com/hvduong2k5/DUT.PBL6/services/traceability-service/pkg/proto/traceability/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strings"
)

func TestScopedProviderRequiresWarehouseAndExactSKU(t *testing.T) {
	const batch = "33333333-3333-4333-8333-333333333333"
	for _, warehouse := range []string{"", "not-a-uuid", "77777777-7777-4777-8777-777777777777"} {
		s := Live{Inventory: batchLookup(func(context.Context, *traceabilityv1.BatchRequest, ...grpc.CallOption) (*traceabilityv1.BatchSnapshot, error) {
			return &traceabilityv1.BatchSnapshot{BatchId: batch, SkuCode: "SKU-1", BatchCode: "LOT", ProductionDate: "2026-11-01", ExpiryDate: "2027-11-01", BatchVersion: 1, WarehouseId: warehouse}, nil
		})}
		result, err := s.Batch(context.Background(), "inventory", batch, "SKU-1")
		if domain.ValidUUID(warehouse) {
			if err != nil || result.WarehouseID != warehouse {
				t.Fatal("verified warehouse lost", err)
			}
		} else if !errors.Is(err, domain.ErrUnavailable) {
			t.Fatal("missing provider capability fabricated", err)
		}
	}
	for _, sku := range []string{"SKU-1", "SKU-1/other", ".."} {
		var calls atomic.Int32
		catalog := catalogClient(t, func(ctx context.Context, r *catalogv1.GetTraceabilityBindingRequest) (*catalogv1.GetTraceabilityBindingResponse, error) {
			calls.Add(1)
			return &catalogv1.GetTraceabilityBindingResponse{ProductId: "22222222-2222-4222-8222-222222222222", SkuCode: "SKU-1", VersionFingerprint: "variant:1:1"}, nil
		})
		source := Live{CatalogSource: catalog, CatalogToken: "synthetic"}
		product, version, err := source.Catalog(context.Background(), sku)
		if sku == "SKU-1" {
			if err != nil || product == "" || version != "variant:1:1" {
				t.Fatal("binding not returned", err)
			}
		} else if err == nil || calls.Load() != 0 {
			t.Fatal("unsafe SKU reached Catalog")
		}
	}
}

func TestCatalogRPCFailuresDoNotFallback(t *testing.T) {
	for _, tc := range []struct {
		name         string
		code         codes.Code
		sku, version string
		want         error
	}{
		{"missing", codes.NotFound, "SKU-1", "1", domain.ErrNotFound},
		{"invalid", codes.InvalidArgument, "SKU-1", "1", domain.ErrInvalidCommand},
		{"old provider", codes.Unimplemented, "SKU-1", "1", domain.ErrUnavailable},
		{"service denied", codes.PermissionDenied, "SKU-1", "1", domain.ErrUnavailable},
		{"timeout", codes.DeadlineExceeded, "SKU-1", "1", domain.ErrUnavailable},
		{"mismatched SKU", codes.OK, "OTHER", "1", domain.ErrConflict},
		{"oversize reply", codes.OK, "SKU-1", strings.Repeat("v", 8193), domain.ErrUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := catalogClient(t, func(context.Context, *catalogv1.GetTraceabilityBindingRequest) (*catalogv1.GetTraceabilityBindingResponse, error) {
				if tc.code != codes.OK {
					return nil, status.Error(tc.code, "Safe provider error")
				}
				return &catalogv1.GetTraceabilityBindingResponse{ProductId: "22222222-2222-4222-8222-222222222222", SkuCode: tc.sku, VersionFingerprint: tc.version}, nil
			})
			_, _, err := (Live{CatalogSource: client, CatalogToken: "synthetic"}).Catalog(context.Background(), "SKU-1")
			if !errors.Is(err, tc.want) {
				t.Fatal("incorrect provider error mapping", err)
			}
		})
	}
}
