package worker

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

type Publisher interface {
	Publish(context.Context, string, string, []byte) error
}

type Outbox struct {
	pool         *pgxpool.Pool
	publisher    Publisher
	pollInterval time.Duration
	batchSize    int
	logger       zerolog.Logger
}

func NewOutbox(pool *pgxpool.Pool, publisher Publisher, pollInterval time.Duration, logger zerolog.Logger) *Outbox {
	return &Outbox{pool: pool, publisher: publisher, pollInterval: pollInterval, batchSize: 50, logger: logger}
}

func (w *Outbox) Run(ctx context.Context) {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()
	for {
		if err := w.publishBatch(ctx); err != nil && ctx.Err() == nil {
			w.logger.Error().Err(err).Msg("outbox batch failed")
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

type outboxRecord struct {
	id           uuid.UUID
	topic        string
	partitionKey string
	payload      []byte
}

func (w *Outbox) publishBatch(ctx context.Context) error {
	tx, err := w.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	rows, err := tx.Query(ctx, `
		SELECT id, topic, partition_key, payload
		FROM outbox_events
		WHERE published_at IS NULL
		ORDER BY occurred_at
		LIMIT $1
		FOR UPDATE SKIP LOCKED`, w.batchSize)
	if err != nil {
		return err
	}
	var records []outboxRecord
	for rows.Next() {
		var record outboxRecord
		if err := rows.Scan(&record.id, &record.topic, &record.partitionKey, &record.payload); err != nil {
			rows.Close()
			return err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	for _, record := range records {
		publishCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		publishErr := w.publisher.Publish(publishCtx, record.topic, record.partitionKey, record.payload)
		cancel()
		if publishErr != nil {
			_, _ = tx.Exec(ctx, `UPDATE outbox_events SET attempts=attempts+1, last_error=$2 WHERE id=$1`, record.id, publishErr.Error())
			if err := tx.Commit(ctx); err != nil {
				return err
			}
			return publishErr
		}
		if _, err := tx.Exec(ctx, `UPDATE outbox_events SET published_at=now(), attempts=attempts+1, last_error='' WHERE id=$1`, record.id); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
