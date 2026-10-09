package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	carekafka "github.com/omamx/care-service/internal/infrastructure/kafka"
	"github.com/rs/zerolog/log"
)

type OutboxRecord struct {
	ID            uuid.UUID
	AggregateType string
	AggregateID   string
	EventType     string
	Topic         string
	PartitionKey  string
	Payload       json.RawMessage
	Traceparent   string
	CreatedAt     time.Time
	Attempts      int
}

type Outbox struct {
	pool         *pgxpool.Pool
	producer     carekafka.Producer
	pollInterval time.Duration
	batchSize    int
	stop         chan struct{}
	stopOnce     sync.Once
	wg           sync.WaitGroup
	onLag        func(float64)
}

func NewOutbox(pool *pgxpool.Pool, producer carekafka.Producer, pollInterval time.Duration, batchSize int) *Outbox {
	return &Outbox{
		pool: pool, producer: producer, pollInterval: pollInterval, batchSize: batchSize,
		stop: make(chan struct{}),
	}
}

func (w *Outbox) SetLagCallback(callback func(float64)) {
	w.onLag = callback
}

func (w *Outbox) Start(ctx context.Context) {
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		ticker := time.NewTicker(w.pollInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-w.stop:
				return
			case <-ticker.C:
				batchCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
				err := w.publishBatch(batchCtx)
				cancel()
				if err != nil && ctx.Err() == nil {
					log.Error().Err(err).Msg("outbox batch failed; events remain available for retry")
				}
			}
		}
	}()
}

func (w *Outbox) Stop() {
	w.stopOnce.Do(func() { close(w.stop) })
	w.wg.Wait()
}

func (w *Outbox) publishBatch(ctx context.Context) error {
	tx, err := w.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `
		SELECT id, aggregate_type, aggregate_id, event_type, topic, partition_key,
		       payload, COALESCE(traceparent, ''), created_at, attempts
		FROM outbox_events
		WHERE published_at IS NULL AND next_attempt_at <= NOW()
		ORDER BY created_at
		LIMIT $1
		FOR UPDATE SKIP LOCKED
	`, w.batchSize)
	if err != nil {
		return err
	}
	var records []OutboxRecord
	for rows.Next() {
		var record OutboxRecord
		if err := rows.Scan(
			&record.ID, &record.AggregateType, &record.AggregateID, &record.EventType,
			&record.Topic, &record.PartitionKey, &record.Payload, &record.Traceparent,
			&record.CreatedAt, &record.Attempts,
		); err != nil {
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
		envelope := map[string]any{
			"specversion":     "1.0",
			"id":              record.ID.String(),
			"source":          "https://omama.vn/services/care-service",
			"type":            record.EventType,
			"subject":         record.AggregateType + ":" + record.AggregateID,
			"time":            record.CreatedAt.UTC().Format(time.RFC3339Nano),
			"datacontenttype": "application/json",
			"data":            record.Payload,
		}
		if record.Traceparent != "" {
			envelope["traceparent"] = record.Traceparent
		}
		payload, marshalErr := json.Marshal(envelope)
		if marshalErr != nil {
			return marshalErr
		}
		if publishErr := w.producer.Publish(ctx, record.Topic, record.PartitionKey, payload); publishErr != nil {
			nextAttempt := retryAt(record.Attempts + 1)
			_, updateErr := tx.Exec(ctx, `
				UPDATE outbox_events
				SET attempts = attempts + 1, last_error = $2, next_attempt_at = $3
				WHERE id = $1
			`, record.ID, truncateError(publishErr.Error()), nextAttempt)
			if updateErr != nil {
				return updateErr
			}
			if err := tx.Commit(ctx); err != nil {
				return err
			}
			w.reportLag(ctx)
			return fmt.Errorf("publish outbox event %s: %w", record.ID, publishErr)
		}
		if _, err := tx.Exec(ctx, `
			UPDATE outbox_events
			SET published_at = NOW(), attempts = attempts + 1, last_error = NULL
			WHERE id = $1
		`, record.ID); err != nil {
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}
	w.reportLag(ctx)
	return nil
}

func (w *Outbox) reportLag(ctx context.Context) {
	if w.onLag == nil {
		return
	}
	var count int64
	if err := w.pool.QueryRow(ctx, `SELECT COUNT(*) FROM outbox_events WHERE published_at IS NULL`).Scan(&count); err == nil {
		w.onLag(float64(count))
	}
}

func retryAt(attempt int) time.Time {
	delay := time.Second << min(attempt, 8)
	return time.Now().UTC().Add(delay)
}

func truncateError(value string) string {
	if len(value) <= 2000 {
		return value
	}
	return value[:2000]
}
