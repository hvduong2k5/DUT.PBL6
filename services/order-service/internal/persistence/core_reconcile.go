package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"time"
)

// Reconcile dispositions for a parked checkout operation, decided from the
// authoritative Inventory proof re-queried with the original reserve key.
const (
	reconcileStale      = "STALE"
	reconcileResume     = "RESUME"
	reconcileCompensate = "COMPENSATE"
	reconcileReview     = "REVIEW"
)

// reconcileDisposition maps the remote proof state to the local obligation.
// A reservation that is still RESERVED and unexpired can resume the parked
// checkout; a released/expired (or lapsed) reservation means compensation is
// already complete remotely; anything else (notably an unexpected COMMITTED)
// stays in MANUAL_REVIEW for a human.
func reconcileDisposition(opStatus, proofState string, proofExpired bool) string {
	if opStatus == "SUCCEEDED" || opStatus == "FAILED" {
		return reconcileStale
	}
	switch proofState {
	case "RESERVED":
		if proofExpired {
			return reconcileCompensate
		}
		return reconcileResume
	case "RELEASED", "EXPIRED":
		return reconcileCompensate
	default:
		return reconcileReview
	}
}

type reconcileWork struct {
	LeaseFence
	Workflow string
	Input    []byte
	Attempts int
}

func (r *CheckoutRuntime) claimReconcile(ctx context.Context) (reconcileWork, error) {
	w := reconcileWork{}
	w.Owner = uuid.NewString()
	e := r.Store.db.QueryRow(ctx, `WITH pick AS(SELECT id FROM jobs WHERE cohort=$1 AND kind='RESOURCE_RECONCILE' AND ((status IN ('READY','RETRY_WAIT') AND run_after<=clock_timestamp()) OR(status='LEASED' AND lease_until<clock_timestamp())) ORDER BY run_after,id FOR UPDATE SKIP LOCKED LIMIT 1) UPDATE jobs j SET status='LEASED',lease_owner=$2,lease_epoch=j.lease_epoch+1,lease_until=clock_timestamp()+interval '30 seconds',attempts=j.attempts+1 FROM pick WHERE j.id=pick.id RETURNING j.id,j.lease_epoch,j.workflow_id,j.input_cipher,j.attempts`, Cohort, w.Owner).Scan(&w.JobID, &w.Epoch, &w.Workflow, &w.Input, &w.Attempts)
	return w, e
}

func (r *CheckoutRuntime) fenceReconcile(ctx context.Context, w reconcileWork, fn func(pgx.Tx) error) error {
	return r.Store.runner.Do(ctx, func(tx pgx.Tx) error {
		tag, e := tx.Exec(ctx, "UPDATE jobs SET lease_until=clock_timestamp()+interval '30 seconds' WHERE id=$1 AND lease_owner=$2 AND lease_epoch=$3 AND lease_until>clock_timestamp() AND status='LEASED'", w.JobID, w.Owner, w.Epoch)
		if e != nil {
			return e
		}
		if tag.RowsAffected() != 1 {
			return ErrLease
		}
		return fn(tx)
	})
}

// retryReconcile reschedules a reconcile job after a transient failure
// (for example Inventory Lookup being unavailable). After 20 attempts the
// job parks; the operation stays in MANUAL_REVIEW so the obligation is
// never silently dropped.
func (r *CheckoutRuntime) retryReconcile(ctx context.Context, w reconcileWork) error {
	if w.Attempts >= 20 {
		return r.fenceReconcile(ctx, w, func(tx pgx.Tx) error {
			_, e := tx.Exec(ctx, "UPDATE jobs SET status='PARKED',error_code='RECONCILE_LOOKUP_FAILED',lease_owner=NULL,lease_until=NULL WHERE id=$1", w.JobID)
			return e
		})
	}
	_, e := r.Store.db.Exec(ctx, "UPDATE jobs SET status='RETRY_WAIT',run_after=clock_timestamp()+interval '5 seconds',error_code='RECONCILE_LOOKUP_FAILED',lease_owner=NULL,lease_until=NULL WHERE id=$1 AND lease_owner=$2 AND lease_epoch=$3 AND status='LEASED'", w.JobID, w.Owner, w.Epoch)
	return e
}

// WorkReconcile executes one RESOURCE_RECONCILE job: it re-queries Inventory
// with the original reserve key of a parked checkout operation and settles
// the local obligation from the returned proof (resume, compensated, or
// keep for manual review). Lookup errors are retryable: persistence stays
// decoupled from provider error shapes, and the admin Recover path remains
// available while reconcile retries.
func (r *CheckoutRuntime) WorkReconcile(ctx context.Context) (bool, error) {
	if r.Inventory == nil {
		return false, ErrCapability
	}
	w, e := r.claimReconcile(ctx)
	if errors.Is(e, pgx.ErrNoRows) {
		return false, nil
	}
	if e != nil {
		return false, e
	}
	raw, e := r.Store.cipher.Open("job:"+w.JobID, w.Input)
	if e != nil {
		return true, e
	}
	var input map[string]string
	if json.Unmarshal(raw, &input) != nil || input["operation_id"] == "" {
		return true, ErrIntegrity
	}
	operation, planned := input["operation_id"], input["order_id"]

	var opStatus, opStage, scope string
	var acceptedRaw, canonicalRaw []byte
	var checkoutWorkflow string
	e = r.Store.db.QueryRow(ctx, "SELECT status,stage,scope,input_cipher,canonical_cipher FROM checkout_operations WHERE id=$1 AND cohort=$2", operation, Cohort).Scan(&opStatus, &opStage, &scope, &acceptedRaw, &canonicalRaw)
	if e != nil {
		return true, e
	}
	if e = r.Store.db.QueryRow(ctx, "SELECT id FROM workflows WHERE kind='CHECKOUT' AND aggregate_id=$1", operation).Scan(&checkoutWorkflow); e != nil {
		return true, e
	}
	if opStatus == "SUCCEEDED" || opStatus == "FAILED" {
		return true, r.finishReconcile(ctx, w, reconcileStale, operation, checkoutWorkflow, opStage)
	}
	raw, e = r.Store.cipher.Open("accepted:"+operation+":"+scope, acceptedRaw)
	if e != nil {
		return true, e
	}
	var accepted AcceptedCheckout
	if json.Unmarshal(raw, &accepted) != nil {
		return true, ErrIntegrity
	}
	terms := accepted.Terms
	if canonicalRaw != nil {
		raw, e = r.Store.cipher.Open("canonical:"+operation, canonicalRaw)
		if e != nil {
			return true, e
		}
		if json.Unmarshal(raw, &terms) != nil {
			return true, ErrIntegrity
		}
	}
	terms.OrderID = planned
	proof, e := r.Inventory.Lookup(ctx, terms, "inventory:reserve:"+planned)
	if e != nil {
		_ = r.retryReconcile(context.WithoutCancel(ctx), w)
		return true, e
	}
	var now time.Time
	if e = r.Store.db.QueryRow(ctx, "SELECT clock_timestamp()").Scan(&now); e != nil {
		_ = r.retryReconcile(context.WithoutCancel(ctx), w)
		return true, e
	}
	disposition := reconcileDisposition(opStatus, proof.State, !proof.ExpiresAt.After(now))
	return true, r.finishReconcile(ctx, w, disposition, operation, checkoutWorkflow, opStage)
}

func (r *CheckoutRuntime) finishReconcile(ctx context.Context, w reconcileWork, disposition, operation, checkoutWorkflow, opStage string) error {
	return r.fenceReconcile(ctx, w, func(tx pgx.Tx) error {
		var status string
		if e := tx.QueryRow(ctx, "SELECT status FROM checkout_operations WHERE id=$1 FOR UPDATE", operation).Scan(&status); e != nil {
			return e
		}
		if status == "SUCCEEDED" || status == "FAILED" {
			disposition = reconcileStale
		}
		switch disposition {
		case reconcileResume:
			tag, e := tx.Exec(ctx, "UPDATE jobs SET status='READY',attempts=0,run_after=clock_timestamp(),lease_owner=NULL,lease_until=NULL,lease_epoch=lease_epoch+1 WHERE cohort=$1 AND status='PARKED' AND workflow_id=$2", Cohort, checkoutWorkflow)
			if e != nil {
				return e
			}
			if tag.RowsAffected() != 1 {
				return ErrConflict
			}
			if _, e = tx.Exec(ctx, "UPDATE workflows SET state='RUNNING',stage_started_at=clock_timestamp(),stage_deadline=clock_timestamp()+interval '15 minutes',version=version+1 WHERE id=$1", checkoutWorkflow); e != nil {
				return e
			}
			if _, e = tx.Exec(ctx, "UPDATE checkout_operations SET status='WAITING_RETRY' WHERE id=$1", operation); e != nil {
				return e
			}
		case reconcileCompensate:
			if _, e := tx.Exec(ctx, "UPDATE checkout_operations SET status='FAILED',error_code='CHECKOUT_COMPENSATED',error_http=409,completed_at=clock_timestamp() WHERE id=$1", operation); e != nil {
				return e
			}
			if _, e := tx.Exec(ctx, "UPDATE workflows SET stage='DONE',state='COMPLETED',completion_outcome='COMPENSATED',version=version+1 WHERE id=$1", checkoutWorkflow); e != nil {
				return e
			}
			if _, e := tx.Exec(ctx, "UPDATE jobs SET status='DONE',lease_owner=NULL,lease_until=NULL WHERE workflow_id=$1 AND status='PARKED'", checkoutWorkflow); e != nil {
				return e
			}
			if _, e := tx.Exec(ctx, "UPDATE checkout_submission_guards SET state='RELEASED',version=version+1 WHERE operation_id=$1 AND state='ACTIVE'", operation); e != nil {
				return e
			}
		case reconcileReview:
			if _, e := tx.Exec(ctx, "UPDATE jobs SET error_code='RECONCILE_REVIEW_REQUIRED' WHERE id=$1", w.JobID); e != nil {
				return e
			}
		}
		outcome := "SUCCEEDED"
		if disposition == reconcileCompensate {
			outcome = "COMPENSATED"
		}
		if _, e := tx.Exec(ctx, "UPDATE workflows SET stage='DONE',state='COMPLETED',completion_outcome=$2,version=version+1 WHERE id=$1", w.Workflow, outcome); e != nil {
			return e
		}
		if _, e := tx.Exec(ctx, "UPDATE jobs SET status='DONE',lease_owner=NULL,lease_until=NULL WHERE id=$1", w.JobID); e != nil {
			return e
		}
		return r.Store.audit(ctx, tx, "", "", "CHECKOUT_RECONCILE_"+disposition, map[string]any{"operation_id": operation, "stage": opStage})
	})
}
