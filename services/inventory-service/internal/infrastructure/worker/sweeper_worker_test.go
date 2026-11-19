package worker_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"dut-pbl6/inventory-service/internal/application/port"
	"dut-pbl6/inventory-service/internal/application/usecase"
	"dut-pbl6/inventory-service/internal/domain/entity"
	"dut-pbl6/inventory-service/internal/infrastructure/worker"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockInventoryRepository struct {
	mock.Mock
}

func (m *MockInventoryRepository) GetItemBySKU(ctx context.Context, sku string) (*entity.InventoryItem, error) {
	args := m.Called(ctx, sku)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.InventoryItem), args.Error(1)
}

func (m *MockInventoryRepository) GetItemBySKUForUpdate(ctx context.Context, tx port.Transaction, sku string) (*entity.InventoryItem, error) {
	args := m.Called(ctx, tx, sku)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.InventoryItem), args.Error(1)
}

func (m *MockInventoryRepository) UpdateItem(ctx context.Context, tx port.Transaction, item *entity.InventoryItem) error {
	args := m.Called(ctx, tx, item)
	return args.Error(0)
}

func (m *MockInventoryRepository) GetItemsBySKUs(ctx context.Context, skus []string) ([]*entity.InventoryItem, error) {
	args := m.Called(ctx, skus)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.InventoryItem), args.Error(1)
}

func (m *MockInventoryRepository) GetBatchesBySKU(ctx context.Context, sku string) ([]*entity.Batch, error) {
	args := m.Called(ctx, sku)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.Batch), args.Error(1)
}

func (m *MockInventoryRepository) GetActiveBatchesBySKUForUpdate(ctx context.Context, tx port.Transaction, sku string) ([]*entity.Batch, error) {
	args := m.Called(ctx, tx, sku)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.Batch), args.Error(1)
}

func (m *MockInventoryRepository) GetBatchByID(ctx context.Context, tx port.Transaction, batchID uuid.UUID) (*entity.Batch, error) {
	args := m.Called(ctx, tx, batchID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Batch), args.Error(1)
}

func (m *MockInventoryRepository) GetBatchesByIDsForUpdate(ctx context.Context, tx port.Transaction, batchIDs []uuid.UUID) ([]*entity.Batch, error) {
	args := m.Called(ctx, tx, batchIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.Batch), args.Error(1)
}

func (m *MockInventoryRepository) UpdateBatch(ctx context.Context, tx port.Transaction, batch *entity.Batch) error {
	args := m.Called(ctx, tx, batch)
	return args.Error(0)
}

func (m *MockInventoryRepository) CreateReservation(ctx context.Context, tx port.Transaction, res *entity.StockReservation) error {
	args := m.Called(ctx, tx, res)
	return args.Error(0)
}

func (m *MockInventoryRepository) GetReservationByID(ctx context.Context, id uuid.UUID) (*entity.StockReservation, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.StockReservation), args.Error(1)
}

func (m *MockInventoryRepository) GetReservationByOrderID(ctx context.Context, orderID string) (*entity.StockReservation, error) {
	args := m.Called(ctx, orderID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.StockReservation), args.Error(1)
}

func (m *MockInventoryRepository) GetReservationByOrderIDForUpdate(ctx context.Context, tx port.Transaction, orderID string) (*entity.StockReservation, error) {
	args := m.Called(ctx, tx, orderID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.StockReservation), args.Error(1)
}

func (m *MockInventoryRepository) UpdateReservation(ctx context.Context, tx port.Transaction, res *entity.StockReservation) error {
	args := m.Called(ctx, tx, res)
	return args.Error(0)
}

func (m *MockInventoryRepository) GetExpiredPendingReservations(ctx context.Context, now time.Time, limit int) ([]*entity.StockReservation, error) {
	args := m.Called(ctx, now, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.StockReservation), args.Error(1)
}

func (m *MockInventoryRepository) GetBatchesNearExpiry(ctx context.Context, thresholdDate time.Time, limit int) ([]*entity.Batch, error) {
	args := m.Called(ctx, thresholdDate, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.Batch), args.Error(1)
}

type MockTransactionManager struct {
	mock.Mock
}

func (m *MockTransactionManager) ExecuteInTransaction(ctx context.Context, fn func(tx port.Transaction) error) error {
	args := m.Called(ctx, fn)
	if args.Get(0) != nil {
		return args.Error(0)
	}
	return nil
}

type MockTransaction struct {
	mock.Mock
}

func (m *MockTransaction) Commit() error {
	return m.Called().Error(0)
}

func (m *MockTransaction) Rollback() error {
	return m.Called().Error(0)
}

type MockLockService struct {
	mock.Mock
}

func (m *MockLockService) AcquireLock(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	args := m.Called(ctx, key, ttl)
	return args.Bool(0), args.Error(1)
}

func (m *MockLockService) ReleaseLock(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

// Tests for TTLReservationCleanupWorker
func TestTTLReservationCleanupWorker_RunOnce_Success(t *testing.T) {
	mockRepo := new(MockInventoryRepository)
	mockTxManager := new(MockTransactionManager)
	mockLockService := new(MockLockService)
	ctx := context.Background()
	now := time.Now().UTC()

	orderID1 := "ORD-CLEANUP-1"
	orderID2 := "ORD-CLEANUP-2"
	sku := "MX-GION-500G"
	whID := entity.DefaultWarehouseID
	batchID := uuid.New()

	alloc1 := entity.ReservationItemAllocation{SKU: sku, BatchID: batchID, AllocatedQty: 2}
	res1, _ := entity.NewStockReservation(orderID1, 15*time.Minute, []entity.ReservationItemAllocation{alloc1})
	alloc2 := entity.ReservationItemAllocation{SKU: sku, BatchID: batchID, AllocatedQty: 3}
	res2, _ := entity.NewStockReservation(orderID2, 15*time.Minute, []entity.ReservationItemAllocation{alloc2})

	mockRepo.On("GetExpiredPendingReservations", ctx, now, 50).Return([]*entity.StockReservation{res1, res2}, nil)

	// ReleaseReservationUseCase setup for order 1
	mockRepo.On("GetReservationByOrderID", ctx, orderID1).Return(res1, nil)
	lockKey := "lock:inventory:sku:" + sku
	mockLockService.On("AcquireLock", ctx, lockKey, 3*time.Second).Return(true, nil)
	mockLockService.On("ReleaseLock", mock.Anything, lockKey).Return(nil)

	mockTx := new(MockTransaction)
	mockTxManager.On("ExecuteInTransaction", ctx, mock.Anything).Run(func(args mock.Arguments) {
		fn := args.Get(1).(func(port.Transaction) error)
		_ = fn(mockTx)
	}).Return(nil)

	item, _ := entity.NewInventoryItem(sku, whID, 100)
	_ = item.Reserve(10)
	batch, _ := entity.NewBatch("LOT-1", sku, uuid.New(), now.Add(-10*24*time.Hour), now.Add(60*24*time.Hour), 100)
	_ = batch.Reserve(10)

	mockRepo.On("GetReservationByOrderIDForUpdate", ctx, mockTx, orderID1).Return(res1, nil)
	mockRepo.On("GetReservationByOrderIDForUpdate", ctx, mockTx, orderID2).Return(res2, nil)
	mockRepo.On("UpdateReservation", ctx, mockTx, mock.Anything).Return(nil).Times(2)
	mockRepo.On("GetItemBySKUForUpdate", ctx, mockTx, sku).Return(item, nil).Times(2)
	mockRepo.On("UpdateItem", ctx, mockTx, mock.Anything).Return(nil).Times(2)
	mockRepo.On("GetBatchByID", ctx, mockTx, batchID).Return(batch, nil).Times(2)
	mockRepo.On("UpdateBatch", ctx, mockTx, mock.Anything).Return(nil).Times(2)

	// ReleaseReservationUseCase setup for order 2
	mockRepo.On("GetReservationByOrderID", ctx, orderID2).Return(res2, nil)

	releaseUC := usecase.NewReleaseReservationUseCase(mockRepo, mockTxManager, mockLockService)
	workerObj := worker.NewTTLReservationCleanupWorker(mockRepo, releaseUC)

	released, err := workerObj.RunOnce(ctx, now)
	require.NoError(t, err)
	assert.Equal(t, 2, released)

	mockRepo.AssertExpectations(t)
}

func TestTTLReservationCleanupWorker_RunOnce_Empty(t *testing.T) {
	mockRepo := new(MockInventoryRepository)
	mockTxManager := new(MockTransactionManager)
	mockLockService := new(MockLockService)
	ctx := context.Background()
	now := time.Now().UTC()

	mockRepo.On("GetExpiredPendingReservations", ctx, now, 50).Return([]*entity.StockReservation{}, nil)

	releaseUC := usecase.NewReleaseReservationUseCase(mockRepo, mockTxManager, mockLockService)
	workerObj := worker.NewTTLReservationCleanupWorker(mockRepo, releaseUC)

	released, err := workerObj.RunOnce(ctx, now)
	require.NoError(t, err)
	assert.Equal(t, 0, released)

	mockRepo.AssertExpectations(t)
}

func TestTTLReservationCleanupWorker_Lifecycle(t *testing.T) {
	mockRepo := new(MockInventoryRepository)
	mockTxManager := new(MockTransactionManager)
	mockLockService := new(MockLockService)

	mockRepo.On("GetExpiredPendingReservations", mock.Anything, mock.Anything, mock.Anything).
		Return([]*entity.StockReservation{}, nil).Maybe()

	releaseUC := usecase.NewReleaseReservationUseCase(mockRepo, mockTxManager, mockLockService)
	cfg := worker.CleanupWorkerConfig{
		Interval:  10 * time.Millisecond,
		BatchSize: 10,
	}
	workerObj := worker.NewTTLReservationCleanupWorker(mockRepo, releaseUC, cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := workerObj.Start(ctx)
	require.NoError(t, err)

	err = workerObj.Start(ctx)
	assert.ErrorIs(t, err, worker.ErrWorkerAlreadyRunning)

	time.Sleep(30 * time.Millisecond)

	err = workerObj.Stop(1 * time.Second)
	require.NoError(t, err)

	err = workerObj.Stop(1 * time.Second)
	assert.ErrorIs(t, err, worker.ErrWorkerNotRunning)
}

// Tests for ExpiryCheckWorker
func TestExpiryCheckWorker_NearExpiryBatch_TransitionsAndEmitsOutbox(t *testing.T) {
	mockRepo := new(MockInventoryRepository)
	mockTxManager := new(MockTransactionManager)
	mockOutboxRepo := new(MockOutboxRepository)
	ctx := context.Background()
	now := time.Now().UTC()

	sku := "MX-GION-500G"
	batchID := uuid.New()
	supplierID := uuid.New()
	expDate := now.Add(30 * 24 * time.Hour) // 30 days left (< 45 days)
	mfgDate := now.Add(-30 * 24 * time.Hour)

	batch, _ := entity.NewBatch("LOT-2026-30DAYS", sku, supplierID, mfgDate, expDate, 50)
	batch.ID = batchID
	assert.Equal(t, entity.BatchStatusActive, batch.Status)

	thresholdDate := now.Add(45 * 24 * time.Hour)
	mockRepo.On("GetBatchesNearExpiry", ctx, thresholdDate, 50).Return([]*entity.Batch{batch}, nil)

	mockTx := new(MockTransaction)
	mockTxManager.On("ExecuteInTransaction", ctx, mock.Anything).Run(func(args mock.Arguments) {
		fn := args.Get(1).(func(port.Transaction) error)
		_ = fn(mockTx)
	}).Return(nil)

	mockRepo.On("GetBatchByID", ctx, mockTx, batchID).Return(batch, nil)
	mockRepo.On("UpdateBatch", ctx, mockTx, mock.MatchedBy(func(b *entity.Batch) bool {
		return b.Status == entity.BatchStatusNearExpiry
	})).Return(nil)

	mockOutboxRepo.On("SaveEvent", ctx, mockTx, mock.MatchedBy(func(ev *entity.OutboxEvent) bool {
		if ev.AggregateType != "Batch" || ev.EventType != entity.EventTypeExpiryWarning {
			return false
		}
		var ce entity.CloudEvent[entity.ExpiryWarningEventData]
		if err := json.Unmarshal(ev.Payload, &ce); err != nil {
			return false
		}
		return ce.Data.BatchCode == "LOT-2026-30DAYS" && ce.Data.DaysUntilExpiry == 30
	})).Return(nil)

	w := worker.NewExpiryCheckWorker(mockRepo, mockTxManager, mockOutboxRepo)
	processed, err := w.RunOnce(ctx, now)
	require.NoError(t, err)
	assert.Equal(t, 1, processed)

	mockRepo.AssertExpectations(t)
	mockOutboxRepo.AssertExpectations(t)
}

func TestExpiryCheckWorker_ExpiredBatch_TransitionsToExpired(t *testing.T) {
	mockRepo := new(MockInventoryRepository)
	mockTxManager := new(MockTransactionManager)
	mockOutboxRepo := new(MockOutboxRepository)
	ctx := context.Background()
	now := time.Now().UTC()

	sku := "MX-GION-500G"
	batchID := uuid.New()
	supplierID := uuid.New()
	expDate := now.Add(-1 * time.Hour) // Already expired
	mfgDate := now.Add(-60 * 24 * time.Hour)

	batch, _ := entity.NewBatch("LOT-EXPIRED", sku, supplierID, mfgDate, expDate, 50)
	batch.ID = batchID
	assert.Equal(t, entity.BatchStatusActive, batch.Status)

	thresholdDate := now.Add(45 * 24 * time.Hour)
	mockRepo.On("GetBatchesNearExpiry", ctx, thresholdDate, 50).Return([]*entity.Batch{batch}, nil)

	mockTx := new(MockTransaction)
	mockTxManager.On("ExecuteInTransaction", ctx, mock.Anything).Run(func(args mock.Arguments) {
		fn := args.Get(1).(func(port.Transaction) error)
		_ = fn(mockTx)
	}).Return(nil)

	mockRepo.On("GetBatchByID", ctx, mockTx, batchID).Return(batch, nil)
	mockRepo.On("UpdateBatch", ctx, mockTx, mock.MatchedBy(func(b *entity.Batch) bool {
		return b.Status == entity.BatchStatusExpired
	})).Return(nil)

	w := worker.NewExpiryCheckWorker(mockRepo, mockTxManager, mockOutboxRepo)
	processed, err := w.RunOnce(ctx, now)
	require.NoError(t, err)
	assert.Equal(t, 1, processed)

	mockRepo.AssertExpectations(t)
}

func TestExpiryCheckWorker_ExpiredBatch_WithReservedQty_TransitionsToQuarantine(t *testing.T) {
	mockRepo := new(MockInventoryRepository)
	mockTxManager := new(MockTransactionManager)
	mockOutboxRepo := new(MockOutboxRepository)
	ctx := context.Background()
	now := time.Now().UTC()

	sku := "MX-GION-500G"
	batchID := uuid.New()
	supplierID := uuid.New()
	expDate := now.Add(-1 * time.Hour) // Already expired
	mfgDate := now.Add(-60 * 24 * time.Hour)

	batch, _ := entity.NewBatch("LOT-RESERVED-EXPIRED", sku, supplierID, mfgDate, expDate, 50)
	batch.ID = batchID
	batch.ReservedQty = 10 // Reserved items present

	thresholdDate := now.Add(45 * 24 * time.Hour)
	mockRepo.On("GetBatchesNearExpiry", ctx, thresholdDate, 50).Return([]*entity.Batch{batch}, nil)

	mockTx := new(MockTransaction)
	mockTxManager.On("ExecuteInTransaction", ctx, mock.Anything).Run(func(args mock.Arguments) {
		fn := args.Get(1).(func(port.Transaction) error)
		_ = fn(mockTx)
	}).Return(nil)

	mockRepo.On("GetBatchByID", ctx, mockTx, batchID).Return(batch, nil)
	mockRepo.On("UpdateBatch", ctx, mockTx, mock.MatchedBy(func(b *entity.Batch) bool {
		return b.Status == entity.BatchStatusQuarantine
	})).Return(nil)

	w := worker.NewExpiryCheckWorker(mockRepo, mockTxManager, mockOutboxRepo)
	processed, err := w.RunOnce(ctx, now)
	require.NoError(t, err)
	assert.Equal(t, 1, processed)

	mockRepo.AssertExpectations(t)
}

func TestExpiryCheckWorker_Lifecycle(t *testing.T) {
	mockRepo := new(MockInventoryRepository)
	mockTxManager := new(MockTransactionManager)
	mockOutboxRepo := new(MockOutboxRepository)

	mockRepo.On("GetBatchesNearExpiry", mock.Anything, mock.Anything, mock.Anything).
		Return([]*entity.Batch{}, nil).Maybe()

	cfg := worker.ExpiryCheckWorkerConfig{
		Interval:  10 * time.Millisecond,
		BatchSize: 10,
	}
	w := worker.NewExpiryCheckWorker(mockRepo, mockTxManager, mockOutboxRepo, cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := w.Start(ctx)
	require.NoError(t, err)

	err = w.Start(ctx)
	assert.ErrorIs(t, err, worker.ErrWorkerAlreadyRunning)

	time.Sleep(30 * time.Millisecond)

	err = w.Stop(1 * time.Second)
	require.NoError(t, err)

	err = w.Stop(1 * time.Second)
	assert.ErrorIs(t, err, worker.ErrWorkerNotRunning)
}
