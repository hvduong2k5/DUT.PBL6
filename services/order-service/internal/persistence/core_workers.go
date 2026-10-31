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

type CoreCleanup interface {
	Cleanup(context.Context, string, string, int64) (string, error)
}

func (s *Store) RootScope(ctx context.Context, id string) (string, error) {
	var scope string
	e := s.db.QueryRow(ctx, "SELECT scope FROM orders WHERE id=$1 AND cohort=$2", id, Cohort).Scan(&scope)
	return scope, e
}
func (s *Store) CancelPending(ctx context.Context, id, scope, key string, version int64, reason string) (core.Aggregate, error) {
	if !validID(id) || !commandKey(key) || len(reason) > 1000 {
		return core.Aggregate{}, ErrInput
	}
	var result core.Aggregate
	request, _ := json.Marshal([]any{id, version, reason})
	requestHash := hash(request)
	e := s.runner.Do(ctx, func(tx pgx.Tx) error {
		var locked string
		if e := tx.QueryRow(ctx, "SELECT id FROM orders WHERE id=$1 AND scope=$2 AND cohort=$3 FOR UPDATE", id, scope, Cohort).Scan(&locked); e != nil {
			return e
		}
		a, e := s.loadOrder(ctx, tx, id, scope)
		if e != nil {
			return e
		}
		var prior string
		e = tx.QueryRow(ctx, "SELECT request_hash FROM command_registry WHERE scope=$1 AND kind=$2 AND key=$3", scope, "CORE_CANCEL:"+id, key).Scan(&prior)
		if e == nil {
			if prior != requestHash {
				return ErrConflict
			}
			result = a
			return nil
		}
		if !errors.Is(e, pgx.ErrNoRows) {
			return e
		}
		if a.Version() != version {
			return ErrConflict
		}
		var now time.Time
		if e = tx.QueryRow(ctx, "SELECT clock_timestamp()").Scan(&now); e != nil {
			return e
		}
		next, e := a.CancelPending(core.DecisionContext{ExpectedVersion: version, At: now}, a.Owner())
		if e != nil {
			return e
		}
		if e = s.finishCancellation(ctx, tx, a, next, now); e != nil {
			return e
		}
		result = next
		_, e = tx.Exec(ctx, "INSERT INTO command_registry(scope,kind,key,request_hash,order_id,response) VALUES($1,$2,$3,$4,$5,'{}')", scope, "CORE_CANCEL:"+id, key, requestHash, id)
		return e
	})
	return result, e
}
func (s *Store) finishCancellation(ctx context.Context, tx pgx.Tx, a, next core.Aggregate, now time.Time) error {
	if e := s.savePaymentRoot(ctx, tx, a, next, "UNPAID", "CANCEL_UNPAID"); e != nil {
		return e
	}
	if _, e := tx.Exec(ctx, "UPDATE payment_intents SET status='CLOSED',version=version+1 WHERE order_id=$1 AND status='ACTIVE'", a.ID()); e != nil {
		return e
	}
	if e := s.followup(ctx, tx, uuid.NewString(), uuid.NewString(), "STOCK_RELEASE", a.ID(), a.Snapshot().Data().OperationID, "RECOVERY", now); e != nil {
		return e
	}
	return s.cancelledEvent(ctx, tx, next, "PENDING", now)
}
func (s *Store) cancelledEvent(ctx context.Context, tx pgx.Tx, a core.Aggregate, compensation string, now time.Time) error {
	data := map[string]any{"aggregate_id": a.ID(), "version": a.Version(), "ordinal": 0, "status": a.State(), "payment_status": "UNPAID", "stock_status": a.Record().Reservation.State, "cancellation_id": uuid.NewSHA1(uuid.NameSpaceOID, []byte("cancel:"+a.ID())).String(), "compensation_status": compensation}
	return s.paymentEvent(ctx, tx, "ORDER", a.ID(), "order.events.v3", "vn.omama.order.cancelled.v3", "https://contracts.omama.example/order/v3/cancelled.event.json", CancelledSchemaHash, a.Version(), 0, data, a.ID(), now)
}
func (s *Store) SweepCoreTimeouts(ctx context.Context) error {
	rows, e := s.db.Query(ctx, "SELECT id,scope FROM orders WHERE cohort=$1 AND status='PENDING_PAYMENT' AND payment_expires_at<=clock_timestamp() ORDER BY payment_expires_at LIMIT 100", Cohort)
	if e != nil {
		return e
	}
	type candidate struct{ id, scope string }
	list := []candidate{}
	for rows.Next() {
		var c candidate
		if e = rows.Scan(&c.id, &c.scope); e != nil {
			rows.Close()
			return e
		}
		list = append(list, c)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	for _, c := range list {
		e = s.runner.Do(ctx, func(tx pgx.Tx) error {
			var id string
			if e := tx.QueryRow(ctx, "SELECT id FROM orders WHERE id=$1 FOR UPDATE", c.id).Scan(&id); e != nil {
				return e
			}
			a, e := s.loadOrder(ctx, tx, c.id, c.scope)
			if e != nil {
				return e
			}
			if a.State() != core.PendingPayment {
				return nil
			}
			var now time.Time
			if e = tx.QueryRow(ctx, "SELECT clock_timestamp()").Scan(&now); e != nil {
				return e
			}
			next, e := a.Timeout(core.DecisionContext{ExpectedVersion: a.Version(), At: now})
			if e != nil {
				return e
			}
			return s.finishCancellation(ctx, tx, a, next, now)
		})
		if e != nil {
			return e
		}
	}
	return nil
}
func (r *CheckoutRuntime) WorkFollowup(ctx context.Context, cleanup CoreCleanup) (bool, error) {
	j := PaymentJob{Owner: uuid.NewString()}
	var kind string
	e := r.Store.db.QueryRow(ctx, `WITH pick AS(SELECT id FROM jobs WHERE cohort=$1 AND kind IN ('CART_CLEANUP','FULFILLMENT_READY','STOCK_RELEASE') AND ((status IN ('READY','RETRY_WAIT') AND run_after<=clock_timestamp()) OR(status='LEASED' AND lease_until<clock_timestamp())) ORDER BY run_after,id FOR UPDATE SKIP LOCKED LIMIT 1) UPDATE jobs j SET status='LEASED',lease_owner=$2,lease_epoch=lease_epoch+1,lease_until=clock_timestamp()+interval '30 seconds',attempts=attempts+1 FROM pick WHERE j.id=pick.id RETURNING j.id,j.order_id,j.workflow_id,j.lease_epoch,j.kind`, Cohort, j.Owner).Scan(&j.ID, &j.Order, &j.Workflow, &j.Epoch, &kind)
	if errors.Is(e, pgx.ErrNoRows) {
		return false, nil
	}
	if e != nil {
		return false, e
	}
	scope, e := r.Store.RootScope(ctx, j.Order)
	if e != nil {
		return true, e
	}
	a, e := r.Store.LoadOrder(ctx, j.Order, scope)
	if e != nil {
		return true, e
	}
	var released core.ResourceProof
	cleanupOutcome := ""
	switch kind {
	case "CART_CLEANUP":
		if cleanup == nil {
			e = ErrCapability
		} else {
			cleanupOutcome, e = cleanup.Cleanup(ctx, scope, a.Snapshot().Data().Provenance.CartID, a.Snapshot().Data().Provenance.CartRevision)
		}
	case "STOCK_RELEASE":
		released, e = r.Inventory.Release(ctx, a.Snapshot().Data(), a.Record().Reservation, "inventory:release:"+a.ID())
	}
	p := &Payment{Store: r.Store}
	if e != nil {
		_ = p.retry(context.WithoutCancel(ctx), j, time.Second, "FOLLOWUP_RETRY")
		return true, e
	}
	e = r.Store.runner.Do(ctx, func(tx pgx.Tx) error {
		var id string
		if e := tx.QueryRow(ctx, "SELECT id FROM orders WHERE id=$1 FOR UPDATE", a.ID()).Scan(&id); e != nil {
			return e
		}
		current, e := r.Store.loadOrder(ctx, tx, a.ID(), scope)
		if e != nil {
			return e
		}
		if e = p.fence(ctx, tx, j); e != nil {
			return e
		}
		var now time.Time
		if e = tx.QueryRow(ctx, "SELECT clock_timestamp()").Scan(&now); e != nil {
			return e
		}
		switch kind {
		case "CART_CLEANUP":
			if _, e = tx.Exec(ctx, "UPDATE jobs SET payload=jsonb_set(payload,'{cleanup_status}',to_jsonb($2::text),true) WHERE id=$1", j.ID, cleanupOutcome); e != nil {
				return e
			}
		case "FULFILLMENT_READY":
			var activeCase bool
			if e = tx.QueryRow(ctx, "SELECT has_active_case FROM orders WHERE id=$1", current.ID()).Scan(&activeCase); e != nil {
				return e
			}
			if current.ReadyEligible() && !activeCase {
				next, e := current.AuthorizeReady(core.DecisionContext{ExpectedVersion: current.Version(), At: now})
				if e != nil {
					return e
				}
				payment := "CONFIRMED"
				if current.Snapshot().Data().Payment.Method == "COD" {
					payment = "UNPAID"
				}
				if e = r.Store.savePaymentRoot(ctx, tx, current, next, payment, "FULFILLMENT_READY"); e != nil {
					return e
				}
				var generation int64
				if e = tx.QueryRow(ctx, "UPDATE orders SET ready_authorized=true,ready_generation=ready_generation+1 WHERE id=$1 RETURNING ready_generation", current.ID()).Scan(&generation); e != nil {
					return e
				}
				s := next.Snapshot().Data()
				proof := next.Record().Reservation
				data := map[string]any{"aggregate_id": next.ID(), "version": next.Version(), "ordinal": 0, "status": next.State(), "payment_status": payment, "stock_status": "COMMITTED", "items_digest": s.Provenance.ItemsDigest, "warehouse_id": s.WarehouseID, "reservation_id": proof.ReservationID, "stock_resource_version": proof.ResourceVersion, "ready_generation": generation, "policy_version": s.Policies.Order}
				if e = r.Store.paymentEvent(ctx, tx, "ORDER", next.ID(), "order.events.v3", "vn.omama.order.fulfillment_ready.v3", "https://contracts.omama.example/order/v3/fulfillment_ready.event.json", ReadySchemaHash, next.Version(), 0, data, next.ID(), now); e != nil {
					return e
				}
			} else if !current.State().Terminal() {
				_, e = tx.Exec(ctx, "UPDATE jobs SET status='PARKED',error_code='READY_BLOCKED',lease_owner=NULL,lease_until=NULL WHERE id=$1", j.ID)
				return e
			}
		case "STOCK_RELEASE":
			next, e := current.ConfirmReleased(core.DecisionContext{ExpectedVersion: current.Version(), At: now}, released)
			if e != nil {
				return e
			}
			if e = r.Store.savePaymentRoot(ctx, tx, current, next, "UNPAID", "STOCK_RELEASED"); e != nil {
				return e
			}
			if e = r.Store.cancelledEvent(ctx, tx, next, "SUCCEEDED", now); e != nil {
				return e
			}
		}
		if _, e = tx.Exec(ctx, "UPDATE workflows SET state='COMPLETED',completion_outcome='SUCCEEDED',version=version+1 WHERE id=$1", j.Workflow); e != nil {
			return e
		}
		_, e = tx.Exec(ctx, "UPDATE jobs SET status='DONE',lease_owner=NULL,lease_until=NULL WHERE id=$1", j.ID)
		return e
	})
	if e != nil {
		_ = p.retry(context.WithoutCancel(ctx), j, time.Second, "FOLLOWUP_RETRY")
	}
	return true, e
}
