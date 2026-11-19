package grpc_test

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"dut-pbl6/inventory-service/internal/application/port"
	"dut-pbl6/inventory-service/internal/application/usecase"
	"dut-pbl6/inventory-service/internal/domain/entity"
	presentationgrpc "dut-pbl6/inventory-service/internal/presentation/grpc"
	commonv1 "dut-pbl6/inventory-service/pkg/proto/common/v1"
	inventoryv1 "dut-pbl6/inventory-service/pkg/proto/inventory/v1"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

// --- Mocks ---

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
	return fn(nil)
}

type MockLockService struct {
	mock.Mock
}

func (m *MockLockService) AcquireLock(ctx context.Context, resource string, ttl time.Duration) (bool, error) {
	args := m.Called(ctx, resource, ttl)
	return args.Bool(0), args.Error(1)
}

func (m *MockLockService) ReleaseLock(ctx context.Context, resource string) error {
	args := m.Called(ctx, resource)
	return args.Error(0)
}

type testFixture struct {
	mockRepo    *MockInventoryRepository
	mockTxMgr   *MockTransactionManager
	mockLock    *MockLockService
	handler     *presentationgrpc.InventoryGRPCHandler
}

func setupFixture() *testFixture {
	mockRepo := new(MockInventoryRepository)
	mockTxMgr := new(MockTransactionManager)
	mockLock := new(MockLockService)

	reserveUC := usecase.NewReserveStockUseCase(mockRepo, mockTxMgr, mockLock)
	releaseUC := usecase.NewReleaseReservationUseCase(mockRepo, mockTxMgr, mockLock)
	stockLevelUC := usecase.NewGetStockLevelUseCase(mockRepo)
	batchFEFOUC := usecase.NewGetBatchFEFODetailsUseCase(mockRepo)

	handler := presentationgrpc.NewInventoryGRPCHandler(reserveUC, releaseUC, stockLevelUC, batchFEFOUC)

	return &testFixture{
		mockRepo:  mockRepo,
		mockTxMgr: mockTxMgr,
		mockLock:  mockLock,
		handler:   handler,
	}
}

func extractErrorDetail(t *testing.T, err error) *commonv1.ErrorDetail {
	st, ok := status.FromError(err)
	require.True(t, ok, "error should be a gRPC status error")
	for _, detail := range st.Details() {
		if ed, ok := detail.(*commonv1.ErrorDetail); ok {
			return ed
		}
	}
	return nil
}

// --- 1. ReserveStock Tests ---

func TestInventoryGRPCHandler_ReserveStock(t *testing.T) {
	ctx := context.Background()

	t.Run("HappyPath", func(t *testing.T) {
		f := setupFixture()
		orderID := "ORD-GRPC-001"
		sku := "MX-GION-500G"
		reqQty := int32(5)

		item := &entity.InventoryItem{
			SKU:         sku,
			PhysicalQty: 100,
			ReservedQty: 0,
			Status:      entity.ItemStatusActive,
		}
		now := time.Now().UTC()
		batch := &entity.Batch{
			ID:          uuid.New(),
			BatchCode:   "LOT-01",
			SKU:         sku,
			PhysicalQty: 100,
			ReservedQty: 0,
			Status:      entity.BatchStatusActive,
			ExpDate:     now.Add(30 * 24 * time.Hour),
		}

		f.mockRepo.On("GetReservationByOrderID", mock.Anything, orderID).Return(nil, nil)
		f.mockLock.On("AcquireLock", mock.Anything, "lock:inventory:sku:"+sku, mock.Anything).Return(true, nil)
		f.mockLock.On("ReleaseLock", mock.Anything, "lock:inventory:sku:"+sku).Return(nil)
		f.mockTxMgr.On("ExecuteInTransaction", mock.Anything, mock.Anything).Return(nil)
		f.mockRepo.On("GetItemBySKUForUpdate", mock.Anything, mock.Anything, sku).Return(item, nil)
		f.mockRepo.On("GetActiveBatchesBySKUForUpdate", mock.Anything, mock.Anything, sku).Return([]*entity.Batch{batch}, nil)
		f.mockRepo.On("UpdateItem", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		f.mockRepo.On("UpdateBatch", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		f.mockRepo.On("CreateReservation", mock.Anything, mock.Anything, mock.Anything).Return(nil)

		req := &inventoryv1.ReserveStockRequest{
			OrderId: orderID,
			Items: []*inventoryv1.ReserveItem{
				{SkuCode: sku, Quantity: reqQty},
			},
		}

		resp, err := f.handler.ReserveStock(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, inventoryv1.ReservationStatus_RESERVATION_STATUS_SUCCESS, resp.Status)
		assert.NotEmpty(t, resp.GetReservationId())
		assert.NotNil(t, resp.GetExpiresAt())
	})

	t.Run("InputValidationError_NilRequest", func(t *testing.T) {
		f := setupFixture()
		resp, err := f.handler.ReserveStock(ctx, nil)
		assert.Nil(t, resp)
		require.Error(t, err)

		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())

		ed := extractErrorDetail(t, err)
		require.NotNil(t, ed)
		assert.Equal(t, commonv1.ErrorCode_ERR_INVALID_ARGUMENT, ed.ErrorCode)
		assert.Equal(t, "inventory", ed.Domain)
	})

	t.Run("InputValidationError_EmptyOrderID", func(t *testing.T) {
		f := setupFixture()
		req := &inventoryv1.ReserveStockRequest{
			OrderId: "",
			Items:   []*inventoryv1.ReserveItem{{SkuCode: "SKU-01", Quantity: 5}},
		}
		resp, err := f.handler.ReserveStock(ctx, req)
		assert.Nil(t, resp)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())

		reqWhitespace := &inventoryv1.ReserveStockRequest{
			OrderId: "   ",
			Items:   []*inventoryv1.ReserveItem{{SkuCode: "SKU-01", Quantity: 5}},
		}
		resp, err = f.handler.ReserveStock(ctx, reqWhitespace)
		assert.Nil(t, resp)
		st, ok = status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("InputValidationError_EmptyItems", func(t *testing.T) {
		f := setupFixture()
		req := &inventoryv1.ReserveStockRequest{
			OrderId: "ORD-001",
			Items:   []*inventoryv1.ReserveItem{},
		}
		resp, err := f.handler.ReserveStock(ctx, req)
		assert.Nil(t, resp)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("InputValidationError_InvalidQuantityOrSKU", func(t *testing.T) {
		f := setupFixture()
		req := &inventoryv1.ReserveStockRequest{
			OrderId: "ORD-001",
			Items:   []*inventoryv1.ReserveItem{{SkuCode: "SKU-01", Quantity: -1}},
		}
		resp, err := f.handler.ReserveStock(ctx, req)
		assert.Nil(t, resp)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())

		reqEmptySKU := &inventoryv1.ReserveStockRequest{
			OrderId: "ORD-001",
			Items:   []*inventoryv1.ReserveItem{{SkuCode: "", Quantity: 10}},
		}
		resp, err = f.handler.ReserveStock(ctx, reqEmptySKU)
		assert.Nil(t, resp)
		st, ok = status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())

		reqWhitespaceSKU := &inventoryv1.ReserveStockRequest{
			OrderId: "ORD-001",
			Items:   []*inventoryv1.ReserveItem{{SkuCode: "   ", Quantity: 10}},
		}
		resp, err = f.handler.ReserveStock(ctx, reqWhitespaceSKU)
		assert.Nil(t, resp)
		st, ok = status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("InsufficientStockError", func(t *testing.T) {
		f := setupFixture()
		orderID := "ORD-GRPC-INSUFFICIENT"
		sku := "MX-GION-500G"

		item := &entity.InventoryItem{
			SKU:         sku,
			PhysicalQty: 2, // only 2 available
			ReservedQty: 0,
			Status:      entity.ItemStatusActive,
		}

		f.mockRepo.On("GetReservationByOrderID", mock.Anything, orderID).Return(nil, nil)
		f.mockLock.On("AcquireLock", mock.Anything, "lock:inventory:sku:"+sku, mock.Anything).Return(true, nil)
		f.mockLock.On("ReleaseLock", mock.Anything, "lock:inventory:sku:"+sku).Return(nil)
		f.mockTxMgr.On("ExecuteInTransaction", mock.Anything, mock.Anything).Return(nil)
		f.mockRepo.On("GetItemBySKUForUpdate", mock.Anything, mock.Anything, sku).Return(item, nil)

		req := &inventoryv1.ReserveStockRequest{
			OrderId: orderID,
			Items: []*inventoryv1.ReserveItem{
				{SkuCode: sku, Quantity: 10}, // Requesting 10
			},
		}

		resp, err := f.handler.ReserveStock(ctx, req)
		assert.Nil(t, resp)
		require.Error(t, err)

		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.ResourceExhausted, st.Code())

		ed := extractErrorDetail(t, err)
		require.NotNil(t, ed)
		assert.Equal(t, commonv1.ErrorCode_ERR_INVENTORY_INSUFFICIENT_STOCK, ed.ErrorCode)
		assert.Equal(t, "inventory", ed.Domain)
	})

	t.Run("NotFoundError_SKU", func(t *testing.T) {
		f := setupFixture()
		orderID := "ORD-NOT-FOUND"
		sku := "SKU-NON-EXISTENT"

		f.mockRepo.On("GetReservationByOrderID", mock.Anything, orderID).Return(nil, nil)
		f.mockLock.On("AcquireLock", mock.Anything, "lock:inventory:sku:"+sku, mock.Anything).Return(true, nil)
		f.mockLock.On("ReleaseLock", mock.Anything, "lock:inventory:sku:"+sku).Return(nil)
		f.mockTxMgr.On("ExecuteInTransaction", mock.Anything, mock.Anything).Return(nil)
		f.mockRepo.On("GetItemBySKUForUpdate", mock.Anything, mock.Anything, sku).Return(nil, entity.ErrItemNotFound)

		req := &inventoryv1.ReserveStockRequest{
			OrderId: orderID,
			Items:   []*inventoryv1.ReserveItem{{SkuCode: sku, Quantity: 5}},
		}

		resp, err := f.handler.ReserveStock(ctx, req)
		assert.Nil(t, resp)
		require.Error(t, err)

		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.NotFound, st.Code())

		ed := extractErrorDetail(t, err)
		require.NotNil(t, ed)
		assert.Equal(t, commonv1.ErrorCode_ERR_INVENTORY_SKU_NOT_FOUND, ed.ErrorCode)
	})

	t.Run("InternalError", func(t *testing.T) {
		f := setupFixture()
		orderID := "ORD-INTERNAL-ERR"
		sku := "SKU-INTERNAL"

		f.mockRepo.On("GetReservationByOrderID", mock.Anything, orderID).Return(nil, errors.New("db pool exhausted"))

		req := &inventoryv1.ReserveStockRequest{
			OrderId: orderID,
			Items:   []*inventoryv1.ReserveItem{{SkuCode: sku, Quantity: 5}},
		}

		resp, err := f.handler.ReserveStock(ctx, req)
		assert.Nil(t, resp)
		require.Error(t, err)

		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.Internal, st.Code())
	})
}

// --- 2. ReleaseReservation Tests ---

func TestInventoryGRPCHandler_ReleaseReservation(t *testing.T) {
	ctx := context.Background()

	t.Run("HappyPath", func(t *testing.T) {
		f := setupFixture()
		orderID := "ORD-RELEASE-001"
		sku := "MX-GION-500G"
		batchID := uuid.New()

		alloc := entity.ReservationItemAllocation{
			ID:           uuid.New(),
			SKU:          sku,
			BatchID:      batchID,
			AllocatedQty: 5,
		}
		res, err := entity.NewStockReservation(orderID, entity.DefaultReservationTTL, []entity.ReservationItemAllocation{alloc})
		require.NoError(t, err)

		item := &entity.InventoryItem{SKU: sku, PhysicalQty: 100, ReservedQty: 5, Status: entity.ItemStatusActive}
		batch := &entity.Batch{ID: batchID, SKU: sku, PhysicalQty: 100, ReservedQty: 5, Status: entity.BatchStatusActive}

		f.mockRepo.On("GetReservationByOrderID", mock.Anything, orderID).Return(res, nil)
		f.mockLock.On("AcquireLock", mock.Anything, "lock:inventory:sku:"+sku, mock.Anything).Return(true, nil)
		f.mockLock.On("ReleaseLock", mock.Anything, "lock:inventory:sku:"+sku).Return(nil)
		f.mockTxMgr.On("ExecuteInTransaction", mock.Anything, mock.Anything).Return(nil)
		f.mockRepo.On("GetReservationByOrderIDForUpdate", mock.Anything, mock.Anything, orderID).Return(res, nil)
		f.mockRepo.On("GetItemBySKUForUpdate", mock.Anything, mock.Anything, sku).Return(item, nil)
		f.mockRepo.On("UpdateItem", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		f.mockRepo.On("GetBatchByID", mock.Anything, mock.Anything, batchID).Return(batch, nil)
		f.mockRepo.On("UpdateBatch", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		f.mockRepo.On("UpdateReservation", mock.Anything, mock.Anything, mock.Anything).Return(nil)

		req := &inventoryv1.ReleaseReservationRequest{
			OrderId:       orderID,
			ReleaseReason: "ORDER_CANCELLED",
		}

		resp, err := f.handler.ReleaseReservation(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.True(t, resp.IsReleased)
		assert.Equal(t, int32(5), resp.TotalItemsRestored)
		assert.NotEmpty(t, resp.Message)
	})

	t.Run("HappyPath_ReservationIDOnly", func(t *testing.T) {
		f := setupFixture()
		orderID := "ORD-RELEASE-002"
		resID := uuid.New()
		sku := "MX-GION-500G"
		batchID := uuid.New()

		alloc := entity.ReservationItemAllocation{
			ID:           uuid.New(),
			SKU:          sku,
			BatchID:      batchID,
			AllocatedQty: 8,
		}
		res, err := entity.NewStockReservation(orderID, entity.DefaultReservationTTL, []entity.ReservationItemAllocation{alloc})
		require.NoError(t, err)
		res.ID = resID

		item := &entity.InventoryItem{SKU: sku, PhysicalQty: 100, ReservedQty: 8, Status: entity.ItemStatusActive}
		batch := &entity.Batch{ID: batchID, SKU: sku, PhysicalQty: 100, ReservedQty: 8, Status: entity.BatchStatusActive}

		f.mockRepo.On("GetReservationByID", mock.Anything, resID).Return(res, nil)
		f.mockLock.On("AcquireLock", mock.Anything, "lock:inventory:sku:"+sku, mock.Anything).Return(true, nil)
		f.mockLock.On("ReleaseLock", mock.Anything, "lock:inventory:sku:"+sku).Return(nil)
		f.mockTxMgr.On("ExecuteInTransaction", mock.Anything, mock.Anything).Return(nil)
		f.mockRepo.On("GetReservationByOrderIDForUpdate", mock.Anything, mock.Anything, orderID).Return(res, nil)
		f.mockRepo.On("GetItemBySKUForUpdate", mock.Anything, mock.Anything, sku).Return(item, nil)
		f.mockRepo.On("UpdateItem", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		f.mockRepo.On("GetBatchByID", mock.Anything, mock.Anything, batchID).Return(batch, nil)
		f.mockRepo.On("UpdateBatch", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		f.mockRepo.On("UpdateReservation", mock.Anything, mock.Anything, mock.Anything).Return(nil)

		req := &inventoryv1.ReleaseReservationRequest{
			ReservationId: resID.String(),
			ReleaseReason: "ORDER_CANCELLED",
		}

		resp, err := f.handler.ReleaseReservation(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.True(t, resp.IsReleased)
		assert.Equal(t, int32(8), resp.TotalItemsRestored)
	})

	t.Run("InputValidationError_NilRequest", func(t *testing.T) {
		f := setupFixture()
		resp, err := f.handler.ReleaseReservation(ctx, nil)
		assert.Nil(t, resp)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("InputValidationError_MissingIdentifiers", func(t *testing.T) {
		f := setupFixture()
		req := &inventoryv1.ReleaseReservationRequest{
			OrderId:       "",
			ReservationId: "",
		}
		resp, err := f.handler.ReleaseReservation(ctx, req)
		assert.Nil(t, resp)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("NotFoundError_Reservation", func(t *testing.T) {
		f := setupFixture()
		orderID := "ORD-NON-EXISTENT"

		f.mockRepo.On("GetReservationByOrderID", mock.Anything, orderID).Return(nil, entity.ErrReservationNotFound)

		req := &inventoryv1.ReleaseReservationRequest{OrderId: orderID}
		resp, err := f.handler.ReleaseReservation(ctx, req)
		assert.Nil(t, resp)
		require.Error(t, err)

		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.NotFound, st.Code())

		ed := extractErrorDetail(t, err)
		require.NotNil(t, ed)
		assert.Equal(t, commonv1.ErrorCode_ERR_INVENTORY_RESERVATION_NOT_FOUND, ed.ErrorCode)
	})

	t.Run("InternalError", func(t *testing.T) {
		f := setupFixture()
		orderID := "ORD-INTERNAL-ERR"

		f.mockRepo.On("GetReservationByOrderID", mock.Anything, orderID).Return(nil, errors.New("db error"))

		req := &inventoryv1.ReleaseReservationRequest{OrderId: orderID}
		resp, err := f.handler.ReleaseReservation(ctx, req)
		assert.Nil(t, resp)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.Internal, st.Code())
	})
}

// --- 3. GetStockLevel Tests ---

func TestInventoryGRPCHandler_GetStockLevel(t *testing.T) {
	ctx := context.Background()

	t.Run("HappyPath", func(t *testing.T) {
		f := setupFixture()
		skus := []string{"MX-GION-500G", "MX-DEO-300G"}

		item1 := &entity.InventoryItem{
			SKU:         "MX-GION-500G",
			PhysicalQty: 100,
			ReservedQty: 20,
			Status:      entity.ItemStatusActive,
		}
		item2 := &entity.InventoryItem{
			SKU:         "MX-DEO-300G",
			PhysicalQty: 8,
			ReservedQty: 0,
			Status:      entity.ItemStatusActive,
		}

		f.mockRepo.On("GetItemsBySKUs", mock.Anything, skus).
			Return([]*entity.InventoryItem{item1, item2}, nil)

		req := &inventoryv1.GetStockLevelRequest{
			SkuCodes: skus,
		}

		resp, err := f.handler.GetStockLevel(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Len(t, resp.Items, 2)

		assert.Equal(t, "MX-GION-500G", resp.Items[0].SkuCode)
		assert.Equal(t, int32(100), resp.Items[0].PhysicalQuantity)
		assert.Equal(t, int32(20), resp.Items[0].ReservedQuantity)
		assert.Equal(t, int32(80), resp.Items[0].AvailableQuantity)
		assert.Equal(t, "IN_STOCK", resp.Items[0].StockStatus)

		assert.Equal(t, "MX-DEO-300G", resp.Items[1].SkuCode)
		assert.Equal(t, int32(8), resp.Items[1].AvailableQuantity)
		assert.Equal(t, "LOW_STOCK", resp.Items[1].StockStatus)
	})

	t.Run("InputValidationError_NilRequest", func(t *testing.T) {
		f := setupFixture()
		resp, err := f.handler.GetStockLevel(ctx, nil)
		assert.Nil(t, resp)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("InputValidationError_EmptySKUs", func(t *testing.T) {
		f := setupFixture()
		req := &inventoryv1.GetStockLevelRequest{
			SkuCodes: []string{},
		}
		resp, err := f.handler.GetStockLevel(ctx, req)
		assert.Nil(t, resp)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())

		reqWhitespace := &inventoryv1.GetStockLevelRequest{
			SkuCodes: []string{"", "   "},
		}
		resp, err = f.handler.GetStockLevel(ctx, reqWhitespace)
		assert.Nil(t, resp)
		st, ok = status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("NotFoundError", func(t *testing.T) {
		f := setupFixture()
		f.mockRepo.On("GetItemsBySKUs", mock.Anything, []string{"SKU-NOT-FOUND"}).
			Return(nil, entity.ErrItemNotFound)

		req := &inventoryv1.GetStockLevelRequest{
			SkuCodes: []string{"SKU-NOT-FOUND"},
		}
		resp, err := f.handler.GetStockLevel(ctx, req)
		assert.Nil(t, resp)
		require.Error(t, err)

		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.NotFound, st.Code())

		ed := extractErrorDetail(t, err)
		require.NotNil(t, ed)
		assert.Equal(t, commonv1.ErrorCode_ERR_INVENTORY_SKU_NOT_FOUND, ed.ErrorCode)
	})

	t.Run("InternalError", func(t *testing.T) {
		f := setupFixture()
		f.mockRepo.On("GetItemsBySKUs", mock.Anything, []string{"SKU-ERR"}).
			Return(nil, errors.New("db query error"))

		req := &inventoryv1.GetStockLevelRequest{
			SkuCodes: []string{"SKU-ERR"},
		}
		resp, err := f.handler.GetStockLevel(ctx, req)
		assert.Nil(t, resp)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.Internal, st.Code())
	})
}

// --- 4. GetBatchFEFODetails Tests ---

func TestInventoryGRPCHandler_GetBatchFEFODetails(t *testing.T) {
	ctx := context.Background()

	t.Run("HappyPath", func(t *testing.T) {
		f := setupFixture()
		sku := "MX-GION-500G"
		now := time.Now().UTC()

		item := &entity.InventoryItem{
			SKU:         sku,
			PhysicalQty: 100,
			Status:      entity.ItemStatusActive,
		}
		b1 := &entity.Batch{
			ID:          uuid.New(),
			BatchCode:   "LOT-20261001-01",
			SKU:         sku,
			MfgDate:     now.AddDate(0, -1, 0),
			ExpDate:     now.AddDate(0, 1, 0), // Expiring in ~30 days (< 45 days -> EXPIRING_SOON)
			PhysicalQty: 40,
			ReservedQty: 0,
			Status:      entity.BatchStatusActive,
			CreatedAt:   now.AddDate(0, -1, 0),
		}
		b2 := &entity.Batch{
			ID:          uuid.New(),
			BatchCode:   "LOT-20261101-02",
			SKU:         sku,
			MfgDate:     now.AddDate(0, -1, 0),
			ExpDate:     now.AddDate(0, 4, 0), // Expiring in ~120 days (> 45 days -> ACTIVE_FEFO)
			PhysicalQty: 60,
			ReservedQty: 10,
			Status:      entity.BatchStatusActive,
			CreatedAt:   now.AddDate(0, -1, 0),
		}

		f.mockRepo.On("GetItemBySKU", mock.Anything, sku).Return(item, nil)
		f.mockRepo.On("GetBatchesBySKU", mock.Anything, sku).Return([]*entity.Batch{b1, b2}, nil)

		req := &inventoryv1.GetBatchFEFODetailsRequest{
			SkuCode: sku,
		}

		resp, err := f.handler.GetBatchFEFODetails(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, sku, resp.SkuCode)
		require.Len(t, resp.Batches, 2)

		assert.Equal(t, "LOT-20261001-01", resp.Batches[0].BatchCode)
		assert.Equal(t, int32(40), resp.Batches[0].RemainingQuantity)
		assert.Equal(t, inventoryv1.BatchQualityStatus_BATCH_QUALITY_STATUS_EXPIRING_SOON, resp.Batches[0].QualityStatus)
		assert.Contains(t, resp.Batches[0].OcopTraceQrCode, "LOT-20261001-01")

		assert.Equal(t, "LOT-20261101-02", resp.Batches[1].BatchCode)
		assert.Equal(t, int32(50), resp.Batches[1].RemainingQuantity)
		assert.Equal(t, inventoryv1.BatchQualityStatus_BATCH_QUALITY_STATUS_ACTIVE_FEFO, resp.Batches[1].QualityStatus)
	})

	t.Run("ExpiredBatch_Quarantined", func(t *testing.T) {
		f := setupFixture()
		sku := "MX-EXPIRED-TEST"
		item := &entity.InventoryItem{
			SKU:         sku,
			PhysicalQty: 50,
			ReservedQty: 0,
			Status:      entity.ItemStatusActive,
		}

		now := time.Now().UTC()
		bExpired := &entity.Batch{
			ID:          uuid.New(),
			BatchCode:   "LOT-EXPIRED-01",
			SKU:         sku,
			MfgDate:     now.AddDate(0, -6, 0),
			ExpDate:     now.AddDate(0, 0, -2), // Expired 2 days ago!
			PhysicalQty: 50,
			ReservedQty: 0,
			Status:      entity.BatchStatusActive, // In DB it was marked active
			CreatedAt:   now.AddDate(0, -6, 0),
		}

		f.mockRepo.On("GetItemBySKU", mock.Anything, sku).Return(item, nil)
		f.mockRepo.On("GetBatchesBySKU", mock.Anything, sku).Return([]*entity.Batch{bExpired}, nil)

		req := &inventoryv1.GetBatchFEFODetailsRequest{
			SkuCode: sku,
		}

		resp, err := f.handler.GetBatchFEFODetails(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Len(t, resp.Batches, 1)

		assert.Equal(t, "LOT-EXPIRED-01", resp.Batches[0].BatchCode)
		assert.Equal(t, int32(0), resp.Batches[0].DaysUntilExpiry)
		// Crucial verification: Expired batch MUST be QUARANTINED, NOT EXPIRING_SOON!
		assert.Equal(t, inventoryv1.BatchQualityStatus_BATCH_QUALITY_STATUS_QUARANTINED, resp.Batches[0].QualityStatus)
	})

	t.Run("InputValidationError_NilRequest", func(t *testing.T) {
		f := setupFixture()
		resp, err := f.handler.GetBatchFEFODetails(ctx, nil)
		assert.Nil(t, resp)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("InputValidationError_EmptySKU", func(t *testing.T) {
		f := setupFixture()
		req := &inventoryv1.GetBatchFEFODetailsRequest{SkuCode: ""}
		resp, err := f.handler.GetBatchFEFODetails(ctx, req)
		assert.Nil(t, resp)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())

		reqWhitespace := &inventoryv1.GetBatchFEFODetailsRequest{SkuCode: "   "}
		resp, err = f.handler.GetBatchFEFODetails(ctx, reqWhitespace)
		assert.Nil(t, resp)
		st, ok = status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("NotFoundError_SKU", func(t *testing.T) {
		f := setupFixture()
		sku := "MX-UNKNOWN"
		f.mockRepo.On("GetItemBySKU", mock.Anything, sku).Return(nil, entity.ErrItemNotFound)

		req := &inventoryv1.GetBatchFEFODetailsRequest{SkuCode: sku}
		resp, err := f.handler.GetBatchFEFODetails(ctx, req)
		assert.Nil(t, resp)
		require.Error(t, err)

		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.NotFound, st.Code())

		ed := extractErrorDetail(t, err)
		require.NotNil(t, ed)
		assert.Equal(t, commonv1.ErrorCode_ERR_INVENTORY_SKU_NOT_FOUND, ed.ErrorCode)
	})

	t.Run("InternalError", func(t *testing.T) {
		f := setupFixture()
		sku := "MX-ERR"
		item := &entity.InventoryItem{SKU: sku, Status: entity.ItemStatusActive}
		f.mockRepo.On("GetItemBySKU", mock.Anything, sku).Return(item, nil)
		f.mockRepo.On("GetBatchesBySKU", mock.Anything, sku).Return(nil, errors.New("timeout reading batches"))

		req := &inventoryv1.GetBatchFEFODetailsRequest{SkuCode: sku}
		resp, err := f.handler.GetBatchFEFODetails(ctx, req)
		assert.Nil(t, resp)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.Internal, st.Code())
	})
}

// --- 5. In-Memory gRPC Wire Round-Trip Integration Test (bufconn) ---

func TestInventoryGRPCHandler_RealGRPCRoundTrip(t *testing.T) {
	ctx := context.Background()
	bufferSize := 1024 * 1024
	lis := bufconn.Listen(bufferSize)

	f := setupFixture()
	grpcServer := grpc.NewServer()
	inventoryv1.RegisterInventoryServiceServer(grpcServer, f.handler)

	go func() {
		_ = grpcServer.Serve(lis)
	}()
	defer grpcServer.Stop()

	dialer := func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}

	conn, err := grpc.DialContext(ctx, "bufnet",
		grpc.WithContextDialer(dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	defer conn.Close()

	client := inventoryv1.NewInventoryServiceClient(conn)

	// Test 1: Real gRPC ReserveStock HappyPath over network wire
	orderID := "ORD-BUFCONN-001"
	sku := "MX-GION-500G"
	item := &entity.InventoryItem{SKU: sku, PhysicalQty: 100, ReservedQty: 0, Status: entity.ItemStatusActive}
	now := time.Now().UTC()
	batch := &entity.Batch{
		ID:          uuid.New(),
		SKU:         sku,
		BatchCode:   "LOT-BUF-01",
		MfgDate:     now.Add(-10 * 24 * time.Hour),
		ExpDate:     now.Add(90 * 24 * time.Hour),
		PhysicalQty: 100,
		ReservedQty: 0,
		Status:      entity.BatchStatusActive,
	}

	f.mockRepo.On("GetReservationByOrderID", mock.Anything, orderID).Return(nil, entity.ErrReservationNotFound)
	f.mockLock.On("AcquireLock", mock.Anything, "lock:inventory:sku:"+sku, mock.Anything).Return(true, nil)
	f.mockLock.On("ReleaseLock", mock.Anything, "lock:inventory:sku:"+sku).Return(nil)
	f.mockTxMgr.On("ExecuteInTransaction", mock.Anything, mock.Anything).Return(nil)
	f.mockRepo.On("GetItemBySKUForUpdate", mock.Anything, mock.Anything, sku).Return(item, nil)
	f.mockRepo.On("GetActiveBatchesBySKUForUpdate", mock.Anything, mock.Anything, sku).Return([]*entity.Batch{batch}, nil)
	f.mockRepo.On("UpdateItem", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	f.mockRepo.On("UpdateBatch", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	f.mockRepo.On("CreateReservation", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	reserveResp, err := client.ReserveStock(ctx, &inventoryv1.ReserveStockRequest{
		IdempotencyKey: "IDEM-BUF-001",
		OrderId:        orderID,
		Items: []*inventoryv1.ReserveItem{
			{SkuCode: sku, Quantity: 10},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, reserveResp)
	assert.Equal(t, inventoryv1.ReservationStatus_RESERVATION_STATUS_SUCCESS, reserveResp.Status)
	assert.NotEmpty(t, reserveResp.GetReservationId())

	// Test 2: Real gRPC Error with ErrorDetail unmarshaling over wire
	errOrderID := "ORD-BUF-ERR-002"
	errSKU := "MX-BUF-ERR-SKU"
	f.mockRepo.On("GetReservationByOrderID", mock.Anything, errOrderID).Return(nil, entity.ErrReservationNotFound)
	f.mockLock.On("AcquireLock", mock.Anything, "lock:inventory:sku:"+errSKU, mock.Anything).Return(true, nil)
	f.mockLock.On("ReleaseLock", mock.Anything, "lock:inventory:sku:"+errSKU).Return(nil)
	f.mockTxMgr.On("ExecuteInTransaction", mock.Anything, mock.Anything).Return(nil)
	f.mockRepo.On("GetItemBySKUForUpdate", mock.Anything, mock.Anything, errSKU).Return(&entity.InventoryItem{
		SKU: errSKU, PhysicalQty: 2, ReservedQty: 0, Status: entity.ItemStatusActive,
	}, nil)

	_, err = client.ReserveStock(ctx, &inventoryv1.ReserveStockRequest{
		IdempotencyKey: "IDEM-BUF-002",
		OrderId:        errOrderID,
		Items: []*inventoryv1.ReserveItem{
			{SkuCode: errSKU, Quantity: 50}, // exceeds 2 available
		},
	})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.ResourceExhausted, st.Code())

	var errDetail *commonv1.ErrorDetail
	for _, detail := range st.Details() {
		if ed, ok := detail.(*commonv1.ErrorDetail); ok {
			errDetail = ed
			break
		}
	}
	require.NotNil(t, errDetail)
	assert.Equal(t, commonv1.ErrorCode_ERR_INVENTORY_INSUFFICIENT_STOCK, errDetail.ErrorCode)

	// Test 3: Real gRPC ReleaseReservation over wire
	relOrderID := "ORD-BUF-REL-003"
	relSKU := "MX-BUF-REL-SKU"
	relBatchID := uuid.New()
	relBatch := &entity.Batch{
		ID:          relBatchID,
		SKU:         relSKU,
		BatchCode:   "LOT-BUF-REL",
		PhysicalQty: 100,
		ReservedQty: 10,
		Status:      entity.BatchStatusActive,
	}
	relItem := &entity.InventoryItem{
		SKU:         relSKU,
		PhysicalQty: 100,
		ReservedQty: 10,
		Status:      entity.ItemStatusActive,
	}

	f.mockRepo.On("GetReservationByOrderID", mock.Anything, relOrderID).Return(&entity.StockReservation{
		ID:        uuid.New(),
		OrderID:   relOrderID,
		Status:    entity.ReservationStatusPending,
		ExpiresAt: now.Add(15 * time.Minute),
		Allocations: []entity.ReservationItemAllocation{
			{ID: uuid.New(), SKU: relSKU, BatchID: relBatchID, AllocatedQty: 10},
		},
	}, nil)
	f.mockLock.On("AcquireLock", mock.Anything, "lock:inventory:sku:"+relSKU, mock.Anything).Return(true, nil)
	f.mockLock.On("ReleaseLock", mock.Anything, "lock:inventory:sku:"+relSKU).Return(nil)
	f.mockRepo.On("GetReservationByOrderIDForUpdate", mock.Anything, mock.Anything, relOrderID).Return(&entity.StockReservation{
		ID:        uuid.New(),
		OrderID:   relOrderID,
		Status:    entity.ReservationStatusPending,
		ExpiresAt: now.Add(15 * time.Minute),
		Allocations: []entity.ReservationItemAllocation{
			{ID: uuid.New(), SKU: relSKU, BatchID: relBatchID, AllocatedQty: 10},
		},
	}, nil)
	f.mockRepo.On("GetItemBySKUForUpdate", mock.Anything, mock.Anything, relSKU).Return(relItem, nil)
	f.mockRepo.On("GetBatchByID", mock.Anything, mock.Anything, relBatchID).Return(relBatch, nil)
	f.mockRepo.On("UpdateReservation", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	releaseResp, err := client.ReleaseReservation(ctx, &inventoryv1.ReleaseReservationRequest{
		IdempotencyKey: "IDEM-REL-BUF",
		OrderId:        relOrderID,
		ReleaseReason:  "ORDER_CANCELLED",
	})
	require.NoError(t, err)
	require.NotNil(t, releaseResp)
	assert.True(t, releaseResp.IsReleased)
	assert.Equal(t, int32(10), releaseResp.TotalItemsRestored)
}
