package application

import (
	"context"
	"errors"
	"time"

	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/domain"
)

type OutboxQueue interface {
	Claim(context.Context) (domain.OutboxMessage, bool, error)
	Ack(context.Context, domain.OutboxMessage) error
	Fail(context.Context, domain.OutboxMessage, string) error
}
type EventPublisher interface {
	Publish(context.Context, domain.OutboxMessage) error
}
type Relay struct {
	Queue          OutboxQueue
	Publisher      EventPublisher
	PublishTimeout time.Duration
}

// Tick processes one event. A missing broker is an error, never a log-only send.
func (r *Relay) Tick(ctx context.Context) (bool, error) {
	if r == nil || r.Queue == nil || r.Publisher == nil || r.PublishTimeout <= 0 || r.PublishTimeout > 5*time.Second {
		return false, domain.ErrUnavailable
	}
	claimCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	message, found, err := r.Queue.Claim(claimCtx)
	cancel()
	if err != nil || !found {
		return false, err
	}
	publishCtx, done := context.WithTimeout(ctx, r.PublishTimeout)
	err = r.Publisher.Publish(publishCtx, message)
	done()
	if err != nil {
		code := "BROKER_PUBLISH_FAILED"
		if ctx.Err() != nil {
			code = "DELIVERY_INTERRUPTED"
		}
		// Cancellation still releases the lease within a bounded cleanup window.
		clean, finish := context.WithTimeout(context.Background(), 2*time.Second)
		defer finish()
		if nack := r.Queue.Fail(clean, message, code); nack != nil {
			return true, errors.Join(err, nack)
		}
		return true, err
	}
	ackCtx, finish := context.WithTimeout(ctx, 2*time.Second)
	defer finish()
	if err = r.Queue.Ack(ackCtx, message); err != nil {
		return true, err
	}
	return true, nil
}
