package postgres_test

import (
	"context"
	"regexp"
	"testing"
	"time"

	"dut-pbl6/inventory-service/internal/domain/entity"
	"dut-pbl6/inventory-service/internal/infrastructure/postgres"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostgresOutboxRepository_SaveEvent_Standalone(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := postgres.NewPostgresOutboxRepository(db)
	ctx := context.Background()

	ev, err := entity.NewOutboxEvent("StockReservation", "res-001", entity.EventTypeStockReserved, map[string]string{"foo": "bar"})
	require.NoError(t, err)

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO outbox_events (id, aggregate_type, aggregate_id, event_type, payload, status, retry_count, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)")).
		WithArgs(ev.ID, ev.AggregateType, ev.AggregateID, ev.EventType, []byte(ev.Payload), string(ev.Status), ev.RetryCount, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.SaveEvent(ctx, nil, ev)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresOutboxRepository_SaveEvent_WithTx(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := postgres.NewPostgresOutboxRepository(db)
	ctx := context.Background()

	ev, err := entity.NewOutboxEvent("StockReservation", "res-001", entity.EventTypeStockReserved, map[string]string{"foo": "bar"})
	require.NoError(t, err)

	mock.ExpectBegin()
	sqlTx, err := db.Begin()
	require.NoError(t, err)
	tx := &postgres.SQLTransaction{Tx: sqlTx}

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO outbox_events (id, aggregate_type, aggregate_id, event_type, payload, status, retry_count, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)")).
		WithArgs(ev.ID, ev.AggregateType, ev.AggregateID, ev.EventType, []byte(ev.Payload), string(ev.Status), ev.RetryCount, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.SaveEvent(ctx, tx, ev)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresOutboxRepository_GetPendingEvents(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := postgres.NewPostgresOutboxRepository(db)
	ctx := context.Background()

	evID := uuid.New()
	now := time.Now().UTC()

	rows := sqlmock.NewRows([]string{
		"id", "aggregate_type", "aggregate_id", "event_type", "payload",
		"status", "retry_count", "created_at", "processed_at", "error_message",
	}).AddRow(evID, "StockReservation", "res-001", entity.EventTypeStockReserved, []byte(`{"test":true}`), "PENDING", 0, now, nil, nil)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, aggregate_type, aggregate_id, event_type, payload, status, retry_count, created_at, processed_at, error_message FROM outbox_events WHERE status = 'PENDING' ORDER BY created_at ASC LIMIT $1")).
		WithArgs(10).
		WillReturnRows(rows)

	events, err := repo.GetPendingEvents(ctx, 10)
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, evID, events[0].ID)
	assert.Equal(t, entity.OutboxStatusPending, events[0].Status)
	assert.Equal(t, "StockReservation", events[0].AggregateType)
	assert.Nil(t, events[0].ProcessedAt)
	assert.Nil(t, events[0].ErrorMessage)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresOutboxRepository_MarkPublished(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := postgres.NewPostgresOutboxRepository(db)
	ctx := context.Background()
	evID := uuid.New()
	now := time.Now().UTC()

	mock.ExpectExec(regexp.QuoteMeta("UPDATE outbox_events SET status = $1, processed_at = $2, error_message = NULL WHERE id = $3")).
		WithArgs(string(entity.OutboxStatusPublished), now, evID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.MarkPublished(ctx, evID, now)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresOutboxRepository_MarkFailed(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := postgres.NewPostgresOutboxRepository(db)
	ctx := context.Background()
	evID := uuid.New()
	now := time.Now().UTC()

	// Transient failure (still PENDING)
	mock.ExpectExec(regexp.QuoteMeta("UPDATE outbox_events SET status = $1, retry_count = $2, error_message = $3, processed_at = $4 WHERE id = $5")).
		WithArgs(string(entity.OutboxStatusPending), 1, "kafka timeout", now, evID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.MarkFailed(ctx, evID, 1, "kafka timeout", now, false)
	require.NoError(t, err)

	// Final failure (FAILED)
	mock.ExpectExec(regexp.QuoteMeta("UPDATE outbox_events SET status = $1, retry_count = $2, error_message = $3, processed_at = $4 WHERE id = $5")).
		WithArgs(string(entity.OutboxStatusFailed), 5, "max retries exceeded", now, evID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.MarkFailed(ctx, evID, 5, "max retries exceeded", now, true)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}
