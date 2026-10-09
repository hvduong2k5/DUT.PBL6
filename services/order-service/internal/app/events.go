package app

import (
	"context"
	"crypto/hmac"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/omamx/order-service/internal/domain"
	"time"
)

func (s *Service) ProcessEvent(ctx context.Context, event domain.Event) (string, error) {
	if !hmac.Equal([]byte(event.Signature), []byte(domain.EventSignature(event, s.C.InternalToken))) {
		return "", domain.E("INVALID_EVENT_SIGNATURE", 422)
	}
	if event.ID == "" || len(event.ID) > 128 || event.ResourceID == "" || event.Version <= 0 {
		return "", domain.E("INVALID_EVENT", 422)
	}
	if event.Source != "fulfillment" && event.Source != "shipping" && event.Source != "care" {
		return "", domain.E("INVALID_EVENT_SOURCE", 422)
	}
	if _, e := uuid.Parse(event.OrderID); e != nil {
		return "", domain.E("INVALID_EVENT", 422)
	}
	var resource struct {
		OrderID    string `json:"order_id"`
		Source     string `json:"source"`
		Version    int64  `json:"version"`
		Generation int64  `json:"generation"`
	}
	if e := s.Call(ctx, "GET", "/resources/"+event.ResourceID, nil, &resource); e != nil {
		code, status := s.ErrorCode(e)
		if status == 404 {
			return "", domain.E("DEPENDENCY_UNKNOWN", 503)
		}
		_ = code
		return "", e
	}
	if resource.OrderID != event.OrderID || resource.Source != event.Source || event.Version > resource.Version || event.Generation != resource.Generation {
		return "", domain.E("RESOURCE_MISMATCH", 422)
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return "", e
	}
	defer tx.Rollback(ctx)
	o, orderErr := s.lockOrder(ctx, tx, event.OrderID)
	if orderErr != nil && !errors.Is(orderErr, pgx.ErrNoRows) {
		return "", orderErr
	}
	b, e := json.Marshal(event)
	if e != nil {
		return "", e
	}
	hash := domain.Hash(event)
	_, e = tx.Exec(ctx, `INSERT INTO inbox(source,event_id,hash,status,order_id,resource_id,source_version,payload) VALUES($1,$2,$3,'DEFERRED',$4,$5,$6,$7) ON CONFLICT DO NOTHING`, event.Source, event.ID, hash, event.OrderID, event.ResourceID, event.Version, b)
	if e != nil {
		return "", e
	}
	var old, state string
	if e = tx.QueryRow(ctx, "SELECT hash,status FROM inbox WHERE source=$1 AND event_id=$2 FOR UPDATE", event.Source, event.ID).Scan(&old, &state); e != nil {
		return "", e
	}
	if old != hash {
		return "", domain.E("EVENT_ID_CONFLICT", 409)
	}
	if state == "APPLIED" || state == "REJECTED" {
		return state, tx.Commit(ctx)
	}
	if orderErr != nil {
		return "DEFERRED", tx.Commit(ctx)
	}
	var last int64
	e = tx.QueryRow(ctx, "SELECT source_version FROM source_projections WHERE order_id=$1 AND source=$2 AND resource_id=$3", o.ID, event.Source, event.ResourceID).Scan(&last)
	if e != nil && !errors.Is(e, pgx.ErrNoRows) {
		return "", e
	}
	apply := true
	rejected := false
	changed := false
	target := ""
	if event.Version > last {
		switch event.Type {
		case "PACKING_ACCEPTED":
			if event.Source != "fulfillment" || event.Generation != o.Generation || o.Hold || !o.Ready || o.Stock != "COMMITTED" {
				if o.Status == "CANCELLED_BY_USER" || o.Status == "CANCELLED_BY_ADMIN" || o.Status == "CANCELLED_TIMEOUT" {
					rejected = true
				} else {
					apply = false
				}
			} else if o.Status == "PAID" || o.Status == "CONFIRMED_COD" {
				o.Task = &event.ResourceID
				target = "PROCESSING"
			} else if o.Status != "PROCESSING" {
				rejected = true
			}
		case "PACKING_SEALED":
			if event.Source != "fulfillment" || event.Generation != o.Generation {
				rejected = true
			} else if o.Task == nil || o.Status != "PROCESSING" {
				apply = false
			} else if *o.Task != event.ResourceID || event.Seal == "" {
				rejected = true
			} else {
				o.Seal = &event.Seal
				target = "PACKED"
			}
		case "SHIPMENT_CREATED":
			if event.Source != "shipping" {
				rejected = true
			} else if o.Status != "PACKED" {
				apply = false
			} else {
				o.Shipment = &event.ResourceID
				changed = true
			}
		case "SHIPMENT_DISPATCHED":
			if event.Source != "shipping" {
				rejected = true
			} else if o.Shipment == nil || (o.Status != "PACKED" && o.Status != "DELIVERY_FAILED") {
				apply = false
			} else if *o.Shipment != event.ResourceID {
				rejected = true
			} else {
				target = "SHIPPED"
			}
		case "SHIPMENT_DELIVERED", "SHIPMENT_FAILED":
			if event.Source != "shipping" {
				rejected = true
			} else if o.Shipment == nil || o.Status != "SHIPPED" {
				apply = false
			} else if *o.Shipment != event.ResourceID {
				rejected = true
			} else {
				target = "DELIVERY_FAILED"
				if event.Type == "SHIPMENT_DELIVERED" {
					target = "DELIVERED"
					due := time.Now().UTC().Add(7 * 24 * time.Hour)
					o.CompletionDue = &due
				}
			}
		case "CASE_OPENED", "CASE_CLOSED":
			if event.Source != "care" {
				rejected = true
			} else {
				o.ActiveCase = event.Type == "CASE_OPENED"
				changed = true
			}
		default:
			rejected = true
		}
		if apply && !rejected {
			if target != "" {
				if e = s.transition(ctx, tx, &o, target, event.Source, "SOURCE_RESULT"); e != nil {
					return "", e
				}
			} else if changed {
				if e = s.touch(ctx, tx, &o, event.Source); e != nil {
					return "", e
				}
			}
			_, e = tx.Exec(ctx, `INSERT INTO source_projections(order_id,source,resource_id,source_version) VALUES($1,$2,$3,$4) ON CONFLICT(order_id,source,resource_id) DO UPDATE SET source_version=EXCLUDED.source_version`, o.ID, event.Source, event.ResourceID, event.Version)
			if e != nil {
				return "", e
			}
		}
	}
	state = "APPLIED"
	if !apply {
		state = "DEFERRED"
	}
	if rejected {
		state = "REJECTED"
		if e = s.Audit(ctx, tx, o.ID, event.Source, "EVENT_REJECTED", map[string]any{"event_id": event.ID, "type": event.Type}); e != nil {
			return "", e
		}
	}
	_, e = tx.Exec(ctx, "UPDATE inbox SET status=$3,run_after=clock_timestamp()+interval '1 second' WHERE source=$1 AND event_id=$2", event.Source, event.ID, state)
	if e != nil {
		return "", e
	}
	return state, tx.Commit(ctx)
}
func (s *Service) ReplayDeferred(ctx context.Context) error {
	rows, e := s.DB.Query(ctx, "SELECT payload FROM inbox WHERE status='DEFERRED' AND run_after<=clock_timestamp() ORDER BY source_version LIMIT 100")
	if e != nil {
		return e
	}
	events := []domain.Event{}
	for rows.Next() {
		var b []byte
		if e = rows.Scan(&b); e != nil {
			rows.Close()
			return e
		}
		var event domain.Event
		if e = json.Unmarshal(b, &event); e != nil {
			rows.Close()
			return e
		}
		events = append(events, event)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	for _, event := range events {
		if _, e = s.ProcessEvent(ctx, event); e != nil {
			return e
		}
	}
	return nil
}
func (s *Service) CompleteOrders(ctx context.Context) error {
	if !s.C.AutoComplete {
		return nil
	}
	rows, e := s.DB.Query(ctx, "SELECT id FROM orders WHERE status='DELIVERED' AND completion_due_at<=clock_timestamp() AND payment_status IN ('CONFIRMED','PARTIALLY_REFUNDED') AND operational_hold=false AND NOT EXISTS(SELECT 1 FROM refunds f WHERE f.order_id=orders.id AND f.status IN ('APPROVED','SUBMITTED','UNKNOWN','MANUAL_REVIEW')) LIMIT 100")
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
		var barrier struct {
			Eligible bool `json:"eligible"`
		}
		if e = s.Call(ctx, "POST", "/care/completion-barrier", map[string]any{"order_id": id}, &barrier); e != nil || !barrier.Eligible {
			continue
		}
		tx, e := s.DB.Begin(ctx)
		if e != nil {
			return e
		}
		o, e := s.lockOrder(ctx, tx, id)
		if e != nil {
			_ = tx.Rollback(ctx)
			continue
		}
		if o.Status == "DELIVERED" && !o.ActiveCase && !o.Hold && (o.Payment == "CONFIRMED" || o.Payment == "PARTIALLY_REFUNDED") {
			e = s.transition(ctx, tx, &o, "COMPLETED", "CARE_BARRIER", "WINDOW_CLOSED")
			if e == nil {
				e = s.Emit(ctx, tx, o.ID, "ORDER", o.Version, 0, "completed", o.Status, o.Payment, o.Stock, o.Generation)
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
func (s *Service) Recover(ctx context.Context, p domain.Principal, id string) error {
	if !p.Manager() {
		return domain.E("PERMISSION_DENIED", 403)
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	var order *string
	var scope, status, stage string
	e = tx.QueryRow(ctx, "SELECT order_id,scope,status,stage FROM checkout_operations WHERE id=$1 FOR UPDATE", id).Scan(&order, &scope, &status, &stage)
	if e == nil {
		if p.Role != "ADMIN" {
			return domain.E("PERMISSION_DENIED", 403)
		}
		if status != "MANUAL_REVIEW" {
			return domain.E("RECOVERY_NOT_ALLOWED", 409)
		}
		_, e = tx.Exec(ctx, "UPDATE checkout_operations SET status='WAITING_RETRY',attempts=0,lease_owner=NULL,lease_until=NULL,lease_epoch=lease_epoch+1,run_after=clock_timestamp() WHERE id=$1", id)
		if e != nil {
			return e
		}
		if e = s.Audit(ctx, tx, "", p.ID, "CHECKOUT_RESUME", map[string]any{"operation_id": id, "stage": stage}); e != nil {
			return e
		}
	} else if errors.Is(e, pgx.ErrNoRows) {
		var oid string
		e = tx.QueryRow(ctx, "SELECT order_id FROM jobs WHERE id=$1 AND status='MANUAL_REVIEW'", id).Scan(&oid)
		if e != nil {
			return e
		}
		o, e := s.lockOrder(ctx, tx, oid)
		if e != nil {
			return e
		}
		if !CanRead(p, o) {
			return domain.E("ORDER_NOT_FOUND", 404)
		}
		_, e = tx.Exec(ctx, "UPDATE jobs SET status='RETRY',attempts=0,lease_owner=NULL,lease_until=NULL,lease_epoch=lease_epoch+1,run_after=clock_timestamp() WHERE id=$1", id)
		if e != nil {
			return e
		}
		if e = s.Audit(ctx, tx, oid, p.ID, "JOB_RESUME", map[string]any{"job_id": id}); e != nil {
			return e
		}
	} else {
		return e
	}
	return tx.Commit(ctx)
}
func (s *Service) Quarantine(ctx context.Context, key string, payload []byte, code string) error {
	_, e := s.DB.Exec(ctx, "INSERT INTO quarantine(message_key,payload_hash,error_code) VALUES($1,$2,$3) ON CONFLICT DO NOTHING", key, domain.Hash(string(payload)), code)
	return e
}
