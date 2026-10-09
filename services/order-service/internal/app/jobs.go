package app

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/omamx/order-service/internal/domain"
	"time"
)

type jobWork struct {
	ID, OrderID, Kind, Key string
	Payload                []byte
	Epoch                  int64
	Attempts               int
}

func (s *Service) claimJob(ctx context.Context) (*jobWork, error) {
	var j jobWork
	e := s.DB.QueryRow(ctx, `WITH picked AS (SELECT id FROM jobs WHERE status IN ('PENDING','RETRY') AND run_after<=clock_timestamp() AND (lease_until IS NULL OR lease_until<clock_timestamp()) ORDER BY created_at FOR UPDATE SKIP LOCKED LIMIT 1) UPDATE jobs j SET lease_owner=$1,lease_until=clock_timestamp()+interval '10 seconds',lease_epoch=lease_epoch+1,attempts=attempts+1 FROM picked WHERE j.id=picked.id RETURNING j.id,j.order_id,j.kind,j.key,j.payload,j.lease_epoch,j.attempts`, s.workerID).Scan(&j.ID, &j.OrderID, &j.Kind, &j.Key, &j.Payload, &j.Epoch, &j.Attempts)
	if errors.Is(e, pgx.ErrNoRows) {
		return nil, nil
	}
	return &j, e
}
func (s *Service) jobTransaction(ctx context.Context, j *jobWork, fn func(pgx.Tx, *domain.Order) error) error {
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	o, e := s.lockOrder(ctx, tx, j.OrderID)
	if e != nil {
		return e
	}
	var ok bool
	e = tx.QueryRow(ctx, "SELECT lease_owner=$2 AND lease_epoch=$3 AND lease_until>clock_timestamp() FROM jobs WHERE id=$1 FOR UPDATE", j.ID, s.workerID, j.Epoch).Scan(&ok)
	if e != nil {
		return e
	}
	if !ok {
		return domain.E("STALE_LEASE", 409)
	}
	if e = fn(tx, &o); e != nil {
		return e
	}
	return tx.Commit(ctx)
}
func finishJob(ctx context.Context, tx pgx.Tx, j *jobWork, status, code string) error {
	_, e := tx.Exec(ctx, "UPDATE jobs SET status=$2,error_code=$3,lease_owner=NULL,lease_until=NULL WHERE id=$1", j.ID, status, code)
	return e
}
func (s *Service) retryJob(ctx context.Context, j *jobWork, cause error) {
	code, _ := s.ErrorCode(cause)
	status := "RETRY"
	if j.Attempts >= 10 {
		status = "MANUAL_REVIEW"
	}
	_ = s.jobTransaction(ctx, j, func(tx pgx.Tx, o *domain.Order) error {
		_, e := tx.Exec(ctx, "UPDATE jobs SET status=$2,error_code=$3,run_after=clock_timestamp()+($4::double precision*interval '1 second'),lease_until=NULL,lease_owner=NULL WHERE id=$1", j.ID, status, code, backoff(j.Attempts))
		if e != nil {
			return e
		}
		if status == "MANUAL_REVIEW" {
			o.Hold = true
			return s.touch(ctx, tx, o, "RECOVERY_REQUIRED")
		}
		return nil
	})
}
func (s *Service) WorkJob(ctx context.Context) (bool, error) {
	j, e := s.claimJob(ctx)
	if e != nil || j == nil {
		return false, e
	}
	o, e := scanOrder(s.DB.QueryRow(ctx, "SELECT "+orderColumns+" FROM orders WHERE id=$1", j.OrderID))
	if e != nil {
		s.retryJob(ctx, j, e)
		return true, e
	}
	switch j.Kind {
	case "FINALIZE":
		if o.Status != "PAYMENT_FINALIZING" {
			e = s.jobTransaction(ctx, j, func(tx pgx.Tx, _ *domain.Order) error { return finishJob(ctx, tx, j, "DONE", "") })
			break
		}
		var r domain.Reservation
		e = s.Call(ctx, "POST", "/inventory/finalize", map[string]any{"order_id": o.ID, "operation_key": j.Key}, &r)
		if e != nil {
			s.retryJob(ctx, j, e)
			return true, nil
		}
		e = s.jobTransaction(ctx, j, func(tx pgx.Tx, current *domain.Order) error {
			if current.Status != "PAYMENT_FINALIZING" {
				return finishJob(ctx, tx, j, "DONE", "")
			}
			if r.OrderID != current.ID {
				return domain.E("RESOURCE_MISMATCH", 409)
			}
			if r.State != "COMMITTED" {
				if r.State != "EXPIRED" && r.State != "RELEASED" {
					return domain.E("DEPENDENCY_UNKNOWN", 503)
				}
				current.Stock = r.State
				current.Payment = "RECONCILIATION_REQUIRED"
				current.Hold = true
				if err := s.transition(ctx, tx, current, "CANCELLED_TIMEOUT", "INVENTORY", "EXPIRED_BEFORE_FINALIZE"); err != nil {
					return err
				}
				_, err := tx.Exec(ctx, `INSERT INTO reconciliation_cases(id,receipt_id,reason) SELECT gen_random_uuid(),receipt_id,'EXPIRED_BEFORE_FINALIZE' FROM allocations WHERE order_id=$1 ON CONFLICT(receipt_id) DO NOTHING`, current.ID)
				if err != nil {
					return err
				}
				if err = s.Emit(ctx, tx, current.ID, "ORDER", current.Version, 0, "cancelled", current.Status, current.Payment, current.Stock, current.Generation); err != nil {
					return err
				}
				return finishJob(ctx, tx, j, "DONE", "")
			}
			var amount int64
			var confirmed bool
			if err := tx.QueryRow(ctx, `SELECT COALESCE(sum(a.amount),0),COALESCE(bool_and(r.provider_status='SUCCEEDED'),false) FROM allocations a JOIN receipts r ON r.id=a.receipt_id WHERE a.order_id=$1`, current.ID).Scan(&amount, &confirmed); err != nil {
				return err
			}
			if amount != current.Final || !confirmed {
				return domain.E("PAYMENT_INVARIANT_FAILED", 409)
			}
			current.Stock = "COMMITTED"
			current.Payment = "CONFIRMED"
			if err := s.transition(ctx, tx, current, "PAID", "PAYMENT", "STOCK_FINALIZED"); err != nil {
				return err
			}
			if err := s.Emit(ctx, tx, current.ID, "ORDER", current.Version, 0, "paid", current.Status, current.Payment, current.Stock, current.Generation); err != nil {
				return err
			}
			if err := s.insertJob(ctx, tx, current.ID, "READY", "ready:"+current.ID, nil); err != nil {
				return err
			}
			var started time.Time
			if err := tx.QueryRow(ctx, "SELECT created_at FROM receipts WHERE id=(SELECT receipt_id FROM allocations WHERE order_id=$1)", current.ID).Scan(&started); err == nil {
				s.Metrics.WithLabelValues("payment_finalization", "PAID").Observe(time.Since(started).Seconds())
			}
			return finishJob(ctx, tx, j, "DONE", "")
		})
	case "READY":
		if o.Hold || o.Stock != "COMMITTED" || (o.Status != "PAID" && o.Status != "CONFIRMED_COD") {
			e = s.jobTransaction(ctx, j, func(tx pgx.Tx, _ *domain.Order) error { return finishJob(ctx, tx, j, "DONE", "NOT_ELIGIBLE") })
			break
		}
		generation := o.Generation
		if generation == 0 {
			generation = 1
		}
		var ready struct {
			Authorized bool `json:"authorized"`
		}
		e = s.Call(ctx, "POST", "/fulfillment/authorize", map[string]any{"order_id": o.ID, "generation": generation}, &ready)
		if e != nil {
			s.retryJob(ctx, j, e)
			return true, nil
		}
		e = s.jobTransaction(ctx, j, func(tx pgx.Tx, current *domain.Order) error {
			if !ready.Authorized || current.Generation != o.Generation || current.Hold || current.Stock != "COMMITTED" || (current.Status != "PAID" && current.Status != "CONFIRMED_COD") {
				return finishJob(ctx, tx, j, "DONE", "NOT_ELIGIBLE")
			}
			if current.Method != "COD" && current.Payment != "CONFIRMED" && current.Payment != "PARTIALLY_REFUNDED" {
				return domain.E("PAYMENT_INVARIANT_FAILED", 409)
			}
			current.Generation = generation
			current.Ready = true
			if err := s.touch(ctx, tx, current, "FULFILLMENT_READY"); err != nil {
				return err
			}
			if err := s.Emit(ctx, tx, current.ID, "ORDER", current.Version, 0, "fulfillment_ready", current.Status, current.Payment, current.Stock, current.Generation); err != nil {
				return err
			}
			return finishJob(ctx, tx, j, "DONE", "")
		})
	case "CANCEL":
		var payload struct {
			Target   string `json:"target"`
			Approval string `json:"approval_id"`
			Actor    string `json:"actor"`
		}
		if e = json.Unmarshal(j.Payload, &payload); e != nil {
			return true, e
		}
		if o.Status == payload.Target {
			e = s.jobTransaction(ctx, j, func(tx pgx.Tx, _ *domain.Order) error { return finishJob(ctx, tx, j, "DONE", "") })
			break
		}
		var barrier struct {
			Cancelled bool `json:"cancelled"`
		}
		e = s.Call(ctx, "POST", "/fulfillment/cancel", map[string]any{"order_id": o.ID, "operation_key": j.Key}, &barrier)
		if e != nil {
			s.retryJob(ctx, j, e)
			return true, nil
		}
		if !barrier.Cancelled {
			e = s.jobTransaction(ctx, j, func(tx pgx.Tx, current *domain.Order) error {
				current.Hold = false
				current.Ready = true
				if err := s.touch(ctx, tx, current, "CANCEL_REJECTED_TASK_ACCEPTED"); err != nil {
					return err
				}
				return finishJob(ctx, tx, j, "REJECTED", "CANCELLATION_NOT_ALLOWED")
			})
			break
		}
		path := "/inventory/release"
		if o.Stock == "COMMITTED" {
			if payload.Approval == "" && o.Method != "COD" {
				s.retryJob(ctx, j, domain.E("REVERSAL_APPROVAL_REQUIRED", 409))
				return true, nil
			}
			path = "/inventory/reverse"
		}
		var r domain.Reservation
		e = s.Call(ctx, "POST", path, map[string]any{"order_id": o.ID, "operation_key": j.Key + ":stock", "approval_id": payload.Approval, "cod_unpaid": o.Method == "COD" && o.Payment == "UNPAID"}, &r)
		if e != nil {
			s.retryJob(ctx, j, e)
			return true, nil
		}
		if r.State != "RELEASED" && r.State != "EXPIRED" {
			s.retryJob(ctx, j, domain.E("DEPENDENCY_UNKNOWN", 503))
			return true, nil
		}
		e = s.jobTransaction(ctx, j, func(tx pgx.Tx, current *domain.Order) error {
			current.Stock = r.State
			current.Ready = false
			current.Hold = false
			if err := s.transition(ctx, tx, current, payload.Target, payload.Actor, "CANCELLATION_CONFIRMED"); err != nil {
				return err
			}
			if err := s.Emit(ctx, tx, current.ID, "ORDER", current.Version, 0, "cancelled", current.Status, current.Payment, current.Stock, current.Generation); err != nil {
				return err
			}
			return finishJob(ctx, tx, j, "DONE", "")
		})
	case "REFUND":
		var payload struct {
			ID string `json:"refund_id"`
		}
		if e = json.Unmarshal(j.Payload, &payload); e != nil {
			return true, e
		}
		var amount int64
		var receiptID, status string
		e = s.DB.QueryRow(ctx, "SELECT amount,receipt_id,status FROM refunds WHERE id=$1", payload.ID).Scan(&amount, &receiptID, &status)
		if e != nil {
			return true, e
		}
		if status == "SUCCEEDED" || status == "FAILED" {
			e = s.jobTransaction(ctx, j, func(tx pgx.Tx, _ *domain.Order) error { return finishJob(ctx, tx, j, "DONE", "") })
			break
		}
		var result struct {
			Status    string `json:"status"`
			Reference string `json:"reference"`
		}
		var providerTransaction string
		if err := s.DB.QueryRow(ctx, "SELECT transaction_id FROM receipts WHERE id=$1", receiptID).Scan(&providerTransaction); err != nil {
			return true, err
		}
		e = s.Call(ctx, "GET", "/provider/refunds/"+j.Key, nil, &result)
		if e == nil && result.Status == "NOT_FOUND" {
			e = s.Call(ctx, "POST", "/provider/refunds", map[string]any{"operation_key": j.Key, "transaction_id": providerTransaction, "amount": amount}, &result)
		}
		if e != nil || result.Status == "UNKNOWN" {
			_ = s.jobTransaction(ctx, j, func(tx pgx.Tx, _ *domain.Order) error {
				_, err := tx.Exec(ctx, "UPDATE refunds SET status='UNKNOWN' WHERE id=$1", payload.ID)
				return err
			})
			s.retryJob(ctx, j, domain.E("REFUND_UNKNOWN", 503))
			return true, nil
		}
		if result.Status != "SUCCEEDED" && result.Status != "FAILED" {
			s.retryJob(ctx, j, domain.E("REFUND_UNKNOWN", 503))
			return true, nil
		}
		e = s.jobTransaction(ctx, j, func(tx pgx.Tx, current *domain.Order) error {
			var before string
			var ver int64
			if err := tx.QueryRow(ctx, "SELECT status,version FROM refunds WHERE id=$1 FOR UPDATE", payload.ID).Scan(&before, &ver); err != nil {
				return err
			}
			if before == "SUCCEEDED" || before == "FAILED" {
				return finishJob(ctx, tx, j, "DONE", "")
			}
			_, err := tx.Exec(ctx, "UPDATE refunds SET status=$2,provider_ref=$3,version=version+1 WHERE id=$1", payload.ID, result.Status, result.Reference)
			if err != nil {
				return err
			}
			if result.Status == "SUCCEEDED" {
				if _, err = tx.Exec(ctx, "UPDATE reconciliation_cases SET status='RESOLVED' WHERE receipt_id=$1 AND (SELECT COALESCE(sum(amount),0) FROM refunds WHERE receipt_id=$1 AND status='SUCCEEDED') >= (SELECT amount FROM receipts WHERE id=$1)", receiptID); err != nil {
					return err
				}
				var total int64
				if err = tx.QueryRow(ctx, `SELECT COALESCE(sum(f.amount),0) FROM refunds f JOIN allocations a ON a.receipt_id=f.receipt_id WHERE f.order_id=$1 AND f.status='SUCCEEDED'`, current.ID).Scan(&total); err != nil {
					return err
				}
				if total >= current.Final {
					current.Payment = "REFUNDED"
				} else if total > 0 {
					current.Payment = "PARTIALLY_REFUNDED"
				}
				if err = s.touch(ctx, tx, current, "REFUND_SETTLED"); err != nil {
					return err
				}
				if err = s.Emit(ctx, tx, payload.ID, "REFUND", ver+1, 0, "refund_succeeded", "SUCCEEDED", "", "", 0); err != nil {
					return err
				}
			}
			if err = s.Audit(ctx, tx, current.ID, "PROVIDER", "REFUND_RESULT", map[string]any{"refund_id": payload.ID, "status": result.Status}); err != nil {
				return err
			}
			return finishJob(ctx, tx, j, "DONE", "")
		})
	default:
		e = domain.E("UNKNOWN_JOB", 500)
	}
	if e != nil {
		s.retryJob(ctx, j, e)
	}
	return true, e
}
func (s *Service) ApproveCancellation(ctx context.Context, id string) error {
	var a domain.Approval
	if e := s.Call(ctx, "GET", "/care/approvals/"+id, nil, &a); e != nil {
		return e
	}
	if a.Type != "CANCEL" || a.State != "APPROVED" || a.Actor != "SALES_MANAGER" {
		return domain.E("INVALID_APPROVAL", 422)
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	o, e := s.lockOrder(ctx, tx, a.OrderID)
	if e != nil {
		return e
	}
	if o.Status == "CANCELLED_BY_ADMIN" {
		return nil
	}
	if o.Status != "PAID" && o.Status != "CONFIRMED_COD" {
		return domain.E("CANCELLATION_NOT_ALLOWED", 409)
	}
	var already bool
	if e = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM jobs WHERE key=$1)", "cancel-approved:"+a.ID).Scan(&already); e != nil {
		return e
	}
	if already {
		return nil
	}
	o.Hold = true
	o.Ready = false
	if e = s.touch(ctx, tx, &o, "CANCEL_APPROVAL"); e != nil {
		return e
	}
	if e = s.insertJob(ctx, tx, o.ID, "CANCEL", "cancel-approved:"+a.ID, map[string]any{"target": "CANCELLED_BY_ADMIN", "approval_id": a.ID, "actor": a.Actor}); e != nil {
		return e
	}
	return tx.Commit(ctx)
}
func (s *Service) SweepTimeouts(ctx context.Context) error {
	rows, e := s.DB.Query(ctx, "SELECT id FROM orders WHERE status='PENDING_PAYMENT' AND operational_hold=false AND payment_expires_at<=clock_timestamp() LIMIT 100")
	if e != nil {
		return e
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if e = rows.Scan(&id); e != nil {
			rows.Close()
			return e
		}
		ids = append(ids, id)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	for _, id := range ids {
		tx, e := s.DB.Begin(ctx)
		if e != nil {
			return e
		}
		o, e := s.lockOrder(ctx, tx, id)
		if e != nil {
			_ = tx.Rollback(ctx)
			continue
		}
		var due bool
		e = tx.QueryRow(ctx, "SELECT clock_timestamp()>=$1", o.ExpiresAt).Scan(&due)
		if e == nil && due && o.Status == "PENDING_PAYMENT" && !o.Hold {
			o.Hold = true
			o.Ready = false
			e = s.touch(ctx, tx, &o, "TIMEOUT_INTENT")
			if e == nil {
				e = s.insertJob(ctx, tx, id, "CANCEL", "timeout:"+id, map[string]any{"target": "CANCELLED_TIMEOUT", "actor": "SYSTEM"})
			}
		}
		if e != nil {
			_ = tx.Rollback(ctx)
			return e
		}
		if e = tx.Commit(ctx); e != nil {
			return e
		}
	}
	return nil
}
