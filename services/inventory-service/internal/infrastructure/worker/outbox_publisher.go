package worker

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strings"
	"sync/atomic"
	"time"

	"dut-pbl6/inventory-service/internal/application/port"
	"dut-pbl6/inventory-service/internal/domain/entity"
)

var (
	ErrWorkerAlreadyRunning = errors.New("worker is already running")
	ErrWorkerNotRunning     = errors.New("worker is not running")
)

// OutboxPublisherConfig configures the OutboxPublisherWorker.
type OutboxPublisherConfig struct {
	PollInterval time.Duration
	BatchSize    int
	MaxRetries   int
	BaseBackoff  time.Duration
	DefaultTopic string
}

// DefaultOutboxPublisherConfig returns standard production defaults.
func DefaultOutboxPublisherConfig() OutboxPublisherConfig {
	return OutboxPublisherConfig{
		PollInterval: 500 * time.Millisecond,
		BatchSize:    50,
		MaxRetries:   5,
		BaseBackoff:  1 * time.Second,
		DefaultTopic: entity.TopicInventoryEvents,
	}
}

// OutboxPublisherWorker polls outbox_events table for PENDING events and publishes them to Kafka.
type OutboxPublisherWorker struct {
	outboxRepo port.OutboxRepository
	publisher  port.EventPublisher
	config     OutboxPublisherConfig
	isRunning  atomic.Bool
	stopChan   chan struct{}
	doneChan   chan struct{}
}

// NewOutboxPublisherWorker creates a new OutboxPublisherWorker instance.
func NewOutboxPublisherWorker(
	outboxRepo port.OutboxRepository,
	publisher port.EventPublisher,
	cfg ...OutboxPublisherConfig,
) *OutboxPublisherWorker {
	c := DefaultOutboxPublisherConfig()
	if len(cfg) > 0 {
		c = cfg[0]
		if c.PollInterval <= 0 {
			c.PollInterval = 500 * time.Millisecond
		}
		if c.BatchSize <= 0 {
			c.BatchSize = 50
		}
		if c.MaxRetries <= 0 {
			c.MaxRetries = 5
		}
		if c.BaseBackoff <= 0 {
			c.BaseBackoff = 1 * time.Second
		}
		if c.DefaultTopic == "" {
			c.DefaultTopic = entity.TopicInventoryEvents
		}
	}

	return &OutboxPublisherWorker{
		outboxRepo: outboxRepo,
		publisher:  publisher,
		config:     c,
		stopChan:   make(chan struct{}),
		doneChan:   make(chan struct{}),
	}
}

// Start launches the background polling loop.
func (w *OutboxPublisherWorker) Start(ctx context.Context) error {
	if !w.isRunning.CompareAndSwap(false, true) {
		return ErrWorkerAlreadyRunning
	}

	w.stopChan = make(chan struct{})
	w.doneChan = make(chan struct{})

	go w.run(ctx)
	return nil
}

// Stop gracefully shuts down the worker, waiting for in-flight batch to complete.
func (w *OutboxPublisherWorker) Stop(timeout ...time.Duration) error {
	if !w.isRunning.CompareAndSwap(true, false) {
		return ErrWorkerNotRunning
	}

	close(w.stopChan)

	to := 5 * time.Second
	if len(timeout) > 0 && timeout[0] > 0 {
		to = timeout[0]
	}

	select {
	case <-w.doneChan:
		return nil
	case <-time.After(to):
		return errors.New("timeout waiting for outbox publisher worker to stop")
	}
}

func (w *OutboxPublisherWorker) run(ctx context.Context) {
	defer close(w.doneChan)

	ticker := time.NewTicker(w.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopChan:
			return
		case <-ticker.C:
			_, _ = w.ProcessBatch(ctx)
		}
	}
}

type lightPayload struct {
	Subject string `json:"subject"`
	Data    struct {
		SKUCode   string `json:"sku_code"`
		BatchCode string `json:"batch_code"`
		OrderID   string `json:"order_id"`
	} `json:"data"`
}

// ProcessBatch scans and dispatches one batch of pending outbox events.
func (w *OutboxPublisherWorker) ProcessBatch(ctx context.Context) (int, error) {
	if w.outboxRepo == nil || w.publisher == nil {
		return 0, nil
	}

	events, err := w.outboxRepo.GetPendingEvents(ctx, w.config.BatchSize)
	if err != nil {
		log.Printf("[OutboxPublisherWorker] Failed to query pending outbox events: %v", err)
		return 0, err
	}

	if len(events) == 0 {
		return 0, nil
	}

	publishedCount := 0
	now := time.Now().UTC()

	for _, ev := range events {
		// 1. Max retries guard
		if ev.RetryCount >= w.config.MaxRetries {
			_ = w.outboxRepo.MarkFailed(ctx, ev.ID, ev.RetryCount, "maximum retries exceeded", now, true)
			continue
		}

		// 2. Exponential backoff check
		if ev.RetryCount > 0 {
			backoff := w.config.BaseBackoff * time.Duration(1<<(ev.RetryCount-1))
			lastAttempt := ev.CreatedAt
			if ev.ProcessedAt != nil {
				lastAttempt = *ev.ProcessedAt
			}
			if now.Before(lastAttempt.Add(backoff)) {
				// Backoff period has not elapsed yet; skip for now
				continue
			}
		}

		// 3. Extract Partition Key and Topic
		topic := w.config.DefaultTopic
		partitionKey := ev.AggregateID

		var lp lightPayload
		if err := json.Unmarshal(ev.Payload, &lp); err == nil {
			if lp.Data.SKUCode != "" {
				partitionKey = lp.Data.SKUCode
			} else if lp.Data.BatchCode != "" {
				partitionKey = lp.Data.BatchCode
			} else if lp.Data.OrderID != "" {
				partitionKey = lp.Data.OrderID
			} else if strings.HasPrefix(lp.Subject, "sku:") {
				partitionKey = strings.TrimPrefix(lp.Subject, "sku:")
			}
		}

		// 4. Publish to Message Broker
		pubErr := w.publisher.Publish(ctx, topic, partitionKey, []byte(ev.Payload))
		if pubErr != nil {
			newRetry := ev.RetryCount + 1
			finalFailure := newRetry >= w.config.MaxRetries
			log.Printf("[OutboxPublisherWorker] Publish error for event %s (retry %d/%d): %v",
				ev.ID, newRetry, w.config.MaxRetries, pubErr)

			_ = w.outboxRepo.MarkFailed(ctx, ev.ID, newRetry, pubErr.Error(), time.Now().UTC(), finalFailure)
			continue
		}

		// 5. Success -> mark published
		if err := w.outboxRepo.MarkPublished(ctx, ev.ID, time.Now().UTC()); err != nil {
			log.Printf("[OutboxPublisherWorker] Failed to mark event %s published: %v", ev.ID, err)
		} else {
			publishedCount++
		}
	}

	return publishedCount, nil
}
