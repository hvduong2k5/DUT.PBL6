package repository

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/hvduong2k5/DUT.PBL6/services/content-service/internal/domain"
	"github.com/jackc/pgx/v5"
)

var _ domain.IdempotencyStore = (*Repository)(nil)

func (r *Repository) Begin(ctx context.Context, principal uuid.UUID, key, hash string) (domain.IdempotencyRecord, bool, error) {
	tag, err := r.pool.Exec(ctx, `INSERT INTO idempotency_keys(principal_id,key,request_hash,status,created_at) VALUES($1,$2,$3,'IN_PROGRESS',now()) ON CONFLICT DO NOTHING`, principal, key, hash)
	if err != nil {
		return domain.IdempotencyRecord{}, false, err
	}
	if tag.RowsAffected() == 1 {
		return domain.IdempotencyRecord{RequestHash: hash}, true, nil
	}
	// A key still IN_PROGRESS after two minutes belongs to a crashed request and
	// may be taken over by a retry with the same payload.
	tag, err = r.pool.Exec(ctx, `UPDATE idempotency_keys SET created_at=now() WHERE principal_id=$1 AND key=$2 AND status='IN_PROGRESS' AND request_hash=$3 AND created_at < now() - interval '2 minutes'`, principal, key, hash)
	if err != nil {
		return domain.IdempotencyRecord{}, false, err
	}
	if tag.RowsAffected() == 1 {
		return domain.IdempotencyRecord{RequestHash: hash}, true, nil
	}
	var rec domain.IdempotencyRecord
	var status string
	var code *int
	var headers []byte
	err = r.pool.QueryRow(ctx, `SELECT request_hash,status,response_status,response_headers,response_body FROM idempotency_keys WHERE principal_id=$1 AND key=$2`, principal, key).Scan(&rec.RequestHash, &status, &code, &headers, &rec.Body)
	if errors.Is(err, pgx.ErrNoRows) {
		// Abandoned between the two statements: the next retry will own it.
		return domain.IdempotencyRecord{RequestHash: hash}, false, nil
	}
	if err != nil {
		return domain.IdempotencyRecord{}, false, err
	}
	rec.Completed = status == "COMPLETED"
	if code != nil {
		rec.Status = *code
	}
	if len(headers) > 0 {
		if err = json.Unmarshal(headers, &rec.Headers); err != nil {
			return domain.IdempotencyRecord{}, false, err
		}
	}
	return rec, false, nil
}
func (r *Repository) Complete(ctx context.Context, principal uuid.UUID, key string, status int, headers map[string]string, body []byte) error {
	raw, err := json.Marshal(headers)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `UPDATE idempotency_keys SET status='COMPLETED',response_status=$3,response_headers=$4,response_body=$5 WHERE principal_id=$1 AND key=$2`, principal, key, status, raw, body)
	return err
}
func (r *Repository) Abandon(ctx context.Context, principal uuid.UUID, key string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM idempotency_keys WHERE principal_id=$1 AND key=$2 AND status='IN_PROGRESS'`, principal, key)
	return err
}
