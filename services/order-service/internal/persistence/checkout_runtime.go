package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/omamx/order-service/internal/domain/core"
	"time"
)

// AcceptedCheckout is encrypted admission evidence, not a public request body.
type AcceptedCheckout struct {
	Terms        core.SnapshotInput `json:"terms"`
	QuoteExpires time.Time          `json:"quote_expires_at"`
}
type CheckoutSource interface {
	Revalidate(context.Context, core.SnapshotInput) error
}

var ErrTermsChanged = errors.New("CHECKOUT_TERMS_CHANGED")

type CheckoutInventory interface {
	Lookup(context.Context, core.SnapshotInput, string) (core.ResourceProof, error)
	Reserve(context.Context, core.SnapshotInput, string) (core.ResourceProof, error)
	Commit(context.Context, core.SnapshotInput, core.ResourceProof, string) (core.ResourceProof, error)
	Release(context.Context, core.SnapshotInput, core.ResourceProof, string) (core.ResourceProof, error)
}
type CheckoutRuntime struct {
	Store     *Store
	Source    CheckoutSource
	Inventory CheckoutInventory
}
type checkoutWork struct {
	LeaseFence
	Operation, Planned, Scope, Stage, Workflow, Method string
	Input, Canonical                                   []byte
	Accepted                                           time.Time
	Attempts                                           int
}

func (r *CheckoutRuntime) claim(ctx context.Context) (w checkoutWork, e error) {
	w.Owner = uuid.NewString()
	e = r.Store.db.QueryRow(ctx, `WITH pick AS(SELECT j.id FROM jobs j JOIN workflows f ON f.id=j.workflow_id WHERE j.cohort=$1 AND f.kind='CHECKOUT' AND f.state<>'COMPLETED' AND ((j.status IN ('READY','RETRY_WAIT') AND j.run_after<=clock_timestamp()) OR(j.status='LEASED' AND j.lease_until<clock_timestamp())) ORDER BY j.run_after,j.id FOR UPDATE OF j SKIP LOCKED LIMIT 1) UPDATE jobs j SET status='LEASED',lease_owner=$2,lease_epoch=j.lease_epoch+1,lease_until=clock_timestamp()+interval '30 seconds',attempts=j.attempts+1 FROM pick,workflows f,checkout_operations c WHERE j.id=pick.id AND f.id=j.workflow_id AND c.id=f.aggregate_id RETURNING j.id,j.lease_epoch,f.id,c.id,c.planned_order_id,c.scope,c.stage,c.input_cipher,c.canonical_cipher,c.accepted_at,c.payment_method,j.attempts`, Cohort, w.Owner).Scan(&w.JobID, &w.Epoch, &w.Workflow, &w.Operation, &w.Planned, &w.Scope, &w.Stage, &w.Input, &w.Canonical, &w.Accepted, &w.Method, &w.Attempts)
	return
}
func (r *CheckoutRuntime) fenced(ctx context.Context, w checkoutWork, fn func(pgx.Tx) error) error {
	return r.Store.runner.Do(ctx, func(tx pgx.Tx) error {
		var id string
		if e := tx.QueryRow(ctx, "SELECT id FROM checkout_operations WHERE id=$1 AND cohort=$2 FOR UPDATE", w.Operation, Cohort).Scan(&id); e != nil {
			return e
		}
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
func (r *CheckoutRuntime) retry(ctx context.Context, w checkoutWork, cause error) error {
	return r.fenced(ctx, w, func(tx pgx.Tx) error {
		status := "WAITING_RETRY"
		if w.Stage == "COMPENSATE" {
			status = "COMPENSATING"
		}
		if w.Attempts >= 20 {
			status = "MANUAL_REVIEW"
		}
		if _, e := tx.Exec(ctx, "UPDATE checkout_operations SET status=$2,error_code='CHECKOUT_DEPENDENCY_RETRY',error_http=503 WHERE id=$1", w.Operation, status); e != nil {
			return e
		}
		if w.Attempts >= 20 {
			if _, e := tx.Exec(ctx, "UPDATE checkout_operations SET status='MANUAL_REVIEW',error_code='CHECKOUT_DEPENDENCY_RETRY',error_http=503 WHERE id=$1", w.Operation); e != nil {
				return e
			}
			if _, e := tx.Exec(ctx, "UPDATE workflows SET state='MANUAL_REVIEW',version=version+1 WHERE id=$1", w.Workflow); e != nil {
				return e
			}
		}
		_, e := tx.Exec(ctx, `UPDATE jobs SET status=CASE WHEN attempts>=20 THEN 'PARKED' ELSE 'RETRY_WAIT' END,run_after=clock_timestamp()+interval '1 second',error_code='CHECKOUT_DEPENDENCY_RETRY',lease_owner=NULL,lease_until=NULL WHERE id=$1`, w.JobID)
		return e
	})
}
func (r *CheckoutRuntime) advance(ctx context.Context, w checkoutWork, stage string, terms core.SnapshotInput, proof *core.ResourceProof) error {
	raw, _ := json.Marshal(terms)
	encrypted, e := r.Store.cipher.Seal("canonical:"+w.Operation, raw)
	if e != nil {
		return e
	}
	return r.fenced(ctx, w, func(tx pgx.Tx) error {
		var reservation any
		var expires any
		if proof != nil {
			reservation = proof.ReservationID
			expires = proof.ExpiresAt
		}
		if _, e := tx.Exec(ctx, `UPDATE checkout_operations SET status='PROCESSING',stage=$2,error_code=NULL,error_http=NULL,canonical_cipher=$3,canonical_hash=$4,reservation_id=COALESCE($5,reservation_id),reservation_expires_at=COALESCE($6,reservation_expires_at) WHERE id=$1`, w.Operation, stage, encrypted, hash(raw), reservation, expires); e != nil {
			return e
		}
		if _, e := tx.Exec(ctx, "UPDATE workflows SET stage=$2,version=version+1 WHERE id=$1", w.Workflow, stage); e != nil {
			return e
		}
		kind := "CHECKOUT_VALIDATE"
		if stage == "PLACE" {
			kind = "ORDER_PLACE"
		}
		_, e := tx.Exec(ctx, "UPDATE jobs SET kind=$2,stage=$3,status='READY',attempts=0,lease_owner=NULL,lease_until=NULL,run_after=clock_timestamp() WHERE id=$1", w.JobID, kind, stage)
		return e
	})
}
func (r *CheckoutRuntime) fail(ctx context.Context, w checkoutWork, reason string) error {
	return r.fenced(ctx, w, func(tx pgx.Tx) error {
		if _, e := tx.Exec(ctx, "UPDATE checkout_operations SET status='FAILED',stage=$3,error_code=$2,error_http=409,completed_at=clock_timestamp() WHERE id=$1", w.Operation, reason, w.Stage); e != nil {
			return e
		}
		outcome := "REJECTED"
		if w.Stage == "COMPENSATE" {
			outcome = "COMPENSATED"
		}
		if _, e := tx.Exec(ctx, "UPDATE workflows SET stage='DONE',state='COMPLETED',completion_outcome=$2,version=version+1 WHERE id=$1", w.Workflow, outcome); e != nil {
			return e
		}
		if _, e := tx.Exec(ctx, "UPDATE checkout_submission_guards SET state='RELEASED',version=version+1 WHERE operation_id=$1 AND state='ACTIVE'", w.Operation); e != nil {
			return e
		}
		_, e := tx.Exec(ctx, "UPDATE jobs SET status='DONE',lease_owner=NULL,lease_until=NULL WHERE id=$1", w.JobID)
		if e != nil {
			return e
		}
		return r.Store.audit(ctx, tx, "", "ORDER_CORE", "CHECKOUT_TERMINAL", map[string]any{"operation_id": w.Operation, "reason": reason, "outcome": outcome})
	})
}
func (r *CheckoutRuntime) Work(ctx context.Context) (bool, error) {
	if r.Source == nil || r.Inventory == nil {
		return false, ErrCapability
	}
	w, e := r.claim(ctx)
	if errors.Is(e, pgx.ErrNoRows) {
		return false, nil
	}
	if e != nil {
		return false, e
	}
	raw, e := r.Store.cipher.Open("accepted:"+w.Operation+":"+w.Scope, w.Input)
	if e != nil {
		return true, e
	}
	var accepted AcceptedCheckout
	if json.Unmarshal(raw, &accepted) != nil {
		return true, ErrIntegrity
	}
	terms := accepted.Terms
	if w.Canonical != nil {
		raw, e = r.Store.cipher.Open("canonical:"+w.Operation, w.Canonical)
		if e != nil {
			return true, e
		}
		if json.Unmarshal(raw, &terms) != nil {
			return true, ErrIntegrity
		}
	}
	switch w.Stage {
	case "VALIDATE":
		e = r.Source.Revalidate(ctx, terms)
		if e != nil {
			if errors.Is(e, ErrTermsChanged) {
				return true, r.fail(ctx, w, "TERMS_CHANGED")
			}
			break
		}
		var now time.Time
		e = r.Store.db.QueryRow(ctx, "SELECT clock_timestamp()").Scan(&now)
		if e != nil {
			break
		}
		// A backward VM/DB clock is transient. Preserve the accepted evidence and
		// retry; it must not be relabelled as invalid customer terms.
		if now.Before(w.Accepted) {
			e = core.ErrDeadline
			break
		}
		if !now.Before(accepted.QuoteExpires) || !now.Add(30*time.Second).Before(terms.Catalog.ValidUntil) || !now.Add(30*time.Second).Before(terms.Shipping.ValidUntil) {
			return true, r.fail(ctx, w, "CHECKOUT_TERMS_EXPIRED")
		}
		terms.OrderID = w.Planned
		terms.OperationID = w.Operation
		terms.SnapshotID = uuid.NewString()
		terms.PlacedAt = now
		terms.Provenance.AcceptedAt = w.Accepted
		terms.Provenance.CanonicalizedAt = now
		if terms.Payment.Method == "VIETQR" {
			expiry := now.Add(15 * time.Minute)
			if terms.Catalog.ValidUntil.Before(expiry) {
				expiry = terms.Catalog.ValidUntil
			}
			if terms.Shipping.ValidUntil.Before(expiry) {
				expiry = terms.Shipping.ValidUntil
			}
			terms.Payment.ExpiresAt = &expiry
		}
		if _, e = core.NewSnapshot(terms); e != nil {
			return true, r.fail(ctx, w, "INVALID_CANONICAL_TERMS")
		}
		return true, r.advance(ctx, w, "RESERVE", terms, nil)
	case "RESERVE":
		var now time.Time
		if e = r.Store.db.QueryRow(ctx, "SELECT clock_timestamp()").Scan(&now); e != nil {
			break
		}
		if !now.Add(30*time.Second).Before(terms.Catalog.ValidUntil) || !now.Add(30*time.Second).Before(terms.Shipping.ValidUntil) || (terms.Payment.ExpiresAt != nil && !now.Add(30*time.Second).Before(*terms.Payment.ExpiresAt)) {
			return true, r.advance(ctx, w, "COMPENSATE", terms, nil)
		}
		var proof core.ResourceProof
		proof, e = r.Inventory.Reserve(ctx, terms, "inventory:reserve:"+w.Planned)
		if e != nil {
			break
		}
		if proof.State != "RESERVED" {
			return true, r.advance(ctx, w, "COMPENSATE", terms, &proof)
		}
		if terms.Payment.ExpiresAt != nil && proof.ExpiresAt.Before(*terms.Payment.ExpiresAt) {
			terms.Payment.ExpiresAt = &proof.ExpiresAt
		}
		stage := "PLACE"
		if terms.Payment.Method == "COD" {
			stage = "COD_FINALIZE"
		}
		return true, r.advance(ctx, w, stage, terms, &proof)
	case "PLACE", "COD_FINALIZE", "COMPENSATE":
		// Re-query the original reserve key after every retry: the remote proof,
		// never a reconstructed local reservation, is the placement authority.
		proof, err := r.Inventory.Lookup(ctx, terms, "inventory:reserve:"+w.Planned)
		if err != nil {
			e = err
			break
		}
		if w.Stage == "COMPENSATE" {
			proof, e = r.Inventory.Release(ctx, terms, proof, "inventory:release:"+w.Planned)
			if e != nil {
				break
			}
			if proof.State != "RELEASED" && proof.State != "EXPIRED" {
				e = ErrIntegrity
				break
			}
			return true, r.fail(ctx, w, "CHECKOUT_COMPENSATED")
		}
		if w.Stage == "COD_FINALIZE" {
			proof, e = r.Inventory.Commit(ctx, terms, proof, "inventory:cod-finalize:"+w.Planned)
			if e != nil {
				break
			}
			if proof.State != "COMMITTED" {
				return true, r.advance(ctx, w, "COMPENSATE", terms, &proof)
			}
			return true, r.advance(ctx, w, "PLACE", terms, &proof)
		}
		if terms.Payment.Method == "COD" {
			proof, e = r.Inventory.Commit(ctx, terms, proof, "inventory:cod-finalize:"+w.Planned)
			if e != nil {
				break
			}
		}
		_, e = r.Store.Place(ctx, Placement{w.Operation, w.LeaseFence, proof})
		if errors.Is(e, core.ErrDeadline) || errors.Is(e, core.ErrSnapshot) || errors.Is(e, core.ErrProof) {
			return true, r.advance(ctx, w, "COMPENSATE", terms, &proof)
		}
		if e == nil {
			return true, nil
		}
	default:
		e = ErrIntegrity
	}
	_ = r.retry(context.WithoutCancel(ctx), w, e)
	return true, e
}
func (r *CheckoutRuntime) Recover(ctx context.Context, id, actor, warehouse, merchant, key, reason string) error {
	if !validID(id) || !commandKey(key) || reason == "" || len(reason) > 1000 {
		return ErrInput
	}
	body, _ := json.Marshal([]string{id, reason})
	requestHash := hash(body)
	return r.Store.runner.Do(ctx, func(tx pgx.Tx) error {
		var state, scope string
		var raw []byte
		if e := tx.QueryRow(ctx, "SELECT status,scope,input_cipher FROM checkout_operations WHERE id=$1 AND cohort=$2 FOR UPDATE", id, Cohort).Scan(&state, &scope, &raw); e != nil {
			return e
		}
		clear, e := r.Store.cipher.Open("accepted:"+id+":"+scope, raw)
		if e != nil {
			return e
		}
		var accepted AcceptedCheckout
		if json.Unmarshal(clear, &accepted) != nil {
			return ErrIntegrity
		}
		if accepted.Terms.MerchantID != merchant || warehouse != "" && accepted.Terms.WarehouseID != warehouse {
			return ErrForbidden
		}
		var prior string
		e = tx.QueryRow(ctx, "SELECT request_hash FROM command_registry WHERE scope=$1 AND kind=$2 AND key=$3", actor, "CHECKOUT_RECOVER:"+id, key).Scan(&prior)
		if e == nil {
			if prior != requestHash {
				return ErrConflict
			}
			return nil
		}
		if !errors.Is(e, pgx.ErrNoRows) {
			return e
		}
		if state != "MANUAL_REVIEW" {
			return ErrConflict
		}
		tag, e := tx.Exec(ctx, "UPDATE jobs SET status='READY',attempts=0,run_after=clock_timestamp(),lease_owner=NULL,lease_until=NULL,lease_epoch=lease_epoch+1 WHERE cohort=$1 AND status='PARKED' AND workflow_id IN(SELECT id FROM workflows WHERE kind='CHECKOUT' AND aggregate_id=$2)", Cohort, id)
		if e != nil {
			return e
		}
		if tag.RowsAffected() != 1 {
			return ErrConflict
		}
		if _, e = tx.Exec(ctx, "UPDATE workflows SET state='RUNNING',version=version+1 WHERE kind='CHECKOUT' AND aggregate_id=$1", id); e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, "UPDATE checkout_operations SET status='WAITING_RETRY' WHERE id=$1", id); e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, "INSERT INTO command_registry(scope,kind,key,request_hash,response) VALUES($1,$2,$3,$4,'{}')", actor, "CHECKOUT_RECOVER:"+id, key, requestHash); e != nil {
			return e
		}
		return r.Store.audit(ctx, tx, "", actor, "CHECKOUT_RECOVER", map[string]any{"operation_id": id, "reason": reason})
	})
}

func (s *Store) ReadAcceptance(ctx context.Context, id, scope string) (Acceptance, error) {
	var a Acceptance
	e := s.db.QueryRow(ctx, `SELECT c.id,c.planned_order_id,o.id,c.status,c.accepted_at FROM checkout_operations c LEFT JOIN orders o ON o.operation_id=c.id WHERE c.id=$1 AND c.scope=$2 AND c.cohort=$3`, id, scope, Cohort).Scan(&a.OperationID, &a.PlannedOrderID, &a.OrderID, &a.Status, &a.AcceptedAt)
	return a, e
}
