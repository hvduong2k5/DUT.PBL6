package worker

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/profile-service/internal/infrastructure/kafka"
	kafkaGo "github.com/segmentio/kafka-go"
)

type OutboxRecord struct {
	ID            uuid.UUID
	AggregateType string
	AggregateID   string
	EventType     string
	Payload       json.RawMessage
	Topic         string
}

type OutboxPublisher struct {
	pool          *pgxpool.Pool
	producer      kafka.Producer
	batchSize     int
	pollInterval  time.Duration
	stopChan      chan struct{}
	wg            sync.WaitGroup
	lagCountGauge func(count float64) // Optional metric hook
}

func NewOutboxPublisher(
	pool *pgxpool.Pool,
	producer kafka.Producer,
	batchSize int,
	pollInterval time.Duration,
) *OutboxPublisher {
	return &OutboxPublisher{
		pool:         pool,
		producer:     producer,
		batchSize:    batchSize,
		pollInterval: pollInterval,
		stopChan:     make(chan struct{}),
	}
}

func (p *OutboxPublisher) SetLagMetricCallback(cb func(count float64)) {
	p.lagCountGauge = cb
}

func (p *OutboxPublisher) Start(ctx context.Context) {
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		ticker := time.NewTicker(p.pollInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-p.stopChan:
				return
			case <-ticker.C:
				_ = p.publishBatch(ctx)
			}
		}
	}()
}

func (p *OutboxPublisher) Stop() {
	select {
	case <-p.stopChan:
	default:
		close(p.stopChan)
	}
	p.wg.Wait()
}

func (p *OutboxPublisher) publishBatch(ctx context.Context) error {
	tx, err := p.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// SELECT FOR UPDATE SKIP LOCKED ensures multiple pod replicas can poll concurrently without collision
	query := `
		SELECT id, aggregate_type, aggregate_id, event_type, payload, topic
		FROM outbox_events
		WHERE published_at IS NULL
		ORDER BY created_at ASC
		LIMIT $1
		FOR UPDATE SKIP LOCKED
	`
	rows, err := tx.Query(ctx, query, p.batchSize)
	if err != nil {
		return err
	}
	defer rows.Close()

	var records []OutboxRecord
	for rows.Next() {
		var rec OutboxRecord
		if err := rows.Scan(&rec.ID, &rec.AggregateType, &rec.AggregateID, &rec.EventType, &rec.Payload, &rec.Topic); err == nil {
			records = append(records, rec)
		}
	}
	rows.Close()

	if len(records) == 0 {
		if p.lagCountGauge != nil {
			p.lagCountGauge(0)
		}
		return nil
	}

	// Publish to Kafka
	var publishedIDs []uuid.UUID
	for _, rec := range records {
		cloudEvent := map[string]any{
			"specversion":     "1.0",
			"id":              rec.ID.String(),
			"source":          "/profile-service",
			"type":            rec.EventType,
			"time":            time.Now().UTC().Format(time.RFC3339),
			"datacontenttype": "application/json",
			"data":            rec.Payload,
			"aggregate_type":  rec.AggregateType,
			"aggregate_id":    rec.AggregateID,
		}
		eventBytes, err := json.Marshal(cloudEvent)
		if err != nil {
			continue
		}

		msg := kafkaGo.Message{
			Topic: rec.Topic,
			Key:   []byte(rec.AggregateID),
			Value: eventBytes,
			Time:  time.Now().UTC(),
		}

		if err := p.producer.Publish(ctx, rec.Topic, rec.AggregateID, eventBytes); err != nil {
			// Stop batch on first publishing failure to preserve order
			break
		}
		_ = msg
		publishedIDs = append(publishedIDs, rec.ID)
	}

	if len(publishedIDs) > 0 {
		updateQuery := `UPDATE outbox_events SET published_at = NOW() WHERE id = ANY($1)`
		if _, err := tx.Exec(ctx, updateQuery, publishedIDs); err != nil {
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	// Update lag metric
	if p.lagCountGauge != nil {
		var remainingLag int
		_ = p.pool.QueryRow(ctx, `SELECT COUNT(*) FROM outbox_events WHERE published_at IS NULL`).Scan(&remainingLag)
		p.lagCountGauge(float64(remainingLag))
	}

	return nil
}

// GetUnpublishedCount checks remaining outbox events for SRE monitoring.
func (p *OutboxPublisher) GetUnpublishedCount(ctx context.Context) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM outbox_events WHERE published_at IS NULL`
	err := p.pool.QueryRow(ctx, query).Scan(&count)
	return count, err
}
