package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/omamx/order-service/internal/domain"
	"github.com/omamx/order-service/internal/domain/core"
	"time"
)

type SourceEvidence struct {
	Order      string    `json:"order_id"`
	Resource   string    `json:"resource_id"`
	Source     string    `json:"source"`
	Kind       string    `json:"kind"`
	Version    int64     `json:"version"`
	Generation int64     `json:"generation"`
	Seal       string    `json:"seal_code"`
	Occurred   time.Time `json:"occurred_at"`
	Settled    bool      `json:"settlement_verified"`
}
type CompletionDecision struct {
	Order      string    `json:"order_id"`
	Version    int64     `json:"order_version"`
	Allowed    bool      `json:"allowed"`
	Decision   string    `json:"decision_id"`
	Expires    time.Time `json:"expires_at"`
	Settlement bool      `json:"settlement_verified"`
}
type CompletionAuthority interface {
	Completion(context.Context, string, int64) (CompletionDecision, error)
}

func (s *Store) ApplySource(ctx context.Context, event domain.Event, proof SourceEvidence) (out string, err error) {
	expectedSource := "fulfillment"
	switch event.Type {
	case "SHIPMENT_CREATED", "SHIPMENT_DISPATCHED", "SHIPMENT_DELIVERED", "SHIPMENT_FAILED":
		expectedSource = "shipping"
	case "CASE_OPENED", "CASE_CLOSED":
		expectedSource = "care"
	}
	if !validID(event.OrderID) || !validID(event.ResourceID) || event.Source != expectedSource || proof.Order != event.OrderID || proof.Resource != event.ResourceID || proof.Source != event.Source || proof.Kind != event.Type || proof.Version != event.Version || proof.Generation != event.Generation || proof.Seal != event.Seal {
		return "", ErrInput
	}
	raw, _ := json.Marshal(event)
	requestHash := hash(raw)
	source := "core:" + event.Source
	canonical, _ := json.Marshal(proof)
	cipher, e := s.cipher.Seal("source:"+source+":"+event.ID, canonical)
	if e != nil {
		return "", e
	}
	err = s.runner.Do(ctx, func(tx pgx.Tx) error {
		var scope string
		if e := tx.QueryRow(ctx, "SELECT scope FROM orders WHERE id=$1 AND cohort=$2 FOR UPDATE", event.OrderID, Cohort).Scan(&scope); e != nil {
			if !errors.Is(e, pgx.ErrNoRows) {
				return e
			}
			var operation string
			if e = tx.QueryRow(ctx, "SELECT id FROM checkout_operations WHERE planned_order_id=$1 AND cohort=$2 FOR SHARE", event.OrderID, Cohort).Scan(&operation); e != nil {
				return ErrCapability
			}
			if _, e = tx.Exec(ctx, "INSERT INTO inbox(source,event_id,hash,status,order_id,resource_id,source_version,payload) VALUES($1,$2,$3,'DEFERRED',$4,$5,$6,$7) ON CONFLICT DO NOTHING", source, event.ID, requestHash, event.OrderID, event.ResourceID, event.Version, raw); e != nil {
				return e
			}
			var prior string
			if e = tx.QueryRow(ctx, "SELECT hash FROM inbox WHERE source=$1 AND event_id=$2", source, event.ID).Scan(&prior); e != nil {
				return e
			}
			if prior != requestHash {
				return ErrConflict
			}
			out = "DEFERRED"
			return nil
		}
		a, e := s.loadOrder(ctx, tx, event.OrderID, scope)
		if e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, "INSERT INTO inbox(source,event_id,hash,status,order_id,resource_id,source_version,payload) VALUES($1,$2,$3,'DEFERRED',$4,$5,$6,$7) ON CONFLICT DO NOTHING", source, event.ID, requestHash, event.OrderID, event.ResourceID, event.Version, raw); e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, "INSERT INTO core_source_evidence(source,event_id,order_id,resource_id,source_version,evidence_hash,evidence_cipher,occurred_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT DO NOTHING", source, event.ID, event.OrderID, event.ResourceID, event.Version, hash(canonical), cipher, proof.Occurred); e != nil {
			return e
		}
		var pinned string
		if e = tx.QueryRow(ctx, "SELECT evidence_hash FROM core_source_evidence WHERE source=$1 AND event_id=$2", source, event.ID).Scan(&pinned); e != nil {
			return e
		}
		if pinned != hash(canonical) {
			return ErrIntegrity
		}
		var prior, state string
		if e = tx.QueryRow(ctx, "SELECT hash,status FROM inbox WHERE source=$1 AND event_id=$2 FOR UPDATE", source, event.ID).Scan(&prior, &state); e != nil {
			return e
		}
		if prior != requestHash {
			return ErrConflict
		}
		if state == "APPLIED" || state == "REJECTED" {
			out = state
			return nil
		}
		var last int64
		e = tx.QueryRow(ctx, "SELECT source_version FROM source_projections WHERE order_id=$1 AND source=$2 AND resource_id=$3", event.OrderID, source, event.ResourceID).Scan(&last)
		if e != nil && !errors.Is(e, pgx.ErrNoRows) {
			return e
		}
		var ready, activeCase bool
		var generation int64
		var task, shipment, seal *string
		if e = tx.QueryRow(ctx, "SELECT ready_authorized,ready_generation,task_id,shipment_id,seal_code,has_active_case FROM orders WHERE id=$1", event.OrderID).Scan(&ready, &generation, &task, &shipment, &seal, &activeCase); e != nil {
			return e
		}
		var now time.Time
		if e = tx.QueryRow(ctx, "SELECT clock_timestamp()").Scan(&now); e != nil {
			return e
		}
		next := a
		out = "APPLIED"
		if event.Version > last {
			switch event.Type {
			case "PACKING_ACCEPTED":
				if !ready || activeCase || event.Generation != generation {
					out = "DEFERRED"
				}
				if task != nil && *task != event.ResourceID {
					out = "REJECTED"
				}
			case "PACKING_SEALED":
				if task == nil || a.State() != core.Processing {
					out = "DEFERRED"
				} else if *task != event.ResourceID || generation != event.Generation {
					out = "REJECTED"
				}
			case "SHIPMENT_CREATED":
				if a.State() != core.Packed {
					out = "DEFERRED"
				}
				if shipment != nil && *shipment != event.ResourceID {
					out = "REJECTED"
				}
			case "SHIPMENT_DISPATCHED", "SHIPMENT_DELIVERED", "SHIPMENT_FAILED":
				if shipment == nil {
					out = "DEFERRED"
				} else if *shipment != event.ResourceID {
					out = "REJECTED"
				}
			}
			if a.State().Terminal() {
				out = "REJECTED"
			}
			if out == "APPLIED" {
				next, e = a.ApplyFulfillment(core.DecisionContext{ExpectedVersion: a.Version(), At: now}, core.FulfillmentProof{OrderID: proof.Order, ResourceID: proof.Resource, Kind: proof.Kind, Version: proof.Version, Generation: proof.Generation, Seal: proof.Seal, At: proof.Occurred})
				if errors.Is(e, core.ErrState) || errors.Is(e, core.ErrBlocked) {
					out = "DEFERRED"
				} else if e != nil {
					return e
				}
			}
			if out == "APPLIED" {
				var payment string
				if e = tx.QueryRow(ctx, "SELECT payment_status FROM orders WHERE id=$1", a.ID()).Scan(&payment); e != nil {
					return e
				}
				if e = s.savePaymentRoot(ctx, tx, a, next, payment, event.Type); e != nil {
					return e
				}
				switch event.Type {
				case "PACKING_ACCEPTED":
					_, e = tx.Exec(ctx, "UPDATE orders SET task_id=$2 WHERE id=$1", a.ID(), event.ResourceID)
				case "PACKING_SEALED":
					_, e = tx.Exec(ctx, "UPDATE orders SET seal_code=$2 WHERE id=$1", a.ID(), event.Seal)
				case "SHIPMENT_CREATED":
					_, e = tx.Exec(ctx, "UPDATE orders SET shipment_id=$2 WHERE id=$1", a.ID(), event.ResourceID)
				case "SHIPMENT_DELIVERED":
					_, e = tx.Exec(ctx, "UPDATE orders SET completion_due_at=$2::timestamptz+interval '7 days' WHERE id=$1", a.ID(), proof.Occurred)
					if e == nil {
						_, e = tx.Exec(ctx, "INSERT INTO core_lifecycle_facts(order_id,delivered_at,source,event_id) VALUES($1,$2,$3,$4)", a.ID(), proof.Occurred, source, event.ID)
					}
				case "CASE_OPENED", "CASE_CLOSED":
					_, e = tx.Exec(ctx, "UPDATE orders SET has_active_case=$2 WHERE id=$1", a.ID(), event.Type == "CASE_OPENED")
					if e == nil && event.Type == "CASE_CLOSED" && next.ReadyEligible() {
						_, e = tx.Exec(ctx, "UPDATE jobs SET status='READY',lease_owner=NULL,lease_until=NULL,run_after=clock_timestamp() WHERE order_id=$1 AND kind='FULFILLMENT_READY' AND status IN ('DONE','PARKED')", a.ID())
					}
				}
				if e != nil {
					return e
				}
				_, e = tx.Exec(ctx, "INSERT INTO source_projections(order_id,source,resource_id,source_version) VALUES($1,$2,$3,$4) ON CONFLICT(order_id,source,resource_id) DO UPDATE SET source_version=excluded.source_version", a.ID(), source, event.ResourceID, event.Version)
				if e != nil {
					return e
				}
			}
		}
		_, e = tx.Exec(ctx, "UPDATE inbox SET status=$3,run_after=clock_timestamp()+interval '1 second' WHERE source=$1 AND event_id=$2", source, event.ID, out)
		return e
	})
	return
}
func (s *Store) CompleteDue(ctx context.Context, authority CompletionAuthority) error {
	if authority == nil {
		return ErrCapability
	}
	rows, e := s.db.Query(ctx, "SELECT id,scope,version FROM orders WHERE cohort=$1 AND status='DELIVERED' AND completion_due_at<=clock_timestamp() AND NOT operational_hold AND NOT has_active_case ORDER BY completion_due_at LIMIT 50", Cohort)
	if e != nil {
		return e
	}
	type candidate struct {
		id, scope string
		version   int64
	}
	list := []candidate{}
	for rows.Next() {
		var c candidate
		if e = rows.Scan(&c.id, &c.scope, &c.version); e != nil {
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
		decision, e := authority.Completion(ctx, c.id, c.version)
		if e != nil {
			return e
		}
		if decision.Order != c.id || decision.Version != c.version || !decision.Allowed || decision.Decision == "" {
			continue
		}
		e = s.runner.Do(ctx, func(tx pgx.Tx) error {
			var id string
			if e := tx.QueryRow(ctx, "SELECT id FROM orders WHERE id=$1 FOR UPDATE", c.id).Scan(&id); e != nil {
				return e
			}
			a, e := s.loadOrder(ctx, tx, c.id, c.scope)
			if e != nil {
				return e
			}
			if a.State() != core.Delivered || a.Version() != c.version {
				return nil
			}
			var due, now, delivered time.Time
			var active, held, refund bool
			if e = tx.QueryRow(ctx, `SELECT completion_due_at,clock_timestamp(),(SELECT delivered_at FROM core_lifecycle_facts WHERE order_id=$1),has_active_case,operational_hold,EXISTS(SELECT 1 FROM refunds WHERE order_id=$1 AND budget_held AND status<>'SUCCEEDED') FROM orders WHERE id=$1`, c.id).Scan(&due, &now, &delivered, &active, &held, &refund); e != nil {
				return e
			}
			if !decision.Expires.After(now) {
				return ErrExpired
			}
			settled := a.HasMilestone(core.FirstPaid) || a.Snapshot().Data().Payment.Method == "COD" && decision.Settlement
			eligible := core.CompletionEligibility{State: a.State(), At: now, DeliveredAt: delivered, WindowEndsAt: due, CareOrderID: decision.Order, OrderID: c.id, CareDecisionID: decision.Decision, CareAllowed: true, SettlementVerified: settled, BlockingHold: held || a.Record().Blocks != 0, ActiveCase: active, UnresolvedRefund: refund, UnresolvedStock: a.Record().Reservation.State != "COMMITTED"}
			next, e := a.Complete(core.DecisionContext{ExpectedVersion: a.Version(), At: now}, eligible)
			if errors.Is(e, core.ErrBlocked) {
				return nil
			}
			if e != nil {
				return e
			}
			var payment string
			if e = tx.QueryRow(ctx, "SELECT payment_status FROM orders WHERE id=$1", c.id).Scan(&payment); e != nil {
				return e
			}
			if e = s.savePaymentRoot(ctx, tx, a, next, payment, "COMPLETED"); e != nil {
				return e
			}
			if _, e = tx.Exec(ctx, "INSERT INTO order_milestones(order_id,kind,decision_ref,occurred_at) VALUES($1,'COMPLETED',$2,$3)", c.id, decision.Decision, now); e != nil {
				return e
			}
			data := map[string]any{"aggregate_id": c.id, "version": next.Version(), "ordinal": 0, "status": "COMPLETED", "payment_status": payment, "stock_status": "COMMITTED", "completion_decision_id": decision.Decision}
			return s.paymentEvent(ctx, tx, "ORDER", c.id, "order.events.v3", "vn.omama.order.completed.v3", "https://contracts.omama.example/order/v3/completed.event.json", CompletedSchemaHash, next.Version(), 0, data, c.id, now)
		})
		if e != nil {
			return e
		}
	}
	return nil
}
