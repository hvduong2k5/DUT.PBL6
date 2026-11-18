package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Outbox struct {
	db                  *pgxpool.Pool
	lease               time.Duration
	attempts            int
	baseRetry, maxRetry time.Duration
}
type OutboxOptions struct {
	Lease               time.Duration
	MaxAttempts         int
	BaseRetry, MaxRetry time.Duration
}

func NewOutbox(db *pgxpool.Pool, options OutboxOptions) (*Outbox, error) {
	if db == nil || options.Lease < 10*time.Second || options.Lease > 10*time.Minute || options.MaxAttempts < 1 || options.MaxAttempts > 100 || options.BaseRetry < time.Millisecond || options.MaxRetry < options.BaseRetry || options.MaxRetry > time.Hour {
		return nil, ErrInvalidCommand
	}
	return &Outbox{db: db, lease: options.Lease, attempts: options.MaxAttempts, baseRetry: options.BaseRetry, maxRetry: options.MaxRetry}, nil
}

func (s *Outbox) Claim(ctx context.Context) (domain.OutboxMessage, bool, error) {
	token, err := domain.NewUUID()
	if err != nil {
		return domain.OutboxMessage{}, false, err
	}
	// A crashed last attempt must become FAILED, rather than remain invisible
	// forever behind the max-attempt filter. Failed heads block only their profile.
	_, err = s.db.Exec(ctx, `UPDATE outbox_events SET status='FAILED',lease_owner=NULL,lease_until=NULL,error_code='LEASE_EXPIRED_MAX_ATTEMPTS'
  WHERE status='PENDING' AND attempts >= $1 AND (lease_until IS NULL OR lease_until<=clock_timestamp())`, s.attempts)
	if err != nil {
		return domain.OutboxMessage{}, false, err
	}
	var m domain.OutboxMessage
	err = s.db.QueryRow(ctx, `WITH candidate AS (
  SELECT e.id FROM outbox_events e
  WHERE e.status='PENDING' AND e.attempts<$1 AND e.retry_at<=clock_timestamp()
   AND (e.lease_until IS NULL OR e.lease_until<=clock_timestamp())
   AND NOT EXISTS(SELECT 1 FROM outbox_events prior WHERE prior.profile_id=e.profile_id AND prior.status<>'PUBLISHED'
    AND (prior.aggregate_version,prior.created_at,prior.id)<(e.aggregate_version,e.created_at,e.id))
  ORDER BY e.created_at,e.id FOR UPDATE OF e SKIP LOCKED LIMIT 1
 ) UPDATE outbox_events e SET lease_owner=$2,lease_until=clock_timestamp()+$3::double precision*interval '1 second',attempts=attempts+1
 FROM candidate WHERE e.id=candidate.id
 RETURNING e.id::text,e.profile_id::text,e.aggregate_version,e.event_type,e.topic,e.partition_key,e.payload,e.attempts,e.lease_owner::text,e.lease_until`, s.attempts, token, s.lease.Seconds()).Scan(&m.ID, &m.ProfileID, &m.Version, &m.Type, &m.Topic, &m.Key, &m.Payload, &m.Attempts, &m.LeaseToken, &m.LeaseUntil)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.OutboxMessage{}, false, nil
	}
	return m, err == nil, err
}

func (s *Outbox) Ack(ctx context.Context, m domain.OutboxMessage) error {
	if !domain.ValidUUID(m.ID) || !domain.ValidUUID(m.LeaseToken) {
		return ErrInvalidCommand
	}
	tag, err := s.db.Exec(ctx, `UPDATE outbox_events SET status='PUBLISHED',published_at=clock_timestamp(),lease_owner=NULL,lease_until=NULL,error_code=NULL
 WHERE id=$1 AND lease_owner=$2 AND status='PENDING' AND lease_until>clock_timestamp()`, m.ID, m.LeaseToken)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return domain.ErrLeaseLost
	}
	return nil
}

func (s *Outbox) Fail(ctx context.Context, m domain.OutboxMessage, code string) error {
	if !domain.ValidUUID(m.ID) || !domain.ValidUUID(m.LeaseToken) || (code != "BROKER_PUBLISH_FAILED" && code != "DELIVERY_INTERRUPTED") {
		return ErrInvalidCommand
	}
	delay := s.baseRetry
	for i := 1; i < m.Attempts && delay < s.maxRetry; i++ {
		delay *= 2
		if delay > s.maxRetry {
			delay = s.maxRetry
		}
	}
	tag, err := s.db.Exec(ctx, `UPDATE outbox_events SET status=CASE WHEN attempts >= $3 THEN 'FAILED' ELSE 'PENDING' END,
  retry_at=clock_timestamp()+$4::double precision*interval '1 second',lease_owner=NULL,lease_until=NULL,error_code=$5
 WHERE id=$1 AND lease_owner=$2 AND status='PENDING' AND lease_until>clock_timestamp()`, m.ID, m.LeaseToken, s.attempts, delay.Seconds(), code)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return domain.ErrLeaseLost
	}
	return nil
}

func (s *Outbox) Stats(ctx context.Context) (domain.OutboxStats, error) {
	var stats domain.OutboxStats
	err := s.db.QueryRow(ctx, `SELECT count(*) FILTER(WHERE status='PENDING'),count(*) FILTER(WHERE status='FAILED'),
 COALESCE(GREATEST(0,extract(epoch FROM clock_timestamp()-min(created_at) FILTER(WHERE status='PENDING'))),0)::double precision FROM outbox_events`).Scan(&stats.Pending, &stats.Failed, &stats.OldestPendingSeconds)
	return stats, err
}
