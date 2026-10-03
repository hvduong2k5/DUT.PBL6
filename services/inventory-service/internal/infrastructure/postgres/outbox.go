package postgres

import (
	"context"
	"database/sql"
	"time"

	"dut-pbl6/inventory-service/internal/application/port"
	"dut-pbl6/inventory-service/internal/domain/entity"
	"github.com/google/uuid"
)

// PostgresOutboxRepository implements port.OutboxRepository using PostgreSQL.
type PostgresOutboxRepository struct {
	db *sql.DB
}

// NewPostgresOutboxRepository creates a new PostgresOutboxRepository.
func NewPostgresOutboxRepository(db *sql.DB) *PostgresOutboxRepository {
	return &PostgresOutboxRepository{db: db}
}

func (r *PostgresOutboxRepository) getExecutor(tx port.Transaction) queryExecutor {
	if tx != nil {
		if sqlTx, ok := tx.(*SQLTransaction); ok && sqlTx.Tx != nil {
			return sqlTx.Tx
		}
	}
	return r.db
}

// SaveEvent inserts a new outbox event record within an existing transaction or standalone.
func (r *PostgresOutboxRepository) SaveEvent(ctx context.Context, tx port.Transaction, event *entity.OutboxEvent) error {
	query := `INSERT INTO outbox_events (id, aggregate_type, aggregate_id, event_type, payload, status, retry_count, created_at)
              VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	createdAt := event.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}

	_, err := r.getExecutor(tx).ExecContext(
		ctx,
		query,
		event.ID,
		event.AggregateType,
		event.AggregateID,
		event.EventType,
		[]byte(event.Payload),
		string(event.Status),
		event.RetryCount,
		createdAt,
	)
	return err
}

// GetPendingEvents retrieves up to batchSize PENDING outbox events ordered by created_at ASC.
func (r *PostgresOutboxRepository) GetPendingEvents(ctx context.Context, batchSize int) ([]*entity.OutboxEvent, error) {
	if batchSize <= 0 {
		batchSize = 50
	}

	query := `SELECT id, aggregate_type, aggregate_id, event_type, payload, status, retry_count, created_at, processed_at, error_message
              FROM outbox_events
              WHERE status = 'PENDING'
              ORDER BY created_at ASC
              LIMIT $1`

	rows, err := r.db.QueryContext(ctx, query, batchSize)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*entity.OutboxEvent
	for rows.Next() {
		ev := &entity.OutboxEvent{}
		var statusStr string
		var rawPayload []byte
		var processedAt sql.NullTime
		var errorMsg sql.NullString

		if err := rows.Scan(
			&ev.ID,
			&ev.AggregateType,
			&ev.AggregateID,
			&ev.EventType,
			&rawPayload,
			&statusStr,
			&ev.RetryCount,
			&ev.CreatedAt,
			&processedAt,
			&errorMsg,
		); err != nil {
			return nil, err
		}

		ev.Status = entity.OutboxStatus(statusStr)
		ev.Payload = rawPayload
		if processedAt.Valid {
			t := processedAt.Time
			ev.ProcessedAt = &t
		}
		if errorMsg.Valid {
			s := errorMsg.String
			ev.ErrorMessage = &s
		}
		events = append(events, ev)
	}

	return events, rows.Err()
}

// MarkPublished marks the outbox event as PUBLISHED and records the processed timestamp.
func (r *PostgresOutboxRepository) MarkPublished(ctx context.Context, id uuid.UUID, processedAt time.Time) error {
	query := `UPDATE outbox_events
              SET status = $1, processed_at = $2, error_message = NULL
              WHERE id = $3`

	_, err := r.db.ExecContext(ctx, query, string(entity.OutboxStatusPublished), processedAt.UTC(), id)
	return err
}

// MarkFailed updates the retry count, error message, and sets status to FAILED if finalFailure is true.
func (r *PostgresOutboxRepository) MarkFailed(ctx context.Context, id uuid.UUID, retryCount int, errorMessage string, processedAt time.Time, finalFailure bool) error {
	status := entity.OutboxStatusPending
	if finalFailure {
		status = entity.OutboxStatusFailed
	}

	query := `UPDATE outbox_events
              SET status = $1, retry_count = $2, error_message = $3, processed_at = $4
              WHERE id = $5`

	_, err := r.db.ExecContext(ctx, query, string(status), retryCount, errorMessage, processedAt.UTC(), id)
	return err
}
