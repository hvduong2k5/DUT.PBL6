package persistence

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/omamx/order-service/internal/domain/core"
	"time"
)

type PackingPermit struct {
	ID, Order, Digest, Warehouse, Key, Policy string
	Generation, OrderVersion                  int64
	Expires                                   time.Time
}
type PackingRecord struct {
	Snapshot            core.SnapshotInput
	Stock               core.ResourceProof
	Version, Generation int64
}

// Order is the only authority for eligibility. The permit and its original
// expiry are persisted under the order lock; replays never extend its lifetime.
func (s *Store) PackingInputs(ctx context.Context, id string, generation int64, digest, key, policy string, supplied *PackingPermit) (out PackingRecord, permit PackingPermit, err error) {
	if !validID(id) || generation < 1 || (digest != "" && !validHash(digest)) {
		return out, permit, ErrInput
	}
	generated := uuid.NewString()
	err = s.runner.Do(ctx, func(tx pgx.Tx) error {
		var scope string
		var ready, activeCase bool
		var currentGeneration int64
		if e := tx.QueryRow(ctx, `SELECT scope,ready_authorized,ready_generation,has_active_case FROM orders WHERE id=$1 AND cohort=$2 FOR UPDATE`, id, Cohort).Scan(&scope, &ready, &currentGeneration, &activeCase); e != nil {
			return e
		}
		current, e := s.loadOrder(ctx, tx, id, scope)
		if e != nil {
			return e
		}
		if !ready || activeCase || currentGeneration != generation || !current.ReadyEligible() {
			return core.ErrBlocked
		}
		snapshot := current.Snapshot().Data()
		if digest != "" && digest != string(snapshot.Provenance.ItemsDigest) {
			return ErrIntegrity
		}
		out = PackingRecord{snapshot, current.Record().Reservation, current.Version(), currentGeneration}
		if key == "" && supplied == nil {
			return nil
		}
		if key == "" || len(key) < 16 || len(key) > 128 || policy == "" {
			return ErrInput
		}
		var now time.Time
		if e = tx.QueryRow(ctx, `SELECT clock_timestamp()`).Scan(&now); e != nil {
			return e
		}
		if supplied == nil {
			_, e = tx.Exec(ctx, `INSERT INTO fulfillment_permits(id,order_id,generation,task_request_key,items_digest,warehouse_id,order_version,policy_version,expires_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT(order_id,generation,task_request_key) DO NOTHING`, generated, id, generation, key, string(snapshot.Provenance.ItemsDigest), snapshot.WarehouseID, current.Version(), policy, now.Add(2*time.Minute))
			if e != nil {
				return e
			}
		}
		e = tx.QueryRow(ctx, `SELECT id::text,order_id::text,generation,task_request_key,items_digest,warehouse_id,order_version,policy_version,expires_at FROM fulfillment_permits WHERE order_id=$1 AND generation=$2 AND task_request_key=$3`, id, generation, key).Scan(&permit.ID, &permit.Order, &permit.Generation, &permit.Key, &permit.Digest, &permit.Warehouse, &permit.OrderVersion, &permit.Policy, &permit.Expires)
		if errors.Is(e, pgx.ErrNoRows) {
			return ErrScope
		}
		if e != nil {
			return e
		}
		if permit.Digest != string(snapshot.Provenance.ItemsDigest) || permit.Warehouse != snapshot.WarehouseID || permit.OrderVersion != current.Version() || !now.Add(2*time.Second).Before(permit.Expires) {
			return core.ErrBlocked
		}
		if supplied != nil {
			expected := *supplied
			if !expected.Expires.Equal(permit.Expires) {
				return ErrIntegrity
			}
			expected.Expires = permit.Expires
			if expected != permit {
				return ErrIntegrity
			}
		}
		return nil
	})
	return
}
