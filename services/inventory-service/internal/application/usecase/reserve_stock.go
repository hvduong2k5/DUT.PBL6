package usecase

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"dut-pbl6/inventory-service/internal/application/port"
	"dut-pbl6/inventory-service/internal/domain/entity"
	"dut-pbl6/inventory-service/internal/domain/service"
)

// ReserveItemInput specifies the item SKU and quantity to reserve.
type ReserveItemInput struct {
	SKU string `json:"sku"`
	Qty int    `json:"qty"`
}

// ReserveStockUseCase coordinates atomic stock reservation with distributed locking and FEFO allocation.
type ReserveStockUseCase struct {
	repo        port.InventoryRepository
	txManager   port.TransactionManager
	lockService port.LockService
}

// NewReserveStockUseCase creates a new ReserveStockUseCase.
func NewReserveStockUseCase(
	repo port.InventoryRepository,
	txManager port.TransactionManager,
	lockService port.LockService,
) *ReserveStockUseCase {
	return &ReserveStockUseCase{
		repo:        repo,
		txManager:   txManager,
		lockService: lockService,
	}
}

// Execute performs stock reservation.
// Enforces:
//  1. Input validation and canonicalization: validates quantities and consolidates duplicate SKUs,
//     sorting SKUs lexicographically to prevent deadlocks in distributed locks and DB row locks.
//  2. Idempotency check: if orderID is already reserved, returns existing reservation immediately (UT-INV-APP-03).
//  3. Distributed lock acquisition: locks SKU resources in canonical order via Redlock; returns ErrConcurrentUpdate if lock fails (UT-INV-APP-02).
//  4. Database transaction: checks item availability, selects active batches FOR UPDATE, allocates via FEFO,
//     persists reservation and updates inventory atomically (UT-INV-APP-01).
//  5. Lock release upon completion.
func (uc *ReserveStockUseCase) Execute(
	ctx context.Context,
	orderID string,
	items []ReserveItemInput,
) (*entity.StockReservation, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if orderID == "" {
		return nil, entity.ErrInvalidBatchData
	}
	if len(items) == 0 {
		return nil, entity.ErrInvalidAllocationRequest
	}

	// Consolidate duplicate SKUs and validate inputs upfront
	consolidated := make(map[string]int)
	for _, item := range items {
		if item.SKU == "" || item.Qty <= 0 {
			return nil, entity.ErrInvalidAllocationRequest
		}
		consolidated[item.SKU] += item.Qty
	}

	// Sort SKUs lexicographically to guarantee canonical lock order (prevents circular deadlocks)
	sortedSKUs := make([]string, 0, len(consolidated))
	for sku := range consolidated {
		sortedSKUs = append(sortedSKUs, sku)
	}
	sort.Strings(sortedSKUs)

	// 1. Idempotency check: check if reservation already exists for this orderID
	existing, err := uc.repo.GetReservationByOrderID(ctx, orderID)
	if err == nil && existing != nil {
		// Already processed; return existing reservation immediately (UT-INV-APP-03)
		return existing, nil
	}
	if err != nil && !errors.Is(err, entity.ErrReservationNotFound) {
		return nil, err
	}

	// 2. Distributed Locking in canonical order per SKU (INV-BI-01 Anti-overselling, deadlock-free)
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
				return nil, entity.ErrConcurrentUpdate
			}
			lockedKeys = append(lockedKeys, lockKey)
		}
	}

	// 3. Execute within DB transaction
	var reservation *entity.StockReservation

	txErr := uc.txManager.ExecuteInTransaction(ctx, func(tx port.Transaction) error {
		var allAllocations []entity.ReservationItemAllocation

		for _, sku := range sortedSKUs {
			reqQty := consolidated[sku]

			// Lock item row
			itemEntity, err := uc.repo.GetItemBySKUForUpdate(ctx, tx, sku)
			if err != nil {
				return err
			}
			if itemEntity.AvailableQty() < reqQty {
				return entity.ErrInsufficientStock
			}

			// Lock active batches for this SKU ordered by exp_date ASC
			batches, err := uc.repo.GetActiveBatchesBySKUForUpdate(ctx, tx, sku)
			if err != nil {
				return err
			}

			// Execute FEFO Allocation
			allocResults, err := service.AllocateByFEFO(batches, reqQty)
			if err != nil {
				return err
			}

			// Update Item reservation
			if err := itemEntity.Reserve(reqQty); err != nil {
				return err
			}
			if err := uc.repo.UpdateItem(ctx, tx, itemEntity); err != nil {
				return err
			}

			// Update each allocated batch
			batchMap := make(map[string]*entity.Batch)
			for _, b := range batches {
				batchMap[b.ID.String()] = b
			}

			for _, ar := range allocResults {
				if b, ok := batchMap[ar.BatchID.String()]; ok {
					if err := uc.repo.UpdateBatch(ctx, tx, b); err != nil {
						return err
					}
				}
				allAllocations = append(allAllocations, entity.ReservationItemAllocation{
					SKU:          ar.SKU,
					BatchID:      ar.BatchID,
					AllocatedQty: ar.AllocatedQty,
				})
			}
		}

		// Create reservation entity with 15-minute TTL (INV-BI-05)
		newRes, err := entity.NewStockReservation(orderID, entity.DefaultReservationTTL, allAllocations)
		if err != nil {
			return err
		}

		if err := uc.repo.CreateReservation(ctx, tx, newRes); err != nil {
			return err
		}

		reservation = newRes
		return nil
	})

	if txErr != nil {
		return nil, txErr
	}

	return reservation, nil
}
