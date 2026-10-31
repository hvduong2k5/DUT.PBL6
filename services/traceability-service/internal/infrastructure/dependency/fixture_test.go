package dependency

import (
	"context"
	"errors"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/domain"
	"testing"
	"time"
)

func TestFixtureCannotSubstituteMVPProvider(t *testing.T) {
	now := time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)
	b := domain.Binding{ProductID: "11111111-1111-4111-8111-111111111111", SKUCode: "A", SourceNamespace: "fixture", BatchID: "batch"}
	snapshots := []domain.DependencySnapshot{{Binding: b, BatchCode: "LOT", CatalogVersion: "1", BatchVersion: "1"}}
	if _, err := NewFixture("integration", snapshots, func() time.Time { return now }); err == nil {
		t.Fatal("integration must not use synthetic provider")
	}
	f, err := NewFixture("sandbox", snapshots, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	for _, namespace := range []string{"inventory-service", "catalog-service", "fulfillment-service", "shipping-service"} {
		requested := b
		requested.SourceNamespace = namespace
		if _, err := f.Snapshot(context.Background(), requested); !errors.Is(err, domain.ErrUnavailable) {
			t.Fatalf("%s silently fell back: %v", namespace, err)
		}
	}
	wrong := b
	wrong.SKUCode = "B"
	if _, err := f.Snapshot(context.Background(), wrong); !errors.Is(err, domain.ErrConflict) {
		t.Fatal("mismatched SKU accepted")
	}
	missing := b
	missing.BatchID = "unknown"
	if _, err := f.Snapshot(context.Background(), missing); !errors.Is(err, domain.ErrNotFound) {
		t.Fatal("missing batch synthesized")
	}
	snapshot, err := f.Snapshot(context.Background(), b)
	if err != nil || snapshot.Binding != b || !snapshot.CheckedAt.Equal(now) {
		t.Fatalf("explicit fixture failed: %+v %v", snapshot, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := f.Snapshot(ctx, b); !errors.Is(err, context.Canceled) {
		t.Fatal("cancelled provider request executed")
	}
}
