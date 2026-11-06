package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"time"
)

func (p *Payment) CheckRefundAttempt(ctx context.Context, actor PaymentActor, id, attempt string) error {
	if !actor.allows(p, "payment.refund.execute", false) && !actor.allows(p, "payment.refund.verify", true) {
		return ErrForbidden
	}
	v, e := p.refundView(ctx, p.Store.db, id)
	if e != nil {
		return e
	}
	if v.AttemptID == nil || *v.AttemptID != attempt {
		return ErrConflict
	}
	return nil
}
func (p *Payment) AttachRefundEvidenceBatch(ctx context.Context, actor PaymentActor, authority EvidenceAuthority, id, attempt, key string, version int64, reason string, refs []string) (out RefundView, err error) {
	if !actor.allows(p, "payment.evidence.write", false) {
		return out, ErrForbidden
	}
	if authority == nil {
		return out, ErrCapability
	}
	if !commandKey(key) || len(refs) < 1 || len(refs) > 100 || reason == "" || len(reason) > 1000 {
		return out, ErrInput
	}
	initial, e := p.refundView(ctx, p.Store.db, id)
	if e != nil {
		return out, e
	}
	if initial.Executor == nil || *initial.Executor != actor.ID {
		return out, ErrForbidden
	}
	certificates := []RefundEvidence{}
	seen := map[string]bool{}
	for _, ref := range refs {
		if !validID(ref) || seen[ref] {
			return out, ErrInput
		}
		seen[ref] = true
		certificate, e := authority.RefundEvidence(ctx, ref)
		if e != nil {
			return out, e
		}
		if certificate.Reference != ref || certificate.Refund != id || certificate.Attempt != attempt || !validHash(certificate.SHA256) || certificate.Scan != "CLEAN" {
			return out, ErrInput
		}
		certificates = append(certificates, certificate)
	}
	request, _ := json.Marshal([]any{version, reason, attempt, refs})
	requestHash := hash(request)
	kind := "REFUND_EVIDENCE:" + id
	err = p.Store.runner.Do(ctx, func(tx pgx.Tx) error {
		v, a, _, e := p.lockRefund(ctx, tx, id)
		if e != nil {
			return e
		}
		if v.Executor == nil || *v.Executor != actor.ID || actor.ID == a.Approver {
			return ErrForbidden
		}
		var previous string
		var prior []byte
		e = tx.QueryRow(ctx, "SELECT request_hash,response FROM command_registry WHERE scope=$1 AND kind=$2 AND key=$3", "staff:"+actor.ID, kind, key).Scan(&previous, &prior)
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
		if v.Version != version || v.AttemptID == nil || *v.AttemptID != attempt || (v.Execution != "STARTED" && v.Execution != "EVIDENCE_PENDING" && v.Execution != "UNKNOWN") {
			return ErrConflict
		}
		for _, certificate := range certificates {
			raw, _ := json.Marshal(certificate)
			cipher, e := p.Store.cipher.Seal("refund-evidence:"+certificate.Reference, raw)
			if e != nil {
				return e
			}
			tag, e := tx.Exec(ctx, `INSERT INTO payment_evidence(id,refund_id,attempt_id,uploader,source,sha256,evidence_cipher) VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT(id) DO NOTHING`, certificate.Reference, id, attempt, actor.ID, certificate.Source, certificate.SHA256, cipher)
			if e != nil {
				return e
			}
			if tag.RowsAffected() == 0 {
				var oldCipher []byte
				if e = tx.QueryRow(ctx, "SELECT evidence_cipher FROM payment_evidence WHERE id=$1 AND refund_id=$2 AND attempt_id=$3", certificate.Reference, id, attempt).Scan(&oldCipher); e != nil {
					return e
				}
				old, e := p.Store.cipher.Open("refund-evidence:"+certificate.Reference, oldCipher)
				if e != nil {
					return e
				}
				if hash(old) != hash(raw) {
					return ErrConflict
				}
			}
		}
		if _, e = tx.Exec(ctx, `UPDATE refunds SET execution_state=CASE WHEN execution_state='UNKNOWN' THEN execution_state ELSE 'EVIDENCE_PENDING' END,version=version+1 WHERE id=$1`, id); e != nil {
			return e
		}
		out, e = p.refundView(ctx, tx, id)
		if e != nil {
			return e
		}
		response, _ := json.Marshal(out)
		if _, e = tx.Exec(ctx, `INSERT INTO command_registry(scope,kind,key,request_hash,order_id,response) VALUES($1,$2,$3,$4,$5,$6)`, "staff:"+actor.ID, kind, key, requestHash, out.Order, response); e != nil {
			return e
		}
		return p.Store.audit(ctx, tx, "", actor.ID, "REFUND_EVIDENCE_ATTACHED", map[string]any{"refund_id": id, "evidence_ids": refs, "reason": reason})
	})
	return
}

type RefundBankEvidence struct {
	Source          string    `json:"source_bank_account_id"`
	Reference       string    `json:"outgoing_reference"`
	Amount          int64     `json:"amount"`
	Currency        string    `json:"currency"`
	BeneficiaryHash string    `json:"beneficiary_snapshot_hash"`
	At              time.Time `json:"confirmed_at"`
}
type checkedRefundEvidence struct {
	source       EvidenceAuthority
	bank         string
	presentation *RefundBankEvidence
}

func (c checkedRefundEvidence) RefundEvidence(ctx context.Context, ref string) (RefundEvidence, error) {
	certificate, e := c.source.RefundEvidence(ctx, ref)
	if e != nil {
		return certificate, e
	}
	if c.presentation != nil {
		b := c.presentation
		raw, _ := json.Marshal(map[string]string{"account_number": certificate.Beneficiary, "bank_code": certificate.Bank, "account_holder": certificate.Holder})
		if b.Source != c.bank || certificate.SourceBank != c.bank || b.Reference != certificate.BankReference || b.Amount != certificate.Amount || b.Currency != certificate.Currency || b.BeneficiaryHash != hash(raw) || !b.At.Equal(certificate.Occurred) {
			return RefundEvidence{}, ErrInput
		}
	}
	return certificate, nil
}
func (p *Payment) VerifyRefundEvidenceSet(ctx context.Context, actor PaymentActor, authority EvidenceAuthority, id, attempt, decision, key string, version int64, reason string, refs []string, bank *RefundBankEvidence) (RefundView, error) {
	if !actor.allows(p, "payment.refund.verify", true) {
		return RefundView{}, ErrForbidden
	}
	if authority == nil {
		return RefundView{}, ErrCapability
	}
	if len(refs) < 1 || len(refs) > 100 || !validID(attempt) {
		return RefundView{}, ErrInput
	}
	if (decision == "SENT") != (bank != nil) {
		return RefundView{}, ErrInput
	}
	seen := map[string]bool{}
	for _, ref := range refs {
		if !validID(ref) || seen[ref] {
			return RefundView{}, ErrInput
		}
		seen[ref] = true
	}
	if e := p.CheckRefundAttempt(ctx, actor, id, attempt); e != nil {
		return RefundView{}, e
	}
	var count int
	if e := p.Store.db.QueryRow(ctx, "SELECT count(*) FROM payment_evidence WHERE refund_id=$1 AND attempt_id=$2 AND id=ANY($3::uuid[])", id, attempt, refs).Scan(&count); e != nil {
		return RefundView{}, e
	}
	if count != len(refs) {
		return RefundView{}, ErrInput
	}
	var sourceBank string
	if e := p.Store.db.QueryRow(ctx, "SELECT bank_account_uuid FROM payment_receiver_profiles WHERE id=$1 AND connection_id=$2", p.Receiver, p.Connection).Scan(&sourceBank); e != nil {
		return RefundView{}, e
	}
	if bank != nil && (!validID(bank.Source) || bank.Reference == "" || bank.Amount < 1 || bank.Currency != "VND" || !validHash(bank.BeneficiaryHash) || bank.At.IsZero()) {
		return RefundView{}, ErrInput
	}
	request, _ := json.Marshal([]any{attempt, refs, bank})
	return p.VerifyManualRefund(ctx, actor, checkedRefundEvidence{authority, sourceBank, bank}, id, refs[0], decision, key, version, reason, string(request))
}

type RefundStartInstruction struct {
	Refund   RefundView `json:"refund"`
	Attempt  string     `json:"attempt_id"`
	Amount   int64      `json:"amount"`
	Currency string     `json:"currency"`
	Bank     string     `json:"bank_code"`
	Account  string     `json:"beneficiary_account"`
	Holder   string     `json:"beneficiary_holder"`
	Content  string     `json:"transfer_content"`
	Started  time.Time  `json:"started_at"`
}

func (p *Payment) RefundStartInstruction(ctx context.Context, actor PaymentActor, reader ApprovalReader, id string) (RefundStartInstruction, error) {
	a, e := p.ManualRefundInstructions(ctx, actor, reader, id)
	if e != nil {
		return RefundStartInstruction{}, e
	}
	v, e := p.refundView(ctx, p.Store.db, id)
	if e != nil {
		return RefundStartInstruction{}, e
	}
	if v.AttemptID == nil {
		return RefundStartInstruction{}, ErrIntegrity
	}
	var at time.Time
	var key string
	if e = p.Store.db.QueryRow(ctx, "SELECT started_at,operation_key FROM refund_execution_attempts WHERE id=$1 AND refund_id=$2", *v.AttemptID, id).Scan(&at, &key); e != nil {
		return RefundStartInstruction{}, e
	}
	return RefundStartInstruction{v, *v.AttemptID, a.Amount, "VND", a.Bank, a.Beneficiary, a.Holder, key, at}, nil
}
