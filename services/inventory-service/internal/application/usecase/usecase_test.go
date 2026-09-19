package usecase_test

import (
	"context"
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

// --- MOCKS ---

type MockTransaction struct {
	mock.Mock
}

func (m *MockTransaction) Commit() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockTransaction) Rollback() error {
	args := m.Called()
	return args.Error(0)
}

type MockTransactionManager struct {
	mock.Mock
}

func (m *MockTransactionManager) ExecuteInTransaction(ctx context.Context, fn func(tx port.Transaction) error) error {
	args := m.Called(ctx, fn)
	if args.Get(0) != nil {
		return args.Error(0)
	}
	mockTx := &MockTransaction{}
	return fn(mockTx)
}

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

// --- UNIT TESTS UT-INV-APP-01..06 ---

// UT-INV-APP-01: Luồng cơ bản (Happy path) cho việc giữ chỗ tồn kho
func TestUT_INV_APP_01_ReserveStock_HappyPath(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockInventoryRepository)
	mockTxManager := new(MockTransactionManager)
	mockLock := new(MockLockService)

	sku := "SKU-OC-MEXUNG-001"
	orderID := "ORDER-2026-0001"
	items := []usecase.ReserveItemInput{{SKU: sku, Qty: 20}}

	// Item state
	item, err := entity.NewInventoryItem(sku, entity.DefaultWarehouseID, 100)
	require.NoError(t, err)

	// Batch state
	now := time.Now()
	batch, err := entity.NewBatch("LOT-1", sku, uuid.New(), now.Add(-10*24*time.Hour), now.Add(60*24*time.Hour), 100)
	require.NoError(t, err)

	// Expectations
	mockRepo.On("GetReservationByOrderID", ctx, orderID).Return(nil, entity.ErrReservationNotFound)
	mockLock.On("AcquireLock", ctx, "lock:inventory:sku:"+sku, 3*time.Second).Return(true, nil)
	mockLock.On("ReleaseLock", mock.Anything, "lock:inventory:sku:"+sku).Return(nil)

	mockTxManager.On("ExecuteInTransaction", ctx, mock.Anything).Return(nil)
	mockRepo.On("GetItemBySKUForUpdate", ctx, mock.Anything, sku).Return(item, nil)
	mockRepo.On("GetActiveBatchesBySKUForUpdate", ctx, mock.Anything, sku).Return([]*entity.Batch{batch}, nil)
	mockRepo.On("UpdateItem", ctx, mock.Anything, mock.MatchedBy(func(i *entity.InventoryItem) bool {
		return i.ReservedQty == 20
	})).Return(nil)
	mockRepo.On("UpdateBatch", ctx, mock.Anything, mock.MatchedBy(func(b *entity.Batch) bool {
		return b.ReservedQty == 20
	})).Return(nil)
	mockRepo.On("CreateReservation", ctx, mock.Anything, mock.MatchedBy(func(r *entity.StockReservation) bool {
		return r.OrderID == orderID && len(r.Allocations) == 1 && r.Allocations[0].AllocatedQty == 20
	})).Return(nil)

	uc := usecase.NewReserveStockUseCase(mockRepo, mockTxManager, mockLock)

	// When
	res, err := uc.Execute(ctx, orderID, items)

	// Then
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, orderID, res.OrderID)
	assert.Equal(t, entity.ReservationStatusPending, res.Status)

	mockLock.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockTxManager.AssertExpectations(t)
}

// UT-INV-APP-02: Xử lý tranh chấp khóa (Lock contention)
func TestUT_INV_APP_02_ReserveStock_LockContention(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockInventoryRepository)
	mockTxManager := new(MockTransactionManager)
	mockLock := new(MockLockService)

	sku := "SKU-OC-MEXUNG-001"
	orderID := "ORDER-2026-0002"
	items := []usecase.ReserveItemInput{{SKU: sku, Qty: 10}}

	// Expectations:
	// 1. Check reservation: not exists
	mockRepo.On("GetReservationByOrderID", ctx, orderID).Return(nil, entity.ErrReservationNotFound)
	// 2. LockService returns false (lock already held by another pod/process)
	mockLock.On("AcquireLock", ctx, "lock:inventory:sku:"+sku, 3*time.Second).Return(false, nil)

	uc := usecase.NewReserveStockUseCase(mockRepo, mockTxManager, mockLock)

	// When: Gọi Execute
	res, err := uc.Execute(ctx, orderID, items)

	// Then: Trả về lỗi ErrConcurrentUpdate. Không thực hiện bất kỳ query nào vào CSDL Repository.
	assert.ErrorIs(t, err, entity.ErrConcurrentUpdate)
	assert.Nil(t, res)

	mockLock.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "GetItemBySKUForUpdate", mock.Anything, mock.Anything, mock.Anything)
	mockRepo.AssertNotCalled(t, "GetActiveBatchesBySKUForUpdate", mock.Anything, mock.Anything, mock.Anything)
	mockTxManager.AssertNotCalled(t, "ExecuteInTransaction", mock.Anything, mock.Anything)
}

// UT-INV-APP-03: Đảm bảo tính Idempotency khi mạng bị chập chờn
func TestUT_INV_APP_03_ReserveStock_Idempotency(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockInventoryRepository)
	mockTxManager := new(MockTransactionManager)
	mockLock := new(MockLockService)

	sku := "SKU-OC-MEXUNG-001"
	orderID := "ORDER-2026-EXISTS"
	items := []usecase.ReserveItemInput{{SKU: sku, Qty: 10}}

	existingRes, err := entity.NewStockReservation(orderID, entity.DefaultReservationTTL, []entity.ReservationItemAllocation{
		{
			ID:           uuid.New(),
			SKU:          sku,
			BatchID:      uuid.New(),
			AllocatedQty: 10,
		},
	})
	require.NoError(t, err)

	// Given: Reservation cho order_id đã tồn tại trong DB
	mockRepo.On("GetReservationByOrderID", ctx, orderID).Return(existingRes, nil)

	uc := usecase.NewReserveStockUseCase(mockRepo, mockTxManager, mockLock)

	// When: Gửi lại request Execute(order_id=X, items)
	res, err := uc.Execute(ctx, orderID, items)

	// Then: Trả về kết quả thành công ngay lập tức dựa trên bản ghi cũ. Tồn kho không bị trừ/giữ thêm lần 2.
	require.NoError(t, err)
	assert.Equal(t, existingRes.ID, res.ID)
	assert.Equal(t, orderID, res.OrderID)

	mockLock.AssertNotCalled(t, "AcquireLock", mock.Anything, mock.Anything, mock.Anything)
	mockTxManager.AssertNotCalled(t, "ExecuteInTransaction", mock.Anything, mock.Anything)
	mockRepo.AssertNotCalled(t, "UpdateItem", mock.Anything, mock.Anything, mock.Anything)
}

// UT-INV-APP-04: Giải phóng tồn kho thành công do đơn hàng bị hủy hoặc timeout
func TestUT_INV_APP_04_ReleaseReservation_HappyPath(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockInventoryRepository)
	mockTxManager := new(MockTransactionManager)
	mockLock := new(MockLockService)

	orderID := "ORDER-2026-CANCEL"
	sku := "SKU-OC-MEXUNG-001"
	batchID := uuid.New()

	existingRes, err := entity.NewStockReservation(orderID, entity.DefaultReservationTTL, []entity.ReservationItemAllocation{
		{
			ID:           uuid.New(),
			SKU:          sku,
			BatchID:      batchID,
			AllocatedQty: 25,
		},
	})
	require.NoError(t, err)
	assert.Equal(t, entity.ReservationStatusPending, existingRes.Status)

	// Batch and Item with reserved_qty = 25
	batch := &entity.Batch{
		ID:          batchID,
		SKU:         sku,
		PhysicalQty: 100,
		ReservedQty: 25,
		Status:      entity.BatchStatusActive,
	}
	item := &entity.InventoryItem{
		SKU:         sku,
		PhysicalQty: 100,
		ReservedQty: 25,
		Status:      entity.ItemStatusActive,
	}

	mockRepo.On("GetReservationByOrderID", ctx, orderID).Return(existingRes, nil)
	mockLock.On("AcquireLock", ctx, "lock:inventory:sku:"+sku, 3*time.Second).Return(true, nil)
	mockLock.On("ReleaseLock", mock.Anything, "lock:inventory:sku:"+sku).Return(nil)

	mockTxManager.On("ExecuteInTransaction", ctx, mock.Anything).Return(nil)
	mockRepo.On("GetReservationByOrderIDForUpdate", ctx, mock.Anything, orderID).Return(existingRes, nil)
	mockRepo.On("UpdateReservation", ctx, mock.Anything, mock.MatchedBy(func(r *entity.StockReservation) bool {
		return r.Status == entity.ReservationStatusReleased
	})).Return(nil)

	// Lock Hierarchy: Item updated first, Batch updated second
	mockRepo.On("GetItemBySKUForUpdate", ctx, mock.Anything, sku).Return(item, nil)
	mockRepo.On("UpdateItem", ctx, mock.Anything, mock.MatchedBy(func(i *entity.InventoryItem) bool {
		return i.ReservedQty == 0
	})).Return(nil)

	mockRepo.On("GetBatchByID", ctx, mock.Anything, batchID).Return(batch, nil)
	mockRepo.On("UpdateBatch", ctx, mock.Anything, mock.MatchedBy(func(b *entity.Batch) bool {
		return b.ReservedQty == 0
	})).Return(nil)

	uc := usecase.NewReleaseReservationUseCase(mockRepo, mockTxManager, mockLock)

	// When: Gọi Execute(order_id)
	err = uc.Execute(ctx, orderID)

	// Then: Các lô được hoàn trả reserved_qty, Reservation record chuyển sang RELEASED
	require.NoError(t, err)
	assert.Equal(t, entity.ReservationStatusReleased, existingRes.Status)
	assert.Equal(t, 0, batch.ReservedQty)
	assert.Equal(t, 0, item.ReservedQty)

	mockLock.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockTxManager.AssertExpectations(t)
}

// UT-INV-APP-05: Đảm bảo Idempotency khi giải phóng
func TestUT_INV_APP_05_ReleaseReservation_Idempotency(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockInventoryRepository)
	mockTxManager := new(MockTransactionManager)
	mockLock := new(MockLockService)

	orderID := "ORDER-2026-ALREADY-RELEASED"

	existingRes, err := entity.NewStockReservation(orderID, entity.DefaultReservationTTL, []entity.ReservationItemAllocation{
		{
			ID:           uuid.New(),
			SKU:          "SKU-OC-MEXUNG-001",
			BatchID:      uuid.New(),
			AllocatedQty: 10,
		},
	})
	require.NoError(t, err)
	existingRes.Status = entity.ReservationStatusReleased // Already RELEASED

	mockRepo.On("GetReservationByOrderID", ctx, orderID).Return(existingRes, nil)

	uc := usecase.NewReleaseReservationUseCase(mockRepo, mockTxManager, mockLock)

	// When: Gọi Execute(order_id)
	err = uc.Execute(ctx, orderID)

	// Then: Kết thúc thành công sớm, không gọi DB Update hay Lock. Không có số lượng nào bị trừ/cộng sai lệnh.
	require.NoError(t, err)
	mockLock.AssertNotCalled(t, "AcquireLock", mock.Anything, mock.Anything, mock.Anything)
	mockTxManager.AssertNotCalled(t, "ExecuteInTransaction", mock.Anything, mock.Anything)
	mockRepo.AssertNotCalled(t, "UpdateReservation", mock.Anything, mock.Anything, mock.Anything)
	mockRepo.AssertNotCalled(t, "UpdateBatch", mock.Anything, mock.Anything, mock.Anything)
	mockRepo.AssertNotCalled(t, "UpdateItem", mock.Anything, mock.Anything, mock.Anything)
}

// UT-INV-APP-06: Đơn hàng thanh toán thành công, xác nhận trừ thật số lượng
func TestUT_INV_APP_06_CommitStockDeduction(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockInventoryRepository)
	mockTxManager := new(MockTransactionManager)
	mockLock := new(MockLockService)

	orderID := "ORDER-2026-COMMIT"
	sku := "SKU-OC-MEXUNG-001"
	batchID := uuid.New()

	existingRes, err := entity.NewStockReservation(orderID, entity.DefaultReservationTTL, []entity.ReservationItemAllocation{
		{
			ID:           uuid.New(),
			SKU:          sku,
			BatchID:      batchID,
			AllocatedQty: 30,
		},
	})
	require.NoError(t, err)
	existingRes.Status = entity.ReservationStatusPending

	batch := &entity.Batch{
		ID:          batchID,
		SKU:         sku,
		PhysicalQty: 100,
		ReservedQty: 30,
		Status:      entity.BatchStatusActive,
	}
	item := &entity.InventoryItem{
		SKU:         sku,
		PhysicalQty: 100,
		ReservedQty: 30,
		Status:      entity.ItemStatusActive,
	}

	mockRepo.On("GetReservationByOrderID", ctx, orderID).Return(existingRes, nil)
	mockLock.On("AcquireLock", ctx, "lock:inventory:sku:"+sku, 3*time.Second).Return(true, nil)
	mockLock.On("ReleaseLock", mock.Anything, "lock:inventory:sku:"+sku).Return(nil)

	mockTxManager.On("ExecuteInTransaction", ctx, mock.Anything).Return(nil)
	mockRepo.On("GetReservationByOrderIDForUpdate", ctx, mock.Anything, orderID).Return(existingRes, nil)
	mockRepo.On("UpdateReservation", ctx, mock.Anything, mock.MatchedBy(func(r *entity.StockReservation) bool {
		return r.Status == entity.ReservationStatusCommitted
	})).Return(nil)

	// Lock Hierarchy: Item updated first, Batch updated second
	mockRepo.On("GetItemBySKUForUpdate", ctx, mock.Anything, sku).Return(item, nil)
	mockRepo.On("UpdateItem", ctx, mock.Anything, mock.MatchedBy(func(i *entity.InventoryItem) bool {
		return i.PhysicalQty == 70 && i.ReservedQty == 0
	})).Return(nil)

	mockRepo.On("GetBatchByID", ctx, mock.Anything, batchID).Return(batch, nil)
	mockRepo.On("UpdateBatch", ctx, mock.Anything, mock.MatchedBy(func(b *entity.Batch) bool {
		return b.PhysicalQty == 70 && b.ReservedQty == 0
	})).Return(nil)

	uc := usecase.NewCommitStockDeductionUseCase(mockRepo, mockTxManager, mockLock)

	// When: Gọi Execute(order_id)
	err = uc.Execute(ctx, orderID)

	// Then: Q_phy giảm xuống, Q_res giảm tương ứng cho các lô đã phân bổ. Chuyển Reservation status thành COMMITTED.
	require.NoError(t, err)
	assert.Equal(t, entity.ReservationStatusCommitted, existingRes.Status)
	assert.Equal(t, 70, batch.PhysicalQty)
	assert.Equal(t, 0, batch.ReservedQty)
	assert.Equal(t, 70, item.PhysicalQty)
	assert.Equal(t, 0, item.ReservedQty)

	mockLock.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockTxManager.AssertExpectations(t)
}

// Test multi-item reservation with duplicate SKUs consolidating quantities and locking once
func TestReserveStock_DuplicateSKU_ConsolidationAndSingleLock(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockInventoryRepository)
	mockTxManager := new(MockTransactionManager)
	mockLock := new(MockLockService)

	sku := "SKU-OC-DUP"
	orderID := "ORDER-DUP-001"
	// Order requests same SKU twice (e.g. 10 units + 15 units = 25 total)
	items := []usecase.ReserveItemInput{
		{SKU: sku, Qty: 10},
		{SKU: sku, Qty: 15},
	}

	item, err := entity.NewInventoryItem(sku, entity.DefaultWarehouseID, 100)
	require.NoError(t, err)

	now := time.Now()
	batch, err := entity.NewBatch("LOT-DUP-1", sku, uuid.New(), now.Add(-10*24*time.Hour), now.Add(60*24*time.Hour), 100)
	require.NoError(t, err)

	mockRepo.On("GetReservationByOrderID", ctx, orderID).Return(nil, entity.ErrReservationNotFound)
	// Must lock EXACTLY ONCE for this SKU with total 25 units
	mockLock.On("AcquireLock", ctx, "lock:inventory:sku:"+sku, 3*time.Second).Return(true, nil).Once()
	mockLock.On("ReleaseLock", mock.Anything, "lock:inventory:sku:"+sku).Return(nil).Once()

	mockTxManager.On("ExecuteInTransaction", ctx, mock.Anything).Return(nil)
	mockRepo.On("GetItemBySKUForUpdate", ctx, mock.Anything, sku).Return(item, nil).Once()
	mockRepo.On("GetActiveBatchesBySKUForUpdate", ctx, mock.Anything, sku).Return([]*entity.Batch{batch}, nil).Once()
	mockRepo.On("UpdateItem", ctx, mock.Anything, mock.MatchedBy(func(i *entity.InventoryItem) bool {
		return i.ReservedQty == 25
	})).Return(nil).Once()
	mockRepo.On("UpdateBatch", ctx, mock.Anything, mock.MatchedBy(func(b *entity.Batch) bool {
		return b.ReservedQty == 25
	})).Return(nil).Once()
	mockRepo.On("CreateReservation", ctx, mock.Anything, mock.MatchedBy(func(r *entity.StockReservation) bool {
		return r.OrderID == orderID && len(r.Allocations) == 1 && r.Allocations[0].AllocatedQty == 25
	})).Return(nil).Once()

	uc := usecase.NewReserveStockUseCase(mockRepo, mockTxManager, mockLock)
	res, err := uc.Execute(ctx, orderID, items)

	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, 25, res.Allocations[0].AllocatedQty)

	mockLock.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

// Test multi-SKU canonical lock ordering (alphabetical order)
func TestReserveStock_CanonicalLockOrder_MultiSKU(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockInventoryRepository)
	mockTxManager := new(MockTransactionManager)
	mockLock := new(MockLockService)

	orderID := "ORDER-MULTI-001"
	// Unordered input: SKU-Z comes before SKU-A
	items := []usecase.ReserveItemInput{
		{SKU: "SKU-Z", Qty: 5},
		{SKU: "SKU-A", Qty: 10},
	}

	mockRepo.On("GetReservationByOrderID", ctx, orderID).Return(nil, entity.ErrReservationNotFound)

	// Locks must be acquired in alphabetical order: SKU-A first, then SKU-Z
	var lockOrder []string
	mockLock.On("AcquireLock", ctx, "lock:inventory:sku:SKU-A", 3*time.Second).Run(func(args mock.Arguments) {
		lockOrder = append(lockOrder, "SKU-A")
	}).Return(true, nil)
	mockLock.On("AcquireLock", ctx, "lock:inventory:sku:SKU-Z", 3*time.Second).Run(func(args mock.Arguments) {
		lockOrder = append(lockOrder, "SKU-Z")
	}).Return(true, nil)

	mockLock.On("ReleaseLock", mock.Anything, "lock:inventory:sku:SKU-A").Return(nil)
	mockLock.On("ReleaseLock", mock.Anything, "lock:inventory:sku:SKU-Z").Return(nil)

	now := time.Now()
	itemA, _ := entity.NewInventoryItem("SKU-A", entity.DefaultWarehouseID, 100)
	itemZ, _ := entity.NewInventoryItem("SKU-Z", entity.DefaultWarehouseID, 100)
	batchA, _ := entity.NewBatch("LOT-A", "SKU-A", uuid.New(), now.Add(-10*24*time.Hour), now.Add(60*24*time.Hour), 100)
	batchZ, _ := entity.NewBatch("LOT-Z", "SKU-Z", uuid.New(), now.Add(-10*24*time.Hour), now.Add(60*24*time.Hour), 100)

	mockTxManager.On("ExecuteInTransaction", ctx, mock.Anything).Return(nil)
	mockRepo.On("GetItemBySKUForUpdate", ctx, mock.Anything, "SKU-A").Return(itemA, nil)
	mockRepo.On("GetItemBySKUForUpdate", ctx, mock.Anything, "SKU-Z").Return(itemZ, nil)
	mockRepo.On("GetActiveBatchesBySKUForUpdate", ctx, mock.Anything, "SKU-A").Return([]*entity.Batch{batchA}, nil)
	mockRepo.On("GetActiveBatchesBySKUForUpdate", ctx, mock.Anything, "SKU-Z").Return([]*entity.Batch{batchZ}, nil)
	mockRepo.On("UpdateItem", ctx, mock.Anything, mock.Anything).Return(nil)
	mockRepo.On("UpdateBatch", ctx, mock.Anything, mock.Anything).Return(nil)
	mockRepo.On("CreateReservation", ctx, mock.Anything, mock.Anything).Return(nil)

	uc := usecase.NewReserveStockUseCase(mockRepo, mockTxManager, mockLock)
	_, err := uc.Execute(ctx, orderID, items)

	require.NoError(t, err)
	// Assert canonical lock acquisition order
	assert.Equal(t, []string{"SKU-A", "SKU-Z"}, lockOrder)
}

// Test early input validation in ReserveStockUseCase
func TestReserveStock_InputValidation(t *testing.T) {
	ctx := context.Background()
	uc := usecase.NewReserveStockUseCase(nil, nil, nil)

	// Empty order ID
	res1, err1 := uc.Execute(ctx, "", []usecase.ReserveItemInput{{SKU: "SKU-1", Qty: 5}})
	assert.ErrorIs(t, err1, entity.ErrInvalidBatchData)
	assert.Nil(t, res1)

	// Empty items
	res2, err2 := uc.Execute(ctx, "ORD-1", nil)
	assert.ErrorIs(t, err2, entity.ErrInvalidAllocationRequest)
	assert.Nil(t, res2)

	// Non-positive quantity
	res3, err3 := uc.Execute(ctx, "ORD-1", []usecase.ReserveItemInput{{SKU: "SKU-1", Qty: 0}})
	assert.ErrorIs(t, err3, entity.ErrInvalidAllocationRequest)
	assert.Nil(t, res3)

	// Empty SKU
	res4, err4 := uc.Execute(ctx, "ORD-1", []usecase.ReserveItemInput{{SKU: "", Qty: 5}})
	assert.ErrorIs(t, err4, entity.ErrInvalidAllocationRequest)
	assert.Nil(t, res4)
}

// Test release reservation on EXPIRED status is idempotent and prevents double-release
func TestReleaseReservation_ExpiredIdempotency_NoDoubleRelease(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockInventoryRepository)
	mockTxManager := new(MockTransactionManager)
	mockLock := new(MockLockService)

	orderID := "ORDER-EXPIRED"
	res := &entity.StockReservation{
		ID:        uuid.New(),
		OrderID:   orderID,
		Status:    entity.ReservationStatusExpired, // Already EXPIRED by worker
		CreatedAt: time.Now().Add(-20 * time.Minute),
		Allocations: []entity.ReservationItemAllocation{
			{SKU: "SKU-1", BatchID: uuid.New(), AllocatedQty: 20},
		},
	}

	mockRepo.On("GetReservationByOrderID", ctx, orderID).Return(res, nil)

	uc := usecase.NewReleaseReservationUseCase(mockRepo, mockTxManager, mockLock)
	err := uc.Execute(ctx, orderID)

	// Must succeed idempotently without calling UpdateReservation, UpdateBatch, or UpdateItem
	require.NoError(t, err)
	mockLock.AssertNotCalled(t, "AcquireLock", mock.Anything, mock.Anything, mock.Anything)
	mockTxManager.AssertNotCalled(t, "ExecuteInTransaction", mock.Anything, mock.Anything)
	mockRepo.AssertNotCalled(t, "UpdateReservation", mock.Anything, mock.Anything, mock.Anything)
	mockRepo.AssertNotCalled(t, "UpdateBatch", mock.Anything, mock.Anything, mock.Anything)
	mockRepo.AssertNotCalled(t, "UpdateItem", mock.Anything, mock.Anything, mock.Anything)
}

// Test CommitStockDeduction with invalid states
func TestCommitStockDeduction_InvalidStates(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockInventoryRepository)
	mockTxManager := new(MockTransactionManager)
	mockLock := new(MockLockService)

	uc := usecase.NewCommitStockDeductionUseCase(mockRepo, mockTxManager, mockLock)

	// Empty orderID
	assert.ErrorIs(t, uc.Execute(ctx, ""), entity.ErrInvalidBatchData)

	// Reservation not found
	mockRepo.On("GetReservationByOrderID", ctx, "ORD-NOT-FOUND").Return(nil, entity.ErrReservationNotFound).Once()
	assert.ErrorIs(t, uc.Execute(ctx, "ORD-NOT-FOUND"), entity.ErrReservationNotFound)

	// Reservation in RELEASED state
	resReleased := &entity.StockReservation{
		ID:      uuid.New(),
		OrderID: "ORD-REL",
		Status:  entity.ReservationStatusReleased,
	}
	mockRepo.On("GetReservationByOrderID", ctx, "ORD-REL").Return(resReleased, nil).Once()
	assert.ErrorIs(t, uc.Execute(ctx, "ORD-REL"), entity.ErrReservationAlreadyProcessed)
}

// Test ReleaseReservation lock contention
func TestReleaseReservation_LockContention(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockInventoryRepository)
	mockTxManager := new(MockTransactionManager)
	mockLock := new(MockLockService)

	orderID := "ORDER-LOCK-CONT-REL"
	sku := "SKU-OC-MEXUNG-001"
	res := &entity.StockReservation{
		ID:      uuid.New(),
		OrderID: orderID,
		Status:  entity.ReservationStatusPending,
		Allocations: []entity.ReservationItemAllocation{
			{SKU: sku, BatchID: uuid.New(), AllocatedQty: 10},
		},
	}

	mockRepo.On("GetReservationByOrderID", ctx, orderID).Return(res, nil)
	mockLock.On("AcquireLock", ctx, "lock:inventory:sku:"+sku, 3*time.Second).Return(false, nil)

	uc := usecase.NewReleaseReservationUseCase(mockRepo, mockTxManager, mockLock)
	err := uc.Execute(ctx, orderID)

	assert.ErrorIs(t, err, entity.ErrConcurrentUpdate)
	mockTxManager.AssertNotCalled(t, "ExecuteInTransaction", mock.Anything, mock.Anything)
}

// Test CommitStockDeduction lock contention
func TestCommitStockDeduction_LockContention(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockInventoryRepository)
	mockTxManager := new(MockTransactionManager)
	mockLock := new(MockLockService)

	orderID := "ORDER-LOCK-CONT-COMMIT"
	sku := "SKU-OC-MEXUNG-001"
	res := &entity.StockReservation{
		ID:      uuid.New(),
		OrderID: orderID,
		Status:  entity.ReservationStatusPending,
		Allocations: []entity.ReservationItemAllocation{
			{SKU: sku, BatchID: uuid.New(), AllocatedQty: 10},
		},
	}

	mockRepo.On("GetReservationByOrderID", ctx, orderID).Return(res, nil)
	mockLock.On("AcquireLock", ctx, "lock:inventory:sku:"+sku, 3*time.Second).Return(false, nil)

	uc := usecase.NewCommitStockDeductionUseCase(mockRepo, mockTxManager, mockLock)
	err := uc.Execute(ctx, orderID)

	assert.ErrorIs(t, err, entity.ErrConcurrentUpdate)
	mockTxManager.AssertNotCalled(t, "ExecuteInTransaction", mock.Anything, mock.Anything)
}

// Test ReleaseReservation multi-SKU and multi-batch canonical lock hierarchy (Items -> Batches)
func TestReleaseReservation_MultiSKU_MultiBatch_CanonicalLockHierarchy(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockInventoryRepository)
	mockTxManager := new(MockTransactionManager)
	mockLock := new(MockLockService)

	orderID := "ORDER-MULTI-REL"
	skuA := "SKU-AAA"
	skuZ := "SKU-ZZZ"

	batchID1 := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	batchID2 := uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff")

	res := &entity.StockReservation{
		ID:      uuid.New(),
		OrderID: orderID,
		Status:  entity.ReservationStatusPending,
		Allocations: []entity.ReservationItemAllocation{
			{SKU: skuZ, BatchID: batchID2, AllocatedQty: 15},
			{SKU: skuA, BatchID: batchID1, AllocatedQty: 10},
		},
	}

	mockRepo.On("GetReservationByOrderID", ctx, orderID).Return(res, nil)

	var lockOrder []string
	mockLock.On("AcquireLock", ctx, "lock:inventory:sku:"+skuA, 3*time.Second).Run(func(args mock.Arguments) {
		lockOrder = append(lockOrder, "lock:"+skuA)
	}).Return(true, nil)
	mockLock.On("AcquireLock", ctx, "lock:inventory:sku:"+skuZ, 3*time.Second).Run(func(args mock.Arguments) {
		lockOrder = append(lockOrder, "lock:"+skuZ)
	}).Return(true, nil)
	mockLock.On("ReleaseLock", mock.Anything, "lock:inventory:sku:"+skuA).Return(nil)
	mockLock.On("ReleaseLock", mock.Anything, "lock:inventory:sku:"+skuZ).Return(nil)

	mockTxManager.On("ExecuteInTransaction", ctx, mock.Anything).Return(nil)
	mockRepo.On("GetReservationByOrderIDForUpdate", ctx, mock.Anything, orderID).Return(res, nil)
	mockRepo.On("UpdateReservation", ctx, mock.Anything, mock.Anything).Return(nil)

	var dbLockOrder []string
	itemA := &entity.InventoryItem{SKU: skuA, ReservedQty: 10}
	itemZ := &entity.InventoryItem{SKU: skuZ, ReservedQty: 15}
	batch1 := &entity.Batch{ID: batchID1, SKU: skuA, ReservedQty: 10}
	batch2 := &entity.Batch{ID: batchID2, SKU: skuZ, ReservedQty: 15}

	mockRepo.On("GetItemBySKUForUpdate", ctx, mock.Anything, skuA).Run(func(args mock.Arguments) {
		dbLockOrder = append(dbLockOrder, "item:"+skuA)
	}).Return(itemA, nil)
	mockRepo.On("UpdateItem", ctx, mock.Anything, itemA).Return(nil)

	mockRepo.On("GetItemBySKUForUpdate", ctx, mock.Anything, skuZ).Run(func(args mock.Arguments) {
		dbLockOrder = append(dbLockOrder, "item:"+skuZ)
	}).Return(itemZ, nil)
	mockRepo.On("UpdateItem", ctx, mock.Anything, itemZ).Return(nil)

	mockRepo.On("GetBatchByID", ctx, mock.Anything, batchID1).Run(func(args mock.Arguments) {
		dbLockOrder = append(dbLockOrder, "batch:"+batchID1.String())
	}).Return(batch1, nil)
	mockRepo.On("UpdateBatch", ctx, mock.Anything, batch1).Return(nil)

	mockRepo.On("GetBatchByID", ctx, mock.Anything, batchID2).Run(func(args mock.Arguments) {
		dbLockOrder = append(dbLockOrder, "batch:"+batchID2.String())
	}).Return(batch2, nil)
	mockRepo.On("UpdateBatch", ctx, mock.Anything, batch2).Return(nil)

	uc := usecase.NewReleaseReservationUseCase(mockRepo, mockTxManager, mockLock)
	err := uc.Execute(ctx, orderID)

	require.NoError(t, err)
	assert.Equal(t, []string{"lock:" + skuA, "lock:" + skuZ}, lockOrder)
	expectedDBOrder := []string{
		"item:" + skuA,
		"item:" + skuZ,
		"batch:" + batchID1.String(),
		"batch:" + batchID2.String(),
	}
	assert.Equal(t, expectedDBOrder, dbLockOrder)
}

// Test CommitStockDeduction multi-SKU and multi-batch canonical lock hierarchy (Items -> Batches)
func TestCommitStockDeduction_MultiSKU_MultiBatch_CanonicalLockHierarchy(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockInventoryRepository)
	mockTxManager := new(MockTransactionManager)
	mockLock := new(MockLockService)

	orderID := "ORDER-MULTI-COMMIT"
	skuA := "SKU-AAA"
	skuZ := "SKU-ZZZ"

	batchID1 := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	batchID2 := uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff")

	res := &entity.StockReservation{
		ID:      uuid.New(),
		OrderID: orderID,
		Status:  entity.ReservationStatusPending,
		Allocations: []entity.ReservationItemAllocation{
			{SKU: skuZ, BatchID: batchID2, AllocatedQty: 15},
			{SKU: skuA, BatchID: batchID1, AllocatedQty: 10},
		},
	}

	mockRepo.On("GetReservationByOrderID", ctx, orderID).Return(res, nil)

	var lockOrder []string
	mockLock.On("AcquireLock", ctx, "lock:inventory:sku:"+skuA, 3*time.Second).Run(func(args mock.Arguments) {
		lockOrder = append(lockOrder, "lock:"+skuA)
	}).Return(true, nil)
	mockLock.On("AcquireLock", ctx, "lock:inventory:sku:"+skuZ, 3*time.Second).Run(func(args mock.Arguments) {
		lockOrder = append(lockOrder, "lock:"+skuZ)
	}).Return(true, nil)
	mockLock.On("ReleaseLock", mock.Anything, "lock:inventory:sku:"+skuA).Return(nil)
	mockLock.On("ReleaseLock", mock.Anything, "lock:inventory:sku:"+skuZ).Return(nil)

	mockTxManager.On("ExecuteInTransaction", ctx, mock.Anything).Return(nil)
	mockRepo.On("GetReservationByOrderIDForUpdate", ctx, mock.Anything, orderID).Return(res, nil)
	mockRepo.On("UpdateReservation", ctx, mock.Anything, mock.Anything).Return(nil)

	var dbLockOrder []string
	itemA := &entity.InventoryItem{SKU: skuA, PhysicalQty: 50, ReservedQty: 10}
	itemZ := &entity.InventoryItem{SKU: skuZ, PhysicalQty: 50, ReservedQty: 15}
	batch1 := &entity.Batch{ID: batchID1, SKU: skuA, PhysicalQty: 50, ReservedQty: 10}
	batch2 := &entity.Batch{ID: batchID2, SKU: skuZ, PhysicalQty: 50, ReservedQty: 15}

	mockRepo.On("GetItemBySKUForUpdate", ctx, mock.Anything, skuA).Run(func(args mock.Arguments) {
		dbLockOrder = append(dbLockOrder, "item:"+skuA)
	}).Return(itemA, nil)
	mockRepo.On("UpdateItem", ctx, mock.Anything, itemA).Return(nil)

	mockRepo.On("GetItemBySKUForUpdate", ctx, mock.Anything, skuZ).Run(func(args mock.Arguments) {
		dbLockOrder = append(dbLockOrder, "item:"+skuZ)
	}).Return(itemZ, nil)
	mockRepo.On("UpdateItem", ctx, mock.Anything, itemZ).Return(nil)

	mockRepo.On("GetBatchByID", ctx, mock.Anything, batchID1).Run(func(args mock.Arguments) {
		dbLockOrder = append(dbLockOrder, "batch:"+batchID1.String())
	}).Return(batch1, nil)
	mockRepo.On("UpdateBatch", ctx, mock.Anything, batch1).Return(nil)

	mockRepo.On("GetBatchByID", ctx, mock.Anything, batchID2).Run(func(args mock.Arguments) {
		dbLockOrder = append(dbLockOrder, "batch:"+batchID2.String())
	}).Return(batch2, nil)
	mockRepo.On("UpdateBatch", ctx, mock.Anything, batch2).Return(nil)

	uc := usecase.NewCommitStockDeductionUseCase(mockRepo, mockTxManager, mockLock)
	err := uc.Execute(ctx, orderID)

	require.NoError(t, err)
	assert.Equal(t, []string{"lock:" + skuA, "lock:" + skuZ}, lockOrder)
	expectedDBOrder := []string{
		"item:" + skuA,
		"item:" + skuZ,
		"batch:" + batchID1.String(),
		"batch:" + batchID2.String(),
	}
	assert.Equal(t, expectedDBOrder, dbLockOrder)
}

// Test ReleaseReservation with invalid inputs and errors
func TestReleaseReservation_InvalidStates(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockInventoryRepository)
	mockTxManager := new(MockTransactionManager)
	mockLock := new(MockLockService)

	uc := usecase.NewReleaseReservationUseCase(mockRepo, mockTxManager, mockLock)

	// Empty orderID
	assert.ErrorIs(t, uc.Execute(ctx, ""), entity.ErrInvalidBatchData)

	// Reservation not found
	mockRepo.On("GetReservationByOrderID", ctx, "ORD-NOT-FOUND").Return(nil, entity.ErrReservationNotFound).Once()
	assert.ErrorIs(t, uc.Execute(ctx, "ORD-NOT-FOUND"), entity.ErrReservationNotFound)

	// DB repository error
	dbErr := assert.AnError
	mockRepo.On("GetReservationByOrderID", ctx, "ORD-DB-ERR").Return(nil, dbErr).Once()
	assert.ErrorIs(t, uc.Execute(ctx, "ORD-DB-ERR"), dbErr)
}

// Test ReleaseReservation partial lock contention releases previously acquired locks
func TestReleaseReservation_PartialLockContention_ReleasesAcquiredLocks(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockInventoryRepository)
	mockTxManager := new(MockTransactionManager)
	mockLock := new(MockLockService)

	orderID := "ORDER-PARTIAL-LOCK-REL"
	skuA := "SKU-AAA"
	skuB := "SKU-BBB"

	res := &entity.StockReservation{
		ID:      uuid.New(),
		OrderID: orderID,
		Status:  entity.ReservationStatusPending,
		Allocations: []entity.ReservationItemAllocation{
			{SKU: skuA, BatchID: uuid.New(), AllocatedQty: 5},
			{SKU: skuB, BatchID: uuid.New(), AllocatedQty: 10},
		},
	}

	mockRepo.On("GetReservationByOrderID", ctx, orderID).Return(res, nil)
	// SKU-AAA lock succeeds
	mockLock.On("AcquireLock", ctx, "lock:inventory:sku:"+skuA, 3*time.Second).Return(true, nil).Once()
	// SKU-BBB lock fails (contention)
	mockLock.On("AcquireLock", ctx, "lock:inventory:sku:"+skuB, 3*time.Second).Return(false, nil).Once()

	// SKU-AAA lock MUST be released by defer
	mockLock.On("ReleaseLock", mock.Anything, "lock:inventory:sku:"+skuA).Return(nil).Once()

	uc := usecase.NewReleaseReservationUseCase(mockRepo, mockTxManager, mockLock)
	err := uc.Execute(ctx, orderID)

	assert.ErrorIs(t, err, entity.ErrConcurrentUpdate)
	mockLock.AssertExpectations(t)
	// Ensure DB transaction was NEVER entered
	mockTxManager.AssertNotCalled(t, "ExecuteInTransaction", mock.Anything, mock.Anything)
}

// Test CommitStockDeduction partial lock contention releases previously acquired locks
func TestCommitStockDeduction_PartialLockContention_ReleasesAcquiredLocks(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockInventoryRepository)
	mockTxManager := new(MockTransactionManager)
	mockLock := new(MockLockService)

	orderID := "ORDER-PARTIAL-LOCK-COMMIT"
	skuA := "SKU-AAA"
	skuB := "SKU-BBB"

	res := &entity.StockReservation{
		ID:      uuid.New(),
		OrderID: orderID,
		Status:  entity.ReservationStatusPending,
		Allocations: []entity.ReservationItemAllocation{
			{SKU: skuA, BatchID: uuid.New(), AllocatedQty: 5},
			{SKU: skuB, BatchID: uuid.New(), AllocatedQty: 10},
		},
	}

	mockRepo.On("GetReservationByOrderID", ctx, orderID).Return(res, nil)
	// SKU-AAA lock succeeds
	mockLock.On("AcquireLock", ctx, "lock:inventory:sku:"+skuA, 3*time.Second).Return(true, nil).Once()
	// SKU-BBB lock fails (contention)
	mockLock.On("AcquireLock", ctx, "lock:inventory:sku:"+skuB, 3*time.Second).Return(false, nil).Once()

	// SKU-AAA lock MUST be released by defer
	mockLock.On("ReleaseLock", mock.Anything, "lock:inventory:sku:"+skuA).Return(nil).Once()

	uc := usecase.NewCommitStockDeductionUseCase(mockRepo, mockTxManager, mockLock)
	err := uc.Execute(ctx, orderID)

	assert.ErrorIs(t, err, entity.ErrConcurrentUpdate)
	mockLock.AssertExpectations(t)
	// Ensure DB transaction was NEVER entered
	mockTxManager.AssertNotCalled(t, "ExecuteInTransaction", mock.Anything, mock.Anything)
}

// Test ReleaseReservation with single SKU distributed across multiple batches
func TestReleaseReservation_SameSKU_MultipleBatches(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockInventoryRepository)
	mockTxManager := new(MockTransactionManager)
	mockLock := new(MockLockService)

	orderID := "ORDER-SAME-SKU-MULTI-BATCH-REL"
	sku := "SKU-OC-MEXUNG-001"
	batchID1 := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	batchID2 := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	// 2 allocations for same SKU across 2 different batches (10 + 20 = 30 total)
	res := &entity.StockReservation{
		ID:      uuid.New(),
		OrderID: orderID,
		Status:  entity.ReservationStatusPending,
		Allocations: []entity.ReservationItemAllocation{
			{SKU: sku, BatchID: batchID2, AllocatedQty: 20},
			{SKU: sku, BatchID: batchID1, AllocatedQty: 10},
		},
	}

	item := &entity.InventoryItem{SKU: sku, PhysicalQty: 100, ReservedQty: 30, Status: entity.ItemStatusActive}
	batch1 := &entity.Batch{ID: batchID1, SKU: sku, PhysicalQty: 50, ReservedQty: 10, Status: entity.BatchStatusActive}
	batch2 := &entity.Batch{ID: batchID2, SKU: sku, PhysicalQty: 50, ReservedQty: 20, Status: entity.BatchStatusActive}

	mockRepo.On("GetReservationByOrderID", ctx, orderID).Return(res, nil)
	mockLock.On("AcquireLock", ctx, "lock:inventory:sku:"+sku, 3*time.Second).Return(true, nil).Once()
	mockLock.On("ReleaseLock", mock.Anything, "lock:inventory:sku:"+sku).Return(nil).Once()

	mockTxManager.On("ExecuteInTransaction", ctx, mock.Anything).Return(nil)
	mockRepo.On("GetReservationByOrderIDForUpdate", ctx, mock.Anything, orderID).Return(res, nil)
	mockRepo.On("UpdateReservation", ctx, mock.Anything, mock.MatchedBy(func(r *entity.StockReservation) bool {
		return r.Status == entity.ReservationStatusReleased
	})).Return(nil)

	// Item must be locked and updated ONCE with aggregated 30 units released
	mockRepo.On("GetItemBySKUForUpdate", ctx, mock.Anything, sku).Return(item, nil).Once()
	mockRepo.On("UpdateItem", ctx, mock.Anything, mock.MatchedBy(func(i *entity.InventoryItem) bool {
		return i.ReservedQty == 0
	})).Return(nil).Once()

	// Batches must be updated in canonical order (batchID1 then batchID2)
	var batchUpdateOrder []uuid.UUID
	mockRepo.On("GetBatchByID", ctx, mock.Anything, batchID1).Run(func(args mock.Arguments) {
		batchUpdateOrder = append(batchUpdateOrder, batchID1)
	}).Return(batch1, nil).Once()
	mockRepo.On("UpdateBatch", ctx, mock.Anything, mock.MatchedBy(func(b *entity.Batch) bool {
		return b.ID == batchID1 && b.ReservedQty == 0
	})).Return(nil).Once()

	mockRepo.On("GetBatchByID", ctx, mock.Anything, batchID2).Run(func(args mock.Arguments) {
		batchUpdateOrder = append(batchUpdateOrder, batchID2)
	}).Return(batch2, nil).Once()
	mockRepo.On("UpdateBatch", ctx, mock.Anything, mock.MatchedBy(func(b *entity.Batch) bool {
		return b.ID == batchID2 && b.ReservedQty == 0
	})).Return(nil).Once()

	uc := usecase.NewReleaseReservationUseCase(mockRepo, mockTxManager, mockLock)
	err := uc.Execute(ctx, orderID)

	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{batchID1, batchID2}, batchUpdateOrder)
	assert.Equal(t, 0, item.ReservedQty)
	assert.Equal(t, 0, batch1.ReservedQty)
	assert.Equal(t, 0, batch2.ReservedQty)
	mockLock.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

// Test CommitStockDeduction with single SKU distributed across multiple batches
func TestCommitStockDeduction_SameSKU_MultipleBatches(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockInventoryRepository)
	mockTxManager := new(MockTransactionManager)
	mockLock := new(MockLockService)

	orderID := "ORDER-SAME-SKU-MULTI-BATCH-COMMIT"
	sku := "SKU-OC-MEXUNG-001"
	batchID1 := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	batchID2 := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	// 2 allocations for same SKU across 2 different batches (10 + 20 = 30 total)
	res := &entity.StockReservation{
		ID:      uuid.New(),
		OrderID: orderID,
		Status:  entity.ReservationStatusPending,
		Allocations: []entity.ReservationItemAllocation{
			{SKU: sku, BatchID: batchID2, AllocatedQty: 20},
			{SKU: sku, BatchID: batchID1, AllocatedQty: 10},
		},
	}

	item := &entity.InventoryItem{SKU: sku, PhysicalQty: 100, ReservedQty: 30, Status: entity.ItemStatusActive}
	batch1 := &entity.Batch{ID: batchID1, SKU: sku, PhysicalQty: 50, ReservedQty: 10, Status: entity.BatchStatusActive}
	batch2 := &entity.Batch{ID: batchID2, SKU: sku, PhysicalQty: 50, ReservedQty: 20, Status: entity.BatchStatusActive}

	mockRepo.On("GetReservationByOrderID", ctx, orderID).Return(res, nil)
	mockLock.On("AcquireLock", ctx, "lock:inventory:sku:"+sku, 3*time.Second).Return(true, nil).Once()
	mockLock.On("ReleaseLock", mock.Anything, "lock:inventory:sku:"+sku).Return(nil).Once()

	mockTxManager.On("ExecuteInTransaction", ctx, mock.Anything).Return(nil)
	mockRepo.On("GetReservationByOrderIDForUpdate", ctx, mock.Anything, orderID).Return(res, nil)
	mockRepo.On("UpdateReservation", ctx, mock.Anything, mock.MatchedBy(func(r *entity.StockReservation) bool {
		return r.Status == entity.ReservationStatusCommitted
	})).Return(nil)

	// Item must be locked and committed ONCE with aggregated 30 units deducted
	mockRepo.On("GetItemBySKUForUpdate", ctx, mock.Anything, sku).Return(item, nil).Once()
	mockRepo.On("UpdateItem", ctx, mock.Anything, mock.MatchedBy(func(i *entity.InventoryItem) bool {
		return i.PhysicalQty == 70 && i.ReservedQty == 0
	})).Return(nil).Once()

	// Batches must be committed in canonical order (batchID1 then batchID2)
	var batchUpdateOrder []uuid.UUID
	mockRepo.On("GetBatchByID", ctx, mock.Anything, batchID1).Run(func(args mock.Arguments) {
		batchUpdateOrder = append(batchUpdateOrder, batchID1)
	}).Return(batch1, nil).Once()
	mockRepo.On("UpdateBatch", ctx, mock.Anything, mock.MatchedBy(func(b *entity.Batch) bool {
		return b.ID == batchID1 && b.PhysicalQty == 40 && b.ReservedQty == 0
	})).Return(nil).Once()

	mockRepo.On("GetBatchByID", ctx, mock.Anything, batchID2).Run(func(args mock.Arguments) {
		batchUpdateOrder = append(batchUpdateOrder, batchID2)
	}).Return(batch2, nil).Once()
	mockRepo.On("UpdateBatch", ctx, mock.Anything, mock.MatchedBy(func(b *entity.Batch) bool {
		return b.ID == batchID2 && b.PhysicalQty == 30 && b.ReservedQty == 0
	})).Return(nil).Once()

	uc := usecase.NewCommitStockDeductionUseCase(mockRepo, mockTxManager, mockLock)
	err := uc.Execute(ctx, orderID)

	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{batchID1, batchID2}, batchUpdateOrder)
	assert.Equal(t, 70, item.PhysicalQty)
	assert.Equal(t, 0, item.ReservedQty)
	assert.Equal(t, 40, batch1.PhysicalQty)
	assert.Equal(t, 0, batch1.ReservedQty)
	assert.Equal(t, 30, batch2.PhysicalQty)
	assert.Equal(t, 0, batch2.ReservedQty)
	mockLock.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

// Tests demonstrating that when request context is canceled/timed-out upfront,
// the use cases still release locks via context.WithoutCancel(ctx) without blocking Redis.
func TestReserveStock_ContextCancellation_ReleasesLockSafely(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	type contextKey string
	const traceKey contextKey = "trace_id"
	ctx = context.WithValue(ctx, traceKey, "trace-req-cancelled-1234")

	mockRepo := new(MockInventoryRepository)
	mockTxManager := new(MockTransactionManager)
	mockLock := new(MockLockService)

	orderID := "ORDER-CTX-CANCELLED-001"
	sku := "SKU-CTX-TEST-001"
	lockKey := "lock:inventory:sku:" + sku

	mockRepo.On("GetReservationByOrderID", ctx, orderID).Return(nil, entity.ErrReservationNotFound)
	mockLock.On("AcquireLock", ctx, lockKey, 3*time.Second).Return(true, nil).Once()

	var errDuringRelease error
	var hasDeadlineDuringRelease bool
	mockLock.On("ReleaseLock", mock.MatchedBy(func(c context.Context) bool {
		return c != nil && c.Value(traceKey) == "trace-req-cancelled-1234"
	}), lockKey).Run(func(args mock.Arguments) {
		c := args.Get(0).(context.Context)
		errDuringRelease = c.Err()
		_, hasDeadlineDuringRelease = c.Deadline()
	}).Return(nil).Once()

	mockTxManager.On("ExecuteInTransaction", ctx, mock.Anything).Return(context.Canceled)

	uc := usecase.NewReserveStockUseCase(mockRepo, mockTxManager, mockLock)
	res, err := uc.Execute(ctx, orderID, []usecase.ReserveItemInput{
		{SKU: sku, Qty: 10},
	})

	assert.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, res)

	mockLock.AssertExpectations(t)
	assert.NoError(t, errDuringRelease, "cleanup context must not be canceled during ReleaseLock even when parent was canceled")
	assert.True(t, hasDeadlineDuringRelease, "cleanup context must have a timeout deadline during ReleaseLock")
}

func TestReleaseReservation_ContextCancellation_ReleasesLockSafely(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	type contextKey string
	const traceKey contextKey = "trace_id"
	ctx = context.WithValue(ctx, traceKey, "trace-release-cancelled-5678")

	mockRepo := new(MockInventoryRepository)
	mockTxManager := new(MockTransactionManager)
	mockLock := new(MockLockService)

	orderID := "ORDER-RELEASE-CTX-CANCEL"
	sku := "SKU-RELEASE-001"
	lockKey := "lock:inventory:sku:" + sku

	res := &entity.StockReservation{
		ID:      uuid.New(),
		OrderID: orderID,
		Status:  entity.ReservationStatusPending,
		Allocations: []entity.ReservationItemAllocation{
			{SKU: sku, BatchID: uuid.New(), AllocatedQty: 10},
		},
	}

	mockRepo.On("GetReservationByOrderID", ctx, orderID).Return(res, nil)
	mockLock.On("AcquireLock", ctx, lockKey, 3*time.Second).Return(true, nil).Once()

	var errDuringRelease error
	var hasDeadlineDuringRelease bool
	mockLock.On("ReleaseLock", mock.MatchedBy(func(c context.Context) bool {
		return c != nil && c.Value(traceKey) == "trace-release-cancelled-5678"
	}), lockKey).Run(func(args mock.Arguments) {
		c := args.Get(0).(context.Context)
		errDuringRelease = c.Err()
		_, hasDeadlineDuringRelease = c.Deadline()
	}).Return(nil).Once()

	mockTxManager.On("ExecuteInTransaction", ctx, mock.Anything).Return(context.Canceled)

	uc := usecase.NewReleaseReservationUseCase(mockRepo, mockTxManager, mockLock)
	err := uc.Execute(ctx, orderID)

	assert.ErrorIs(t, err, context.Canceled)
	mockLock.AssertExpectations(t)
	assert.NoError(t, errDuringRelease, "cleanup context must not be canceled during ReleaseLock")
	assert.True(t, hasDeadlineDuringRelease)
}

func TestCommitStockDeduction_ContextCancellation_ReleasesLockSafely(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	type contextKey string
	const traceKey contextKey = "trace_id"
	ctx = context.WithValue(ctx, traceKey, "trace-commit-cancelled-9999")

	mockRepo := new(MockInventoryRepository)
	mockTxManager := new(MockTransactionManager)
	mockLock := new(MockLockService)

	orderID := "ORDER-COMMIT-CTX-CANCEL"
	sku := "SKU-COMMIT-001"
	lockKey := "lock:inventory:sku:" + sku

	res := &entity.StockReservation{
		ID:      uuid.New(),
		OrderID: orderID,
		Status:  entity.ReservationStatusPending,
		Allocations: []entity.ReservationItemAllocation{
			{SKU: sku, BatchID: uuid.New(), AllocatedQty: 10},
		},
	}

	mockRepo.On("GetReservationByOrderID", ctx, orderID).Return(res, nil)
	mockLock.On("AcquireLock", ctx, lockKey, 3*time.Second).Return(true, nil).Once()

	var errDuringRelease error
	var hasDeadlineDuringRelease bool
	mockLock.On("ReleaseLock", mock.MatchedBy(func(c context.Context) bool {
		return c != nil && c.Value(traceKey) == "trace-commit-cancelled-9999"
	}), lockKey).Run(func(args mock.Arguments) {
		c := args.Get(0).(context.Context)
		errDuringRelease = c.Err()
		_, hasDeadlineDuringRelease = c.Deadline()
	}).Return(nil).Once()

	mockTxManager.On("ExecuteInTransaction", ctx, mock.Anything).Return(context.Canceled)

	uc := usecase.NewCommitStockDeductionUseCase(mockRepo, mockTxManager, mockLock)
	err := uc.Execute(ctx, orderID)

	assert.ErrorIs(t, err, context.Canceled)
	mockLock.AssertExpectations(t)
	assert.NoError(t, errDuringRelease, "cleanup context must not be canceled during ReleaseLock")
	assert.True(t, hasDeadlineDuringRelease)
}

func TestReserveStock_NilContext_DoesNotPanic(t *testing.T) {
	mockRepo := new(MockInventoryRepository)
	mockTxManager := new(MockTransactionManager)
	mockLock := new(MockLockService)

	orderID := "ORD-NIL-CTX-01"
	sku := "SKU-NIL-CTX-01"
	lockKey := "lock:inventory:sku:" + sku

	mockRepo.On("GetReservationByOrderID", mock.Anything, orderID).Return(nil, entity.ErrReservationNotFound)
	mockLock.On("AcquireLock", mock.Anything, lockKey, 3*time.Second).Return(true, nil).Once()
	mockLock.On("ReleaseLock", mock.Anything, lockKey).Return(nil).Once()
	mockTxManager.On("ExecuteInTransaction", mock.Anything, mock.Anything).Return(errors.New("tx simulated stop"))

	uc := usecase.NewReserveStockUseCase(mockRepo, mockTxManager, mockLock)
	assert.NotPanics(t, func() {
		res, err := uc.Execute(nil, orderID, []usecase.ReserveItemInput{
			{SKU: sku, Qty: 5},
		})
		assert.Error(t, err)
		assert.Nil(t, res)
	})

	mockLock.AssertExpectations(t)
}

func TestReleaseReservation_NilContext_DoesNotPanic(t *testing.T) {
	mockRepo := new(MockInventoryRepository)
	mockTxManager := new(MockTransactionManager)
	mockLock := new(MockLockService)

	orderID := "ORD-RELEASE-NIL-CTX"
	sku := "SKU-RELEASE-NIL"
	lockKey := "lock:inventory:sku:" + sku

	res := &entity.StockReservation{
		ID:      uuid.New(),
		OrderID: orderID,
		Status:  entity.ReservationStatusPending,
		Allocations: []entity.ReservationItemAllocation{
			{SKU: sku, BatchID: uuid.New(), AllocatedQty: 5},
		},
	}

	mockRepo.On("GetReservationByOrderID", mock.Anything, orderID).Return(res, nil)
	mockLock.On("AcquireLock", mock.Anything, lockKey, 3*time.Second).Return(true, nil).Once()
	mockLock.On("ReleaseLock", mock.Anything, lockKey).Return(nil).Once()
	mockTxManager.On("ExecuteInTransaction", mock.Anything, mock.Anything).Return(errors.New("tx stop"))

	uc := usecase.NewReleaseReservationUseCase(mockRepo, mockTxManager, mockLock)
	assert.NotPanics(t, func() {
		err := uc.Execute(nil, orderID)
		assert.Error(t, err)
	})

	mockLock.AssertExpectations(t)
}

func TestCommitStockDeduction_NilContext_DoesNotPanic(t *testing.T) {
	mockRepo := new(MockInventoryRepository)
	mockTxManager := new(MockTransactionManager)
	mockLock := new(MockLockService)

	orderID := "ORD-COMMIT-NIL-CTX"
	sku := "SKU-COMMIT-NIL"
	lockKey := "lock:inventory:sku:" + sku

	res := &entity.StockReservation{
		ID:      uuid.New(),
		OrderID: orderID,
		Status:  entity.ReservationStatusPending,
		Allocations: []entity.ReservationItemAllocation{
			{SKU: sku, BatchID: uuid.New(), AllocatedQty: 5},
		},
	}

	mockRepo.On("GetReservationByOrderID", mock.Anything, orderID).Return(res, nil)
	mockLock.On("AcquireLock", mock.Anything, lockKey, 3*time.Second).Return(true, nil).Once()
	mockLock.On("ReleaseLock", mock.Anything, lockKey).Return(nil).Once()
	mockTxManager.On("ExecuteInTransaction", mock.Anything, mock.Anything).Return(errors.New("tx stop"))

	uc := usecase.NewCommitStockDeductionUseCase(mockRepo, mockTxManager, mockLock)
	assert.NotPanics(t, func() {
		err := uc.Execute(nil, orderID)
		assert.Error(t, err)
	})

	mockLock.AssertExpectations(t)
}

func TestReserveStock_GetReservationDBError(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockInventoryRepository)
	mockTxManager := new(MockTransactionManager)
	mockLock := new(MockLockService)

	orderID := "ORDER-DB-ERR-001"
	sku := "SKU-TEST-001"

	dbErr := errors.New("db connection timeout")
	mockRepo.On("GetReservationByOrderID", ctx, orderID).Return(nil, dbErr)

	uc := usecase.NewReserveStockUseCase(mockRepo, mockTxManager, mockLock)
	res, err := uc.Execute(ctx, orderID, []usecase.ReserveItemInput{
		{SKU: sku, Qty: 10},
	})

	assert.ErrorIs(t, err, dbErr)
	assert.Nil(t, res)

	// Locks must NOT be acquired if DB check failed with an unexpected error
	mockLock.AssertNotCalled(t, "AcquireLock", mock.Anything, mock.Anything, mock.Anything)
	mockTxManager.AssertNotCalled(t, "ExecuteInTransaction", mock.Anything, mock.Anything)
}



