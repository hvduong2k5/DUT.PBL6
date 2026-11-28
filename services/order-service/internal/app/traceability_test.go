package app

import (
	"testing"

	"github.com/omamx/order-service/internal/domain/core"
	packing "github.com/omamx/order-service/internal/gen/packing/v1"
	"google.golang.org/protobuf/proto"
)

func TestSealedTraceRowsRequireCompleteFrozenLines(t *testing.T) {
	line := "11111111-1111-4111-8111-111111111111"
	a, b := "22222222-2222-4222-8222-222222222222", "33333333-3333-4333-8333-333333333333"
	items := []core.SnapshotItem{{LineID: line, SKU: "SKU-1", Quantity: 5}}
	proof := &packing.PackageRevision{PackedLines: []*packing.PackedBatchLine{{OrderLineId: line, SkuCode: "SKU-1", BatchId: a, BatchCode: "LOT-A", PackedQuantity: 3}, {OrderLineId: line, SkuCode: "SKU-1", BatchId: b, BatchCode: "LOT-B", PackedQuantity: 2}}}
	rows, ok := sealedTraceRows(proof, items)
	if !ok || len(rows) != 2 || rows[0].Quantity != 3 || rows[1].Quantity != 2 {
		t.Fatal(rows, ok)
	}
	for _, tc := range []struct {
		name   string
		mutate func(*packing.PackageRevision)
	}{
		{"legacy empty", func(p *packing.PackageRevision) { p.PackedLines = nil }},
		{"wrong line", func(p *packing.PackageRevision) { p.PackedLines[0].OrderLineId = a }},
		{"wrong sku", func(p *packing.PackageRevision) { p.PackedLines[0].SkuCode = "SKU-2" }},
		{"underpacked", func(p *packing.PackageRevision) { p.PackedLines[0].PackedQuantity = 2 }},
		{"overpacked", func(p *packing.PackageRevision) { p.PackedLines[0].PackedQuantity = 4 }},
		{"negative", func(p *packing.PackageRevision) { p.PackedLines[0].PackedQuantity = -1 }},
		{"duplicate batch", func(p *packing.PackageRevision) { p.PackedLines[1].BatchId = a }},
		{"unknown batch", func(p *packing.PackageRevision) { p.PackedLines[0].BatchId = "unknown" }},
		{"missing code", func(p *packing.PackageRevision) { p.PackedLines[0].BatchCode = "" }},
		{"nil row", func(p *packing.PackageRevision) { p.PackedLines[0] = nil }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := proto.Clone(proof).(*packing.PackageRevision)
			tc.mutate(p)
			if _, ok := sealedTraceRows(p, items); ok {
				t.Fatal("invalid extension accepted")
			}
		})
	}
	if _, ok = sealedTraceRows(proof, append(items, core.SnapshotItem{LineID: a, SKU: "SKU-1", Quantity: 1})); ok {
		t.Fatal("ambiguous SKU bridge")
	}
	if _, ok = sealedTraceRows(proof, append(items, core.SnapshotItem{LineID: a, SKU: "SKU-2", Quantity: 1})); ok {
		t.Fatal("missing frozen line")
	}
}
