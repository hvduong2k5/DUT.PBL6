package usecase_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"dut-pbl6/inventory-service/internal/application/port"
	"dut-pbl6/inventory-service/internal/application/usecase"
	"dut-pbl6/inventory-service/internal/domain/entity"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockOutboxRepository struct {
	mock.Mock
}

func (m *MockOutboxRepository) SaveEvent(ctx context.Context, tx port.Transaction, event *entity.OutboxEvent) error {
	args := m.Called(ctx, tx, event)
	return args.Error(0)
}

func (m *MockOutboxRepository) GetPendingEvents(ctx context.Context, batchSize int) ([]*entity.OutboxEvent, error) {
	args := m.Called(ctx, batchSize)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.OutboxEvent), args.Error(1)
}

func (m *MockOutboxRepository) MarkPublished(ctx context.Context, id uuid.UUID, processedAt time.Time) error {
	args := m.Called(ctx, id, processedAt)
	return args.Error(0)
}

func (m *MockOutboxRepository) MarkFailed(ctx context.Context, id uuid.UUID, retryCount int, errorMessage string, processedAt time.Time, finalFailure bool) error {
	args := m.Called(ctx, id, retryCount, errorMessage, processedAt, finalFailure)
	return args.Error(0)
}

func TestReserveStock_OutboxEvent_EmittedInTransaction(t *testing.T) {
	mockRepo := new(MockInventoryRepository)
	mockTxManager := new(MockTransactionManager)
	mockLockService := new(MockLockService)
	mockOutboxRepo := new(MockOutboxRepository)

	uc := usecase.NewReserveStockUseCase(mockRepo, mockTxManager, mockLockService, mockOutboxRepo)
	ctx := context.Background()

	orderID := "ORD-OUTBOX-01"
	sku := "MX-ME-DEN-250G"
	whID := entity.DefaultWarehouseID
	items := []usecase.ReserveItemInput{{SKU: sku, Qty: 2}}

	// Idempotency: not reserved yet
	mockRepo.On("GetReservationByOrderID", ctx, orderID).Return(nil, entity.ErrReservationNotFound)

	// Lock
	lockKey := "lock:inventory:sku:" + sku
	mockLockService.On("AcquireLock", ctx, lockKey, 3*time.Second).Return(true, nil)
	mockLockService.On("ReleaseLock", mock.Anything, lockKey).Return(nil)

	// In Transaction
	item, _ := entity.NewInventoryItem(sku, whID, 100)
	batch, _ := entity.NewBatch("LOT-1", sku, uuid.New(), time.Now().Add(-10*24*time.Hour), time.Now().Add(60*24*time.Hour), 100)

	mockTx := new(MockTransaction)
	mockTxManager.On("ExecuteInTransaction", ctx, mock.Anything).Run(func(args mock.Arguments) {
		fn := args.Get(1).(func(port.Transaction) error)
		_ = fn(mockTx)
	}).Return(nil)

	mockRepo.On("GetItemBySKUForUpdate", ctx, mockTx, sku).Return(item, nil)
	mockRepo.On("GetActiveBatchesBySKUForUpdate", ctx, mockTx, sku).Return([]*entity.Batch{batch}, nil)
	mockRepo.On("UpdateItem", ctx, mockTx, mock.Anything).Return(nil)
	mockRepo.On("UpdateBatch", ctx, mockTx, mock.Anything).Return(nil)
	mockRepo.On("CreateReservation", ctx, mockTx, mock.Anything).Return(nil)

	// Expect Outbox event save
	mockOutboxRepo.On("SaveEvent", ctx, mockTx, mock.MatchedBy(func(ev *entity.OutboxEvent) bool {
		if ev.AggregateType != "StockReservation" {
			return false
		}
		if ev.EventType != entity.EventTypeStockReserved {
			return false
		}
		var ce entity.CloudEvent[entity.StockReservedEventData]
		if err := json.Unmarshal(ev.Payload, &ce); err != nil {
			return false
		}
		return ce.Data.OrderID == orderID && ce.Data.SKUCode == sku && ce.Data.Quantity == 2
	})).Return(nil)

	res, err := uc.Execute(ctx, orderID, items)
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, orderID, res.OrderID)

	mockOutboxRepo.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestReleaseReservation_OutboxEvent_EmittedInTransaction(t *testing.T) {
	mockRepo := new(MockInventoryRepository)
	mockTxManager := new(MockTransactionManager)
	mockLockService := new(MockLockService)
	mockOutboxRepo := new(MockOutboxRepository)

	uc := usecase.NewReleaseReservationUseCase(mockRepo, mockTxManager, mockLockService, mockOutboxRepo)
	ctx := context.Background()

	orderID := "ORD-OUTBOX-02"
	sku := "MX-GION-500G"
	whID := entity.DefaultWarehouseID
	batchID := uuid.New()

	alloc := entity.ReservationItemAllocation{SKU: sku, BatchID: batchID, AllocatedQty: 3}
	res, _ := entity.NewStockReservation(orderID, 15*time.Minute, []entity.ReservationItemAllocation{alloc})

	mockRepo.On("GetReservationByOrderID", ctx, orderID).Return(res, nil)

	lockKey := "lock:inventory:sku:" + sku
	mockLockService.On("AcquireLock", ctx, lockKey, 3*time.Second).Return(true, nil)
	mockLockService.On("ReleaseLock", mock.Anything, lockKey).Return(nil)

	mockTx := new(MockTransaction)
	mockTxManager.On("ExecuteInTransaction", ctx, mock.Anything).Run(func(args mock.Arguments) {
		fn := args.Get(1).(func(port.Transaction) error)
		_ = fn(mockTx)
	}).Return(nil)

	item, _ := entity.NewInventoryItem(sku, whID, 100)
	_ = item.Reserve(3)
	batch, _ := entity.NewBatch("LOT-2", sku, uuid.New(), time.Now().Add(-10*24*time.Hour), time.Now().Add(60*24*time.Hour), 100)
	_ = batch.Reserve(3)

	mockRepo.On("GetReservationByOrderIDForUpdate", ctx, mockTx, orderID).Return(res, nil)
	mockRepo.On("UpdateReservation", ctx, mockTx, res).Return(nil)
	mockRepo.On("GetItemBySKUForUpdate", ctx, mockTx, sku).Return(item, nil)
	mockRepo.On("UpdateItem", ctx, mockTx, item).Return(nil)
	mockRepo.On("GetBatchByID", ctx, mockTx, batchID).Return(batch, nil)
	mockRepo.On("UpdateBatch", ctx, mockTx, batch).Return(nil)

	// Expect Outbox event save for release
	mockOutboxRepo.On("SaveEvent", ctx, mockTx, mock.MatchedBy(func(ev *entity.OutboxEvent) bool {
		if ev.AggregateType != "StockReservation" {
			return false
		}
		if ev.EventType != entity.EventTypeStockReleased {
			return false
		}
		var ce entity.CloudEvent[entity.StockReleasedEventData]
		if err := json.Unmarshal(ev.Payload, &ce); err != nil {
			return false
		}
		return ce.Data.OrderID == orderID && ce.Data.SKUCode == sku && ce.Data.QuantityReleased == 3
	})).Return(nil)

	totalRestored, err := uc.ExecuteWithResult(ctx, orderID)
	require.NoError(t, err)
	assert.Equal(t, 3, totalRestored)

	mockOutboxRepo.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestReserveStock_OutboxEvent_SaveErrorFailsTransaction(t *testing.T) {
	mockRepo := new(MockInventoryRepository)
	mockTxManager := new(MockTransactionManager)
	mockLockService := new(MockLockService)
	mockOutboxRepo := new(MockOutboxRepository)

	uc := usecase.NewReserveStockUseCase(mockRepo, mockTxManager, mockLockService, mockOutboxRepo)
	ctx := context.Background()

	orderID := "ORD-OUTBOX-03"
	sku := "MX-GION-500G"
	whID := entity.DefaultWarehouseID
	items := []usecase.ReserveItemInput{{SKU: sku, Qty: 1}}

	mockRepo.On("GetReservationByOrderID", ctx, orderID).Return(nil, entity.ErrReservationNotFound)

	lockKey := "lock:inventory:sku:" + sku
	mockLockService.On("AcquireLock", ctx, lockKey, 3*time.Second).Return(true, nil)
	mockLockService.On("ReleaseLock", mock.Anything, lockKey).Return(nil)

	item, _ := entity.NewInventoryItem(sku, whID, 100)
	batch, _ := entity.NewBatch("LOT-1", sku, uuid.New(), time.Now().Add(-10*24*time.Hour), time.Now().Add(60*24*time.Hour), 100)

	mockTx := new(MockTransaction)
	outboxErr := errors.New("database disk full")

	mockTxManager.On("ExecuteInTransaction", ctx, mock.Anything).Run(func(args mock.Arguments) {
		fn := args.Get(1).(func(port.Transaction) error)
		err := fn(mockTx)
		assert.ErrorIs(t, err, outboxErr)
	}).Return(outboxErr)

	mockRepo.On("GetItemBySKUForUpdate", ctx, mockTx, sku).Return(item, nil)
	mockRepo.On("GetActiveBatchesBySKUForUpdate", ctx, mockTx, sku).Return([]*entity.Batch{batch}, nil)
	mockRepo.On("UpdateItem", ctx, mockTx, mock.Anything).Return(nil)
	mockRepo.On("UpdateBatch", ctx, mockTx, mock.Anything).Return(nil)
	mockRepo.On("CreateReservation", ctx, mockTx, mock.Anything).Return(nil)
	mockOutboxRepo.On("SaveEvent", ctx, mockTx, mock.Anything).Return(outboxErr)

	res, err := uc.Execute(ctx, orderID, items)
	require.ErrorIs(t, err, outboxErr)
	assert.Nil(t, res)

	mockOutboxRepo.AssertExpectations(t)
}

func TestCommitStockDeduction_OutboxEvent_EmittedInTransaction(t *testing.T) {
	mockRepo := new(MockInventoryRepository)
	mockTxManager := new(MockTransactionManager)
	mockLockService := new(MockLockService)
	mockOutboxRepo := new(MockOutboxRepository)

	uc := usecase.NewCommitStockDeductionUseCase(mockRepo, mockTxManager, mockLockService, mockOutboxRepo)
	ctx := context.Background()

	orderID := "ORD-COMMIT-OUTBOX-01"
	sku := "MX-GION-500G"
	whID := entity.DefaultWarehouseID
	batchID := uuid.New()

	alloc := entity.ReservationItemAllocation{SKU: sku, BatchID: batchID, AllocatedQty: 4}
	res, _ := entity.NewStockReservation(orderID, 15*time.Minute, []entity.ReservationItemAllocation{alloc})

	mockRepo.On("GetReservationByOrderID", ctx, orderID).Return(res, nil)

	lockKey := "lock:inventory:sku:" + sku
	mockLockService.On("AcquireLock", ctx, lockKey, 3*time.Second).Return(true, nil)
	mockLockService.On("ReleaseLock", mock.Anything, lockKey).Return(nil)

	mockTx := new(MockTransaction)
	mockTxManager.On("ExecuteInTransaction", ctx, mock.Anything).Run(func(args mock.Arguments) {
		fn := args.Get(1).(func(port.Transaction) error)
		_ = fn(mockTx)
	}).Return(nil)

	item, _ := entity.NewInventoryItem(sku, whID, 100)
	_ = item.Reserve(4)
	batch, _ := entity.NewBatch("LOT-1", sku, uuid.New(), time.Now().Add(-10*24*time.Hour), time.Now().Add(60*24*time.Hour), 100)
	_ = batch.Reserve(4)

	mockRepo.On("GetReservationByOrderIDForUpdate", ctx, mockTx, orderID).Return(res, nil)
	mockRepo.On("UpdateReservation", ctx, mockTx, res).Return(nil)
	mockRepo.On("GetItemBySKUForUpdate", ctx, mockTx, sku).Return(item, nil)
	mockRepo.On("UpdateItem", ctx, mockTx, item).Return(nil)
	mockRepo.On("GetBatchByID", ctx, mockTx, batchID).Return(batch, nil)
	mockRepo.On("UpdateBatch", ctx, mockTx, batch).Return(nil)

	// Expect Outbox event save for commit
	mockOutboxRepo.On("SaveEvent", ctx, mockTx, mock.MatchedBy(func(ev *entity.OutboxEvent) bool {
		if ev.AggregateType != "StockReservation" {
			return false
		}
		if ev.EventType != entity.EventTypeStockCommitted {
			return false
		}
		var ce entity.CloudEvent[entity.StockCommittedEventData]
		if err := json.Unmarshal(ev.Payload, &ce); err != nil {
			return false
		}
		return ce.Data.OrderID == orderID &&
			ce.Data.ReservationID == res.ID.String() &&
			len(ce.Data.Items) == 1 &&
			ce.Data.Items[0].SKUCode == sku &&
			ce.Data.Items[0].Quantity == 4
	})).Return(nil)

	err := uc.Execute(ctx, orderID)
	require.NoError(t, err)

	mockOutboxRepo.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestCommitStockDeduction_OutboxEvent_SaveErrorFailsTransaction(t *testing.T) {
	mockRepo := new(MockInventoryRepository)
	mockTxManager := new(MockTransactionManager)
	mockLockService := new(MockLockService)
	mockOutboxRepo := new(MockOutboxRepository)

	uc := usecase.NewCommitStockDeductionUseCase(mockRepo, mockTxManager, mockLockService, mockOutboxRepo)
	ctx := context.Background()

	orderID := "ORD-COMMIT-OUTBOX-02"
	sku := "MX-GION-500G"
	whID := entity.DefaultWarehouseID
	batchID := uuid.New()

	alloc := entity.ReservationItemAllocation{SKU: sku, BatchID: batchID, AllocatedQty: 2}
	res, _ := entity.NewStockReservation(orderID, 15*time.Minute, []entity.ReservationItemAllocation{alloc})

	mockRepo.On("GetReservationByOrderID", ctx, orderID).Return(res, nil)

	lockKey := "lock:inventory:sku:" + sku
	mockLockService.On("AcquireLock", ctx, lockKey, 3*time.Second).Return(true, nil)
	mockLockService.On("ReleaseLock", mock.Anything, lockKey).Return(nil)

	item, _ := entity.NewInventoryItem(sku, whID, 100)
	_ = item.Reserve(2)
	batch, _ := entity.NewBatch("LOT-1", sku, uuid.New(), time.Now().Add(-10*24*time.Hour), time.Now().Add(60*24*time.Hour), 100)
	_ = batch.Reserve(2)

	mockTx := new(MockTransaction)
	outboxErr := errors.New("outbox table write error")

	mockTxManager.On("ExecuteInTransaction", ctx, mock.Anything).Run(func(args mock.Arguments) {
		fn := args.Get(1).(func(port.Transaction) error)
		err := fn(mockTx)
		assert.ErrorIs(t, err, outboxErr)
	}).Return(outboxErr)

	mockRepo.On("GetReservationByOrderIDForUpdate", ctx, mockTx, orderID).Return(res, nil)
	mockRepo.On("UpdateReservation", ctx, mockTx, res).Return(nil)
	mockRepo.On("GetItemBySKUForUpdate", ctx, mockTx, sku).Return(item, nil)
	mockRepo.On("UpdateItem", ctx, mockTx, item).Return(nil)
	mockRepo.On("GetBatchByID", ctx, mockTx, batchID).Return(batch, nil)
	mockRepo.On("UpdateBatch", ctx, mockTx, batch).Return(nil)
	mockOutboxRepo.On("SaveEvent", ctx, mockTx, mock.Anything).Return(outboxErr)

	err := uc.Execute(ctx, orderID)
	require.ErrorIs(t, err, outboxErr)

	mockOutboxRepo.AssertExpectations(t)
}

func TestCommitStockDeduction_AlreadyCommitted_NoOutboxEventEmitted(t *testing.T) {
	mockRepo := new(MockInventoryRepository)
	mockTxManager := new(MockTransactionManager)
	mockLockService := new(MockLockService)
	mockOutboxRepo := new(MockOutboxRepository)

	uc := usecase.NewCommitStockDeductionUseCase(mockRepo, mockTxManager, mockLockService, mockOutboxRepo)
	ctx := context.Background()

	orderID := "ORD-ALREADY-COMMITTED"
	alloc := entity.ReservationItemAllocation{
		ID:           uuid.New(),
		SKU:          "MX-GION-500G",
		BatchID:      uuid.New(),
		AllocatedQty: 1,
	}
	res, err := entity.NewStockReservation(orderID, 15*time.Minute, []entity.ReservationItemAllocation{alloc})
	require.NoError(t, err)
	_ = res.Commit()

	mockRepo.On("GetReservationByOrderID", ctx, orderID).Return(res, nil)

	err = uc.Execute(ctx, orderID)
	require.NoError(t, err)

	mockOutboxRepo.AssertNotCalled(t, "SaveEvent", mock.Anything, mock.Anything, mock.Anything)
	mockTxManager.AssertNotCalled(t, "ExecuteInTransaction", mock.Anything, mock.Anything)
}

