package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/omamx/order-service/internal/domain/core"
	"github.com/omamx/order-service/internal/sepay"
	"strings"
	"time"
)

type PaymentJob struct {
	ID, Owner, Order, Key, Workflow string
	Epoch                           int64
}

func (p *Payment) claim(ctx context.Context, kind string) (PaymentJob, error) {
	j := PaymentJob{Owner: uuid.NewString()}
	e := p.Store.db.QueryRow(ctx, `WITH pick AS (SELECT j.id FROM jobs j LEFT JOIN payment_intents i ON i.order_id=j.order_id WHERE j.cohort=$1 AND j.kind=$2 AND ((j.status IN ('READY','RETRY_WAIT') AND j.run_after<=clock_timestamp()) OR (j.status='LEASED' AND j.lease_until<clock_timestamp())) AND (($2='PAYMENT_SYNC' AND j.key LIKE $3) OR ($2='PAYMENT_FINALIZE' AND i.receiver_profile_id=$4)) ORDER BY j.run_after,j.id FOR UPDATE OF j SKIP LOCKED LIMIT 1) UPDATE jobs j SET status='LEASED',lease_owner=$5,lease_until=clock_timestamp()+interval '30 seconds',lease_epoch=lease_epoch+1,attempts=attempts+1 FROM pick WHERE j.id=pick.id RETURNING j.id,COALESCE(j.order_id::text,''),j.key,j.workflow_id,j.lease_epoch`, Cohort, kind, "sepay:sync:"+p.Receiver+":%", p.Receiver, j.Owner).Scan(&j.ID, &j.Order, &j.Key, &j.Workflow, &j.Epoch)
	return j, e
}
func (p *Payment) fence(ctx context.Context, tx pgx.Tx, j PaymentJob) error {
	tag, e := tx.Exec(ctx, `UPDATE jobs SET lease_until=clock_timestamp()+interval '30 seconds' WHERE id=$1 AND cohort=$2 AND status='LEASED' AND lease_owner=$3 AND lease_epoch=$4 AND lease_until>clock_timestamp()`, j.ID, Cohort, j.Owner, j.Epoch)
	if e != nil {
		return e
	}
	if tag.RowsAffected() != 1 {
		return ErrLease
	}
	return nil
}
func (p *Payment) retry(ctx context.Context, j PaymentJob, delay time.Duration, code string) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	_, e := p.Store.db.Exec(ctx, `UPDATE jobs SET status='RETRY_WAIT',run_after=clock_timestamp()+$5::bigint*interval '1 millisecond',lease_owner=NULL,lease_until=NULL,error_code=$6 WHERE id=$1 AND cohort=$2 AND lease_owner=$3 AND lease_epoch=$4 AND status='LEASED'`, j.ID, Cohort, j.Owner, j.Epoch, delay.Milliseconds(), code)
	return e
}
func (p *Payment) quota(ctx context.Context) error {
	var acquired bool
	e := p.Store.db.QueryRow(ctx, `INSERT INTO payment_dispatch_quotas(egress_scope,next_request_at) VALUES($1,clock_timestamp()+interval '500 milliseconds') ON CONFLICT(egress_scope) DO UPDATE SET next_request_at=clock_timestamp()+interval '500 milliseconds' WHERE payment_dispatch_quotas.next_request_at<=clock_timestamp() RETURNING true`, p.Egress).Scan(&acquired)
	if errors.Is(e, pgx.ErrNoRows) {
		return sepay.ErrUnavailable
	}
	return e
}

// WorkSync runs one immutable-window scan; no SQL transaction spans API calls.
// Lost lease restarts at page1; receipts have their own canonical UUID dedup.
func (p *Payment) WorkSync(ctx context.Context, reader sepay.Reader) (bool, error) {
	j, e := p.claim(ctx, "PAYMENT_SYNC")
	if errors.Is(e, pgx.ErrNoRows) {
		return false, nil
	}
	if e != nil {
		return false, e
	}
	e = p.scan(ctx, j, reader)
	if e != nil {
		delay := time.Second
		var api *sepay.APIError
		if errors.As(e, &api) {
			delay = api.RetryAfter
			if api.Status == 401 || api.Status == 403 {
				_, _ = p.Store.db.Exec(ctx, "UPDATE payment_sync_checkpoints SET circuit='AUTH_FAILED' WHERE receiver_id=$1", p.Receiver)
				delay = time.Minute
			}
		}
		_ = p.retry(context.WithoutCancel(ctx), j, delay, "SYNC_RETRY")
	}
	return true, e
}
func (p *Payment) scan(ctx context.Context, j PaymentJob, reader sepay.Reader) error {
	stream := j.Key[strings.LastIndex(j.Key, ":")+1:]
	runID := uuid.NewString()
	var from, to time.Time
	var bank string
	e := p.Store.runner.Do(ctx, func(tx pgx.Tx) error {
		if e := p.fence(ctx, tx, j); e != nil {
			return e
		}
		var previous *time.Time
		var boot, now time.Time
		if e := tx.QueryRow(ctx, `SELECT c.bootstrap_at,c.last_scanned_end,clock_timestamp(),r.bank_account_uuid FROM payment_sync_checkpoints c JOIN payment_receiver_profiles r ON r.id=c.receiver_id WHERE c.receiver_id=$1 AND c.stream=$2 FOR UPDATE OF c`, p.Receiver, stream).Scan(&boot, &previous, &now, &bank); e != nil {
			return e
		}
		var existing string
		e := tx.QueryRow(ctx, "SELECT id,window_start,window_end FROM payment_sync_runs WHERE receiver_id=$1 AND stream=$2 AND status='RUNNING' FOR UPDATE", p.Receiver, stream).Scan(&existing, &from, &to)
		if e == nil {
			runID = existing
			_, e = tx.Exec(ctx, "UPDATE payment_sync_runs SET lease_epoch=$2,last_durable_page=0,observed_count=0,quarantine_count=0 WHERE id=$1", runID, j.Epoch)
			return e
		}
		if !errors.Is(e, pgx.ErrNoRows) {
			return e
		}
		to = now
		from = boot.Add(-2 * time.Minute)
		if previous != nil {
			from = previous.Add(-2 * time.Minute)
		}
		switch stream {
		case "OVERLAP":
			from = now.Add(-24 * time.Hour)
		case "DAILY":
			from = now.Add(-7 * 24 * time.Hour)
		case "BACKFILL":
			from = boot.Add(-2 * time.Minute)
		}
		if stream != "FAST" && previous != nil && previous.After(from) && previous.Before(now.Add(-time.Minute)) {
			from = previous.Add(-2 * time.Minute)
		}
		// Bounded windows catch up consecutively; never jump an unscanned gap.
		if from.Add(24 * time.Hour).Before(to) {
			to = from.Add(24 * time.Hour)
		}
		_, e = tx.Exec(ctx, `INSERT INTO payment_sync_runs(id,receiver_id,stream,job_id,lease_epoch,window_start,window_end) VALUES($1,$2,$3,$4,$5,$6,$7)`, runID, p.Receiver, stream, j.ID, j.Epoch, from, to)
		return e
	})
	if e != nil {
		return e
	}
	quarantines := 0
	for pageNo := 1; pageNo <= 10000; pageNo++ {
		for attempt := 0; ; attempt++ {
			e = p.quota(ctx)
			if e == nil {
				break
			}
			if !errors.Is(e, sepay.ErrUnavailable) || attempt >= 20 {
				return e
			}
			timer := time.NewTimer(500 * time.Millisecond)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
			if e = p.Store.runner.Do(ctx, func(tx pgx.Tx) error { return p.fence(ctx, tx, j) }); e != nil {
				return e
			}
		}
		page, e := reader.List(ctx, sepay.Window{Bank: bank, From: from, To: to, Page: pageNo})
		if e != nil {
			return e
		}
		if page.Pagination.Page != pageNo {
			return sepay.ErrInvalid
		}
		for _, raw := range page.Rows {
			var t sepay.Transaction
			reason := ""
			if sepay.Decode(raw, &t) != nil || t.Validate() != nil {
				reason = "MALFORMED_API_ROW"
			} else if t.BankAccountID == bank && t.Direction == "out" {
				// The endpoint lists both directions. Outgoing funds are not receipts;
				// persist their disposition without marking the incoming scan incomplete.
				reason = "IGNORED_OUTGOING"
			} else if t.BankAccountID != bank || t.Direction != "in" {
				reason = "RECEIVER_OR_DIRECTION_MISMATCH"
			} else {
				// Heartbeat before each transaction. Epoch loss cannot advance its checkpoint.
				if e = p.Store.runner.Do(ctx, func(tx pgx.Tx) error { return p.fence(ctx, tx, j) }); e != nil {
					return e
				}
				e = p.ingest(ctx, t, &j)
				if errors.Is(e, ErrInput) {
					reason = "RECEIVER_MISMATCH"
				} else if e != nil {
					return e
				}
			}
			if reason != "" {
				cipher, e := p.Store.cipher.Seal("api-quarantine:"+runID+":"+hash(raw), raw)
				if e != nil {
					return e
				}
				e = p.Store.runner.Do(ctx, func(tx pgx.Tx) error {
					if e := p.fence(ctx, tx, j); e != nil {
						return e
					}
					_, e := tx.Exec(ctx, `INSERT INTO payment_api_quarantine(run_id,evidence_hash,evidence_cipher,reason) VALUES($1,$2,$3,$4) ON CONFLICT DO NOTHING`, runID, hash(raw), cipher, reason)
					return e
				})
				if e != nil {
					return e
				}
				if reason != "IGNORED_OUTGOING" {
					quarantines++
				}
			}
		}
		e = p.Store.runner.Do(ctx, func(tx pgx.Tx) error {
			if e := p.fence(ctx, tx, j); e != nil {
				return e
			}
			tag, e := tx.Exec(ctx, `UPDATE payment_sync_runs SET last_durable_page=$3,observed_count=observed_count+$4,quarantine_count=$5 WHERE id=$1 AND lease_epoch=$2 AND last_durable_page=$3-1 AND status='RUNNING'`, runID, j.Epoch, pageNo, len(page.Rows), quarantines)
			if e != nil {
				return e
			}
			if tag.RowsAffected() != 1 {
				return ErrLease
			}
			if _, e = tx.Exec(ctx, "UPDATE payment_sync_checkpoints SET last_fetch_at=clock_timestamp() WHERE receiver_id=$1 AND stream=$2", p.Receiver, stream); e != nil {
				return e
			}
			if page.Pagination.More {
				return nil
			}
			status, circuit := "COMPLETED", "HEALTHY"
			if quarantines > 0 {
				status = "COMPLETED_WITH_REVIEW"
				circuit = "INCOMPLETE_REVIEW"
			}
			if _, e = tx.Exec(ctx, "UPDATE payment_sync_runs SET status=$2 WHERE id=$1", runID, status); e != nil {
				return e
			}
			if _, e = tx.Exec(ctx, `UPDATE payment_sync_checkpoints SET last_scanned_end=GREATEST(COALESCE(last_scanned_end,$3),$3),circuit=$4 WHERE receiver_id=$1 AND stream=$2`, p.Receiver, stream, to, circuit); e != nil {
				return e
			}
			delay := 5 * time.Second
			if stream == "FAST" {
				var active bool
				if e = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM payment_intents WHERE receiver_profile_id=$1 AND status='ACTIVE' AND expires_at>clock_timestamp())", p.Receiver).Scan(&active); e != nil {
					return e
				}
				if !active {
					delay = time.Minute
				}
			}
			switch stream {
			case "OVERLAP":
				delay = 5 * time.Minute
			case "DAILY":
				now := time.Now().In(time.FixedZone("VN", 7*3600))
				next := time.Date(now.Year(), now.Month(), now.Day(), 2, 0, 0, 0, now.Location())
				if !next.After(now) {
					next = next.AddDate(0, 0, 1)
				}
				delay = next.Sub(now)
			case "BACKFILL":
				delay = 24 * time.Hour
			}
			if to.Before(time.Now().Add(-time.Minute)) {
				delay = time.Second
			}
			_, e = tx.Exec(ctx, `UPDATE jobs SET status='RETRY_WAIT',lease_owner=NULL,lease_until=NULL,run_after=clock_timestamp()+$2::bigint*interval '1 millisecond',error_code=NULL WHERE id=$1`, j.ID, delay.Milliseconds())
			return e
		})
		if e != nil {
			return e
		}
		if !page.Pagination.More {
			return nil
		}
	}
	return sepay.ErrInvalid
}

// Inventory is a capability port. No COMMITTED proof may be synthesized from a receipt.
type PaymentInventory interface {
	Finalize(context.Context, core.Aggregate, string) (core.ResourceProof, error)
}

func (p *Payment) WorkFinalize(ctx context.Context, inventory PaymentInventory) (bool, error) {
	if inventory == nil {
		return false, ErrInput
	}
	j, e := p.claim(ctx, "PAYMENT_FINALIZE")
	if errors.Is(e, pgx.ErrNoRows) {
		return false, nil
	}
	if e != nil {
		return false, e
	}
	var scope string
	e = p.Store.db.QueryRow(ctx, "SELECT scope FROM orders WHERE id=$1 AND cohort=$2", j.Order, Cohort).Scan(&scope)
	if e != nil {
		return true, e
	}
	a, e := p.Store.LoadOrder(ctx, j.Order, scope)
	if e != nil {
		return true, e
	}
	// Hold/cancel/refund blocks prevent initiating a new remote effect.
	if a.Record().Blocks != 0 {
		_ = p.retry(ctx, j, time.Minute, "PAYMENT_BLOCKED")
		return true, nil
	}
	proof, e := inventory.Finalize(ctx, a, "inventory:finalize:"+a.ID())
	if e != nil {
		_ = p.retry(context.WithoutCancel(ctx), j, time.Second, "INVENTORY_UNKNOWN")
		return true, e
	}
	e = p.finalized(ctx, j, proof)
	if e != nil {
		_ = p.retry(context.WithoutCancel(ctx), j, time.Second, "FINALIZATION_RETRY")
	}
	return true, e
}
func (p *Payment) finalized(ctx context.Context, j PaymentJob, proof core.ResourceProof) error {
	return p.Store.runner.Do(ctx, func(tx pgx.Tx) error {
		var scope string
		if e := tx.QueryRow(ctx, "SELECT scope FROM orders WHERE id=$1 AND cohort=$2 FOR UPDATE", j.Order, Cohort).Scan(&scope); e != nil {
			return e
		}
		a, e := p.Store.loadOrder(ctx, tx, j.Order, scope)
		if e != nil {
			return e
		}
		if e = p.fence(ctx, tx, j); e != nil {
			return e
		}
		if a.State() == core.Paid {
			_, e = tx.Exec(ctx, "UPDATE jobs SET status='DONE',lease_owner=NULL,lease_until=NULL WHERE id=$1", j.ID)
			return e
		}
		var now time.Time
		if e = tx.QueryRow(ctx, "SELECT clock_timestamp()").Scan(&now); e != nil {
			return e
		}
		// Any refund reservation, even from future manual-refund integrations, fences PAID.
		var budget bool
		if e = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM refunds WHERE receipt_id=$1 AND (budget_held OR status IN ('APPROVED','SUBMITTED','UNKNOWN','MANUAL_REVIEW','SUCCEEDED')))`, a.Record().Allocation.ReceiptID).Scan(&budget); e != nil {
			return e
		}
		if budget || a.Record().Blocks != 0 {
			raw, _ := json.Marshal(proof)
			cipher, e := p.Store.cipher.Seal("blocked-finalization:"+j.ID, raw)
			if e != nil {
				return e
			}
			if _, e = tx.Exec(ctx, `INSERT INTO reconciliation_cases(id,receipt_id,reason,evidence_hash,evidence_cipher) VALUES($1,$2,'BLOCKED_AFTER_RESOURCE_COMMIT',$3,$4) ON CONFLICT DO NOTHING`, uuid.NewString(), a.Record().Allocation.ReceiptID, hash(raw), cipher); e != nil {
				return e
			}
			_, e = tx.Exec(ctx, "UPDATE jobs SET status='PARKED',lease_owner=NULL,lease_until=NULL,error_code='BLOCKED_AFTER_RESOURCE_COMMIT' WHERE id=$1", j.ID)
			return e
		}
		next, e := a.ConfirmPaid(core.DecisionContext{ExpectedVersion: a.Version(), At: now}, proof)
		if e != nil {
			if proof.State != "EXPIRED" && proof.State != "RELEASED" {
				return e
			}
			next, e = a.CancelUncommittedFinalization(core.DecisionContext{ExpectedVersion: a.Version(), At: now}, proof)
			if e != nil {
				return e
			}
			if e = p.Store.savePaymentRoot(ctx, tx, a, next, "RECONCILIATION_REQUIRED", "STOCK_FINALIZATION_FAILED"); e != nil {
				return e
			}
			if _, e = tx.Exec(ctx, `INSERT INTO reconciliation_cases(id,receipt_id,reason) VALUES($1,$2,'STOCK_FINALIZATION_FAILED') ON CONFLICT DO NOTHING`, uuid.NewString(), a.Record().Allocation.ReceiptID); e != nil {
				return e
			}
			if _, e = tx.Exec(ctx, "UPDATE payment_order_projections SET has_open_reconciliation=true,payment_hold=true,version=version+1 WHERE order_id=$1", a.ID()); e != nil {
				return e
			}
		} else {
			if e = p.Store.savePaymentRoot(ctx, tx, a, next, "CONFIRMED", "FIRST_PAID"); e != nil {
				return e
			}
			if _, e = tx.Exec(ctx, `INSERT INTO order_milestones(order_id,kind,decision_ref,occurred_at) VALUES($1,'FIRST_PAID',$2,$3)`, a.ID(), proof.DecisionID, now); e != nil {
				return e
			}
			if e = p.Store.paidEvent(ctx, tx, next, now); e != nil {
				return e
			}
			if e = p.Store.followup(ctx, tx, uuid.NewString(), uuid.NewString(), "FULFILLMENT_READY", a.ID(), a.Snapshot().Data().OperationID, "BARRIER", now); e != nil {
				return e
			}
		}
		if _, e = tx.Exec(ctx, "UPDATE workflows SET state='COMPLETED',completion_outcome='SUCCEEDED',stage='DONE',version=version+1 WHERE id=$1", j.Workflow); e != nil {
			return e
		}
		_, e = tx.Exec(ctx, "UPDATE jobs SET status='DONE',lease_owner=NULL,lease_until=NULL WHERE id=$1", j.ID)
		return e
	})
}
