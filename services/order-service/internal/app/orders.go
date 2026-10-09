package app

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/omamx/order-service/internal/domain"
	"strconv"
	"time"
)

const orderColumns = `id,operation_id,order_code,scope,principal_id,customer_id,warehouse_id,method,status,payment_status,stock_status,subtotal,shipping_fee,final_amount,version,ready_generation,ready_authorized,operational_hold,has_active_case,reservation_id,provider_reference,task_id,shipment_id,seal_code,payment_expires_at,created_at,sla_due_at,completion_due_at,updated_at`

func scanOrder(row pgx.Row) (domain.Order, error) {
	var o domain.Order
	e := row.Scan(&o.ID, &o.OperationID, &o.Code, &o.Scope, &o.PrincipalID, &o.CustomerID, &o.Warehouse, &o.Method, &o.Status, &o.Payment, &o.Stock, &o.Subtotal, &o.Shipping, &o.Final, &o.Version, &o.Generation, &o.Ready, &o.Hold, &o.ActiveCase, &o.Reservation, &o.Reference, &o.Task, &o.Shipment, &o.Seal, &o.ExpiresAt, &o.CreatedAt, &o.SLA, &o.CompletionDue, &o.UpdatedAt)
	return o, e
}
func (s *Service) lockOrder(ctx context.Context, tx pgx.Tx, id string) (domain.Order, error) {
	return scanOrder(tx.QueryRow(ctx, "SELECT "+orderColumns+" FROM orders WHERE id=$1 FOR UPDATE", id))
}
func CanRead(p domain.Principal, o domain.Order) bool {
	if p.Scope() == o.Scope {
		return true
	}
	return p.Staff() && (p.Role == "ADMIN" || p.Warehouse == o.Warehouse)
}
func (s *Service) GetOrder(ctx context.Context, id string, p domain.Principal, detail bool) (domain.Order, error) {
	if _, e := uuid.Parse(id); e != nil {
		return domain.Order{}, domain.E("ORDER_NOT_FOUND", 404)
	}
	o, e := scanOrder(s.DB.QueryRow(ctx, "SELECT "+orderColumns+" FROM orders WHERE id=$1", id))
	if e != nil {
		return o, e
	}
	if !CanRead(p, o) {
		return domain.Order{}, domain.E("ORDER_NOT_FOUND", 404)
	}
	if detail {
		var data []byte
		if e = s.DB.QueryRow(ctx, "SELECT snapshot_cipher FROM orders WHERE id=$1", id).Scan(&data); e != nil {
			return o, e
		}
		var q domain.Quote
		if e = s.Open(data, &q); e != nil {
			return o, e
		}
		if p.Role == "CUSTOMER_SERVICE" {
			q.Address.Street = "[masked]"
			q.Address.Name = "[masked]"
			if len(q.Address.Phone) > 4 {
				q.Address.Phone = "***" + q.Address.Phone[len(q.Address.Phone)-4:]
			}
		}
		o.Snapshot = &q
	}
	return o, nil
}
func (s *Service) ListOrders(ctx context.Context, p domain.Principal, staff, queue bool, status, cursor string, limit int) ([]domain.Order, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if staff && !p.Staff() {
		return nil, domain.E("PERMISSION_DENIED", 403)
	}
	args := []any{}
	query := "SELECT " + orderColumns + " FROM orders WHERE TRUE"
	add := func(v any) string { args = append(args, v); return "$" + strconv.Itoa(len(args)) }
	if !staff {
		query += " AND scope=" + add(p.Scope())
	} else if p.Role != "ADMIN" {
		if p.Warehouse == "" {
			return nil, domain.E("WAREHOUSE_SCOPE_REQUIRED", 403)
		}
		query += " AND warehouse_id=" + add(p.Warehouse)
	}
	if status != "" {
		query += " AND status=" + add(status)
	}
	if queue {
		query += " AND status IN ('PAID','CONFIRMED_COD','PROCESSING','PACKED','SHIPPED','DELIVERY_FAILED') AND operational_hold=false AND (ready_authorized OR status IN ('PROCESSING','PACKED','SHIPPED','DELIVERY_FAILED'))"
		if p.Role == "PACKING_STAFF" {
			query += " AND status IN ('PAID','CONFIRMED_COD','PROCESSING')"
		}
		if p.Role == "WAREHOUSE_STAFF" {
			query += " AND status IN ('PAID','CONFIRMED_COD')"
		}
		if cursor != "" {
			return nil, domain.E("QUEUE_CURSOR_UNSUPPORTED", 400)
		}
		query += " ORDER BY sla_due_at ASC NULLS LAST,created_at,id"
	} else {
		if cursor != "" {
			var boundary time.Time
			var id string
			e := s.decodeCursor(cursor, p.Scope(), status, &boundary, &id)
			if e != nil {
				return nil, e
			}
			query += " AND (created_at,id)<(" + add(boundary) + "," + add(id) + ")"
		}
		query += " ORDER BY created_at DESC,id DESC"
	}
	query += " LIMIT " + add(limit)
	rows, e := s.DB.Query(ctx, query, args...)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []domain.Order{}
	for rows.Next() {
		o, e := scanOrder(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, o)
	}
	return out, rows.Err()
}
func (s *Service) transition(ctx context.Context, tx pgx.Tx, o *domain.Order, to, source, reason string) error {
	if e := domain.RequireTransition(o.Status, to); e != nil {
		return e
	}
	from := o.Status
	o.Status = to
	o.Version++
	tag, e := tx.Exec(ctx, `UPDATE orders SET status=$2,payment_status=$3,stock_status=$4,version=$5,operational_hold=$6,ready_authorized=$7,ready_generation=$8,task_id=$9,shipment_id=$10,seal_code=$11,has_active_case=$12,completion_due_at=$13,sla_due_at=CASE WHEN status<>$2 THEN clock_timestamp()+interval '4 hours' ELSE sla_due_at END,updated_at=clock_timestamp() WHERE id=$1 AND version=$14`, o.ID, o.Status, o.Payment, o.Stock, o.Version, o.Hold, o.Ready, o.Generation, o.Task, o.Shipment, o.Seal, o.ActiveCase, o.CompletionDue, o.Version-1)
	if e != nil {
		return e
	}
	if tag.RowsAffected() != 1 {
		return domain.E("VERSION_CONFLICT", 409)
	}
	_, e = tx.Exec(ctx, "INSERT INTO order_history(id,order_id,version,from_status,to_status,source,reason) VALUES($1,$2,$3,$4,$5,$6,$7)", uuid.NewString(), o.ID, o.Version, from, to, source, reason)
	if e != nil {
		return e
	}
	if e = s.Emit(ctx, tx, o.ID, "ORDER", o.Version, 9, "audit", o.Status, o.Payment, o.Stock, o.Generation); e != nil {
		return e
	}
	return s.Audit(ctx, tx, o.ID, source, "TRANSITION", map[string]any{"from": from, "to": to, "reason_code": reason})
}
func (s *Service) touch(ctx context.Context, tx pgx.Tx, o *domain.Order, source string) error {
	o.Version++
	tag, e := tx.Exec(ctx, `UPDATE orders SET version=$2,payment_status=$3,stock_status=$4,operational_hold=$5,ready_authorized=$6,ready_generation=$7,has_active_case=$8,task_id=$9,shipment_id=$10,seal_code=$11,updated_at=clock_timestamp() WHERE id=$1 AND version=$12`, o.ID, o.Version, o.Payment, o.Stock, o.Hold, o.Ready, o.Generation, o.ActiveCase, o.Task, o.Shipment, o.Seal, o.Version-1)
	if e != nil {
		return e
	}
	if tag.RowsAffected() != 1 {
		return domain.E("VERSION_CONFLICT", 409)
	}
	_, e = tx.Exec(ctx, "INSERT INTO order_history(id,order_id,version,from_status,to_status,source) VALUES($1,$2,$3,$4,$4,$5)", uuid.NewString(), o.ID, o.Version, o.Status, source)
	if e != nil {
		return e
	}
	return s.Emit(ctx, tx, o.ID, "ORDER", o.Version, 9, "audit", o.Status, o.Payment, o.Stock, o.Generation)
}
func (s *Service) Cancel(ctx context.Context, p domain.Principal, id, key string, version int64, reason string, review bool) (map[string]any, error) {
	if len(key) < 16 || len(key) > 128 {
		return nil, domain.E("INVALID_IDEMPOTENCY_KEY", 400)
	}
	if len(reason) > 255 {
		return nil, domain.E("INVALID_REASON", 400)
	}
	hash := domain.Hash(map[string]any{"order_id": id, "version": version, "reason": reason, "review": review})
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback(ctx)
	o, e := s.lockOrder(ctx, tx, id)
	if e != nil {
		return nil, e
	}
	if !CanRead(p, o) || (p.Scope() != o.Scope && !p.Manager()) {
		return nil, domain.E("ORDER_NOT_FOUND", 404)
	}
	var old string
	var saved []byte
	e = tx.QueryRow(ctx, "SELECT request_hash,response FROM command_registry WHERE scope=$1 AND kind='CANCEL' AND key=$2", p.Scope(), key).Scan(&old, &saved)
	if e == nil {
		if old != hash {
			return nil, domain.E("IDEMPOTENCY_CONFLICT", 409)
		}
		var result map[string]any
		e = json.Unmarshal(saved, &result)
		return result, e
	}
	if !errors.Is(e, pgx.ErrNoRows) {
		return nil, e
	}
	if version != o.Version {
		return nil, domain.E("VERSION_CONFLICT", 409)
	}
	result := map[string]any{"order_id": o.ID, "compensation_status": "PENDING"}
	if review {
		if o.Status != "PAID" && !(o.Method == "COD" && o.Payment == "CONFIRMED" && o.Status == "CONFIRMED_COD") {
			return nil, domain.E("CANCELLATION_REVIEW_NOT_ALLOWED", 409)
		}
		requestID := uuid.NewString()
		_, e = tx.Exec(ctx, "INSERT INTO cancellation_requests(id,order_id,actor_id,request_key,reason) VALUES($1,$2,$3,$4,$5)", requestID, o.ID, p.ID, p.Scope()+":"+key, reason)
		if e != nil {
			return nil, e
		}
		result["request_id"] = requestID
		result["status"] = "PENDING_REVIEW"
	} else {
		if o.Method == "COD" && o.Payment != "UNPAID" {
			return nil, domain.E("CANCELLATION_REVIEW_REQUIRED", 409)
		}
		if o.Status != "PENDING_PAYMENT" && o.Status != "CONFIRMED_COD" {
			return nil, domain.E("CANCELLATION_NOT_ALLOWED", 409)
		}
		o.Hold = true
		o.Ready = false
		if e = s.touch(ctx, tx, &o, p.ID); e != nil {
			return nil, e
		}
		if e = s.insertJob(ctx, tx, o.ID, "CANCEL", "cancel:"+o.ID, map[string]any{"target": "CANCELLED_BY_USER", "actor": p.ID}); e != nil {
			return nil, e
		}
		result["status"] = "CANCELLATION_PENDING"
	}
	if e = s.Audit(ctx, tx, o.ID, p.ID, "CANCEL_REQUESTED", map[string]any{"review": review, "reason_hash": domain.Hash(reason)}); e != nil {
		return nil, e
	}
	b, e := json.Marshal(result)
	if e != nil {
		return nil, e
	}
	_, e = tx.Exec(ctx, "INSERT INTO command_registry(scope,kind,key,request_hash,order_id,response) VALUES($1,'CANCEL',$2,$3,$4,$5)", p.Scope(), key, hash, o.ID, b)
	if e != nil {
		return nil, e
	}
	if e = tx.Commit(ctx); e != nil {
		return nil, e
	}
	return result, nil
}
func (s *Service) Hold(ctx context.Context, p domain.Principal, id string, version int64) error {
	return s.SetHold(ctx, p, id, version, true, uuid.NewString())
}
func (s *Service) SetHold(ctx context.Context, p domain.Principal, id string, version int64, active bool, key string) error {
	if !p.Manager() {
		return domain.E("PERMISSION_DENIED", 403)
	}
	if len(key) < 16 || len(key) > 128 {
		return domain.E("INVALID_IDEMPOTENCY_KEY", 400)
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	o, e := s.lockOrder(ctx, tx, id)
	if e != nil {
		return e
	}
	if !CanRead(p, o) {
		return domain.E("ORDER_NOT_FOUND", 404)
	}
	hash := domain.Hash(map[string]any{"order_id": id, "version": version, "active": active})
	var old string
	e = tx.QueryRow(ctx, "SELECT request_hash FROM command_registry WHERE scope=$1 AND kind='HOLD' AND key=$2", p.Scope(), key).Scan(&old)
	if e == nil {
		if old != hash {
			return domain.E("IDEMPOTENCY_CONFLICT", 409)
		}
		return nil
	}
	if !errors.Is(e, pgx.ErrNoRows) {
		return e
	}
	if o.Version != version {
		return domain.E("VERSION_CONFLICT", 409)
	}
	if !active {
		var unresolved bool
		if e = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM jobs WHERE order_id=$1 AND status IN ('PENDING','RETRY','MANUAL_REVIEW') AND kind IN ('FINALIZE','CANCEL'))", o.ID).Scan(&unresolved); e != nil {
			return e
		}
		if unresolved || o.ActiveCase || o.Status == "PAYMENT_FINALIZING" {
			return domain.E("HOLD_RELEASE_NOT_SAFE", 409)
		}
	}
	o.Hold = active
	o.Ready = false
	if !active && (o.Status == "PAID" || o.Status == "CONFIRMED_COD") {
		o.Generation++
		if e = s.insertJob(ctx, tx, o.ID, "READY", "ready:"+o.ID+":"+strconv.FormatInt(o.Generation, 10), nil); e != nil {
			return e
		}
	}
	if e = s.touch(ctx, tx, &o, p.ID); e != nil {
		return e
	}
	if e = s.Audit(ctx, tx, o.ID, p.ID, "OPERATIONAL_HOLD", map[string]any{"active": active}); e != nil {
		return e
	}
	_, e = tx.Exec(ctx, "INSERT INTO command_registry(scope,kind,key,request_hash,order_id,response) VALUES($1,'HOLD',$2,$3,$4,'{}')", p.Scope(), key, hash, o.ID)
	if e != nil {
		return e
	}
	return tx.Commit(ctx)
}
