package port

import (
	"context"

	"dut-pbl6/inventory-service/internal/domain/entity"
	"github.com/google/uuid"
)

// Transaction represents an ongoing database transaction.
type Transaction interface {
	Commit() error
	Rollback() error
}

// TransactionManager executes a function within an ACID transaction.
type TransactionManager interface {
	ExecuteInTransaction(ctx context.Context, fn func(tx Transaction) error) error
}

// InventoryRepository handles persistence for inventory items, batches, and reservations.
type InventoryRepository interface {
	// Item operations with SELECT ... FOR UPDATE support
	GetItemBySKU(ctx context.Context, sku string) (*entity.InventoryItem, error)
	GetItemBySKUForUpdate(ctx context.Context, tx Transaction, sku string) (*entity.InventoryItem, error)
	UpdateItem(ctx context.Context, tx Transaction, item *entity.InventoryItem) error

	// Batch operations with SELECT ... FOR UPDATE support
	GetActiveBatchesBySKUForUpdate(ctx context.Context, tx Transaction, sku string) ([]*entity.Batch, error)
	GetBatchByID(ctx context.Context, tx Transaction, batchID uuid.UUID) (*entity.Batch, error)
	GetBatchesByIDsForUpdate(ctx context.Context, tx Transaction, batchIDs []uuid.UUID) ([]*entity.Batch, error)
	UpdateBatch(ctx context.Context, tx Transaction, batch *entity.Batch) error

	// Reservation operations
	CreateReservation(ctx context.Context, tx Transaction, res *entity.StockReservation) error
	GetReservationByOrderID(ctx context.Context, orderID string) (*entity.StockReservation, error)
	GetReservationByOrderIDForUpdate(ctx context.Context, tx Transaction, orderID string) (*entity.StockReservation, error)
	UpdateReservation(ctx context.Context, tx Transaction, res *entity.StockReservation) error
}

// IdempotencyRepository provides atomic key verification to guarantee exactly-once processing.
type IdempotencyRepository interface {
	// CheckOrSet returns true if the key is newly recorded, false if already exists
	CheckOrSet(ctx context.Context, tx Transaction, key string) (bool, error)
}
