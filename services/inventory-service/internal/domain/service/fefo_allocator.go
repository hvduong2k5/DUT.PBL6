package service

import (
	"sort"
	"time"

	"dut-pbl6/inventory-service/internal/domain/entity"
	"github.com/google/uuid"
)

// AllocationResult represents the outcome of allocating stock from a single batch.
type AllocationResult struct {
	BatchID      uuid.UUID `json:"batch_id"`
	BatchCode    string    `json:"batch_code"`
	SKU          string    `json:"sku"`
	AllocatedQty int       `json:"allocated_qty"`
}

// ToReservationAllocation converts an AllocationResult into a domain entity value object.
func (ar AllocationResult) ToReservationAllocation(resID uuid.UUID) entity.ReservationItemAllocation {
	return entity.ReservationItemAllocation{
		ID:            uuid.New(),
		ReservationID: resID,
		SKU:           ar.SKU,
		BatchID:       ar.BatchID,
		AllocatedQty:  ar.AllocatedQty,
	}
}

// FEFOAllocator coordinates the First-Expired, First-Out allocation algorithm (INV-BI-03).
type FEFOAllocator struct{}

// NewFEFOAllocator creates a new instance of FEFOAllocator.
func NewFEFOAllocator() *FEFOAllocator {
	return &FEFOAllocator{}
}

// AllocateByFEFO allocates requested quantity across given batches according to FEFO rules.
// Rules:
//  1. requestedQty must be strictly > 0 (UT-INV-FEFO-05).
//  2. Only batches with CanAllocate() == true (status ACTIVE/NEAR_EXPIRY, exp_date > now, and AvailableQty > 0) are considered (UT-INV-FEFO-04, INV-BI-02).
//  3. Batches are sorted by ExpDate ascending, then by CreatedAt ascending (INV-BI-03).
//  4. If total available quantity across eligible batches is less than requestedQty, ErrInsufficientStock is returned
//     without mutating any batch state (UT-INV-FEFO-03).
//  5. Eligible batches have their ReservedQty updated via b.ReserveAt(take, now) to reserve the allocated amount atomically (UT-INV-FEFO-01, 02, 06).
func AllocateByFEFO(batches []*entity.Batch, requestedQty int) ([]AllocationResult, error) {
	return AllocateByFEFOAt(batches, requestedQty, time.Now())
}

// AllocateByFEFOAt provides deterministic FEFO allocation evaluated against a specific point in time.
func AllocateByFEFOAt(batches []*entity.Batch, requestedQty int, now time.Time) ([]AllocationResult, error) {
	if requestedQty <= 0 {
		return nil, entity.ErrInvalidAllocationRequest
	}

	// Filter allocatable batches (enforcing INV-BI-02 Zero Expired Sale)
	var candidates []*entity.Batch
	totalAvail := 0
	for _, b := range batches {
		if b != nil && b.CanAllocateAt(now) {
			candidates = append(candidates, b)
			totalAvail += b.AvailableQty()
		}
	}

	// Invariant check: total available stock must satisfy request (INV-BI-01)
	if totalAvail < requestedQty {
		return nil, entity.ErrInsufficientStock
	}

	// Invariant check: candidate batches must belong to the same SKU
	if len(candidates) > 0 {
		firstSKU := candidates[0].SKU
		for _, b := range candidates[1:] {
			if b.SKU != firstSKU {
				return nil, entity.ErrInvalidBatchData
			}
		}
	}

	// Sort candidates according to FEFO: ExpDate ASC, then CreatedAt ASC
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].ExpDate.Equal(candidates[j].ExpDate) {
			return candidates[i].CreatedAt.Before(candidates[j].CreatedAt)
		}
		return candidates[i].ExpDate.Before(candidates[j].ExpDate)
	})

	var results []AllocationResult
	remaining := requestedQty

	for _, b := range candidates {
		if remaining <= 0 {
			break
		}

		avail := b.AvailableQty()
		if avail <= 0 {
			continue
		}

		take := avail
		if remaining < take {
			take = remaining
		}

		// Mutate batch reservation via domain method using the deterministic timestamp
		if err := b.ReserveAt(take, now); err != nil {
			return nil, err
		}
		remaining -= take

		results = append(results, AllocationResult{
			BatchID:      b.ID,
			BatchCode:    b.BatchCode,
			SKU:          b.SKU,
			AllocatedQty: take,
		})
	}

	return results, nil
}

// Allocate is a method wrapper on FEFOAllocator
func (a *FEFOAllocator) Allocate(batches []*entity.Batch, requestedQty int) ([]AllocationResult, error) {
	return AllocateByFEFO(batches, requestedQty)
}
