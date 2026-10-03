package port

import (
	"context"
	"time"

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
	GetItemsBySKUs(ctx context.Context, skus []string) ([]*entity.InventoryItem, error)

	// Batch operations with SELECT ... FOR UPDATE support
	GetActiveBatchesBySKUForUpdate(ctx context.Context, tx Transaction, sku string) ([]*entity.Batch, error)
	GetBatchByID(ctx context.Context, tx Transaction, batchID uuid.UUID) (*entity.Batch, error)
	GetBatchesByIDsForUpdate(ctx context.Context, tx Transaction, batchIDs []uuid.UUID) ([]*entity.Batch, error)
	UpdateBatch(ctx context.Context, tx Transaction, batch *entity.Batch) error
	GetBatchesBySKU(ctx context.Context, sku string) ([]*entity.Batch, error)

	// Reservation operations
	CreateReservation(ctx context.Context, tx Transaction, res *entity.StockReservation) error
	GetReservationByID(ctx context.Context, reservationID uuid.UUID) (*entity.StockReservation, error)
	GetReservationByOrderID(ctx context.Context, orderID string) (*entity.StockReservation, error)
	GetReservationByOrderIDForUpdate(ctx context.Context, tx Transaction, orderID string) (*entity.StockReservation, error)
	UpdateReservation(ctx context.Context, tx Transaction, res *entity.StockReservation) error
	GetExpiredPendingReservations(ctx context.Context, now time.Time, limit int) ([]*entity.StockReservation, error)
	GetBatchesNearExpiry(ctx context.Context, thresholdDate time.Time, limit int) ([]*entity.Batch, error)
}

// IdempotencyRepository provides atomic key verification to guarantee exactly-once processing.
type IdempotencyRepository interface {
	// CheckOrSet returns true if the key is newly recorded, false if already exists
	CheckOrSet(ctx context.Context, tx Transaction, key string) (bool, error)
}

// OutboxRepository handles persistence and lifecycle for transactional outbox events.
type OutboxRepository interface {
	SaveEvent(ctx context.Context, tx Transaction, event *entity.OutboxEvent) error
	GetPendingEvents(ctx context.Context, batchSize int) ([]*entity.OutboxEvent, error)
	MarkPublished(ctx context.Context, id uuid.UUID, processedAt time.Time) error
	MarkFailed(ctx context.Context, id uuid.UUID, retryCount int, errorMessage string, processedAt time.Time, finalFailure bool) error
}
