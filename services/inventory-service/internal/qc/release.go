// Package qc implements the privileged, audited warehouse QC release job.
package qc

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"strings"
	"time"
)

type Request struct {
	Actor, Key, Batch, Warehouse, Manufactured, Expires, Reason string
	Version                                                     int64
}
type Result struct {
	Batch   string
	Version int64
	Status  string
}

func Release(ctx context.Context, db *sql.DB, r Request) (Result, error) {
	var result Result
	for _, id := range []string{r.Actor, r.Batch, r.Warehouse} {
		if n, e := uuid.Parse(id); e != nil || n == uuid.Nil {
			return result, fmt.Errorf("valid actor, batch and warehouse UUIDs required")
		}
	}
	m, em := time.Parse(time.DateOnly, r.Manufactured)
	e, ee := time.Parse(time.DateOnly, r.Expires)
	today := time.Now().UTC().Format(time.DateOnly)
	if em != nil || ee != nil || !m.Before(e) || r.Version < 1 || len(r.Key) < 16 || len(r.Key) > 128 || strings.TrimSpace(r.Reason) == "" || len(r.Reason) > 1000 {
		return result, fmt.Errorf("invalid QC dates, version, reason or idempotency key")
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return result, err
	}
	defer tx.Rollback()
	var authorized bool
	if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='inventory_qc_operator' AND pg_has_role(current_user,oid,'MEMBER'))`).Scan(&authorized); err != nil {
		return result, err
	}
	if !authorized {
		return result, fmt.Errorf("QC requires inventory_qc_operator database role membership")
	}
	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,993))`, r.Actor+":"+r.Key); err != nil {
		return result, err
	}
	raw, _ := json.Marshal(r)
	var matches bool
	var response []byte
	err = tx.QueryRowContext(ctx, `SELECT request=$3::jsonb,response FROM inventory_qc_releases WHERE actor_id=$1 AND idem_key=$2`, r.Actor, r.Key, string(raw)).Scan(&matches, &response)
	if err == nil {
		if !matches {
			return result, fmt.Errorf("QC idempotency conflict")
		}
		err = json.Unmarshal(response, &result)
		return result, err
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return result, err
	}
	if r.Manufactured > today || r.Expires <= today {
		return result, fmt.Errorf("QC dates must describe a currently usable batch")
	}
	var sku string
	if err = tx.QueryRowContext(ctx, `SELECT sku FROM batches WHERE id=$1`, r.Batch).Scan(&sku); err != nil {
		return result, err
	}
	var warehouse string
	if err = tx.QueryRowContext(ctx, `SELECT warehouse_id::text FROM inventory_items WHERE sku=$1 FOR UPDATE`, sku).Scan(&warehouse); err != nil {
		return result, err
	}
	if warehouse != r.Warehouse {
		return result, fmt.Errorf("QC warehouse mismatch")
	}
	var version int64
	var status string
	if err = tx.QueryRowContext(ctx, `SELECT version,status FROM batches WHERE id=$1 AND sku=$2 FOR UPDATE`, r.Batch, sku).Scan(&version, &status); err != nil {
		return result, err
	}
	if version != r.Version || status != "QUARANTINE" {
		return result, fmt.Errorf("QC revision or state conflict")
	}
	result.Batch = r.Batch
	if err = tx.QueryRowContext(ctx, `UPDATE batches SET mfg_date=$2,exp_date=$3,status=CASE WHEN $3::date<=CURRENT_DATE+45 THEN 'NEAR_EXPIRY' ELSE 'ACTIVE' END WHERE id=$1 RETURNING version,status`, r.Batch, r.Manufactured, r.Expires).Scan(&result.Version, &result.Status); err != nil {
		return result, err
	}
	response, _ = json.Marshal(result)
	if _, err = tx.ExecContext(ctx, `INSERT INTO inventory_qc_releases(actor_id,idem_key,batch_id,request,response,reason,warehouse_id) VALUES($1,$2,$3,$4,$5,$6,$7)`, r.Actor, r.Key, r.Batch, string(raw), string(response), r.Reason, r.Warehouse); err != nil {
		return result, err
	}
	err = tx.Commit()
	return result, err
}
