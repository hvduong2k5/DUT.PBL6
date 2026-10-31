package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/omamx/order-service/internal/domain/core"
	"strconv"
	"time"
)

// ApprovalReader must query Care over an authenticated connection. Values in a
// public refund request are never a substitute for this authoritative snapshot.
type RefundApproval struct {
	WithdrawalReference string `json:"withdrawal_reference,omitempty"`
	ID                  string `json:"approval_id"`
	Receipt             string `json:"receipt_id"`
	Connection          string `json:"connection_id"`
	Receiver            string `json:"receiver_profile_id"`
	Approver            string `json:"approver_id"`
	Amount              int64  `json:"amount"`
	Currency            string `json:"currency"`
	State               string `json:"state"`
	Beneficiary         string `json:"beneficiary_account"`
	Bank                string `json:"beneficiary_bank"`
	Holder              string `json:"beneficiary_holder"`
	Policy              string `json:"policy_version"`
	Version             int64  `json:"version"`
}
type ApprovalReader interface {
	RefundApproval(context.Context, string) (RefundApproval, error)
}

var ErrCapability = errors.New("PAYMENT_CAPABILITY_UNAVAILABLE")
var ErrForbidden = errors.New("PAYMENT_FORBIDDEN")

type PaymentActor struct {
	ID, Connection string
	Receivers      []string
	Permissions    map[string]bool
	MFA            time.Time
}

func (a PaymentActor) allows(p *Payment, permission string, mfa bool) bool {
	if !validID(a.ID) || a.Connection != p.Connection || !a.Permissions[permission] {
		return false
	}
	scope := false
	for _, id := range a.Receivers {
		if id == p.Receiver {
			scope = true
		}
	}
	if !scope {
		return false
	}
	return !mfa || !a.MFA.IsZero() && !a.MFA.After(time.Now().Add(time.Minute)) && time.Since(a.MFA) <= 15*time.Minute
}

type RefundView struct {
	Approval   string     `json:"approval_id"`
	Currency   string     `json:"currency"`
	Mode       string     `json:"execution_mode"`
	AttemptID  *string    `json:"attempt_id"`
	Masked     string     `json:"beneficiary_masked"`
	Evidence   []string   `json:"evidence_ids"`
	Obligation string     `json:"obligation_state"`
	ID         string     `json:"refund_id"`
	Receipt    string     `json:"receipt_id"`
	Order      *string    `json:"order_id"`
	Amount     int64      `json:"amount"`
	Status     string     `json:"status"`
	Execution  string     `json:"execution_state"`
	Version    int64      `json:"version"`
	Attempt    int        `json:"attempt_number"`
	Executor   *string    `json:"claim_owner"`
	ClaimUntil *time.Time `json:"claim_expires_at"`
}

func (p *Payment) AcceptManualRefund(ctx context.Context, reader ApprovalReader, approvalID, key string) (RefundView, error) {
	if !commandKey(key) {
		return RefundView{}, ErrInput
	}
	commandHash := hash([]byte(approvalID))
	commandScope := "care:" + p.Connection
	var oldHash string
	var oldResponse []byte
	e := p.Store.db.QueryRow(ctx, "SELECT request_hash,response FROM command_registry WHERE scope=$1 AND kind='REFUND_ACCEPT' AND key=$2", commandScope, key).Scan(&oldHash, &oldResponse)
	if e == nil {
		if oldHash != commandHash {
			return RefundView{}, ErrConflict
		}
		var oldID string
		if json.Unmarshal(oldResponse, &oldID) != nil {
			return RefundView{}, ErrIntegrity
		}
		return p.refundView(ctx, p.Store.db, oldID)
	}
	if !errors.Is(e, pgx.ErrNoRows) {
		return RefundView{}, e
	}
	if reader == nil {
		return RefundView{}, ErrCapability
	}
	approval, e := reader.RefundApproval(ctx, approvalID)
	if e != nil {
		return RefundView{}, e
	}
	if approval.ID != approvalID || approval.State != "APPROVED" || !validID(approval.Receipt) || approval.Connection != p.Connection || approval.Receiver != p.Receiver || !validID(approval.Approver) || approval.Amount < 1 || approval.Currency != "VND" || len(approval.Beneficiary) < 4 || approval.Holder == "" || approval.Bank == "" || approval.Policy == "" || approval.Version < 1 {
		return RefundView{}, ErrInput
	}
	var order *string
	if e = p.Store.db.QueryRow(ctx, "SELECT order_id FROM receipts WHERE id=$1 AND connection_id=$2 AND receiver_profile_id=$3", approval.Receipt, p.Connection, p.Receiver).Scan(&order); e != nil {
		return RefundView{}, e
	}
	id := uuid.NewString()
	raw, _ := json.Marshal(approval)
	cipher, e := p.Store.cipher.Seal("refund-approval:"+id, raw)
	if e != nil {
		return RefundView{}, e
	}
	e = p.Store.runner.Do(ctx, func(tx pgx.Tx) error {
		var a core.Aggregate
		var scope string
		if _, e := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1,0))", commandScope+":"+key); e != nil {
			return e
		}
		var duplicate []byte
		var priorHash string
		if e := tx.QueryRow(ctx, "SELECT request_hash,response FROM command_registry WHERE scope=$1 AND kind='REFUND_ACCEPT' AND key=$2", commandScope, key).Scan(&priorHash, &duplicate); e == nil {
			if priorHash != commandHash {
				return ErrConflict
			}
			if json.Unmarshal(duplicate, &id) != nil {
				return ErrIntegrity
			}
			return nil
		} else if !errors.Is(e, pgx.ErrNoRows) {
			return e
		}
		if order != nil {
			if e := tx.QueryRow(ctx, "SELECT scope FROM orders WHERE id=$1 AND cohort=$2 FOR UPDATE", *order, Cohort).Scan(&scope); e != nil {
				return e
			}
			var e error
			a, e = p.Store.loadOrder(ctx, tx, *order, scope)
			if e != nil {
				return e
			}
		}
		var amount int64
		if e := tx.QueryRow(ctx, "SELECT amount FROM receipts WHERE id=$1 AND connection_id=$2 AND receiver_profile_id=$3 FOR UPDATE", approval.Receipt, p.Connection, p.Receiver).Scan(&amount); e != nil {
			return e
		}
		var existing, approvalHash string
		e := tx.QueryRow(ctx, "SELECT id,approval_hash FROM refunds WHERE approval_id=$1 AND execution_mode='MANUAL'", approval.ID).Scan(&existing, &approvalHash)
		if e == nil {
			if approvalHash != hash(raw) {
				return ErrConflict
			}
			id = existing
			response, _ := json.Marshal(id)
			_, e = tx.Exec(ctx, `INSERT INTO command_registry(scope,kind,key,request_hash,order_id,response) VALUES($1,'REFUND_ACCEPT',$2,$3,$4,$5)`, commandScope, key, commandHash, order, response)
			return e
		}
		if !errors.Is(e, pgx.ErrNoRows) {
			return e
		}
		var held int64
		if e = tx.QueryRow(ctx, "SELECT COALESCE(sum(amount),0) FROM refunds WHERE receipt_id=$1 AND budget_held", approval.Receipt).Scan(&held); e != nil {
			return e
		}
		if held > amount || approval.Amount > amount-held {
			return ErrConflict
		}
		if _, e = tx.Exec(ctx, `INSERT INTO refunds(id,order_id,receipt_id,approval_id,operation_key,amount,status,execution_mode,execution_state,budget_held,approval_cipher,approval_hash,approver_id) VALUES($1,$2,$3,$4,$5,$6,'MANUAL_REVIEW','MANUAL','AWAITING_EXECUTOR',true,$7,$8,$9)`, id, order, approval.Receipt, approval.ID, "manual-refund:"+approval.ID, approval.Amount, cipher, hash(raw), approval.Approver); e != nil {
			return e
		}
		if order != nil && !a.State().Terminal() {
			var now time.Time
			if e = tx.QueryRow(ctx, "SELECT clock_timestamp()").Scan(&now); e != nil {
				return e
			}
			next, e := a.Block(core.DecisionContext{ExpectedVersion: a.Version(), At: now}, core.RefundBlock|core.HoldBlock)
			if e != nil {
				return e
			}
			payment := "CONFIRMED"
			if a.Record().Allocation.AllocationID == "" {
				payment = "RECONCILIATION_REQUIRED"
			}
			if e = p.Store.savePaymentRoot(ctx, tx, a, next, payment, "REFUND_OBLIGATION"); e != nil {
				return e
			}
		}
		if order != nil {
			if _, e = tx.Exec(ctx, `INSERT INTO payment_order_projections(order_id,payment_hold,has_open_reconciliation,support_reference,support_reason) VALUES($1,true,true,$2,'REFUND_OBLIGATION') ON CONFLICT(order_id) DO UPDATE SET payment_hold=true,has_open_reconciliation=true,version=payment_order_projections.version+1`, *order, uuid.NewString()); e != nil {
				return e
			}
		}
		response, _ := json.Marshal(id)
		if _, e = tx.Exec(ctx, `INSERT INTO command_registry(scope,kind,key,request_hash,order_id,response) VALUES($1,'REFUND_ACCEPT',$2,$3,$4,$5)`, commandScope, key, commandHash, order, response); e != nil {
			return e
		}
		return p.Store.audit(ctx, tx, "", "CARE", "REFUND_ACCEPTED", map[string]any{"refund_id": id, "approval_id": approval.ID})
	})
	if e != nil {
		return RefundView{}, e
	}
	return p.refundView(ctx, p.Store.db, id)
}
func (p *Payment) refundView(ctx context.Context, q rowReader, id string) (v RefundView, e error) {
	var encrypted []byte
	var held bool
	e = q.QueryRow(ctx, `SELECT f.id,f.receipt_id,f.order_id,f.amount,f.status,f.execution_state,f.version,f.current_attempt,f.executor_id::text,f.claim_until,f.approval_id,f.approval_cipher,f.budget_held,(SELECT a.id::text FROM refund_execution_attempts a WHERE a.refund_id=f.id AND a.attempt_no=f.current_attempt),ARRAY(SELECT e.id::text FROM payment_evidence e WHERE e.refund_id=f.id ORDER BY e.created_at,e.id) FROM refunds f JOIN receipts r ON r.id=f.receipt_id WHERE f.id=$1 AND f.execution_mode='MANUAL' AND r.connection_id=$2 AND r.receiver_profile_id=$3`, id, p.Connection, p.Receiver).Scan(&v.ID, &v.Receipt, &v.Order, &v.Amount, &v.Status, &v.Execution, &v.Version, &v.Attempt, &v.Executor, &v.ClaimUntil, &v.Approval, &encrypted, &held, &v.AttemptID, &v.Evidence)
	if e != nil {
		return
	}
	raw, err := p.Store.cipher.Open("refund-approval:"+id, encrypted)
	if err != nil {
		e = err
		return
	}
	var a RefundApproval
	if json.Unmarshal(raw, &a) != nil || len(a.Beneficiary) < 4 {
		e = ErrIntegrity
		return
	}
	v.Masked = "****" + a.Beneficiary[len(a.Beneficiary)-4:]
	v.Currency = "VND"
	v.Mode = "MANUAL_BANK"
	v.Obligation = "OPEN"
	if !held {
		v.Obligation = "CANCELLED"
	}
	if v.Execution == "VERIFIED_SENT" {
		v.Obligation = "FULFILLED"
	}
	if v.Execution == "CANCELLED" {
		v.Execution = "AWAITING_EXECUTOR"
	}
	return
}
func (p *Payment) ReadRefund(ctx context.Context, actor PaymentActor, id string) (RefundView, error) {
	if !actor.allows(p, "payment.refund.read", false) {
		return RefundView{}, ErrForbidden
	}
	return p.refundView(ctx, p.Store.db, id)
}

// RefundCommand is local state only; STARTED is committed before sensitive
// instructions may be shown. Unknown effects never become retryable by timeout.
func (p *Payment) RefundCommand(ctx context.Context, actor PaymentActor, reader ApprovalReader, id, action, key string, version int64, reason string, decisionRefs ...string) (RefundView, error) {
	permission := "payment.refund.execute"
	if action == "cancel-obligation" {
		permission = "payment.refund.accept"
	}
	if !actor.allows(p, permission, action == "start") {
		return RefundView{}, ErrForbidden
	}
	if !commandKey(key) || !validID(id) || reason == "" {
		return RefundView{}, ErrInput
	}
	var approvalID string
	if e := p.Store.db.QueryRow(ctx, "SELECT approval_id FROM refunds WHERE id=$1 AND execution_mode='MANUAL'", id).Scan(&approvalID); e != nil {
		return RefundView{}, e
	}
	var approval RefundApproval
	if action == "start" || action == "retry-attempt" || action == "cancel-obligation" {
		if reader == nil {
			return RefundView{}, ErrCapability
		}
		var e error
		approval, e = reader.RefundApproval(ctx, approvalID)
		if e != nil {
			return RefundView{}, e
		}
	}
	scope := "staff:" + actor.ID
	kind := "MANUAL_REFUND:" + id + ":" + action
	body, _ := json.Marshal([]any{version, reason, decisionRefs})
	requestHash := hash(body)
	var out RefundView
	e := p.Store.runner.Do(ctx, func(tx pgx.Tx) error {
		// Global financial lock order: Order -> receipt -> refund.
		initial, e := p.refundView(ctx, tx, id)
		if e != nil {
			return e
		}
		if initial.Order != nil {
			var locked string
			if e = tx.QueryRow(ctx, "SELECT id FROM orders WHERE id=$1 AND cohort=$2 FOR UPDATE", *initial.Order, Cohort).Scan(&locked); e != nil {
				return e
			}
		}
		var received int64
		if e = tx.QueryRow(ctx, "SELECT amount FROM receipts WHERE id=$1 FOR UPDATE", initial.Receipt).Scan(&received); e != nil {
			return e
		}
		var approver string
		var snapshotCipher []byte
		if e = tx.QueryRow(ctx, "SELECT approver_id,approval_cipher FROM refunds WHERE id=$1 FOR UPDATE", id).Scan(&approver, &snapshotCipher); e != nil {
			return e
		}
		var prior []byte
		var previousHash string
		e = tx.QueryRow(ctx, "SELECT request_hash,response FROM command_registry WHERE scope=$1 AND kind=$2 AND key=$3", scope, kind, key).Scan(&previousHash, &prior)
		if e == nil {
			if previousHash != requestHash {
				return ErrConflict
			}
			if json.Unmarshal(prior, &out) != nil {
				return ErrIntegrity
			}
			return nil
		}
		if !errors.Is(e, pgx.ErrNoRows) {
			return e
		}
		current, e := p.refundView(ctx, tx, id)
		if e != nil {
			return e
		}
		if current.Version != version {
			return ErrConflict
		}
		if current.Obligation != "OPEN" {
			return ErrConflict
		}
		var now time.Time
		if e = tx.QueryRow(ctx, "SELECT clock_timestamp()").Scan(&now); e != nil {
			return e
		}
		if action != "cancel-obligation" && actor.ID == approver {
			return ErrScope
		}
		status, state := current.Status, current.Execution
		executor := current.Executor
		claim := current.ClaimUntil
		attempt := current.Attempt
		held := true
		switch action {
		case "claim":
			if state != "AWAITING_EXECUTOR" && (state != "CLAIMED" || claim == nil || claim.After(now)) {
				return ErrConflict
			}
			who := actor.ID
			executor = &who
			until := now.Add(5 * time.Minute)
			claim = &until
			state = "CLAIMED"
		case "start":
			if state != "CLAIMED" || executor == nil || *executor != actor.ID || claim == nil || !claim.After(now) {
				return ErrConflict
			}
			raw, e := p.Store.cipher.Open("refund-approval:"+id, snapshotCipher)
			if e != nil {
				return e
			}
			var pinned RefundApproval
			if json.Unmarshal(raw, &pinned) != nil {
				return ErrIntegrity
			}
			if approval != pinned || approval.State != "APPROVED" {
				return ErrConflict
			}
			state, status = "STARTED", "SUBMITTED"
			attempt++
			raw, _ = json.Marshal([]any{id, current.Amount, attempt, hash(raw)})
			if _, e = tx.Exec(ctx, `INSERT INTO refund_execution_attempts(id,refund_id,attempt_no,executor_id,operation_key,request_hash) VALUES($1,$2,$3,$4,$5,$6)`, uuid.NewString(), id, attempt, actor.ID, "omp-rf-"+id+"-"+strconv.Itoa(attempt), hash(raw)); e != nil {
				return e
			}
		case "report-uncertain":
			if executor == nil || *executor != actor.ID || (state != "STARTED" && state != "EVIDENCE_PENDING" && state != "UNKNOWN") {
				return ErrConflict
			}
			state, status = "UNKNOWN", "UNKNOWN"
		case "retry-attempt":
			if state != "VERIFIED_NOT_SENT" || approval.State != "APPROVED" || approval.ID != approvalID || approval.Receipt != current.Receipt || approval.Amount != current.Amount {
				return ErrConflict
			}
			state, status = "AWAITING_EXECUTOR", "MANUAL_REVIEW"
			executor = nil
			claim = nil
		case "cancel-obligation":
			if len(decisionRefs) != 1 || decisionRefs[0] == "" || len(decisionRefs[0]) > 128 || approval.WithdrawalReference != decisionRefs[0] {
				return ErrInput
			}
			if (state != "AWAITING_EXECUTOR" && state != "CLAIMED" && state != "VERIFIED_NOT_SENT") || approval.State != "WITHDRAWN" || approval.ID != approvalID || approval.Receipt != current.Receipt {
				return ErrConflict
			}
			state, status, held = "CANCELLED", "FAILED", false
			executor = nil
			claim = nil
		default:
			return ErrInput
		}
		if _, e = tx.Exec(ctx, `UPDATE refunds SET status=$2,execution_state=$3,version=version+1,executor_id=$4,claim_until=$5,current_attempt=$6,budget_held=$7 WHERE id=$1`, id, status, state, executor, claim, attempt, held); e != nil {
			return e
		}
		out, e = p.refundView(ctx, tx, id)
		if e != nil {
			return e
		}
		raw, _ := json.Marshal(out)
		if _, e = tx.Exec(ctx, `INSERT INTO command_registry(scope,kind,key,request_hash,order_id,response) VALUES($1,$2,$3,$4,$5,$6)`, scope, kind, key, requestHash, out.Order, raw); e != nil {
			return e
		}
		return p.Store.audit(ctx, tx, "", actor.ID, "REFUND_"+action, map[string]any{"refund_id": id, "reason": reason, "version": out.Version})
	})
	return out, e
}
