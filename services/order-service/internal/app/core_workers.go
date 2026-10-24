package app

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/omamx/order-service/internal/domain"
	"github.com/omamx/order-service/internal/persistence"
	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
	"strconv"
)

type coreCleanup struct{ s *Service }

func (c coreCleanup) Cleanup(ctx context.Context, scope, cartID string, revision int64) (string, error) {
	key := "cart:" + scope
	outcome := "APPLIED"
	e := c.s.Redis.Watch(ctx, func(tx *redis.Tx) error {
		raw, e := tx.Get(ctx, key).Bytes()
		if errors.Is(e, redis.Nil) {
			outcome = "CART_MISSING"
			return nil
		}
		if e != nil {
			return e
		}
		var cart domain.Cart
		if e = json.Unmarshal(raw, &cart); e != nil {
			return e
		}
		if cart.Revision != revision {
			outcome = "SKIPPED_CHANGED"
			return nil
		}
		actual, e := tx.Get(ctx, "core-cart-id:"+scope).Result()
		if e != nil && !errors.Is(e, redis.Nil) {
			return e
		}
		if actual != cartID {
			outcome = "SKIPPED_REPLACED"
			return nil
		}
		_, e = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error { pipe.Del(ctx, key, "core-cart-id:"+scope); return nil })
		return e
	}, key, "core-cart-id:"+scope)
	return outcome, e
}

type corePublisher struct{ s *Service }

func (p corePublisher) Publish(ctx context.Context, message persistence.EventMessage) error {
	return p.s.Writer.WriteMessages(ctx, kafka.Message{Topic: message.Topic, Key: []byte(message.Key), Value: message.Bytes, Headers: []kafka.Header{{Key: "event_id", Value: []byte(message.ID)}, {Key: "schema_hash", Value: []byte(message.SchemaHash)}, {Key: "stream_id", Value: []byte(message.Stream)}, {Key: "stream_sequence", Value: []byte(strconv.FormatInt(message.Sequence, 10))}}})
}
func (s *Service) PublishCoreOne(ctx context.Context) (bool, error) {
	if s.CoreCheckout == nil || s.Writer == nil {
		return false, nil
	}
	return s.CoreCheckout.Store.PublishOne(ctx, corePublisher{s})
}
func (s *Service) WorkCoreFollowup(ctx context.Context) (bool, error) {
	if s.CoreCheckout == nil {
		return false, nil
	}
	return s.CoreCheckout.WorkFollowup(ctx, coreCleanup{s})
}
func (s *Service) getCoreOrder(ctx context.Context, id string, p domain.Principal, detail bool) (domain.Order, error) {
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return domain.Order{}, e
	}
	defer tx.Rollback(ctx)
	o, e := scanOrder(tx.QueryRow(ctx, "SELECT "+orderColumns+" FROM orders WHERE id=$1 AND cohort='order-core-1' FOR SHARE", id))
	if e != nil {
		return o, e
	}
	if !CanRead(p, o) {
		return domain.Order{}, domain.E("ORDER_NOT_FOUND", 404)
	}
	a, e := s.Payment.Store.LoadOrderTx(ctx, tx, id, o.Scope)
	if e != nil {
		return o, e
	}
	data := a.Snapshot().Data()
	o.ItemsDigest = data.Provenance.ItemsDigest
	if detail {
		if p.Role == "CUSTOMER_SERVICE" {
			data.Address.Street = "[masked]"
			data.Address.RecipientName = "[masked]"
			data.Address.PhoneE164 = "[masked]"
		}
		o.CoreSnapshot = &data
	}
	return o, tx.Commit(ctx)
}
func (s *Service) isCoreOrder(ctx context.Context, id string) bool {
	if s.CoreCheckout == nil {
		return false
	}
	var yes bool
	e := s.DB.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM orders WHERE id=$1 AND cohort='order-core-1')", id).Scan(&yes)
	return e == nil && yes
}
func (s *Service) isCoreIdentity(ctx context.Context, id string) bool {
	if s.CoreCheckout == nil {
		return false
	}
	var target bool
	e := s.DB.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM orders WHERE id=$1 AND cohort='order-core-1') OR EXISTS(SELECT 1 FROM checkout_operations WHERE planned_order_id=$1 AND cohort='order-core-1')", id).Scan(&target)
	return e == nil && target
}
