package transport

import (
	"math"
	"sort"
	"strings"

	d "github.com/omamx/fulfillment-service/internal/domain"
	p "github.com/omamx/fulfillment-service/internal/gen/packing"
)

// Only the task snapshot loaded from PACKING_SEALED is used. Invalid legacy
// extensions are omitted without weakening or rejecting the existing base proof.
func sealedBatchLines(task d.Task) []*p.PackedBatchLine {
	if len(task.Lines) == 0 || len(task.Allocations) == 0 || len(task.Lines) > 100 {
		return nil
	}
	lines := map[string]d.Line{}
	skus := map[string]bool{}
	for _, line := range task.Lines {
		if !d.UUID(line.ID) || line.Quantity <= 0 || line.SKU == "" || lines[line.ID].ID != "" || skus[line.SKU] {
			return nil
		}
		lines[line.ID] = line
		skus[line.SKU] = true
	}
	totals := map[string]int64{}
	groups := map[string]*p.PackedBatchLine{}
	for _, slot := range task.Allocations {
		line, ok := lines[slot.LineID]
		if !ok || !d.UUID(slot.BatchID) || slot.Counted <= 0 || slot.Counted != slot.Allocated || strings.TrimSpace(slot.BatchCode) == "" || (slot.SKU != "" && slot.SKU != line.SKU) {
			return nil
		}
		totals[line.ID] += int64(slot.Counted)
		if totals[line.ID] > int64(line.Quantity) {
			return nil
		}
		key := line.ID + ":" + slot.BatchID
		group := groups[key]
		if group == nil {
			group = &p.PackedBatchLine{OrderLineId: line.ID, SkuCode: line.SKU, BatchId: slot.BatchID, BatchCode: slot.BatchCode}
			groups[key] = group
		}
		quantity := int64(group.PackedQuantity) + int64(slot.Counted)
		if group.BatchCode != slot.BatchCode || quantity > math.MaxInt32 {
			return nil
		}
		group.PackedQuantity = int32(quantity)
		if len(groups) > 1000 {
			return nil
		}
	}
	result := make([]*p.PackedBatchLine, 0, len(groups))
	for _, line := range task.Lines {
		if totals[line.ID] != int64(line.Quantity) {
			return nil
		}
		batches := []*p.PackedBatchLine{}
		for _, group := range groups {
			if group.OrderLineId == line.ID {
				batches = append(batches, group)
			}
		}
		sort.Slice(batches, func(i, j int) bool { return batches[i].BatchId < batches[j].BatchId })
		result = append(result, batches...)
	}
	return result
}
