package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/notification-service/internal/domain"
)

type Store struct{ Pool *pgxpool.Pool }

func (s Store) Ping(ctx context.Context) error { return s.Pool.Ping(ctx) }
func (s Store) Get(ctx context.Context, producer, id string) (domain.Request, error) {
	if _, e := uuid.Parse(id); e != nil {
		return domain.Request{}, domain.ErrNotFound
	}
	var r domain.Request
	err := s.Pool.QueryRow(ctx, "SELECT id::text,status,created_at FROM notification_requests WHERE id=$1 AND producer_id=$2", id, producer).Scan(&r.ID, &r.Status, &r.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return r, domain.ErrNotFound
	}
	return r, err
}
func (s Store) Accept(ctx context.Context, producer, key string, c domain.Command) (domain.Result, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return domain.Result{}, err
	}
	defer tx.Rollback(ctx)
	// Lock command identity first and business intent second; no external I/O in TX.
	for _, identity := range []string{"key:" + producer + ":" + key, "intent:" + c.Intent()} {
		if _, err = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1,0))", identity); err != nil {
			return domain.Result{}, err
		}
	}
	var r domain.Result
	var fp string
	err = tx.QueryRow(ctx, `SELECT r.id::text,r.status,r.created_at,k.fingerprint FROM idempotency_records k JOIN notification_requests r ON r.id=k.request_id WHERE k.producer_id=$1 AND k.command_kind='INTAKE' AND k.key=$2`, producer, key).Scan(&r.ID, &r.Status, &r.CreatedAt, &fp)
	if err == nil {
		if fp != c.Fingerprint() {
			return s.conflict(ctx, tx, producer, "key:"+key, c.Fingerprint())
		}
		r.Duplicate = true
		return r, tx.Commit(ctx)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return r, err
	}
	err = tx.QueryRow(ctx, "SELECT id::text,status,created_at,fingerprint FROM notification_requests WHERE intent_key=$1", c.Intent()).Scan(&r.ID, &r.Status, &r.CreatedAt, &fp)
	if err == nil {
		if fp != c.Fingerprint() {
			return s.conflict(ctx, tx, producer, c.Intent(), c.Fingerprint())
		}
		r.Duplicate = true
	} else if errors.Is(err, pgx.ErrNoRows) {
		r.ID = uuid.NewString()
		r.Status = "WAITING_CONTEXT"
		body, _ := json.Marshal(c)
		err = tx.QueryRow(ctx, `INSERT INTO notification_requests(id,producer_id,intent_key,fingerprint,command,status) VALUES($1,$2,$3,$4,$5,$6) RETURNING created_at`, r.ID, producer, c.Intent(), c.Fingerprint(), body, r.Status).Scan(&r.CreatedAt)
		if err != nil {
			return r, err
		}
		if _, err = tx.Exec(ctx, "INSERT INTO audit_outbox(id,request_id,action,actor) VALUES($1,$2,'REQUEST_ACCEPTED',$3)", uuid.NewString(), r.ID, producer); err != nil {
			return r, err
		}
	} else {
		return r, err
	}
	_, err = tx.Exec(ctx, "INSERT INTO idempotency_records(producer_id,key,fingerprint,request_id) VALUES($1,$2,$3,$4)", producer, key, c.Fingerprint(), r.ID)
	if err != nil {
		return r, err
	}
	return r, tx.Commit(ctx)
}
func (s Store) conflict(ctx context.Context, tx pgx.Tx, producer, identity, fp string) (domain.Result, error) {
	_, err := tx.Exec(ctx, "INSERT INTO event_conflicts(producer_id,identity_hash,conflicting_hash) VALUES($1,encode(sha256($2::bytea),'hex'),$3) ON CONFLICT DO NOTHING", producer, []byte(identity), fp)
	if err != nil {
		return domain.Result{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.Result{}, err
	}
	return domain.Result{}, domain.ErrConflict
}
