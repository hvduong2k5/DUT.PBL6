package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/omamx/order-service/internal/domain/core"
	"time"
)

type AdmissionBarrier struct {
	Order      string    `json:"order_id"`
	Version    int64     `json:"order_version"`
	Generation int64     `json:"generation"`
	Purpose    string    `json:"purpose"`
	NoTask     bool      `json:"no_active_task"`
	Decision   string    `json:"decision_id"`
	Expires    time.Time `json:"expires_at"`
}

func (s *Store) AdmissionHold(ctx context.Context, id, scope, actor, key string, version int64, active bool, proof AdmissionBarrier, stock core.ResourceProof) error {
	if !commandKey(key) || !validID(id) {
		return ErrInput
	}
	body, _ := json.Marshal([]any{id, version, active})
	requestHash := hash(body)
	kind := "CORE_HOLD:" + id
	return s.runner.Do(ctx, func(tx pgx.Tx) error {
		var locked string
		if e := tx.QueryRow(ctx, "SELECT id FROM orders WHERE id=$1 AND scope=$2 AND cohort=$3 FOR UPDATE", id, scope, Cohort).Scan(&locked); e != nil {
			return e
		}
		a, e := s.loadOrder(ctx, tx, id, scope)
		if e != nil {
			return e
		}
		var old string
		e = tx.QueryRow(ctx, "SELECT request_hash FROM command_registry WHERE scope=$1 AND kind=$2 AND key=$3", actor, kind, key).Scan(&old)
		if e == nil {
			if old != requestHash {
				return ErrConflict
			}
			return nil
		}
		if !errors.Is(e, pgx.ErrNoRows) {
			return e
		}
		if a.Version() != version {
			return ErrConflict
		}
		var now time.Time
		var generation int64
		var review, activeCase bool
		if e = tx.QueryRow(ctx, "SELECT clock_timestamp(),ready_generation,has_active_case,EXISTS(SELECT 1 FROM payment_order_projections WHERE order_id=$1 AND has_open_reconciliation) FROM orders WHERE id=$1", id).Scan(&now, &generation, &activeCase, &review); e != nil {
			return e
		}
		purpose := "HOLD"
		if !active {
			purpose = "RESUME"
		}
		if proof.Order != id || proof.Version != version || proof.Generation != generation || proof.Purpose != purpose || !proof.NoTask || proof.Decision == "" || !proof.Expires.After(now) {
			return ErrForbidden
		}
		if a.State() != core.PendingPayment && a.State() != core.Paid && a.State() != core.ConfirmedCOD {
			return core.ErrState
		}
		var next core.Aggregate
		if active {
			next, e = a.Block(core.DecisionContext{ExpectedVersion: version, At: now}, core.HoldBlock)
		} else {
			if review || activeCase || stock.OrderID != id || stock.ReservationID != a.Record().Reservation.ReservationID || stock.ItemsDigest != a.Record().Reservation.ItemsDigest || stock.WarehouseID != a.Record().Reservation.WarehouseID || stock.ResourceVersion < a.Record().Reservation.ResourceVersion {
				return ErrForbidden
			}
			if a.State() == core.PendingPayment {
				if stock.State != "RESERVED" || !stock.ExpiresAt.After(now) || !a.Snapshot().Data().Payment.ExpiresAt.After(now) {
					return ErrExpired
				}
			} else if stock.State != "COMMITTED" {
				return ErrForbidden
			}
			next, e = a.ResumeAdmissionHold(core.DecisionContext{ExpectedVersion: version, At: now}, proof.Order, proof.Decision, proof.NoTask)
		}
		if e != nil {
			return e
		}
		var payment string
		if e = tx.QueryRow(ctx, "SELECT payment_status FROM orders WHERE id=$1", id).Scan(&payment); e != nil {
			return e
		}
		if e = s.savePaymentRoot(ctx, tx, a, next, payment, purpose); e != nil {
			return e
		}
		if !active && next.ReadyEligible() {
			if _, e = tx.Exec(ctx, "UPDATE jobs SET status='READY',lease_owner=NULL,lease_until=NULL,run_after=clock_timestamp() WHERE order_id=$1 AND kind='FULFILLMENT_READY' AND status IN ('DONE','PARKED')", id); e != nil {
				return e
			}
		}
		_, e = tx.Exec(ctx, "INSERT INTO command_registry(scope,kind,key,request_hash,order_id,response) VALUES($1,$2,$3,$4,$5,'{}')", actor, kind, key, requestHash, id)
		return e
	})
}
