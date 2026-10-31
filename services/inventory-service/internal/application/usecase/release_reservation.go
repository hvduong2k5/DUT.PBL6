package usecase

import (
	"context"
	"fmt"
	"sort"
	"strings"
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
	outboxRepo  port.OutboxRepository
}

// NewReleaseReservationUseCase creates a new ReleaseReservationUseCase.
func NewReleaseReservationUseCase(
	repo port.InventoryRepository,
	txManager port.TransactionManager,
	lockService port.LockService,
	outboxRepo ...port.OutboxRepository,
) *ReleaseReservationUseCase {
	var obRepo port.OutboxRepository
	if len(outboxRepo) > 0 {
		obRepo = outboxRepo[0]
	}
	return &ReleaseReservationUseCase{
		repo:        repo,
		txManager:   txManager,
		lockService: lockService,
		outboxRepo:  obRepo,
	}
}

// Execute releases the reservation and restores reserved stock.
// Accepts orderID, and optionally reservationID if orderID is not known.
// Enforces:
// 1. Idempotency check: if already CANCELLED, RELEASED, COMMITTED, or EXPIRED, exits early without DB update (UT-INV-APP-05).
// 2. Distributed locking: acquires Redlock for all affected SKUs in canonical order before updating DB state.
// 3. Lock Hierarchy: Locks and updates inventory_items first, then locks and updates batches in canonical order (Items -> Batches).
func (uc *ReleaseReservationUseCase) Execute(ctx context.Context, orderID string, reservationIDs ...string) error {
	_, err := uc.ExecuteWithResult(ctx, orderID, reservationIDs...)
	return err
}

// ExecuteWithResult releases the reservation and restores reserved stock, returning total items restored.
func (uc *ReleaseReservationUseCase) ExecuteWithResult(ctx context.Context, orderID string, reservationIDs ...string) (int, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	orderID = strings.TrimSpace(orderID)
	var reservationID string
	if len(reservationIDs) > 0 {
		reservationID = strings.TrimSpace(reservationIDs[0])
	}
	if orderID == "" && reservationID == "" {
		return 0, entity.ErrInvalidBatchData
	}

	// 1. Fetch reservation to identify affected SKUs and verify existence
	var res *entity.StockReservation
	var err error
	if orderID != "" {
		res, err = uc.repo.GetReservationByOrderID(ctx, orderID)
	} else {
		resUUID, parseErr := uuid.Parse(reservationID)
		if parseErr != nil {
			return 0, entity.ErrInvalidBatchData
		}
		res, err = uc.repo.GetReservationByID(ctx, resUUID)
	}
	if err != nil {
		return 0, err
	}
	if res == nil {
		return 0, entity.ErrReservationNotFound
	}
	orderID = res.OrderID

	// Idempotency check: if already released, cancelled, committed, or expired, exit early (UT-INV-APP-05)
	if res.Status == entity.ReservationStatusReleased ||
		res.Status == entity.ReservationStatusCancelled ||
		res.Status == entity.ReservationStatusCommitted ||
		res.Status == entity.ReservationStatusExpired {
		return 0, nil
	}

	totalRestored := 0
	for _, alloc := range res.Allocations {
		totalRestored += alloc.AllocatedQty
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
				return 0, entity.ErrConcurrentUpdate
			}
			lockedKeys = append(lockedKeys, lockKey)
		}
	}

	// 4. Execute within DB transaction enforcing Lock Hierarchy (Items -> Batches)
	txErr := uc.txManager.ExecuteInTransaction(ctx, func(tx port.Transaction) error {
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

		itemMap := make(map[string]*entity.InventoryItem)
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
			itemMap[s] = item
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

		// Save Outbox Event(s) within the same DB transaction
		if uc.outboxRepo != nil {
			for _, s := range skus {
				item := itemMap[s]
				ce, err := entity.NewStockReleasedCloudEvent(
					res.ID.String(),
					orderID,
					s,
					item.WarehouseID.String(),
					skuAllocMap[s],
					item.AvailableQty(),
					"ORDER_CANCELLED_OR_EXPIRED",
					"",
				)
				if err != nil {
					return err
				}
				outboxEvent, err := entity.NewOutboxEvent(
					"StockReservation",
					res.ID.String(),
					entity.EventTypeStockReleased,
					ce,
				)
				if err != nil {
					return err
				}
				if err := uc.outboxRepo.SaveEvent(ctx, tx, outboxEvent); err != nil {
					return err
				}
			}
		}

		return nil
	})
	if txErr != nil {
		return 0, txErr
	}

	return totalRestored, nil
}
