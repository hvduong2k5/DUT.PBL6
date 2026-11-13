package kafka_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"dut-pbl6/inventory-service/internal/application/port"
	"dut-pbl6/inventory-service/internal/application/usecase"
	"dut-pbl6/inventory-service/internal/domain/entity"
	presentationkafka "dut-pbl6/inventory-service/internal/presentation/kafka"
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

type MockIdempotencyRepository struct {
	mock.Mock
}

func (m *MockIdempotencyRepository) CheckOrSet(ctx context.Context, tx port.Transaction, key string) (bool, error) {
	args := m.Called(ctx, tx, key)
	return args.Bool(0), args.Error(1)
}

type MockMessageReader struct {
	mock.Mock
}

func (m *MockMessageReader) FetchMessage(ctx context.Context) (presentationkafka.KafkaMessage, error) {
	args := m.Called(ctx)
	return args.Get(0).(presentationkafka.KafkaMessage), args.Error(1)
}

func (m *MockMessageReader) CommitMessages(ctx context.Context, msgs ...presentationkafka.KafkaMessage) error {
	args := m.Called(ctx, msgs)
	return args.Error(0)
}

func (m *MockMessageReader) Close() error {
	return m.Called().Error(0)
}

func TestOrderPaidHandler_HappyPath(t *testing.T) {
	mockRepo := new(MockInventoryRepository)
	mockTxManager := new(MockTransactionManager)
	mockLockService := new(MockLockService)
	mockIdemp := new(MockIdempotencyRepository)
	ctx := context.Background()

	commitUC := usecase.NewCommitStockDeductionUseCase(mockRepo, mockTxManager, mockLockService)
	handler := presentationkafka.NewOrderPaidHandler(commitUC, mockIdemp)

	orderID := "ORD-20261015-0042"
	sku := "MX-GION-500G"
	batchID := uuid.New()
	whID := entity.DefaultWarehouseID

	// Idempotency: is new
	idempKey := "idemp:kafka:order.events.v1:order_paid:" + orderID
	mockIdemp.On("CheckOrSet", ctx, nil, idempKey).Return(true, nil)

	// Commit use case mocks
	alloc := entity.ReservationItemAllocation{SKU: sku, BatchID: batchID, AllocatedQty: 2}
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
	_ = item.Reserve(2)
	batch, _ := entity.NewBatch("LOT-1", sku, uuid.New(), time.Now().Add(-10*24*time.Hour), time.Now().Add(60*24*time.Hour), 100)
	_ = batch.Reserve(2)

	mockRepo.On("GetReservationByOrderIDForUpdate", ctx, mockTx, orderID).Return(res, nil)
	mockRepo.On("UpdateReservation", ctx, mockTx, res).Return(nil)
	mockRepo.On("GetItemBySKUForUpdate", ctx, mockTx, sku).Return(item, nil)
	mockRepo.On("UpdateItem", ctx, mockTx, item).Return(nil)
	mockRepo.On("GetBatchByID", ctx, mockTx, batchID).Return(batch, nil)
	mockRepo.On("UpdateBatch", ctx, mockTx, batch).Return(nil)

	payload := `{
		"specversion": "1.0",
		"id": "event-123",
		"type": "vn.omama.order.paid.v1",
		"source": "https://omama.vn/services/order-service",
		"data": {
			"order_id": "ORD-20261015-0042",
			"channel": "D2C_WEB"
		}
	}`

	msg := presentationkafka.KafkaMessage{
		Topic: "order.events.v1",
		Key:   []byte("ORD-20261015-0042"),
		Value: []byte(payload),
	}

	err := handler.Handle(ctx, msg)
	require.NoError(t, err)

	mockIdemp.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestOrderPaidHandler_DuplicateEvent_SkippedByIdempotency(t *testing.T) {
	mockRepo := new(MockInventoryRepository)
	mockTxManager := new(MockTransactionManager)
	mockLockService := new(MockLockService)
	mockIdemp := new(MockIdempotencyRepository)
	ctx := context.Background()

	commitUC := usecase.NewCommitStockDeductionUseCase(mockRepo, mockTxManager, mockLockService)
	handler := presentationkafka.NewOrderPaidHandler(commitUC, mockIdemp)

	orderID := "ORD-DUPLICATE"
	idempKey := "idemp:kafka:order.events.v1:order_paid:" + orderID

	// Reservation is already COMMITTED in DB -> commitUC returns nil idempotently
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

	mockIdemp.On("CheckOrSet", ctx, nil, idempKey).Return(false, nil) // Already recorded

	payload := `{
		"type": "vn.omama.order.paid.v1",
		"data": {
			"order_id": "ORD-DUPLICATE"
		}
	}`
	msg := presentationkafka.KafkaMessage{
		Topic: "order.events.v1",
		Key:   []byte("ORD-DUPLICATE"),
		Value: []byte(payload),
	}

	err = handler.Handle(ctx, msg)
	require.NoError(t, err)

	mockRepo.AssertExpectations(t)
	mockIdemp.AssertExpectations(t)
}

func TestOrderPaidHandler_CommitFails_IdempotencyNotRecorded_ReturnsError(t *testing.T) {
	mockRepo := new(MockInventoryRepository)
	mockTxManager := new(MockTransactionManager)
	mockLockService := new(MockLockService)
	mockIdemp := new(MockIdempotencyRepository)
	ctx := context.Background()

	commitUC := usecase.NewCommitStockDeductionUseCase(mockRepo, mockTxManager, mockLockService)
	handler := presentationkafka.NewOrderPaidHandler(commitUC, mockIdemp)

	orderID := "ORD-COMMIT-FAIL"
	transientErr := errors.New("db connection timeout")
	mockRepo.On("GetReservationByOrderID", ctx, orderID).Return(nil, transientErr)

	payload := `{
		"type": "vn.omama.order.paid.v1",
		"data": {
			"order_id": "ORD-COMMIT-FAIL"
		}
	}`
	msg := presentationkafka.KafkaMessage{
		Topic: "order.events.v1",
		Key:   []byte("ORD-COMMIT-FAIL"),
		Value: []byte(payload),
	}

	err := handler.Handle(ctx, msg)
	require.Error(t, err)
	assert.ErrorIs(t, err, transientErr)

	// Idempotency key MUST NOT be recorded so Kafka can retry safely!
	mockIdemp.AssertNotCalled(t, "CheckOrSet", mock.Anything, mock.Anything, mock.Anything)
}

func TestOrderPaidHandler_ReservationAlreadyProcessed_SucceedsAndRecordsIdempotency(t *testing.T) {
	mockRepo := new(MockInventoryRepository)
	mockTxManager := new(MockTransactionManager)
	mockLockService := new(MockLockService)
	mockIdemp := new(MockIdempotencyRepository)
	ctx := context.Background()

	commitUC := usecase.NewCommitStockDeductionUseCase(mockRepo, mockTxManager, mockLockService)
	handler := presentationkafka.NewOrderPaidHandler(commitUC, mockIdemp)

	orderID := "ORD-ALREADY-RELEASED"
	alloc := entity.ReservationItemAllocation{
		ID:           uuid.New(),
		SKU:          "MX-GION-500G",
		BatchID:      uuid.New(),
		AllocatedQty: 1,
	}
	res, err := entity.NewStockReservation(orderID, 15*time.Minute, []entity.ReservationItemAllocation{alloc})
	require.NoError(t, err)
	_ = res.Release()
	mockRepo.On("GetReservationByOrderID", ctx, orderID).Return(res, nil)

	idempKey := "idemp:kafka:order.events.v1:order_paid:" + orderID
	mockIdemp.On("CheckOrSet", ctx, nil, idempKey).Return(true, nil)

	payload := `{
		"type": "vn.omama.order.paid.v1",
		"data": {
			"order_id": "ORD-ALREADY-RELEASED"
		}
	}`
	msg := presentationkafka.KafkaMessage{
		Topic: "order.events.v1",
		Key:   []byte("ORD-ALREADY-RELEASED"),
		Value: []byte(payload),
	}

	err = handler.Handle(ctx, msg)
	require.NoError(t, err)

	mockRepo.AssertExpectations(t)
	mockIdemp.AssertExpectations(t)
}

func TestOrderPaidHandler_DifferentEventType_Ignored(t *testing.T) {
	mockRepo := new(MockInventoryRepository)
	mockTxManager := new(MockTransactionManager)
	mockLockService := new(MockLockService)
	mockIdemp := new(MockIdempotencyRepository)
	ctx := context.Background()

	commitUC := usecase.NewCommitStockDeductionUseCase(mockRepo, mockTxManager, mockLockService)
	handler := presentationkafka.NewOrderPaidHandler(commitUC, mockIdemp)

	payload := `{
		"type": "vn.omama.order.placed.v1",
		"data": {
			"order_id": "ORD-PLACED"
		}
	}`
	msg := presentationkafka.KafkaMessage{
		Topic: "order.events.v1",
		Key:   []byte("ORD-PLACED"),
		Value: []byte(payload),
	}

	err := handler.Handle(ctx, msg)
	require.NoError(t, err)

	mockIdemp.AssertNotCalled(t, "CheckOrSet", mock.Anything, mock.Anything, mock.Anything)
	mockRepo.AssertNotCalled(t, "GetReservationByOrderID", mock.Anything, mock.Anything)
}

func TestOrderPaidHandler_InvalidPayload_ReturnsError(t *testing.T) {
	mockRepo := new(MockInventoryRepository)
	mockTxManager := new(MockTransactionManager)
	mockLockService := new(MockLockService)
	ctx := context.Background()

	commitUC := usecase.NewCommitStockDeductionUseCase(mockRepo, mockTxManager, mockLockService)
	handler := presentationkafka.NewOrderPaidHandler(commitUC)

	msg := presentationkafka.KafkaMessage{
		Topic: "order.events.v1",
		Key:   []byte("ORD-INVALID"),
		Value: []byte("not json"),
	}

	err := handler.Handle(ctx, msg)
	assert.ErrorIs(t, err, presentationkafka.ErrInvalidEventPayload)
}

func TestOrderPaidHandler_MissingOrderID_ReturnsError(t *testing.T) {
	mockRepo := new(MockInventoryRepository)
	mockTxManager := new(MockTransactionManager)
	mockLockService := new(MockLockService)
	ctx := context.Background()

	commitUC := usecase.NewCommitStockDeductionUseCase(mockRepo, mockTxManager, mockLockService)
	handler := presentationkafka.NewOrderPaidHandler(commitUC)

	payload := `{
		"type": "vn.omama.order.paid.v1",
		"data": {}
	}`
	msg := presentationkafka.KafkaMessage{
		Topic: "order.events.v1",
		Value: []byte(payload),
	}

	err := handler.Handle(ctx, msg)
	assert.ErrorIs(t, err, presentationkafka.ErrInvalidEventPayload)
}

func TestKafkaConsumerListener_Lifecycle(t *testing.T) {
	mockReader := new(MockMessageReader)
	mockRepo := new(MockInventoryRepository)
	mockTxManager := new(MockTransactionManager)
	mockLockService := new(MockLockService)

	commitUC := usecase.NewCommitStockDeductionUseCase(mockRepo, mockTxManager, mockLockService)
	handler := presentationkafka.NewOrderPaidHandler(commitUC)

	listener := presentationkafka.NewKafkaConsumerListener(mockReader, handler)

	mockReader.On("FetchMessage", mock.Anything).Return(presentationkafka.KafkaMessage{}, errors.New("read timeout")).Maybe()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := listener.Start(ctx)
	require.NoError(t, err)

	err = listener.Start(ctx)
	assert.ErrorIs(t, err, presentationkafka.ErrListenerAlreadyRunning)

	time.Sleep(30 * time.Millisecond)

	err = listener.Stop(1 * time.Second)
	require.NoError(t, err)

	err = listener.Stop(1 * time.Second)
	assert.ErrorIs(t, err, presentationkafka.ErrListenerNotRunning)
}

func TestOrderPaidHandler_CheckOrSetFails_ReturnsError(t *testing.T) {
	mockRepo := new(MockInventoryRepository)
	mockTxManager := new(MockTransactionManager)
	mockLockService := new(MockLockService)
	mockIdemp := new(MockIdempotencyRepository)
	ctx := context.Background()

	commitUC := usecase.NewCommitStockDeductionUseCase(mockRepo, mockTxManager, mockLockService)
	handler := presentationkafka.NewOrderPaidHandler(commitUC, mockIdemp)

	orderID := "ORD-IDEMP-FAIL"
	idempKey := "idemp:kafka:order.events.v1:order_paid:" + orderID

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

	idempErr := errors.New("db connection lost during CheckOrSet")
	mockIdemp.On("CheckOrSet", ctx, nil, idempKey).Return(false, idempErr)

	payload := `{
		"type": "vn.omama.order.paid.v1",
		"data": {
			"order_id": "ORD-IDEMP-FAIL"
		}
	}`
	msg := presentationkafka.KafkaMessage{
		Topic: "order.events.v1",
		Key:   []byte("ORD-IDEMP-FAIL"),
		Value: []byte(payload),
	}

	err = handler.Handle(ctx, msg)
	require.Error(t, err)
	assert.ErrorIs(t, err, idempErr)

	mockRepo.AssertExpectations(t)
	mockIdemp.AssertExpectations(t)
}

func TestOrderPaidHandler_CatchesPanic_ReturnsError(t *testing.T) {
	// A handler with nil use case causes a panic during Execute, which must be caught safely
	handler := presentationkafka.NewOrderPaidHandler(nil)
	ctx := context.Background()

	payload := `{
		"type": "vn.omama.order.paid.v1",
		"data": {
			"order_id": "ORD-PANIC-TEST"
		}
	}`
	msg := presentationkafka.KafkaMessage{
		Topic: "order.events.v1",
		Key:   []byte("ORD-PANIC-TEST"),
		Value: []byte(payload),
	}

	assert.NotPanics(t, func() {
		err := handler.Handle(ctx, msg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "panic recovered in OrderPaidHandler")
	})
}
