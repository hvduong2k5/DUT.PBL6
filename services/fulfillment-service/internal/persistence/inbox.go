package persistence

import (
	"context"
	"crypto/sha256"
	"fmt"
	"github.com/google/uuid"
	d "github.com/omamx/fulfillment-service/internal/domain"
)

func (s *Store) IngestReady(ctx context.Context, raw, key []byte, topic string, partition int, offset int64) error {
	event, invalid := d.ParseReady(raw, key)
	hash := fmt.Sprintf("%x", sha256.Sum256(raw))
	source, id := event.Source, event.ID
	if source == "" {
		source = "unverified"
	}
	if id == "" {
		id = hash
	}
	disposition := "ACCEPTED"
	reason := ""
	if invalid != nil {
		disposition = "QUARANTINED"
		reason = "INVALID_READY_EVENT"
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(context.Background())
	result, e := tx.Exec(ctx, `INSERT INTO fulfillment.inbox(source,event_id,payload_hash,payload,topic,partition_id,offset_value,disposition,reason_code,received_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,clock_timestamp()) ON CONFLICT DO NOTHING`, source, id, hash, raw, topic, partition, offset, disposition, reason)
	if e != nil {
		return e
	}
	if result.RowsAffected() == 0 {
		var previous string
		if e = tx.QueryRow(ctx, `SELECT payload_hash FROM fulfillment.inbox WHERE source=$1 AND event_id=$2`, source, id).Scan(&previous); e != nil {
			return e
		}
		if previous != hash {
			_, e = tx.Exec(ctx, `INSERT INTO fulfillment.inbox_conflicts(source,event_id,payload_hash,payload) VALUES($1,$2,$3,$4) ON CONFLICT DO NOTHING`, source, id, hash, raw)
			if e != nil {
				return e
			}
		}
		return tx.Commit(ctx)
	}
	if invalid == nil {
		_, e = tx.Exec(ctx, `INSERT INTO fulfillment.jobs(job_id,kind,dedupe_key,state,stage,order_id,input,command_key,next_run_at,created_at,updated_at) VALUES($1,'READY_ADMISSION',$2,'PENDING','ACCEPT',$3,$4,$5,clock_timestamp(),clock_timestamp(),clock_timestamp()) ON CONFLICT(dedupe_key) DO NOTHING`, uuid.NewString(), fmt.Sprintf("ready:%s:%d", event.Data.Order, event.Data.Generation), event.Data.Order, marshal(event), "ready-"+uuid.NewString())
		if e != nil {
			return e
		}
	}
	return tx.Commit(ctx)
}
func (s *Store) SaveJobRequest(ctx context.Context, owner string, j Job, request []byte) error {
	result, e := s.DB.Exec(ctx, `UPDATE fulfillment.jobs SET original_request=$4 WHERE job_id=$1 AND lease_owner=$2 AND lease_epoch=$3 AND original_request IS NULL`, j.ID, owner, j.Epoch, request)
	if e != nil {
		return e
	}
	if result.RowsAffected() != 1 {
		return d.Fail("STALE_JOB_LEASE", 409)
	}
	return nil
}
