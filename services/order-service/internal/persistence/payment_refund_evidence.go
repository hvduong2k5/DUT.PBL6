package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"time"
)

// EvidenceAuthority is an authenticated Finance/evidence service, not the request
// body or an arbitrary uploaded URL. CLEAN alone is never proof money was sent.
type RefundEvidence struct {
	Reference     string    `json:"reference"`
	Refund        string    `json:"refund_id"`
	Attempt       string    `json:"attempt_id"`
	SHA256        string    `json:"sha256"`
	Source        string    `json:"source"`
	Scan          string    `json:"scan_state"`
	Decision      string    `json:"decision"`
	Definitive    bool      `json:"definitive"`
	Amount        int64     `json:"amount"`
	Currency      string    `json:"currency"`
	Beneficiary   string    `json:"beneficiary_account"`
	Bank          string    `json:"bank"`
	Holder        string    `json:"beneficiary_holder"`
	SourceBank    string    `json:"source_bank_account_id"`
	BankIdentity  string    `json:"bank_identity"`
	BankReference string    `json:"bank_reference"`
	Occurred      time.Time `json:"occurred_at"`
}
type EvidenceAuthority interface {
	RefundEvidence(context.Context, string) (RefundEvidence, error)
}

func (p *Payment) lockRefund(ctx context.Context, tx pgx.Tx, id string) (RefundView, RefundApproval, time.Time, error) {
	v, e := p.refundView(ctx, tx, id)
	if e != nil {
		return v, RefundApproval{}, time.Time{}, e
	}
	if v.Order != nil {
		var order string
		if e = tx.QueryRow(ctx, "SELECT id FROM orders WHERE id=$1 AND cohort=$2 FOR UPDATE", *v.Order, Cohort).Scan(&order); e != nil {
			return v, RefundApproval{}, time.Time{}, e
		}
	}
	var receipt string
	if e = tx.QueryRow(ctx, "SELECT id FROM receipts WHERE id=$1 FOR UPDATE", v.Receipt).Scan(&receipt); e != nil {
		return v, RefundApproval{}, time.Time{}, e
	}
	var encrypted []byte
	var now time.Time
	if e = tx.QueryRow(ctx, "SELECT approval_cipher,clock_timestamp() FROM refunds WHERE id=$1 FOR UPDATE", id).Scan(&encrypted, &now); e != nil {
		return v, RefundApproval{}, now, e
	}
	raw, e := p.Store.cipher.Open("refund-approval:"+id, encrypted)
	if e != nil {
		return v, RefundApproval{}, now, e
	}
	var a RefundApproval
	if json.Unmarshal(raw, &a) != nil {
		return v, a, now, ErrIntegrity
	}
	v, e = p.refundView(ctx, tx, id)
	return v, a, now, e
}
func (p *Payment) VerifyManualRefund(ctx context.Context, actor PaymentActor, authority EvidenceAuthority, id, evidenceID, decision, key string, version int64, reason string, extra ...string) (out RefundView, err error) {
	if !actor.allows(p, "payment.refund.verify", true) {
		return out, ErrForbidden
	}
	if authority == nil {
		return out, ErrCapability
	}
	if !commandKey(key) || !validID(evidenceID) || reason == "" {
		return out, ErrInput
	}
	var encrypted []byte
	if err = p.Store.db.QueryRow(ctx, `SELECT e.evidence_cipher FROM payment_evidence e JOIN refunds f ON f.id=e.refund_id JOIN receipts r ON r.id=f.receipt_id WHERE e.id=$1 AND e.refund_id=$2 AND r.connection_id=$3 AND r.receiver_profile_id=$4`, evidenceID, id, p.Connection, p.Receiver).Scan(&encrypted); err != nil {
		return
	}
	raw, e := p.Store.cipher.Open("refund-evidence:"+evidenceID, encrypted)
	if e != nil {
		return out, e
	}
	var stored RefundEvidence
	if json.Unmarshal(raw, &stored) != nil {
		return out, ErrIntegrity
	}
	latest, e := authority.RefundEvidence(ctx, stored.Reference)
	if e != nil {
		return out, e
	}
	aRaw, _ := json.Marshal(latest)
	if hash(raw) != hash(aRaw) || latest.Scan != "CLEAN" || latest.Refund != id || latest.Decision != decision || latest.Source != "BANK_STATEMENT" {
		return out, ErrInput
	}
	if decision != "SENT" && decision != "NOT_SENT" && decision != "INCONCLUSIVE" {
		return out, ErrInput
	}
	if decision != "INCONCLUSIVE" && !latest.Definitive {
		return out, ErrInput
	}
	request, _ := json.Marshal([]any{version, evidenceID, decision, reason, extra})
	requestHash := hash(request)
	err = p.Store.runner.Do(ctx, func(tx pgx.Tx) error {
		v, approval, now, e := p.lockRefund(ctx, tx, id)
		if e != nil {
			return e
		}
		if v.Executor == nil || actor.ID == *v.Executor || actor.ID == approval.Approver {
			return ErrForbidden
		}
		var prior []byte
		var previous string
		e = tx.QueryRow(ctx, "SELECT request_hash,response FROM command_registry WHERE scope=$1 AND kind=$2 AND key=$3", "staff:"+actor.ID, "REFUND_VERIFY:"+id, key).Scan(&previous, &prior)
		if e == nil {
			if previous != requestHash {
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
		if v.Version != version || (v.Execution != "STARTED" && v.Execution != "EVIDENCE_PENDING" && v.Execution != "UNKNOWN") {
			return ErrConflict
		}
		var attempt string
		var started time.Time
		if e = tx.QueryRow(ctx, "SELECT id,started_at FROM refund_execution_attempts WHERE refund_id=$1 AND attempt_no=$2", id, v.Attempt).Scan(&attempt, &started); e != nil {
			return e
		}
		if latest.Attempt != attempt {
			return ErrConflict
		}
		state, status := "UNKNOWN", "UNKNOWN"
		switch decision {
		case "SENT":
			if latest.BankIdentity == "" || latest.BankReference == "" || latest.Amount != v.Amount || latest.Currency != "VND" || latest.Beneficiary != approval.Beneficiary || latest.Bank != approval.Bank || latest.Holder != approval.Holder || latest.Occurred.Before(started.Add(-time.Second)) || latest.Occurred.After(now.Add(time.Minute)) {
				return ErrInput
			}
			if _, e = tx.Exec(ctx, "INSERT INTO refund_bank_dispositions(connection_id,bank_identity,refund_id) VALUES($1,$2,$3)", p.Connection, latest.BankIdentity, id); e != nil {
				return e
			}
			state, status = "VERIFIED_SENT", "SUCCEEDED"
		case "NOT_SENT":
			state, status = "VERIFIED_NOT_SENT", "FAILED"
		}
		if _, e = tx.Exec(ctx, `INSERT INTO refund_verifications(id,refund_id,attempt_id,verifier_id,result,bank_reference,evidence_id) VALUES($1,$2,$3,$4,$5,$6,$7)`, uuid.NewString(), id, attempt, actor.ID, decision, latest.BankReference, evidenceID); e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, "UPDATE refunds SET execution_state=$2,status=$3,version=version+1 WHERE id=$1", id, state, status); e != nil {
			return e
		}
		if decision == "SENT" {
			if v.Order != nil {
				var allocated, refunded int64
				if e = tx.QueryRow(ctx, `SELECT COALESCE((SELECT amount FROM allocations WHERE order_id=$1 AND receipt_id=$2),0),COALESCE((SELECT sum(f.amount) FROM refunds f JOIN allocations a ON a.receipt_id=f.receipt_id WHERE a.order_id=$1 AND f.status='SUCCEEDED'),0)`, *v.Order, v.Receipt).Scan(&allocated, &refunded); e != nil {
					return e
				}
				if allocated > 0 {
					if refunded > allocated {
						return ErrIntegrity
					}
					payment := "PARTIALLY_REFUNDED"
					if refunded == allocated {
						payment = "REFUNDED"
					}
					if _, e = tx.Exec(ctx, "UPDATE orders SET payment_status=$2 WHERE id=$1", *v.Order, payment); e != nil {
						return e
					}
				}
				if _, e = tx.Exec(ctx, "UPDATE payment_order_projections SET version=version+1 WHERE order_id=$1", *v.Order); e != nil {
					return e
				}
			}
			data := map[string]any{"aggregate_id": id, "version": v.Version + 1, "ordinal": 0, "refund_id": id, "receipt_id": v.Receipt, "order_id": v.Order, "amount": v.Amount, "currency": "VND", "settled_at": latest.Occurred, "evidence_hash": hash(raw)}
			if e = p.Store.paymentEvent(ctx, tx, "REFUND", id, "payment.events.v1", "vn.omama.payment.refund_settled.v1", "https://contracts.omama.example/payment/candidate/events.v1.schema.json", "bf4374af8940203097b703aa80f3c31a7b8dd3aa528518a9a993e04ffd7ff39a", v.Version+1, 0, data, v.Order, now); e != nil {
				return e
			}
		}
		out, e = p.refundView(ctx, tx, id)
		if e != nil {
			return e
		}
		response, _ := json.Marshal(out)
		if _, e = tx.Exec(ctx, `INSERT INTO command_registry(scope,kind,key,request_hash,order_id,response) VALUES($1,$2,$3,$4,$5,$6)`, "staff:"+actor.ID, "REFUND_VERIFY:"+id, key, requestHash, v.Order, response); e != nil {
			return e
		}
		return p.Store.audit(ctx, tx, "", actor.ID, "REFUND_"+decision, map[string]any{"refund_id": id, "evidence_id": evidenceID, "reason": reason})
	})
	return
}

// Refund instructions are never cached in command responses and cannot be
// reissued to a different executor, expired MFA session or UNKNOWN attempt.
func (p *Payment) ManualRefundInstructions(ctx context.Context, actor PaymentActor, reader ApprovalReader, id string) (RefundApproval, error) {
	if !actor.allows(p, "payment.refund.execute", true) {
		return RefundApproval{}, ErrForbidden
	}
	if reader == nil {
		return RefundApproval{}, ErrCapability
	}
	var out RefundApproval
	e := p.Store.runner.Do(ctx, func(tx pgx.Tx) error {
		v, a, _, e := p.lockRefund(ctx, tx, id)
		if e != nil {
			return e
		}
		if v.Executor == nil || *v.Executor != actor.ID || (v.Execution != "STARTED" && v.Execution != "EVIDENCE_PENDING") {
			return ErrForbidden
		}
		out = a
		return nil
	})
	if e != nil {
		return out, e
	}
	current, e := reader.RefundApproval(ctx, out.ID)
	if e != nil {
		return RefundApproval{}, e
	}
	if current != out || current.State != "APPROVED" {
		return RefundApproval{}, ErrConflict
	}
	latest, e := p.refundView(ctx, p.Store.db, id)
	if e != nil {
		return RefundApproval{}, e
	}
	if latest.Executor == nil || *latest.Executor != actor.ID || (latest.Execution != "STARTED" && latest.Execution != "EVIDENCE_PENDING") || !actor.allows(p, "payment.refund.execute", true) {
		return RefundApproval{}, ErrForbidden
	}
	return out, nil
}
