package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	d "github.com/omamx/fulfillment-service/internal/domain"
	"strings"
	"time"
)

type Store struct{ DB *pgxpool.Pool }
type query interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func marshal(v any) []byte { b, _ := json.Marshal(v); return b }
func mapped(e error) error {
	if errors.Is(e, pgx.ErrNoRows) {
		return d.Fail("NOT_FOUND", 404)
	}
	var p *pgconn.PgError
	if errors.As(e, &p) {
		switch p.Code {
		case "23505":
			return d.Fail("RESOURCE_CONFLICT", 409)
		case "40P01", "40001", "55P03":
			return d.Fail("RETRY_SAME_COMMAND", 503)
		}
	}
	return e
}
func (s *Store) orderTx(ctx context.Context, order string, fn func(pgx.Tx, time.Time) error) error {
	tx, e := s.DB.BeginTx(ctx, pgx.TxOptions{})
	if e != nil {
		return e
	}
	defer tx.Rollback(context.Background())
	if _, e = tx.Exec(ctx, `INSERT INTO fulfillment.order_guards(order_id) VALUES($1) ON CONFLICT DO NOTHING`, order); e != nil {
		return mapped(e)
	}
	var id string
	if e = tx.QueryRow(ctx, `SELECT order_id::text FROM fulfillment.order_guards WHERE order_id=$1 FOR UPDATE`, order).Scan(&id); e != nil {
		return mapped(e)
	}
	var now time.Time
	if e = tx.QueryRow(ctx, `SELECT clock_timestamp()`).Scan(&now); e != nil {
		return e
	}
	if e = fn(tx, now); e != nil {
		return mapped(e)
	}
	if e = tx.Commit(ctx); e != nil {
		return d.Fail("OUTCOME_UNKNOWN", 504)
	}
	return nil
}
func registry(ctx context.Context, q query, c d.Command) (json.RawMessage, bool, error) {
	var hash string
	var b []byte
	e := q.QueryRow(ctx, `SELECT request_hash,response FROM fulfillment.command_registry WHERE caller_id=$1 AND action=$2 AND order_id=$3 AND command_key=$4`, c.Caller, c.Action, c.OrderID, c.Key).Scan(&hash, &b)
	if errors.Is(e, pgx.ErrNoRows) {
		return nil, false, nil
	}
	if e != nil {
		return nil, false, e
	}
	if hash != c.Hash {
		return nil, false, d.Fail("IDEMPOTENCY_CONFLICT", 409)
	}
	return b, true, nil
}
func (s *Store) Replay(ctx context.Context, c d.Command) (json.RawMessage, bool, error) {
	return registry(ctx, s.DB, c)
}
func saveCommand(ctx context.Context, tx pgx.Tx, c d.Command, result, state, reason string, task *string, now time.Time, response any) error {
	decisionID := uuid.NewString()
	if value, ok := response.(d.Decision); ok && value.Command != nil {
		decisionID = value.Command.ID
	}
	_, e := tx.Exec(ctx, `INSERT INTO fulfillment.command_registry(caller_id,action,order_id,command_key,request_hash,generation,task_id,decision_id,decision_at,result,reason_code,resulting_state,response) VALUES($1,$2,$3,$4,$5,NULLIF($6,0),$7,$8,$9,$10,$11,$12,$13)`, c.Caller, c.Action, c.OrderID, c.Key, c.Hash, c.Generation, task, decisionID, now, result, reason, state, marshal(response))
	return e
}
func (s *Store) Decision(ctx context.Context, order string, g int64, key, action string) (out d.Decision, e error) {
	if !d.UUID(order) || g < 1 || (key == "") != (action == "") {
		return out, d.Fail("INVALID_QUERY", 400)
	}
	tx, err := s.DB.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return out, err
	}
	defer tx.Rollback(context.Background())
	out, e = decision(ctx, tx, order, g)
	if e != nil {
		return out, mapped(e)
	}
	if key != "" {
		caller := "service:order-service"
		if action == "AcceptTask" {
			caller = "service:fulfillment-service"
		} else if action != "CancelTask" {
			return out, d.Fail("INVALID_ACTION", 400)
		}
		var data []byte
		e = tx.QueryRow(ctx, `SELECT response FROM fulfillment.command_registry WHERE caller_id=$1 AND action=$2 AND order_id=$3 AND command_key=$4 AND generation=$5`, caller, action, order, key, g).Scan(&data)
		if e != nil {
			return out, mapped(e)
		}
		var original d.Decision
		if e = json.Unmarshal(data, &original); e != nil {
			return out, e
		}
		out.Command = original.Command
	}
	return out, tx.Commit(ctx)
}
func decision(ctx context.Context, q query, order string, g int64) (out d.Decision, e error) {
	var dec *string
	var at *time.Time
	out.OrderID = order
	out.Generation = g
	out.Contract = d.Contract
	e = q.QueryRow(ctx, `SELECT state,task_id::text,resource_version,COALESCE(items_digest,''),COALESCE(warehouse_id,''),decision_id::text,decision_at,clock_timestamp() FROM fulfillment.admissions WHERE order_id=$1 AND generation=$2`, order, g).Scan(&out.State, &out.TaskID, &out.Version, &out.Digest, &out.Warehouse, &dec, &at, &out.Observed)
	if e == nil && dec != nil {
		out.Reference = &d.DecisionReference{Service: "fulfillment-service", ID: *dec, OrderID: order, Generation: g, Version: out.Version}
	}
	return
}
func newTask(in d.AdmissionInput, now time.Time) d.Task {
	i := in.Inputs
	t := d.Task{ID: uuid.NewString(), OrderID: i.OrderID, State: "READY", Version: 1, Generation: i.Generation, Warehouse: i.Warehouse, Due: i.Due, Admitted: now, Digest: i.Digest, PolicyVersion: i.Policy.Version, Evidence: i.Policy.Evidence, Lines: i.Lines, Allocations: append([]d.Allocation{}, in.Stock.Slots...), Checklist: append([]d.Checklist{}, i.Policy.Checklist...), Blocks: []d.Block{}, Inputs: i, Stock: in.Stock, Cohort: "DOMAIN_TEST", Channel: i.Channel, DeadlineRevision: i.DeadlineRevision}
	if in.Integrated {
		t.Cohort = "INTEGRATED_V2"
	}
	for n := range t.Checklist {
		t.Checklist[n].Result = "PENDING"
		t.Checklist[n].By = nil
		t.Checklist[n].At = nil
		t.Checklist[n].Note = nil
	}
	for n := range t.Allocations {
		t.Allocations[n].Counted = 0
		t.Allocations[n].Version = in.Stock.Version
	}
	t.Package = d.Package{ID: uuid.NewString(), State: "DRAFT"}
	t.Project(now)
	return t
}
func audit(ctx context.Context, tx pgx.Tx, entity, actor, action, reason string, before, after any, now time.Time) error {
	var sequence int64
	var previous string
	e := tx.QueryRow(ctx, `SELECT sequence,current_hash FROM fulfillment.audit_entries WHERE entity_id=$1 ORDER BY sequence DESC LIMIT 1`, entity).Scan(&sequence, &previous)
	if e != nil && !errors.Is(e, pgx.ErrNoRows) {
		return e
	}
	if previous == "" {
		previous = fmt.Sprintf("%064d", 0)
	}
	id := uuid.NewString()
	hash := d.Hash(map[string]any{"prev_hash": previous, "actor": actor, "action": action, "sequence": sequence + 1, "time": now, "before": before, "after": after, "reason": reason})
	_, e = tx.Exec(ctx, `INSERT INTO fulfillment.audit_entries(audit_id,entity_id,sequence,actor_id,action,reason,redacted_before,redacted_after,prev_hash,current_hash,occurred_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, id, entity, sequence+1, actor, action, reason, marshal(before), marshal(after), previous, hash, now)
	if e != nil {
		return e
	}
	role := "STAFF"
	if strings.HasPrefix(actor, "service:") {
		role = "SERVICE"
	}
	event := d.NewEvent("vn.omama.audit.record.v1", entity, now, map[string]any{"actor_id": actor, "actor_role": role, "ip_address": "", "action": action, "entity_type": "FULFILLMENT_ORDER", "entity_id": entity, "before_state": before, "after_state": after, "change_reason": reason, "prev_entity_hash": previous, "current_entity_hash": hash})
	event.ID = id
	event.Subject = "entity:fulfillment:" + entity
	event.Trace = "00-" + hexID() + "-" + hexID()[:16] + "-01"
	_, e = tx.Exec(ctx, `INSERT INTO fulfillment.outbox(event_id,stream_id,stream_version,ordinal,topic,partition_key,envelope,occurred_at,next_run_at) VALUES($1,$2,$3,0,'audit.events.v1',$4,$5,$6,$6)`, id, "audit:"+entity, sequence+1, entity, marshal(event), now)
	return e
}
