package app

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/omamx/order-service/internal/domain"
	"strconv"
	"time"
)

func (s *Service) RecordReceipt(ctx context.Context, r domain.Receipt) error {
	if e := r.Validate(s.C.Receiver); e != nil {
		return e
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	o, orderErr := scanOrder(tx.QueryRow(ctx, "SELECT "+orderColumns+" FROM orders WHERE provider_reference=$1 FOR UPDATE", r.Reference))
	if orderErr != nil && !errors.Is(orderErr, pgx.ErrNoRows) {
		return orderErr
	}
	var orderID any
	if orderErr == nil {
		orderID = o.ID
	}
	rid := uuid.NewString()
	hash := domain.Hash(r)
	tag, e := tx.Exec(ctx, `INSERT INTO receipts(id,provider,transaction_id,order_id,reference,receiver,amount,currency,payload_hash,status,provider_status,provider_paid_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,'VERIFIED','SUCCEEDED',$10) ON CONFLICT(provider,transaction_id) DO NOTHING`, rid, r.Provider, r.TransactionID, orderID, r.Reference, r.Receiver, r.Amount, r.Currency, hash, r.PaidAt)
	if e != nil {
		return e
	}
	if tag.RowsAffected() == 0 {
		var previous string
		if e = tx.QueryRow(ctx, "SELECT payload_hash FROM receipts WHERE provider=$1 AND transaction_id=$2", r.Provider, r.TransactionID).Scan(&previous); e != nil {
			return e
		}
		if previous != hash {
			if e = s.Audit(ctx, tx, "", "SIMULATOR", "RECEIPT_CONFLICT", map[string]any{"transaction_id": r.TransactionID, "hash": hash}); e != nil {
				return e
			}
			if e = tx.Commit(ctx); e != nil {
				return e
			}
			return domain.E("RECEIPT_CONFLICT", 409)
		}
		return tx.Commit(ctx)
	}
	reason := ""
	var now time.Time
	if e = tx.QueryRow(ctx, "SELECT clock_timestamp()").Scan(&now); e != nil {
		return e
	}
	if orderErr != nil {
		reason = "UNMATCHED"
	} else if o.Method == "COD" {
		if o.Hold {
			reason = "LATE_OR_UNAVAILABLE"
		} else if r.Amount != o.Final {
			reason = "AMOUNT_MISMATCH"
		} else if o.Status == "CANCELLED_BY_USER" || o.Status == "CANCELLED_BY_ADMIN" || o.Status == "CANCELLED_TIMEOUT" {
			reason = "LATE_PAYMENT"
		}
	} else if o.Status != "PENDING_PAYMENT" || !now.Before(o.ExpiresAt) || o.Hold {
		reason = "LATE_OR_UNAVAILABLE"
	} else if r.Amount != o.Final {
		reason = "AMOUNT_MISMATCH"
	}
	if reason == "" {
		var allocated bool
		if e = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM allocations WHERE order_id=$1)", o.ID).Scan(&allocated); e != nil {
			return e
		}
		if allocated {
			reason = "EXTRA_RECEIPT"
		}
	}
	if reason != "" {
		if _, e = tx.Exec(ctx, "UPDATE receipts SET status='RECONCILIATION_REQUIRED' WHERE id=$1", rid); e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, "INSERT INTO reconciliation_cases(id,receipt_id,reason) VALUES($1,$2,$3)", uuid.NewString(), rid, reason); e != nil {
			return e
		}
		if e = s.Emit(ctx, tx, rid, "PAYMENT", 1, 0, "reconciliation_required", "RECONCILIATION_REQUIRED", "", "", 0); e != nil {
			return e
		}
	} else {
		if _, e = tx.Exec(ctx, "INSERT INTO allocations(receipt_id,order_id,amount) VALUES($1,$2,$3)", rid, o.ID, r.Amount); e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, "UPDATE receipts SET status='ALLOCATED' WHERE id=$1", rid); e != nil {
			return e
		}
		o.Payment = "CONFIRMED"
		if o.Method == "COD" {
			if e = s.touch(ctx, tx, &o, "PAYMENT_COD_SETTLEMENT"); e != nil {
				return e
			}
		} else {
			o.Stock = "COMMITTING"
			if e = s.transition(ctx, tx, &o, "PAYMENT_FINALIZING", "PAYMENT", "VERIFIED_EXACT_RECEIPT"); e != nil {
				return e
			}
			if e = s.insertJob(ctx, tx, o.ID, "FINALIZE", "finalize:"+o.ID, map[string]any{"receipt_id": rid}); e != nil {
				return e
			}
		}
	}
	if e = s.Audit(ctx, tx, o.ID, "SIMULATOR", "RECEIPT_RECORDED", map[string]any{"receipt_id": rid, "reason": reason}); e != nil {
		return e
	}
	return tx.Commit(ctx)
}

// Statement cursor is advanced only after every receipt has a durable local outcome.
func (s *Service) ReconcilePayments(ctx context.Context) error {
	var position int64
	e := s.DB.QueryRow(ctx, "SELECT position FROM provider_cursors WHERE provider='SIMULATOR'").Scan(&position)
	if e != nil && !errors.Is(e, pgx.ErrNoRows) {
		return e
	}
	var statement struct {
		Cursor   int64            `json:"cursor"`
		Receipts []domain.Receipt `json:"receipts"`
	}
	if e = s.Call(ctx, "GET", "/provider/statement?after="+strconv.FormatInt(position, 10), nil, &statement); e != nil {
		return e
	}
	if statement.Cursor < position {
		return domain.E("INVALID_PROVIDER_CURSOR", 503)
	}
	for _, receipt := range statement.Receipts {
		if e = s.RecordReceipt(ctx, receipt); e != nil {
			return e
		}
	}
	_, e = s.DB.Exec(ctx, "INSERT INTO provider_cursors(provider,position) VALUES('SIMULATOR',$1) ON CONFLICT(provider) DO UPDATE SET position=GREATEST(provider_cursors.position,EXCLUDED.position)", statement.Cursor)
	return e
}
func (s *Service) AcceptRefund(ctx context.Context, approvalID string) (string, error) {
	var a domain.Approval
	if e := s.Call(ctx, "GET", "/care/approvals/"+approvalID, nil, &a); e != nil {
		return "", e
	}
	if a.State != "APPROVED" || a.Type != "REFUND" || a.Actor != "SALES_MANAGER" || a.Amount <= 0 || a.Amount > domain.MaxMoney {
		return "", domain.E("INVALID_APPROVAL", 422)
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return "", e
	}
	defer tx.Rollback(ctx)
	o, e := s.lockOrder(ctx, tx, a.OrderID)
	if e != nil {
		return "", e
	}
	var existing string
	e = tx.QueryRow(ctx, "SELECT id FROM refunds WHERE approval_id=$1", a.ID).Scan(&existing)
	if e == nil {
		return existing, nil
	}
	if !errors.Is(e, pgx.ErrNoRows) {
		return "", e
	}
	var received int64
	var providerStatus string
	if e = tx.QueryRow(ctx, "SELECT amount,provider_status FROM receipts WHERE id=$1 AND order_id=$2 FOR UPDATE", a.ReceiptID, o.ID).Scan(&received, &providerStatus); e != nil {
		return "", e
	}
	if providerStatus != "SUCCEEDED" {
		return "", domain.E("RECEIPT_NOT_CONFIRMED", 409)
	}
	var held int64
	if e = tx.QueryRow(ctx, "SELECT COALESCE(sum(amount),0) FROM refunds WHERE receipt_id=$1 AND status IN ('APPROVED','SUBMITTED','SUCCEEDED','UNKNOWN','MANUAL_REVIEW')", a.ReceiptID).Scan(&held); e != nil {
		return "", e
	}
	if a.Amount > received-held {
		return "", domain.E("REFUND_BUDGET_EXCEEDED", 409)
	}
	id := uuid.NewString()
	key := "refund:" + a.ID
	if _, e = tx.Exec(ctx, "INSERT INTO refunds(id,order_id,receipt_id,approval_id,operation_key,amount,status) VALUES($1,$2,$3,$4,$5,$6,'APPROVED')", id, o.ID, a.ReceiptID, a.ID, key, a.Amount); e != nil {
		return "", e
	}
	if e = s.insertJob(ctx, tx, o.ID, "REFUND", key, map[string]any{"refund_id": id}); e != nil {
		return "", e
	}
	o.Hold = true
	o.Ready = false
	if e = s.touch(ctx, tx, &o, "REFUND_OBLIGATION"); e != nil {
		return "", e
	}
	if e = s.Audit(ctx, tx, o.ID, a.Actor, "REFUND_APPROVED", map[string]any{"refund_id": id, "amount": a.Amount, "approval_id": a.ID}); e != nil {
		return "", e
	}
	return id, tx.Commit(ctx)
}
