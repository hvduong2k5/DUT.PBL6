package transport

import (
	"testing"

	d "github.com/omamx/fulfillment-service/internal/domain"
)

func TestSealedBatchLinesPreserveExactLineQuantities(t *testing.T) {
	const line = "11111111-1111-4111-8111-111111111111"
	const a = "22222222-2222-4222-8222-222222222222"
	const b = "33333333-3333-4333-8333-333333333333"
	task := d.Task{Lines: []d.Line{{ID: line, SKU: "SKU", Quantity: 5}}, Allocations: []d.Allocation{
		{LineID: line, BatchID: a, BatchCode: "LOT-A", Allocated: 1, Counted: 1},
		{LineID: line, BatchID: b, BatchCode: "LOT-B", Allocated: 2, Counted: 2},
		{LineID: line, BatchID: a, BatchCode: "LOT-A", Allocated: 2, Counted: 2},
	}}
	rows := sealedBatchLines(task)
	if len(rows) != 2 || rows[0].BatchId != a || rows[0].PackedQuantity != 3 || rows[1].PackedQuantity != 2 || rows[0].SkuCode != "SKU" {
		t.Fatal("sealed quantity aggregation changed", rows)
	}
	// The result is independent of later/live quantities and cannot mutate the snapshot.
	rows[0].PackedQuantity = 99
	if sealedBatchLines(task)[0].PackedQuantity != 3 {
		t.Fatal("proof was mutated")
	}
	for _, change := range []func(*d.Task){
		func(x *d.Task) { x.Allocations = nil },
		func(x *d.Task) { x.Allocations[0].Counted = 0 },
		func(x *d.Task) { x.Allocations[0].SKU = "WRONG" },
		func(x *d.Task) { x.Allocations[0].BatchID = "LOT-A" },
		func(x *d.Task) { x.Allocations[0].LineID = "44444444-4444-4444-8444-444444444444" },
		func(x *d.Task) { x.Allocations[0].BatchCode = "OTHER" },
	} {
		copy := task
		copy.Allocations = append([]d.Allocation{}, task.Allocations...)
		change(&copy)
		if len(sealedBatchLines(copy)) != 0 {
			t.Fatal("invalid extension was exposed")
		}
	}
}
