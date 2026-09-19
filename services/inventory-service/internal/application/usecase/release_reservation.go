package usecase

import (
	"context"
	"fmt"
	"sort"
	"time"

	"dut-pbl6/inventory-service/internal/application/port"
	"dut-pbl6/inventory-service/internal/domain/entity"
	"github.com/google/uuid"
)

// ReleaseReservationUseCase handles releasing reserved stock (order cancelled or TTL expired).
type ReleaseReservationUseCase struct {
	repo        port.InventoryRepository
	txManager   port.TransactionManager
	lockService port.LockService
}

// NewReleaseReservationUseCase creates a new ReleaseReservationUseCase.
func NewReleaseReservationUseCase(
	repo port.InventoryRepository,
	txManager port.TransactionManager,
	lockService port.LockService,
) *ReleaseReservationUseCase {
	return &ReleaseReservationUseCase{
		repo:        repo,
		txManager:   txManager,
		lockService: lockService,
	}
}

// Execute releases the reservation and restores reserved stock.
// Enforces:
// 1. Idempotency check: if already CANCELLED, RELEASED, COMMITTED, or EXPIRED, exits early without DB update (UT-INV-APP-05).
// 2. Distributed locking: acquires Redlock for all affected SKUs in canonical order before updating DB state.
// 3. Lock Hierarchy: Locks and updates inventory_items first, then locks and updates batches in canonical order (Items -> Batches).
func (uc *ReleaseReservationUseCase) Execute(ctx context.Context, orderID string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if orderID == "" {
		return entity.ErrInvalidBatchData
	}

	// 1. Fetch reservation to identify affected SKUs and verify existence
	res, err := uc.repo.GetReservationByOrderID(ctx, orderID)
	if err != nil {
		return err
	}
	if res == nil {
		return entity.ErrReservationNotFound
	}

	// Idempotency check: if already released, cancelled, committed, or expired, exit early (UT-INV-APP-05)
	if res.Status == entity.ReservationStatusReleased ||
		res.Status == entity.ReservationStatusCancelled ||
		res.Status == entity.ReservationStatusCommitted ||
		res.Status == entity.ReservationStatusExpired {
		return nil
	}

	// 2. Extract distinct SKUs and sort lexicographically (prevents distributed lock deadlocks)
	skuMap := make(map[string]bool)
	for _, alloc := range res.Allocations {
		skuMap[alloc.SKU] = true
	}
	sortedSKUs := make([]string, 0, len(skuMap))
	for s := range skuMap {
		sortedSKUs = append(sortedSKUs, s)
	}
	sort.Strings(sortedSKUs)

	// 3. Acquire distributed locks in canonical order per SKU
	var lockedKeys []string
	if uc.lockService != nil {
		defer func() {
			if len(lockedKeys) == 0 {
				return
			}
			cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
			defer cancel()
			for _, k := range lockedKeys {
				_ = uc.lockService.ReleaseLock(cleanupCtx, k)
			}
		}()

		for _, sku := range sortedSKUs {
			lockKey := fmt.Sprintf("lock:inventory:sku:%s", sku)
			acquired, err := uc.lockService.AcquireLock(ctx, lockKey, 3*time.Second)
			if err != nil || !acquired {
				return entity.ErrConcurrentUpdate
			}
			lockedKeys = append(lockedKeys, lockKey)
		}
	}

	// 4. Execute within DB transaction enforcing Lock Hierarchy (Items -> Batches)
	return uc.txManager.ExecuteInTransaction(ctx, func(tx port.Transaction) error {
		res, err := uc.repo.GetReservationByOrderIDForUpdate(ctx, tx, orderID)
		if err != nil {
			return err
		}
		if res == nil {
			return entity.ErrReservationNotFound
		}

		// Idempotency check: re-verify within transaction
		if res.Status == entity.ReservationStatusReleased ||
			res.Status == entity.ReservationStatusCancelled ||
			res.Status == entity.ReservationStatusCommitted ||
			res.Status == entity.ReservationStatusExpired {
			return nil
		}

		// Transition reservation status to RELEASED
		if err := res.Release(); err != nil {
			return err
		}
		if err := uc.repo.UpdateReservation(ctx, tx, res); err != nil {
			return err
		}

		// STEP 1: Lock and update inventory_items first in sorted canonical order (Items -> Batches)
		skuAllocMap := make(map[string]int)
		for _, alloc := range res.Allocations {
			skuAllocMap[alloc.SKU] += alloc.AllocatedQty
		}

		var skus []string
		for s := range skuAllocMap {
			skus = append(skus, s)
		}
		sort.Strings(skus)

		for _, s := range skus {
			item, err := uc.repo.GetItemBySKUForUpdate(ctx, tx, s)
			if err != nil {
				return err
			}
			if err := item.Release(skuAllocMap[s]); err != nil {
				return err
			}
			if err := uc.repo.UpdateItem(ctx, tx, item); err != nil {
				return err
			}
		}

		// STEP 2: Aggregate BatchIDs, sort canonical, lock and update batches
		batchAllocMap := make(map[uuid.UUID]int)
		for _, alloc := range res.Allocations {
			batchAllocMap[alloc.BatchID] += alloc.AllocatedQty
		}

		batchIDs := make([]uuid.UUID, 0, len(batchAllocMap))
		for bID := range batchAllocMap {
			batchIDs = append(batchIDs, bID)
		}
		sort.Slice(batchIDs, func(i, j int) bool {
			return batchIDs[i].String() < batchIDs[j].String()
		})

		for _, bID := range batchIDs {
			batch, err := uc.repo.GetBatchByID(ctx, tx, bID)
			if err != nil {
				return err
			}
			if err := batch.Release(batchAllocMap[bID]); err != nil {
				return err
			}
			if err := uc.repo.UpdateBatch(ctx, tx, batch); err != nil {
				return err
			}
		}

		return nil
	})
}
