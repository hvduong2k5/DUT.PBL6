package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/omamx/procurement-service/internal/domain"
)

type GoodsReceivedOutbox interface {
	ClaimGoodsReceivedEvent(context.Context) (domain.OutboxMessage, bool, error)
	MarkGoodsReceivedEventPublished(context.Context, string) error
	RetryGoodsReceivedEvent(context.Context, string, string) error
}

type EventPublisher interface {
	Publish(context.Context, string, string, []byte) error
}

type GoodsReceivedOutboxWorker struct {
	store     GoodsReceivedOutbox
	publisher EventPublisher
	logger    *slog.Logger
	pollEvery time.Duration
}

func NewGoodsReceivedOutboxWorker(store GoodsReceivedOutbox, publisher EventPublisher, logger *slog.Logger) *GoodsReceivedOutboxWorker {
	if logger == nil {
		logger = slog.Default()
	}
	return &GoodsReceivedOutboxWorker{store: store, publisher: publisher, logger: logger, pollEvery: time.Second}
}

func (w *GoodsReceivedOutboxWorker) DispatchOne(ctx context.Context) (bool, error) {
	message, found, err := w.store.ClaimGoodsReceivedEvent(ctx)
	if err != nil || !found {
		return found, err
	}
	if err := w.publisher.Publish(ctx, message.Topic, message.Key, message.Payload); err != nil {
		retryErr := w.store.RetryGoodsReceivedEvent(ctx, message.ID, err.Error())
		if retryErr != nil {
			return true, errors.Join(fmt.Errorf("publish goods received event %s: %w", message.ID, err), fmt.Errorf("schedule event retry: %w", retryErr))
		}
		return true, fmt.Errorf("publish goods received event %s: %w", message.ID, err)
	}
	if err := w.store.MarkGoodsReceivedEventPublished(ctx, message.ID); err != nil {
		// The message may have reached Kafka. Leave it retryable; its stable CloudEvent ID
		// lets consumers deduplicate a publish repeated after this database error.
		return true, fmt.Errorf("mark goods received event %s published: %w", message.ID, err)
	}
	return true, nil
}

func (w *GoodsReceivedOutboxWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.pollEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		found, err := w.DispatchOne(ctx)
		if err != nil {
			w.logger.Error("goods-received outbox delivery failed", "error", err)
		}
		if found && err == nil {
			continue
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
