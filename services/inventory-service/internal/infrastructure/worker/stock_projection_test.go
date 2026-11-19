package worker_test

import (
	"context"
	"dut-pbl6/inventory-service/internal/domain/entity"
	"dut-pbl6/inventory-service/internal/infrastructure/worker"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestCatalogSnapshotRouting(t *testing.T) {
	ctx := context.Background()
	repo := new(MockOutboxRepository)
	pub := new(MockEventPublisher)
	payload := json.RawMessage(`{"sku_code":"SKU-A","sellable_quantity":8,"inventory_version":2,"source_updated_at":"2026-11-04T00:00:00Z"}`)
	ev := &entity.OutboxEvent{ID: uuid.New(), AggregateID: "SKU-A", EventType: "inventory.stock-changed.v1", Payload: payload, CreatedAt: time.Now().Add(-time.Minute)}
	repo.On("GetPendingEvents", ctx, 50).Return([]*entity.OutboxEvent{ev}, nil)
	pub.On("Publish", ctx, "inventory.stock-changed.v1", "SKU-A", []byte(payload)).Return(nil)
	repo.On("MarkPublished", ctx, ev.ID, mock.AnythingOfType("time.Time")).Return(nil)
	count, err := worker.NewOutboxPublisherWorker(repo, pub).ProcessBatch(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count)
	pub.AssertExpectations(t)
	repo.AssertExpectations(t)
}
