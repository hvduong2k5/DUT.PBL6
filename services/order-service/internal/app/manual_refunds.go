package app

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/omamx/order-service/internal/domain"
	"strings"
	"time"
)

type RefundPayee struct {
	Bank         string `json:"bank_code"`
	Account      string `json:"account_number"`
	Name         string `json:"account_name"`
	Verification string `json:"verification_reference"`
}
type ManualRefundInput struct {
	ReceiptID string      `json:"receipt_id"`
	Amount    int64       `json:"amount"`
	Reason    string      `json:"reason"`
	Payee     RefundPayee `json:"payee"`
}
type RefundEvidence struct {
	Bank          string    `json:"source_bank_code"`
	Account       string    `json:"source_account_number"`
	TransactionID string    `json:"bank_transaction_id"`
	Amount        int64     `json:"amount"`
	PaidAt        time.Time `json:"paid_at"`
	Statement     string    `json:"statement_reference"`
}
type ManualRefund struct {
	ID           string          `json:"refund_id"`
	ReceiptID    string          `json:"receipt_id"`
	OrderID      *string         `json:"order_id"`
	Amount       int64           `json:"amount"`
	Status       string          `json:"status"`
	ApprovedBy   string          `json:"approved_by"`
	ClaimedBy    *string         `json:"claimed_by"`
	VerifiedBy   *string         `json:"verified_by"`
	EvidenceHash *string         `json:"evidence_hash"`
	CreatedAt    time.Time       `json:"created_at"`
	Payee        *RefundPayee    `json:"payee,omitempty"`
	Evidence     *RefundEvidence `json:"evidence,omitempty"`
}

const manualRefundColumns = "id,receipt_id,order_id,amount,status,approved_by,claimed_by,verified_by,evidence_hash,created_at"

func scanManualRefund(row pgx.Row) (ManualRefund, error) {
	var f ManualRefund
	e := row.Scan(&f.ID, &f.ReceiptID, &f.OrderID, &f.Amount, &f.Status, &f.ApprovedBy, &f.ClaimedBy, &f.VerifiedBy, &f.EvidenceHash, &f.CreatedAt)
	return f, e
}
func bounded(v string, max int) bool { return strings.TrimSpace(v) != "" && len(v) <= max }
func validCommandKey(v string) bool {
	if len(v) < 16 || len(v) > 128 {
		return false
	}
	for _, c := range v {
		if c < 33 || c > 126 {
			return false
		}
	}
	return true
}
func (s *Service) manualAccess(p domain.Principal) error {
	if !s.MVP() {
		return domain.E("MANUAL_REFUND_DISABLED", 409)
	}
	if !p.Manager() {
		return domain.E("PERMISSION_DENIED", 403)
	}
	return nil
}
func (s *Service) manualReceiptAccess(ctx context.Context, tx pgx.Tx, p domain.Principal, orderID *string) error {
	if orderID == nil {
		if p.Role != "ADMIN" {
			return domain.E("PERMISSION_DENIED", 403)
		}
		return nil
	}
	o, e := s.lockOrder(ctx, tx, *orderID)
	if e != nil {
		return e
	}
	if !CanRead(p, o) {
		return domain.E("NOT_FOUND", 404)
	}
	return nil
}

func (s *Service) ApproveManualRefund(ctx context.Context, p domain.Principal, key string, in ManualRefundInput) (ManualRefund, error) {
	if e := s.manualAccess(p); e != nil {
		return ManualRefund{}, e
	}
	if !validCommandKey(key) {
		return ManualRefund{}, domain.E("INVALID_IDEMPOTENCY_KEY", 400)
	}
	if _, e := uuid.Parse(in.ReceiptID); e != nil {
		return ManualRefund{}, domain.E("INVALID_RECEIPT_ID", 400)
	}
	if in.Amount <= 0 || !bounded(in.Reason, 500) || !bounded(in.Payee.Bank, 32) || !bounded(in.Payee.Account, 64) || !bounded(in.Payee.Name, 255) || !bounded(in.Payee.Verification, 255) {
		return ManualRefund{}, domain.E("INVALID_REFUND", 422)
	}
	hash := domain.Hash(in)
	requestKey := p.Scope() + ":" + key
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return ManualRefund{}, e
	}
	defer tx.Rollback(ctx)
	if _, e = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1,0))", "manual-refund-key:"+requestKey); e != nil {
		return ManualRefund{}, e
	}
	// First find the immutable receipt->Order relation, then lock in Order->receipt order.
	var orderID *string
	if e = tx.QueryRow(ctx, "SELECT order_id FROM receipts WHERE id=$1", in.ReceiptID).Scan(&orderID); e != nil {
		return ManualRefund{}, e
	}
	if e = s.manualReceiptAccess(ctx, tx, p, orderID); e != nil {
		return ManualRefund{}, e
	}
	var amount int64
	if e = tx.QueryRow(ctx, "SELECT amount FROM receipts WHERE id=$1 AND provider_status='SUCCEEDED' FOR UPDATE", in.ReceiptID).Scan(&amount); e != nil {
		return ManualRefund{}, e
	}
	var oldHash string
	e = tx.QueryRow(ctx, "SELECT request_hash FROM manual_refunds WHERE request_key=$1", requestKey).Scan(&oldHash)
	if e == nil {
		if oldHash != hash {
			return ManualRefund{}, domain.E("IDEMPOTENCY_CONFLICT", 409)
		}
		return scanManualRefund(tx.QueryRow(ctx, "SELECT "+manualRefundColumns+" FROM manual_refunds WHERE request_key=$1", requestKey))
	}
	if !errors.Is(e, pgx.ErrNoRows) {
		return ManualRefund{}, e
	}
	if in.Amount != amount {
		return ManualRefund{}, domain.E("FULL_RECEIPT_REFUND_ONLY", 422)
	}
	var allocated bool
	if e = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM allocations WHERE receipt_id=$1)", in.ReceiptID).Scan(&allocated); e != nil {
		return ManualRefund{}, e
	}
	if allocated {
		var allowed bool
		e = tx.QueryRow(ctx, `SELECT status IN ('CANCELLED_TIMEOUT','CANCELLED_BY_USER','CANCELLED_BY_ADMIN') AND stock_status IN ('RELEASED','EXPIRED') AND NOT ready_authorized AND NOT EXISTS(SELECT 1 FROM jobs WHERE order_id=orders.id AND kind IN ('FINALIZE','CANCEL') AND status NOT IN ('DONE','REJECTED')) FROM orders WHERE id=$1`, *orderID).Scan(&allowed)
		if e != nil {
			return ManualRefund{}, e
		}
		if !allowed {
			return ManualRefund{}, domain.E("ORDER_REFUND_NOT_SAFE", 409)
		}
	}
	var held int64
	e = tx.QueryRow(ctx, `SELECT (SELECT COALESCE(sum(amount),0) FROM refunds WHERE receipt_id=$1 AND status<>'FAILED')+(SELECT COALESCE(sum(amount),0) FROM manual_refunds WHERE receipt_id=$1)`, in.ReceiptID).Scan(&held)
	if e != nil {
		return ManualRefund{}, e
	}
	if in.Amount > amount-held {
		return ManualRefund{}, domain.E("REFUND_BUDGET_EXCEEDED", 409)
	}
	sealed, e := s.Seal(in.Payee)
	if e != nil {
		return ManualRefund{}, e
	}
	id := uuid.NewString()
	_, e = tx.Exec(ctx, `INSERT INTO manual_refunds(id,receipt_id,order_id,request_key,request_hash,amount,approved_by,reason,payee_cipher) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, id, in.ReceiptID, orderID, requestKey, hash, in.Amount, p.ID, in.Reason, sealed)
	if e != nil {
		return ManualRefund{}, e
	}
	oid := ""
	if orderID != nil {
		oid = *orderID
	}
	if e = s.Audit(ctx, tx, oid, p.ID, "MANUAL_REFUND_APPROVED", map[string]any{"refund_id": id, "receipt_id": in.ReceiptID, "amount": in.Amount, "request_hash": hash}); e != nil {
		return ManualRefund{}, e
	}
	f, e := scanManualRefund(tx.QueryRow(ctx, "SELECT "+manualRefundColumns+" FROM manual_refunds WHERE id=$1", id))
	if e != nil {
		return f, e
	}
	return f, tx.Commit(ctx)
}

// All manual transitions use Order->receipt->manual_refund lock order, also
// serializing budget changes with legacy refund acceptance.
func (s *Service) changeManualRefund(ctx context.Context, p domain.Principal, id string, fn func(pgx.Tx, *ManualRefund) error) (ManualRefund, error) {
	if e := s.manualAccess(p); e != nil {
		return ManualRefund{}, e
	}
	if _, e := uuid.Parse(id); e != nil {
		return ManualRefund{}, domain.E("NOT_FOUND", 404)
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return ManualRefund{}, e
	}
	defer tx.Rollback(ctx)
	var orderID *string
	var receipt string
	if e = tx.QueryRow(ctx, "SELECT order_id,receipt_id FROM manual_refunds WHERE id=$1", id).Scan(&orderID, &receipt); e != nil {
		return ManualRefund{}, e
	}
	if e = s.manualReceiptAccess(ctx, tx, p, orderID); e != nil {
		return ManualRefund{}, e
	}
	if _, e = tx.Exec(ctx, "SELECT id FROM receipts WHERE id=$1 FOR UPDATE", receipt); e != nil {
		return ManualRefund{}, e
	}
	f, e := scanManualRefund(tx.QueryRow(ctx, "SELECT "+manualRefundColumns+" FROM manual_refunds WHERE id=$1 FOR UPDATE", id))
	if e != nil {
		return f, e
	}
	if e = fn(tx, &f); e != nil {
		return ManualRefund{}, e
	}
	f, e = scanManualRefund(tx.QueryRow(ctx, "SELECT "+manualRefundColumns+" FROM manual_refunds WHERE id=$1", id))
	if e != nil {
		return f, e
	}
	return f, tx.Commit(ctx)
}
func (s *Service) ClaimManualRefund(ctx context.Context, p domain.Principal, id string) (ManualRefund, error) {
	return s.changeManualRefund(ctx, p, id, func(tx pgx.Tx, f *ManualRefund) error {
		if f.ClaimedBy != nil {
			if *f.ClaimedBy == p.ID {
				return nil
			}
			return domain.E("REFUND_ALREADY_CLAIMED", 409)
		}
		if f.Status != "APPROVED" {
			return domain.E("INVALID_REFUND_STATE", 409)
		}
		_, e := tx.Exec(ctx, "UPDATE manual_refunds SET claimed_by=$2,status='SUBMITTED',updated_at=clock_timestamp() WHERE id=$1", id, p.ID)
		if e != nil {
			return e
		}
		return s.Audit(ctx, tx, "", p.ID, "MANUAL_REFUND_CLAIMED", map[string]string{"refund_id": id})
	})
}
func (s *Service) MarkManualRefundUnknown(ctx context.Context, p domain.Principal, id string) (ManualRefund, error) {
	return s.changeManualRefund(ctx, p, id, func(tx pgx.Tx, f *ManualRefund) error {
		if f.ClaimedBy == nil || *f.ClaimedBy != p.ID || f.Status == "SUCCEEDED" {
			return domain.E("REFUND_NOT_OWNED_OR_TERMINAL", 409)
		}
		_, e := tx.Exec(ctx, "UPDATE manual_refunds SET status='UNKNOWN',updated_at=clock_timestamp() WHERE id=$1", id)
		if e != nil {
			return e
		}
		return s.Audit(ctx, tx, "", p.ID, "MANUAL_REFUND_UNKNOWN", map[string]string{"refund_id": id})
	})
}
func (s *Service) SubmitManualRefundEvidence(ctx context.Context, p domain.Principal, id string, in RefundEvidence) (ManualRefund, error) {
	if !bounded(in.Bank, 32) || !bounded(in.Account, 64) || !bounded(in.TransactionID, 128) || !bounded(in.Statement, 255) || in.PaidAt.IsZero() || in.PaidAt.After(time.Now().Add(time.Minute)) {
		return ManualRefund{}, domain.E("INVALID_REFUND_EVIDENCE", 422)
	}
	return s.changeManualRefund(ctx, p, id, func(tx pgx.Tx, f *ManualRefund) error {
		if f.ClaimedBy == nil || *f.ClaimedBy != p.ID {
			return domain.E("REFUND_NOT_OWNED", 409)
		}
		if in.Amount != f.Amount {
			return domain.E("REFUND_EVIDENCE_AMOUNT_MISMATCH", 422)
		}
		hash := domain.Hash(in)
		if f.EvidenceHash != nil {
			if *f.EvidenceHash == hash {
				return nil
			}
			return domain.E("EVIDENCE_CONFLICT", 409)
		}
		if f.Status != "SUBMITTED" && f.Status != "UNKNOWN" {
			return domain.E("INVALID_REFUND_STATE", 409)
		}
		sealed, e := s.Seal(in)
		if e != nil {
			return e
		}
		settlement := domain.Hash([]string{strings.ToUpper(strings.TrimSpace(in.Bank)), strings.TrimSpace(in.Account), strings.TrimSpace(in.TransactionID)})
		_, e = tx.Exec(ctx, "UPDATE manual_refunds SET evidence_cipher=$2,evidence_hash=$3,settlement_key=$4,updated_at=clock_timestamp() WHERE id=$1", id, sealed, hash, settlement)
		var pgerr *pgconn.PgError
		if errors.As(e, &pgerr) && pgerr.Code == "23505" {
			return domain.E("SETTLEMENT_ALREADY_USED", 409)
		}
		if e != nil {
			return e
		}
		return s.Audit(ctx, tx, "", p.ID, "MANUAL_REFUND_EVIDENCE", map[string]string{"refund_id": id, "evidence_hash": hash})
	})
}
func (s *Service) ConfirmManualRefund(ctx context.Context, p domain.Principal, id, hash string) (ManualRefund, error) {
	return s.changeManualRefund(ctx, p, id, func(tx pgx.Tx, f *ManualRefund) error {
		if f.EvidenceHash == nil || *f.EvidenceHash != hash {
			return domain.E("REFUND_EVIDENCE_REQUIRED", 409)
		}
		if f.ClaimedBy == nil || *f.ClaimedBy == p.ID {
			return domain.E("INDEPENDENT_VERIFIER_REQUIRED", 403)
		}
		if f.Status == "SUCCEEDED" {
			return nil
		}
		_, e := tx.Exec(ctx, "UPDATE manual_refunds SET status='SUCCEEDED',verified_by=$2,updated_at=clock_timestamp() WHERE id=$1", id, p.ID)
		if e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, "UPDATE reconciliation_cases SET status='RESOLVED' WHERE receipt_id=$1", f.ReceiptID); e != nil {
			return e
		}
		var allocated bool
		if e = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM allocations WHERE receipt_id=$1)", f.ReceiptID).Scan(&allocated); e != nil {
			return e
		}
		if allocated && f.OrderID != nil {
			o, e := s.lockOrder(ctx, tx, *f.OrderID)
			if e != nil {
				return e
			}
			o.Payment = "REFUNDED"
			o.Ready = false
			if e = s.touch(ctx, tx, &o, "MANUAL_REFUND_VERIFIED"); e != nil {
				return e
			}
		}
		// This is a manual accounting fact, not a fabricated provider v3 refund proof.
		event := map[string]any{"specversion": "1.0", "id": uuid.NewString(), "source": "https://omama.vn/services/order-service", "type": "vn.omama.order.manual_refund_verified.v1", "subject": "refund:" + id, "time": time.Now().UTC().Format(time.RFC3339Nano), "datacontenttype": "application/json", "data": map[string]any{"refund_id": id, "receipt_id": f.ReceiptID, "order_id": f.OrderID, "amount": f.Amount, "currency": "VND", "execution_mode": "MANUAL_BANK_TRANSFER", "evidence_hash": hash, "status": "SUCCEEDED"}}
		payload, e := jsonMarshal(event)
		if e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, `INSERT INTO outbox(id,aggregate_id,aggregate_type,version,ordinal,event_type,topic,payload) VALUES($1,$2,'MANUAL_REFUND',1,0,$3,'order.manual_refunds.v1',$4)`, event["id"], id, event["type"], payload); e != nil {
			return e
		}
		return s.Audit(ctx, tx, "", p.ID, "MANUAL_REFUND_VERIFIED", map[string]string{"refund_id": id, "evidence_hash": hash})
	})
}

func (s *Service) GetManualRefund(ctx context.Context, p domain.Principal, id string) (ManualRefund, error) {
	if e := s.manualAccess(p); e != nil {
		return ManualRefund{}, e
	}
	if _, e := uuid.Parse(id); e != nil {
		return ManualRefund{}, domain.E("NOT_FOUND", 404)
	}
	f, e := scanManualRefund(s.DB.QueryRow(ctx, "SELECT "+manualRefundColumns+" FROM manual_refunds WHERE id=$1", id))
	if e != nil {
		return f, e
	}
	if f.OrderID == nil {
		if p.Role != "ADMIN" {
			return ManualRefund{}, domain.E("NOT_FOUND", 404)
		}
	} else {
		if _, e = s.GetOrder(ctx, *f.OrderID, p, false); e != nil {
			return ManualRefund{}, e
		}
	}
	var payee, evidence []byte
	if e = s.DB.QueryRow(ctx, "SELECT payee_cipher,evidence_cipher FROM manual_refunds WHERE id=$1", id).Scan(&payee, &evidence); e != nil {
		return ManualRefund{}, e
	}
	f.Payee = &RefundPayee{}
	if e = s.Open(payee, f.Payee); e != nil {
		return ManualRefund{}, e
	}
	if len(evidence) > 0 {
		f.Evidence = &RefundEvidence{}
		if e = s.Open(evidence, f.Evidence); e != nil {
			return ManualRefund{}, e
		}
	}
	return f, nil
}
