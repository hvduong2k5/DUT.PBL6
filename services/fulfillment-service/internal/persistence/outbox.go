package persistence

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
)

type Delivery struct {
	ID, Topic, Key string
	Payload        []byte
	Epoch          int64
}

func (s *Store) ClaimDelivery(ctx context.Context, owner string) (out Delivery, found bool, e error) {
	e = s.DB.QueryRow(ctx, `WITH pick AS (SELECT o.event_id FROM fulfillment.outbox o WHERE o.published_at IS NULL AND o.next_run_at<=clock_timestamp() AND (o.lease_until IS NULL OR o.lease_until<clock_timestamp()) AND NOT EXISTS(SELECT 1 FROM fulfillment.outbox h WHERE h.stream_id=o.stream_id AND h.published_at IS NULL AND (h.stream_version,h.ordinal)<(o.stream_version,o.ordinal)) ORDER BY o.occurred_at,o.event_id FOR UPDATE OF o SKIP LOCKED LIMIT 1) UPDATE fulfillment.outbox o SET lease_owner=$1,lease_epoch=lease_epoch+1,lease_until=clock_timestamp()+interval '15 seconds',attempts=attempts+1 FROM pick WHERE o.event_id=pick.event_id RETURNING o.event_id::text,o.topic,o.partition_key,o.envelope,o.lease_epoch`, owner).Scan(&out.ID, &out.Topic, &out.Key, &out.Payload, &out.Epoch)
	if errors.Is(e, pgx.ErrNoRows) {
		return out, false, nil
	}
	return out, e == nil, e
}
func (s *Store) DeliveryResult(ctx context.Context, owner string, v Delivery, ok bool) error {
	_, e := s.DB.Exec(ctx, `UPDATE fulfillment.outbox SET published_at=CASE WHEN $4 THEN clock_timestamp() ELSE NULL END,lease_until=NULL,next_run_at=clock_timestamp()+interval '5 seconds' WHERE event_id=$1 AND lease_owner=$2 AND lease_epoch=$3`, v.ID, owner, v.Epoch, ok)
	return e
}

type Job struct {
	ID, Kind, CommandKey string
	Input                []byte
	Original             []byte
	Epoch                int64
}

func (s *Store) ClaimJob(ctx context.Context, owner, kind string) (j Job, found bool, e error) {
	e = s.DB.QueryRow(ctx, `WITH pick AS (SELECT job_id FROM fulfillment.jobs WHERE kind=$2 AND state<>'SUCCEEDED' AND next_run_at<=clock_timestamp() AND (lease_until IS NULL OR lease_until<clock_timestamp()) ORDER BY next_run_at FOR UPDATE SKIP LOCKED LIMIT 1) UPDATE fulfillment.jobs j SET state='RUNNING',lease_owner=$1,lease_epoch=lease_epoch+1,lease_until=clock_timestamp()+interval '60 seconds',execution_attempts=execution_attempts+1,updated_at=clock_timestamp() FROM pick WHERE j.job_id=pick.job_id RETURNING j.job_id::text,j.kind,j.input,j.lease_epoch,COALESCE(j.command_key,''),j.original_request`, owner, kind).Scan(&j.ID, &j.Kind, &j.Input, &j.Epoch, &j.CommandKey, &j.Original)
	if errors.Is(e, pgx.ErrNoRows) {
		return j, false, nil
	}
	return j, e == nil, e
}
func (s *Store) FinishJob(ctx context.Context, owner string, j Job, ok bool, code string) error {
	_, e := s.DB.Exec(ctx, `UPDATE fulfillment.jobs SET state=CASE WHEN $4 THEN 'SUCCEEDED' WHEN execution_attempts>=10 THEN 'MANUAL_REVIEW' ELSE 'RETRY_WAIT' END,last_error_code=$5,lease_until=NULL,next_run_at=clock_timestamp()+CASE WHEN execution_attempts>=10 THEN interval '5 minutes' ELSE interval '10 seconds' END,updated_at=clock_timestamp() WHERE job_id=$1 AND lease_owner=$2 AND lease_epoch=$3`, j.ID, owner, j.Epoch, ok, code)
	return e
}
func (s *Store) OutboxAge(ctx context.Context) (float64, error) {
	var seconds float64
	e := s.DB.QueryRow(ctx, `SELECT COALESCE(EXTRACT(EPOCH FROM(clock_timestamp()-min(occurred_at))),0)::double precision FROM fulfillment.outbox WHERE published_at IS NULL`).Scan(&seconds)
	return seconds, e
}
