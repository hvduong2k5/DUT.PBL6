package persistence

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"time"
)

// LocalRefundPolicyVersion identifies approval and evidence records
// issued by this service's own refund authority (F-42 slice 2).
const LocalRefundPolicyVersion = "local-refund-authority/1.0"

// localRefundAuthority is the in-service implementation of the refund
// ApprovalReader and EvidenceAuthority contracts. The original design
// read refund approvals from an external care service (SEPAY_CARE_URL)
// and bank-statement evidence certificates from an external finance
// service (SEPAY_FINANCE_URL); neither was ever deployed, so approvals
// and certificates are recorded in this service's own database instead.
//
// Duty separation is per person, not per role: one FINANCE role carries
// the approve, execute and verify grants, and the lifecycle refuses
// self-dealing at every step. CreateRefundApproval makes the actor the
// approver of record; RefundCommand already refuses execution by the
// approver, VerifyManualRefund refuses a verifier who is the approver
// or the executor, and CreateRefundEvidenceRecord refuses an issuer who
// is either. A certificate is sealed into payment_evidence when the
// executor attaches it; verification re-reads the authority record and
// hash-compares it with the sealed copy, so editing a certificate after
// attachment breaks verification instead of rewriting history. The
// evidence file vault (upload/download) deliberately keeps its external
// contract; certificates here attest a bank statement the issuer checked
// in the bank's own channel, identified by its SHA-256.
type localRefundAuthority struct{ p *Payment }

// LocalRefundApproval returns the approval reader backed by this
// service's own payment_refund_approvals table.
func (p *Payment) LocalRefundApproval() ApprovalReader { return localRefundAuthority{p} }

// LocalRefundEvidence returns the evidence authority backed by this
// service's own payment_refund_evidence_records table.
func (p *Payment) LocalRefundEvidence() EvidenceAuthority { return localRefundAuthority{p} }

func (a localRefundAuthority) RefundApproval(ctx context.Context, ref string) (RefundApproval, error) {
	var v RefundApproval
	var withdrawal *string
	e := a.p.Store.db.QueryRow(ctx, `SELECT approval_id,receipt_id::text,connection_id::text,receiver_profile_id::text,approver_id::text,amount,currency,state,beneficiary_account,beneficiary_bank,beneficiary_holder,policy_version,version,withdrawal_reference FROM payment_refund_approvals WHERE approval_id=$1`, ref).Scan(&v.ID, &v.Receipt, &v.Connection, &v.Receiver, &v.Approver, &v.Amount, &v.Currency, &v.State, &v.Beneficiary, &v.Bank, &v.Holder, &v.Policy, &v.Version, &withdrawal)
	if errors.Is(e, pgx.ErrNoRows) {
		// Mirror the external client's contract: an unknown approval
		// reference is a capability failure, not a caller input error.
		return RefundApproval{}, ErrCapability
	}
	if e != nil {
		return RefundApproval{}, e
	}
	if v.Connection != a.p.Connection || v.Receiver != a.p.Receiver {
		return RefundApproval{}, ErrCapability
	}
	if withdrawal != nil {
		v.WithdrawalReference = *withdrawal
	}
	return v, nil
}

func (a localRefundAuthority) RefundEvidence(ctx context.Context, ref string) (RefundEvidence, error) {
	var v RefundEvidence
	e := a.p.Store.db.QueryRow(ctx, `SELECT reference,refund_id::text,attempt_id::text,sha256,source,scan_state,decision,definitive,amount,currency,beneficiary_account,bank,beneficiary_holder,source_bank_account_id,bank_identity,bank_reference,occurred_at FROM payment_refund_evidence_records WHERE reference=$1`, ref).Scan(&v.Reference, &v.Refund, &v.Attempt, &v.SHA256, &v.Source, &v.Scan, &v.Decision, &v.Definitive, &v.Amount, &v.Currency, &v.Beneficiary, &v.Bank, &v.Holder, &v.SourceBank, &v.BankIdentity, &v.BankReference, &v.Occurred)
	if errors.Is(e, pgx.ErrNoRows) {
		return RefundEvidence{}, ErrCapability
	}
	if e != nil {
		return RefundEvidence{}, e
	}
	return v, nil
}

// RefundApprovalView is the staff-facing shape of an approval. The full
// beneficiary account number is never listed; it is revealed only to the
// current executor through the refund start instruction.
type RefundApprovalView struct {
	ID                  string    `json:"approval_id"`
	Receipt             string    `json:"receipt_id"`
	Order               *string   `json:"order_id"`
	Amount              int64     `json:"amount"`
	Currency            string    `json:"currency"`
	State               string    `json:"state"`
	Masked              string    `json:"beneficiary_masked"`
	Bank                string    `json:"beneficiary_bank"`
	Holder              string    `json:"beneficiary_holder"`
	Approver            string    `json:"approver_id"`
	Policy              string    `json:"policy_version"`
	Version             int64     `json:"version"`
	WithdrawalReference string    `json:"withdrawal_reference,omitempty"`
	Reason              string    `json:"reason"`
	CreatedAt           time.Time `json:"created_at"`
}

func approvalView(a RefundApproval, order *string, created time.Time) RefundApprovalView {
	masked := "****"
	if len(a.Beneficiary) >= 4 {
		masked += a.Beneficiary[len(a.Beneficiary)-4:]
	}
	return RefundApprovalView{ID: a.ID, Receipt: a.Receipt, Order: order, Amount: a.Amount, Currency: a.Currency, State: a.State, Masked: masked, Bank: a.Bank, Holder: a.Holder, Approver: a.Approver, Policy: a.Policy, Version: a.Version, WithdrawalReference: a.WithdrawalReference, CreatedAt: created}
}

// CreateRefundApproval records a manual-refund approval against one
// receipt in this store's scope. The approval reserves nothing by
// itself; the receipt budget is held atomically when the approval is
// accepted into a refund obligation. Creation still refuses amounts
// beyond the receipt's remaining refundable balance (settled refunds,
// held obligations and unconsumed approvals counted) so an approval
// that could never be accepted fails here, not at intake.
func (p *Payment) CreateRefundApproval(ctx context.Context, actor PaymentActor, receiptID string, amount int64, beneficiaryAccount, beneficiaryBank, beneficiaryHolder, reason string) (out RefundApprovalView, err error) {
	if !actor.allows(p, "payment.refund.approve", false) {
		return out, ErrForbidden
	}
	if !validID(receiptID) || amount < 1 || len(beneficiaryAccount) < 4 || beneficiaryBank == "" || beneficiaryHolder == "" || reason == "" || len(reason) > 1000 {
		return out, ErrInput
	}
	err = p.Store.runner.Do(ctx, func(tx pgx.Tx) error {
		var received int64
		var order *string
		if e := tx.QueryRow(ctx, "SELECT amount,order_id FROM receipts WHERE id=$1 AND connection_id=$2 AND receiver_profile_id=$3 FOR UPDATE", receiptID, p.Connection, p.Receiver).Scan(&received, &order); e != nil {
			if errors.Is(e, pgx.ErrNoRows) {
				return ErrInput
			}
			return e
		}
		var committed int64
		if e := tx.QueryRow(ctx, `SELECT COALESCE((SELECT sum(amount) FROM refunds WHERE receipt_id=$1 AND budget_held),0)+COALESCE((SELECT sum(amount) FROM payment_refund_approvals WHERE receipt_id=$1 AND state='APPROVED' AND NOT EXISTS(SELECT 1 FROM refunds WHERE refunds.approval_id=payment_refund_approvals.approval_id)),0)`, receiptID).Scan(&committed); e != nil {
			return e
		}
		if amount > received-committed {
			return ErrConflict
		}
		var ref string
		var created time.Time
		if e := tx.QueryRow(ctx, `INSERT INTO payment_refund_approvals(approval_id,receipt_id,connection_id,receiver_profile_id,approver_id,amount,currency,state,beneficiary_account,beneficiary_bank,beneficiary_holder,policy_version,version,reason) VALUES(gen_random_uuid()::text,$1,$2,$3,$4,$5,'VND','APPROVED',$6,$7,$8,$9,1,$10) RETURNING approval_id,created_at`, receiptID, p.Connection, p.Receiver, actor.ID, amount, beneficiaryAccount, beneficiaryBank, beneficiaryHolder, LocalRefundPolicyVersion, reason).Scan(&ref, &created); e != nil {
			return e
		}
		approval := RefundApproval{ID: ref, Receipt: receiptID, Connection: p.Connection, Receiver: p.Receiver, Approver: actor.ID, Amount: amount, Currency: "VND", State: "APPROVED", Beneficiary: beneficiaryAccount, Bank: beneficiaryBank, Holder: beneficiaryHolder, Policy: LocalRefundPolicyVersion, Version: 1}
		out = approvalView(approval, order, created)
		out.Reason = reason
		return p.Store.audit(ctx, tx, "", actor.ID, "PAYMENT_REFUND_APPROVAL", map[string]any{"approval_id": ref, "receipt_id": receiptID, "amount": amount})
	})
	return
}

// WithdrawRefundApproval marks an approval WITHDRAWN with the reference
// an accept-tier actor must present to cancel the refund obligation.
// Only the original approver may withdraw, and never after the refund
// has been verified as sent.
func (p *Payment) WithdrawRefundApproval(ctx context.Context, actor PaymentActor, approvalID, withdrawalReference, reason string) (out RefundApprovalView, err error) {
	if !actor.allows(p, "payment.refund.approve", false) {
		return out, ErrForbidden
	}
	if withdrawalReference == "" || len(withdrawalReference) > 128 || len(reason) > 1000 {
		return out, ErrInput
	}
	err = p.Store.runner.Do(ctx, func(tx pgx.Tx) error {
		var approver, state string
		if e := tx.QueryRow(ctx, "SELECT approver_id::text,state FROM payment_refund_approvals WHERE approval_id=$1 AND connection_id=$2 AND receiver_profile_id=$3 FOR UPDATE", approvalID, p.Connection, p.Receiver).Scan(&approver, &state); e != nil {
			if errors.Is(e, pgx.ErrNoRows) {
				return ErrInput
			}
			return e
		}
		if approver != actor.ID {
			return ErrForbidden
		}
		if state != "APPROVED" {
			return ErrConflict
		}
		var sent int
		if e := tx.QueryRow(ctx, "SELECT count(*) FROM refunds WHERE approval_id=$1 AND status='SUCCEEDED'", approvalID).Scan(&sent); e != nil {
			return e
		}
		if sent > 0 {
			return ErrConflict
		}
		if _, e := tx.Exec(ctx, "UPDATE payment_refund_approvals SET state='WITHDRAWN',withdrawal_reference=$2,withdrawn_at=clock_timestamp() WHERE approval_id=$1", approvalID, withdrawalReference); e != nil {
			return e
		}
		full, e := localRefundAuthority{p}.approvalFrom(ctx, tx, approvalID)
		if e != nil {
			return e
		}
		out = full
		return p.Store.audit(ctx, tx, "", actor.ID, "PAYMENT_REFUND_APPROVAL_WITHDRAWN", map[string]any{"approval_id": approvalID, "withdrawal_reference": withdrawalReference})
	})
	return
}

func (a localRefundAuthority) approvalFrom(ctx context.Context, q rowReader, approvalID string) (RefundApprovalView, error) {
	var v RefundApproval
	var withdrawal *string
	var order *string
	var created time.Time
	var reason string
	e := q.QueryRow(ctx, `SELECT a.approval_id,a.receipt_id::text,a.connection_id::text,a.receiver_profile_id::text,a.approver_id::text,a.amount,a.currency,a.state,a.beneficiary_account,a.beneficiary_bank,a.beneficiary_holder,a.policy_version,a.version,a.withdrawal_reference,a.reason,a.created_at,r.order_id FROM payment_refund_approvals a JOIN receipts r ON r.id=a.receipt_id WHERE a.approval_id=$1`, approvalID).Scan(&v.ID, &v.Receipt, &v.Connection, &v.Receiver, &v.Approver, &v.Amount, &v.Currency, &v.State, &v.Beneficiary, &v.Bank, &v.Holder, &v.Policy, &v.Version, &withdrawal, &reason, &created, &order)
	if e != nil {
		return RefundApprovalView{}, e
	}
	if withdrawal != nil {
		v.WithdrawalReference = *withdrawal
	}
	out := approvalView(v, order, created)
	out.Reason = reason
	return out, nil
}

// ListRefundApprovals returns this store's approvals, newest first,
// optionally filtered to one receipt.
func (p *Payment) ListRefundApprovals(ctx context.Context, actor PaymentActor, receiptID string) ([]RefundApprovalView, error) {
	if !actor.allows(p, "payment.refund.approve", false) && !actor.allows(p, "payment.refund.read", false) {
		return nil, ErrForbidden
	}
	if receiptID != "" && !validID(receiptID) {
		return nil, ErrInput
	}
	query := `SELECT a.approval_id FROM payment_refund_approvals a WHERE a.connection_id=$1 AND a.receiver_profile_id=$2 ORDER BY a.created_at DESC,a.approval_id LIMIT 100`
	args := []any{p.Connection, p.Receiver}
	if receiptID != "" {
		query = `SELECT a.approval_id FROM payment_refund_approvals a WHERE a.connection_id=$1 AND a.receiver_profile_id=$2 AND a.receipt_id=$3 ORDER BY a.created_at DESC,a.approval_id LIMIT 100`
		args = append(args, receiptID)
	}
	rows, e := p.Store.db.Query(ctx, query, args...)
	if e != nil {
		return nil, e
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if e = rows.Scan(&id); e != nil {
			rows.Close()
			return nil, e
		}
		ids = append(ids, id)
	}
	rows.Close()
	if e = rows.Err(); e != nil {
		return nil, e
	}
	out := []RefundApprovalView{}
	authority := localRefundAuthority{p}
	for _, id := range ids {
		view, e := authority.approvalFrom(ctx, p.Store.db, id)
		if e != nil {
			return nil, e
		}
		out = append(out, view)
	}
	return out, nil
}

// AcceptRefundFromApproval is the staff-side intake for an approval:
// an accept-tier actor who is not the approver turns the approval into
// a refund obligation through the same budget-checked accept path the
// internal service route uses.
func (p *Payment) AcceptRefundFromApproval(ctx context.Context, actor PaymentActor, approvalID, key string) (RefundView, error) {
	if !actor.allows(p, "payment.refund.accept", false) {
		return RefundView{}, ErrForbidden
	}
	approval, e := localRefundAuthority{p}.RefundApproval(ctx, approvalID)
	if e != nil {
		return RefundView{}, e
	}
	if approval.Approver == actor.ID {
		return RefundView{}, ErrForbidden
	}
	return p.AcceptManualRefund(ctx, localRefundAuthority{p}, approvalID, key)
}

// RefundEvidenceInput is the issuer's attestation of one bank statement
// line for one refund attempt. Amount and beneficiary fields state what
// the bank shows; verification compares them against the approval the
// executor acted on, so a mis-keyed certificate fails closed.
type RefundEvidenceInput struct {
	Refund        string    `json:"refund_id"`
	Attempt       string    `json:"attempt_id"`
	Decision      string    `json:"decision"`
	Definitive    bool      `json:"definitive"`
	SHA256        string    `json:"sha256"`
	Amount        int64     `json:"amount"`
	Beneficiary   string    `json:"beneficiary_account"`
	Bank          string    `json:"bank"`
	Holder        string    `json:"beneficiary_holder"`
	BankIdentity  string    `json:"bank_identity"`
	BankReference string    `json:"bank_reference"`
	Occurred      time.Time `json:"occurred_at"`
}

// CreateRefundEvidenceRecord issues a bank-statement certificate for a
// refund attempt. The issuer must be neither the refund's approver nor
// its current executor. The source bank identity is taken from this
// store's receiver profile, never from the request.
func (p *Payment) CreateRefundEvidenceRecord(ctx context.Context, actor PaymentActor, in RefundEvidenceInput) (out RefundEvidence, err error) {
	if !actor.allows(p, "payment.evidence.issue", false) {
		return out, ErrForbidden
	}
	if !validID(in.Refund) || !validID(in.Attempt) || !validHash(in.SHA256) || in.Amount < 1 || len(in.Beneficiary) < 4 || in.Bank == "" || in.Holder == "" || in.BankIdentity == "" || in.BankReference == "" || in.Occurred.IsZero() {
		return out, ErrInput
	}
	if in.Decision != "SENT" && in.Decision != "NOT_SENT" && in.Decision != "INCONCLUSIVE" {
		return out, ErrInput
	}
	if in.Decision != "INCONCLUSIVE" && !in.Definitive {
		return out, ErrInput
	}
	err = p.Store.runner.Do(ctx, func(tx pgx.Tx) error {
		var approver string
		var executor *string
		if e := tx.QueryRow(ctx, `SELECT f.approver_id::text,f.executor_id::text FROM refunds f JOIN receipts r ON r.id=f.receipt_id WHERE f.id=$1 AND r.connection_id=$2 AND r.receiver_profile_id=$3`, in.Refund, p.Connection, p.Receiver).Scan(&approver, &executor); e != nil {
			if errors.Is(e, pgx.ErrNoRows) {
				return ErrInput
			}
			return e
		}
		if approver == actor.ID || (executor != nil && *executor == actor.ID) {
			return ErrForbidden
		}
		var started, now time.Time
		if e := tx.QueryRow(ctx, "SELECT started_at,clock_timestamp() FROM refund_execution_attempts WHERE id=$1 AND refund_id=$2", in.Attempt, in.Refund).Scan(&started, &now); e != nil {
			if errors.Is(e, pgx.ErrNoRows) {
				return ErrInput
			}
			return e
		}
		if in.Occurred.Before(started.Add(-time.Second)) || in.Occurred.After(now.Add(time.Minute)) {
			return ErrInput
		}
		var sourceBank string
		if e := tx.QueryRow(ctx, "SELECT bank_account_uuid::text FROM payment_receiver_profiles WHERE id=$1 AND connection_id=$2", p.Receiver, p.Connection).Scan(&sourceBank); e != nil {
			return e
		}
		var ref string
		if e := tx.QueryRow(ctx, `INSERT INTO payment_refund_evidence_records(reference,refund_id,attempt_id,sha256,source,scan_state,decision,definitive,amount,currency,beneficiary_account,bank,beneficiary_holder,source_bank_account_id,bank_identity,bank_reference,occurred_at,issued_by) VALUES(gen_random_uuid()::text,$1,$2,$3,'BANK_STATEMENT','CLEAN',$4,$5,$6,'VND',$7,$8,$9,$10,$11,$12,$13,$14) RETURNING reference`, in.Refund, in.Attempt, in.SHA256, in.Decision, in.Definitive, in.Amount, in.Beneficiary, in.Bank, in.Holder, sourceBank, in.BankIdentity, in.BankReference, in.Occurred, actor.ID).Scan(&ref); e != nil {
			return e
		}
		if e := p.Store.audit(ctx, tx, "", actor.ID, "PAYMENT_REFUND_EVIDENCE_ISSUED", map[string]any{"refund_id": in.Refund, "evidence_reference": ref, "decision": in.Decision}); e != nil {
			return e
		}
		issued, e := localRefundAuthority{p}.evidenceFrom(ctx, tx, ref)
		if e != nil {
			return e
		}
		out = issued
		return nil
	})
	return
}

func (a localRefundAuthority) evidenceFrom(ctx context.Context, q rowReader, ref string) (RefundEvidence, error) {
	var v RefundEvidence
	e := q.QueryRow(ctx, `SELECT reference,refund_id::text,attempt_id::text,sha256,source,scan_state,decision,definitive,amount,currency,beneficiary_account,bank,beneficiary_holder,source_bank_account_id,bank_identity,bank_reference,occurred_at FROM payment_refund_evidence_records WHERE reference=$1`, ref).Scan(&v.Reference, &v.Refund, &v.Attempt, &v.SHA256, &v.Source, &v.Scan, &v.Decision, &v.Definitive, &v.Amount, &v.Currency, &v.Beneficiary, &v.Bank, &v.Holder, &v.SourceBank, &v.BankIdentity, &v.BankReference, &v.Occurred)
	if e != nil {
		return RefundEvidence{}, e
	}
	return v, nil
}

// ListRefundEvidenceRecords returns the certificates issued for one
// refund, newest first. Full beneficiary values stay server-side; the
// web layer masks them before anything reaches a browser.
func (p *Payment) ListRefundEvidenceRecords(ctx context.Context, actor PaymentActor, refundID string) ([]RefundEvidence, error) {
	if !actor.allows(p, "payment.evidence.issue", false) && !actor.allows(p, "payment.refund.verify", false) && !actor.allows(p, "payment.evidence.write", false) {
		return nil, ErrForbidden
	}
	if !validID(refundID) {
		return nil, ErrInput
	}
	rows, e := p.Store.db.Query(ctx, `SELECT reference FROM payment_refund_evidence_records WHERE refund_id=$1 ORDER BY created_at DESC,reference LIMIT 100`, refundID)
	if e != nil {
		return nil, e
	}
	refs := []string{}
	for rows.Next() {
		var ref string
		if e = rows.Scan(&ref); e != nil {
			rows.Close()
			return nil, e
		}
		refs = append(refs, ref)
	}
	rows.Close()
	if e = rows.Err(); e != nil {
		return nil, e
	}
	out := []RefundEvidence{}
	authority := localRefundAuthority{p}
	for _, ref := range refs {
		record, e := authority.evidenceFrom(ctx, p.Store.db, ref)
		if e != nil {
			return nil, e
		}
		out = append(out, record)
	}
	return out, nil
}
