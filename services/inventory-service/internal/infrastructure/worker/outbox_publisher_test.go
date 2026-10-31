package worker_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"dut-pbl6/inventory-service/internal/application/port"
	"dut-pbl6/inventory-service/internal/domain/entity"
	"dut-pbl6/inventory-service/internal/infrastructure/worker"
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

type MockEventPublisher struct {
	mock.Mock
}

func (m *MockEventPublisher) Publish(ctx context.Context, topic string, key string, payload []byte) error {
	args := m.Called(ctx, topic, key, payload)
	return args.Error(0)
}

func (m *MockEventPublisher) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestOutboxPublisherWorker_HappyPath(t *testing.T) {
	mockRepo := new(MockOutboxRepository)
	mockPub := new(MockEventPublisher)
	ctx := context.Background()

	ce, err := entity.NewStockReservedCloudEvent(
		"res-001",
		"ord-001",
		"MX-GION-500G",
		"KHO-HUONG-THUY",
		10,
		90,
		time.Now().Add(15*time.Minute),
		"",
	)
	require.NoError(t, err)

	ev, err := entity.NewOutboxEvent("StockReservation", "res-001", entity.EventTypeStockReserved, ce)
	require.NoError(t, err)

	mockRepo.On("GetPendingEvents", ctx, 50).Return([]*entity.OutboxEvent{ev}, nil)
	mockPub.On("Publish", ctx, entity.TopicInventoryEvents, "MX-GION-500G", []byte(ev.Payload)).Return(nil)
	mockRepo.On("MarkPublished", ctx, ev.ID, mock.AnythingOfType("time.Time")).Return(nil)

	w := worker.NewOutboxPublisherWorker(mockRepo, mockPub)
	n, err := w.ProcessBatch(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, n)

	mockPub.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestOutboxPublisherWorker_StockCommittedEvent_PublishesWithOrderIDPartitionKey(t *testing.T) {
	mockRepo := new(MockOutboxRepository)
	mockPub := new(MockEventPublisher)
	ctx := context.Background()

	orderID := "ORD-COMMIT-PUBLISH-99"
	ce, err := entity.NewStockCommittedCloudEvent(
		"res-commit-001",
		orderID,
		"wh-hue-01",
		[]entity.StockCommittedItem{
			{SKUCode: "MX-GION-500G", Quantity: 5},
		},
		"",
	)
	require.NoError(t, err)

	ev, err := entity.NewOutboxEvent("StockReservation", "res-commit-001", entity.EventTypeStockCommitted, ce)
	require.NoError(t, err)

	mockRepo.On("GetPendingEvents", ctx, 50).Return([]*entity.OutboxEvent{ev}, nil)
	// Order ID must be extracted as the partition key from the CloudEvent data payload
	mockPub.On("Publish", ctx, entity.TopicInventoryEvents, orderID, []byte(ev.Payload)).Return(nil)
	mockRepo.On("MarkPublished", ctx, ev.ID, mock.AnythingOfType("time.Time")).Return(nil)

	w := worker.NewOutboxPublisherWorker(mockRepo, mockPub)
	n, err := w.ProcessBatch(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, n)

	mockPub.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestOutboxPublisherWorker_PublishError_IncrementsRetry(t *testing.T) {
	mockRepo := new(MockOutboxRepository)
	mockPub := new(MockEventPublisher)
	ctx := context.Background()

	ev, err := entity.NewOutboxEvent("StockReservation", "res-001", entity.EventTypeStockReserved, map[string]string{"foo": "bar"})
	require.NoError(t, err)

	mockRepo.On("GetPendingEvents", ctx, 50).Return([]*entity.OutboxEvent{ev}, nil)
	mockPub.On("Publish", ctx, entity.TopicInventoryEvents, "res-001", []byte(ev.Payload)).
		Return(errors.New("kafka broker unreachable"))

	mockRepo.On("MarkFailed", ctx, ev.ID, 1, "kafka broker unreachable", mock.AnythingOfType("time.Time"), false).Return(nil)

	w := worker.NewOutboxPublisherWorker(mockRepo, mockPub)
	n, err := w.ProcessBatch(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, n)

	mockPub.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestOutboxPublisherWorker_MaxRetriesReached_TransitionsToFinalFailure(t *testing.T) {
	mockRepo := new(MockOutboxRepository)
	mockPub := new(MockEventPublisher)
	ctx := context.Background()

	ev, err := entity.NewOutboxEvent("StockReservation", "res-001", entity.EventTypeStockReserved, map[string]string{"foo": "bar"})
	require.NoError(t, err)
	ev.RetryCount = 5 // Already reached max retries

	mockRepo.On("GetPendingEvents", ctx, 50).Return([]*entity.OutboxEvent{ev}, nil)
	mockRepo.On("MarkFailed", ctx, ev.ID, 5, "maximum retries exceeded", mock.AnythingOfType("time.Time"), true).Return(nil)

	cfg := worker.OutboxPublisherConfig{
		BatchSize:  50,
		MaxRetries: 5,
	}
	w := worker.NewOutboxPublisherWorker(mockRepo, mockPub, cfg)
	n, err := w.ProcessBatch(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, n)

	// Publisher should NOT be called
	mockPub.AssertNotCalled(t, "Publish", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	mockRepo.AssertExpectations(t)
}

func TestOutboxPublisherWorker_ExponentialBackoff_SkipsIfBackoffNotElapsed(t *testing.T) {
	mockRepo := new(MockOutboxRepository)
	mockPub := new(MockEventPublisher)
	ctx := context.Background()

	ev, err := entity.NewOutboxEvent("StockReservation", "res-001", entity.EventTypeStockReserved, map[string]string{"foo": "bar"})
	require.NoError(t, err)
	ev.RetryCount = 2
	recentProcessed := time.Now().UTC().Add(-500 * time.Millisecond) // Backoff for retry 2 is 1s * 2^(2-1) = 2s > 500ms
	ev.ProcessedAt = &recentProcessed

	mockRepo.On("GetPendingEvents", ctx, 50).Return([]*entity.OutboxEvent{ev}, nil)

	cfg := worker.OutboxPublisherConfig{
		BatchSize:   50,
		MaxRetries:  5,
		BaseBackoff: 1 * time.Second,
	}
	w := worker.NewOutboxPublisherWorker(mockRepo, mockPub, cfg)
	n, err := w.ProcessBatch(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, n)

	// Publisher must NOT be called since backoff hasn't elapsed
	mockPub.AssertNotCalled(t, "Publish", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	mockRepo.AssertExpectations(t)
}

func TestOutboxPublisherWorker_Lifecycle(t *testing.T) {
	mockRepo := new(MockOutboxRepository)
	mockPub := new(MockEventPublisher)

	mockRepo.On("GetPendingEvents", mock.Anything, mock.Anything).Return([]*entity.OutboxEvent{}, nil).Maybe()

	cfg := worker.OutboxPublisherConfig{
		PollInterval: 10 * time.Millisecond,
		BatchSize:    10,
	}
	w := worker.NewOutboxPublisherWorker(mockRepo, mockPub, cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := w.Start(ctx)
	require.NoError(t, err)

	// Second start should fail
	err = w.Start(ctx)
	assert.ErrorIs(t, err, worker.ErrWorkerAlreadyRunning)

	time.Sleep(50 * time.Millisecond)

	err = w.Stop(1 * time.Second)
	require.NoError(t, err)

	// Second stop should fail
	err = w.Stop(1 * time.Second)
	assert.ErrorIs(t, err, worker.ErrWorkerNotRunning)
}
