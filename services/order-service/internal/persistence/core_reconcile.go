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
	reconcileRelease    = "RELEASE"
	reconcileCompensate = "COMPENSATE"
	reconcileReview     = "REVIEW"
)

// reconcileDisposition maps the remote proof state to the local obligation.
//
//   - An operation that is no longer in MANUAL_REVIEW is owned by someone
//     else (admin Recover, a terminal transition): the proof is stale.
//   - RESERVED and still inside its stage window resumes the parked job,
//     without granting it a fresh attempt budget or stage window.
//   - RESERVED whose stage window has already closed stays in MANUAL_REVIEW:
//     extending the window is a policy grant that belongs to the audited
//     admin Recover path, not to the reconciler.
//   - RESERVED with a lapsed reservation expiry is NOT proof of compensation:
//     the reconciler must first run Release with the original key and only
//     close the obligation on an authoritative RELEASED/EXPIRED proof.
//   - RELEASED/EXPIRED proofs close the obligation as compensated.
//   - Anything else (notably an unexpected COMMITTED) stays with a human.
func reconcileDisposition(opStatus, opError, proofState string, proofExpired, stageExpired bool) string {
	if opStatus != "MANUAL_REVIEW" {
		return reconcileStale
	}
	switch proofState {
	case "RESERVED":
		if proofExpired {
			return reconcileRelease
		}
		if stageExpired || opError == "STAGE_DEADLINE_EXCEEDED" {
			return reconcileReview
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
// (for example Inventory Lookup/Release being unavailable). After 20
// attempts the job parks; the operation stays in MANUAL_REVIEW so the
// obligation is never silently dropped.
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
// the local obligation from the returned proof. The proof is applied only
// after re-verifying, under lock, that the operation is still in
// MANUAL_REVIEW with its checkout job still PARKED (see finishReconcile),
// because an admin Recover may have resumed the checkout in the meantime.
// Provider calls happen outside any SQL transaction.
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

	var opStatus, opStage, opError, scope string
	var acceptedRaw, canonicalRaw []byte
	var checkoutWorkflow string
	var stageDeadline *time.Time
	e = r.Store.db.QueryRow(ctx, "SELECT c.status,c.stage,c.error_code,c.scope,c.input_cipher,c.canonical_cipher,f.id,f.stage_deadline FROM checkout_operations c JOIN workflows f ON f.kind='CHECKOUT' AND f.aggregate_id=c.id WHERE c.id=$1 AND c.cohort=$2", operation, Cohort).Scan(&opStatus, &opStage, &opError, &scope, &acceptedRaw, &canonicalRaw, &checkoutWorkflow, &stageDeadline)
	if e != nil {
		return true, e
	}
	if opStatus != "MANUAL_REVIEW" {
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
	disposition := reconcileDisposition(opStatus, opError, proof.State, !proof.ExpiresAt.After(now), stageDeadlineExceeded(stageDeadline, now))
	if disposition == reconcileRelease {
		// A lapsed RESERVED proof is not compensation evidence. Run the
		// release with the original key and only close the obligation on
		// the authoritative outcome.
		proof, e = r.Inventory.Release(ctx, terms, proof, "inventory:release:"+planned)
		if e != nil {
			_ = r.retryReconcile(context.WithoutCancel(ctx), w)
			return true, e
		}
		if proof.State != "RELEASED" && proof.State != "EXPIRED" {
			_ = r.retryReconcile(context.WithoutCancel(ctx), w)
			return true, ErrIntegrity
		}
		disposition = reconcileCompensate
	}
	return true, r.finishReconcile(ctx, w, disposition, operation, checkoutWorkflow, opStage)
}

func (r *CheckoutRuntime) finishReconcile(ctx context.Context, w reconcileWork, disposition, operation, checkoutWorkflow, opStage string) error {
	return r.fenceReconcile(ctx, w, func(tx pgx.Tx) error {
		// Re-verify ownership under lock before applying the proof: the
		// operation must still be in MANUAL_REVIEW, its workflow must
		// still be in MANUAL_REVIEW, and the checkout job must still be
		// PARKED. An admin Recover (or any other transition) in the
		// meantime makes the proof stale.
		var status string
		if e := tx.QueryRow(ctx, "SELECT status FROM checkout_operations WHERE id=$1 FOR UPDATE", operation).Scan(&status); e != nil {
			return e
		}
		var wfState string
		if e := tx.QueryRow(ctx, "SELECT state FROM workflows WHERE id=$1 FOR UPDATE", checkoutWorkflow).Scan(&wfState); e != nil {
			return e
		}
		parked := 0
		if status == "MANUAL_REVIEW" && wfState == "MANUAL_REVIEW" {
			if e := tx.QueryRow(ctx, "SELECT count(*) FROM jobs WHERE workflow_id=$1 AND status='PARKED'", checkoutWorkflow).Scan(&parked); e != nil {
				return e
			}
		}
		if status != "MANUAL_REVIEW" || wfState != "MANUAL_REVIEW" || (disposition != reconcileReview && parked != 1) {
			disposition = reconcileStale
		}
		switch disposition {
		case reconcileResume:
			// Resume grants no new budget: the job keeps its attempt count
			// and its stage window. Only the audited admin Recover path
			// resets attempts and extends the window.
			tag, e := tx.Exec(ctx, "UPDATE jobs SET status='READY',run_after=clock_timestamp(),lease_owner=NULL,lease_until=NULL,lease_epoch=lease_epoch+1 WHERE cohort=$1 AND status='PARKED' AND workflow_id=$2", Cohort, checkoutWorkflow)
			if e != nil {
				return e
			}
			if tag.RowsAffected() != 1 {
				return ErrConflict
			}
			if _, e = tx.Exec(ctx, "UPDATE workflows SET state='RUNNING',version=version+1 WHERE id=$1", checkoutWorkflow); e != nil {
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
