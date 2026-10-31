package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"time"
)

type EventMessage struct {
	ID, Stream, Topic, Key, SchemaHash string
	Sequence                           int64
	Bytes                              []byte
}
type EventPublisher interface {
	Publish(context.Context, EventMessage) error
}

// PublishOne holds a fenced stream lease, never a SQL transaction over Kafka I/O.
// An ACK loss republishes identical bytes/id. Consumers must deduplicate event_id.
func (s *Store) PublishOne(ctx context.Context, publisher EventPublisher) (bool, error) {
	if publisher == nil {
		return false, ErrCapability
	}
	owner := uuid.NewString()
	var stream string
	var epoch, sequence int64
	e := s.db.QueryRow(ctx, `WITH pick AS(SELECT id FROM event_streams WHERE publish_state='ACTIVE' AND published_sequence<last_sequence AND run_after<=clock_timestamp() AND (lease_until IS NULL OR lease_until<clock_timestamp()) ORDER BY run_after,id FOR UPDATE SKIP LOCKED LIMIT 1) UPDATE event_streams s SET lease_owner=$1,lease_epoch=lease_epoch+1,lease_until=clock_timestamp()+interval '15 seconds',cycle_attempts=cycle_attempts+1 FROM pick WHERE s.id=pick.id RETURNING s.id,s.lease_epoch,s.published_sequence+1`, owner).Scan(&stream, &epoch, &sequence)
	if errors.Is(e, pgx.ErrNoRows) {
		return false, nil
	}
	if e != nil {
		return false, e
	}
	var message EventMessage
	var payloadHash string
	message.Stream = stream
	message.Sequence = sequence
	e = s.db.QueryRow(ctx, `SELECT e.event_id,s.topic,s.partition_key,e.payload_bytes,e.payload_hash,e.schema_hash FROM outbox_events e JOIN event_streams s ON s.id=e.stream_id WHERE e.stream_id=$1 AND e.stream_sequence=$2 AND e.publish_status='PENDING'`, stream, sequence).Scan(&message.ID, &message.Topic, &message.Key, &message.Bytes, &payloadHash, &message.SchemaHash)
	var envelope struct {
		ID      string `json:"id"`
		Subject string `json:"subject"`
		Type    string `json:"type"`
	}
	if e == nil && (hash(message.Bytes) != payloadHash || json.Unmarshal(message.Bytes, &envelope) != nil || envelope.ID != message.ID) {
		e = ErrIntegrity
	}
	if e == nil {
		pins := map[string]string{"vn.omama.order.placed.v3": PlacedSchemaHash, "vn.omama.order.paid.v3": "e77a0a724ce30e27c00dc658ffb3818254db595f8de9d8b4c7bc27523f7cda1d", "vn.omama.order.fulfillment_ready.v3": ReadySchemaHash, "vn.omama.order.cancelled.v3": CancelledSchemaHash, "vn.omama.order.completed.v3": CompletedSchemaHash, "vn.omama.payment.receipt_recorded.v1": "bf4374af8940203097b703aa80f3c31a7b8dd3aa528518a9a993e04ffd7ff39a", "vn.omama.payment.refund_settled.v1": "bf4374af8940203097b703aa80f3c31a7b8dd3aa528518a9a993e04ffd7ff39a"}
		subject := "order:"
		topic := "order.events.v3"
		if envelope.Type == "vn.omama.payment.receipt_recorded.v1" {
			subject = "receipt:"
			topic = "payment.events.v1"
		}
		if envelope.Type == "vn.omama.payment.refund_settled.v1" {
			subject = "refund:"
			topic = "payment.events.v1"
		}
		if pins[envelope.Type] == "" || pins[envelope.Type] != message.SchemaHash || envelope.Subject != subject+message.Key || message.Topic != topic {
			e = ErrIntegrity
		}
	}

	if e != nil {
		_, _ = s.db.Exec(ctx, "UPDATE event_streams SET publish_state='BLOCKED',last_error_code='INVALID_OUTBOX_HEAD',lease_owner=NULL,lease_until=NULL WHERE id=$1 AND lease_owner=$2 AND lease_epoch=$3", stream, owner, epoch)
		return true, e
	}
	call, cancel := context.WithTimeout(ctx, 5*time.Second)
	e = publisher.Publish(call, message)
	cancel()
	if e != nil {
		_, _ = s.db.Exec(context.WithoutCancel(ctx), "UPDATE event_streams SET lease_owner=NULL,lease_until=NULL,run_after=clock_timestamp()+interval '1 second',last_error_code='BROKER_UNKNOWN' WHERE id=$1 AND lease_owner=$2 AND lease_epoch=$3", stream, owner, epoch)
		return true, e
	}
	e = s.runner.Do(ctx, func(tx pgx.Tx) error {
		tag, e := tx.Exec(ctx, "UPDATE event_streams SET published_sequence=$4,lease_owner=NULL,lease_until=NULL,last_error_code=NULL WHERE id=$1 AND lease_owner=$2 AND lease_epoch=$3 AND lease_until>clock_timestamp() AND published_sequence=$4-1", stream, owner, epoch, sequence)
		if e != nil {
			return e
		}
		if tag.RowsAffected() != 1 {
			return ErrLease
		}
		_, e = tx.Exec(ctx, "UPDATE outbox_events SET publish_status='PUBLISHED',published_at=clock_timestamp() WHERE event_id=$1", message.ID)
		return e
	})
	return true, e
}
