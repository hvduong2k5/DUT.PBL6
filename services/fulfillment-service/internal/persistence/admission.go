package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	d "github.com/omamx/fulfillment-service/internal/domain"
	"time"
)

func (s *Store) Accept(ctx context.Context, in d.AdmissionInput) (out d.Decision, e error) {
	c := in.Command
	if e = c.Validate(); e != nil {
		return
	}
	if c.Caller != "service:fulfillment-service" || c.Action != "AcceptTask" {
		return out, d.Fail("FORBIDDEN", 403)
	}
	e = s.orderTx(ctx, c.OrderID, func(tx pgx.Tx, now time.Time) error {
		if b, found, e := registry(ctx, tx, c); e != nil {
			return e
		} else if found {
			if err := json.Unmarshal(b, &out); err != nil {
				return err
			}
			original := out.Command
			current, err := decision(ctx, tx, c.OrderID, c.Generation)
			if err != nil {
				return err
			}
			out = current
			out.Command = original
			return nil
		}
		var high int64
		var accepted *string
		if e := tx.QueryRow(ctx, `SELECT highest_generation,accepted_task_id::text FROM fulfillment.order_guards WHERE order_id=$1`, c.OrderID).Scan(&high, &accepted); e != nil {
			return e
		}
		current, err := decision(ctx, tx, c.OrderID, c.Generation)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		reason := ""
		result := "APPLIED"
		if accepted != nil {
			if err != nil || current.State != "ACCEPTED" || current.Digest != in.Inputs.Digest || current.Warehouse != in.Inputs.Warehouse {
				return d.Fail("ORDER_ALREADY_ACCEPTED", 412)
			}
			out = current
			result = "ALREADY_APPLIED"
		} else {
			if errors.Is(err, pgx.ErrNoRows) {
				if _, err = tx.Exec(ctx, `INSERT INTO fulfillment.admissions(order_id,generation,state,resource_version) VALUES($1,$2,'OPEN',1)`, c.OrderID, c.Generation); err != nil {
					return err
				}
				current, err = decision(ctx, tx, c.OrderID, c.Generation)
				if err != nil {
					return err
				}
			}
			if current.State == "CANCELLED" {
				reason = "GENERATION_CANCELLED"
			} else if c.Generation < high {
				reason = "STALE_GENERATION"
			} else {
				var open int
				if e := tx.QueryRow(ctx, `SELECT count(*) FROM fulfillment.admissions WHERE order_id=$1 AND generation<$2 AND state<>'CANCELLED'`, c.OrderID, c.Generation).Scan(&open); e != nil {
					return e
				}
				if open > 0 {
					reason = "OLDER_GENERATION_OPEN"
				}
			}
			p := in.Permit
			if reason == "" && (!p.Allowed || p.ID == "" || p.OrderVersion < 1 || p.OrderID != c.OrderID || p.Generation != c.Generation || p.Key != c.Key || p.Audience != "fulfillment-service" || p.Digest != in.Inputs.Digest || p.Warehouse != in.Inputs.Warehouse || p.Policy != in.Inputs.Policy.Version || !now.Add(2*time.Second).Before(p.Expires)) {
				reason = "PERMIT_REJECTED_OR_EXPIRED"
			}
			if reason != "" {
				out = current
				result = "REJECTED"
			} else {
				if e := in.Inputs.Validate(in.Stock, now, in.Integrated); e != nil {
					return e
				}
				if in.Inputs.OrderID != c.OrderID || in.Inputs.Generation != c.Generation {
					return d.Fail("INPUT_PROOF_MISMATCH", 412)
				}
				t := newTask(in, now)
				if e := insertTask(ctx, tx, t); e != nil {
					return e
				}
				if _, e := tx.Exec(ctx, `UPDATE fulfillment.admissions SET state='ACCEPTED',task_id=$3,resource_version=resource_version+1,items_digest=$4,warehouse_id=$5,permit_id=$6,decision_id=$7,decision_at=$8 WHERE order_id=$1 AND generation=$2`, c.OrderID, c.Generation, t.ID, t.Digest, t.Warehouse, p.ID, uuid.NewString(), now); e != nil {
					return e
				}
				if _, e := tx.Exec(ctx, `UPDATE fulfillment.order_guards SET highest_generation=GREATEST(highest_generation,$2),accepted_task_id=$3,updated_at=$4 WHERE order_id=$1`, c.OrderID, c.Generation, t.ID, now); e != nil {
					return e
				}
				if e := checkpoint(ctx, tx, t, "PACKING_ACCEPTED", c.Caller, "AcceptTask", "", now); e != nil {
					return e
				}
				out, err = decision(ctx, tx, c.OrderID, c.Generation)
				if err != nil {
					return err
				}
			}
		}
		out.Command = &d.Evidence{Key: c.Key, Action: c.Action, Hash: c.Hash, Result: result, Reason: reason, ID: uuid.NewString(), At: now, State: out.State}
		return saveCommand(ctx, tx, c, result, out.State, reason, out.TaskID, now, out)
	})
	return
}
func (s *Store) Cancel(ctx context.Context, c d.Command, reason string) (out d.Decision, e error) {
	if e = c.Validate(); e != nil {
		return
	}
	if c.Caller != "service:order-service" || c.Action != "CancelTask" {
		return out, d.Fail("FORBIDDEN", 403)
	}
	e = s.orderTx(ctx, c.OrderID, func(tx pgx.Tx, now time.Time) error {
		if b, found, e := registry(ctx, tx, c); e != nil {
			return e
		} else if found {
			if err := json.Unmarshal(b, &out); err != nil {
				return err
			}
			original := out.Command
			current, err := decision(ctx, tx, c.OrderID, c.Generation)
			if err != nil {
				return err
			}
			out = current
			out.Command = original
			return nil
		}
		var accepted *string
		if e := tx.QueryRow(ctx, `SELECT accepted_task_id::text FROM fulfillment.order_guards WHERE order_id=$1`, c.OrderID).Scan(&accepted); e != nil {
			return e
		}
		current, err := decision(ctx, tx, c.OrderID, c.Generation)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		if accepted != nil && (err != nil || current.State != "ACCEPTED") {
			return d.Fail("ORDER_ALREADY_ACCEPTED", 412)
		}
		result, code := "APPLIED", ""
		if err == nil && current.State == "ACCEPTED" {
			result = "REJECTED"
			code = "TASK_ALREADY_ACCEPTED"
		} else if err == nil && current.State == "CANCELLED" {
			result = "ALREADY_APPLIED"
		} else {
			_, err = tx.Exec(ctx, `INSERT INTO fulfillment.admissions(order_id,generation,state,resource_version,decision_id,decision_at) VALUES($1,$2,'CANCELLED',1,$3,$4) ON CONFLICT(order_id,generation) DO UPDATE SET state='CANCELLED',resource_version=fulfillment.admissions.resource_version+1,decision_id=EXCLUDED.decision_id,decision_at=EXCLUDED.decision_at`, c.OrderID, c.Generation, uuid.NewString(), now)
			if err != nil {
				return err
			}
			if _, err = tx.Exec(ctx, `UPDATE fulfillment.order_guards SET highest_generation=GREATEST(highest_generation,$2),updated_at=$3 WHERE order_id=$1`, c.OrderID, c.Generation, now); err != nil {
				return err
			}
		}
		out, err = decision(ctx, tx, c.OrderID, c.Generation)
		if err != nil {
			return err
		}
		out.Command = &d.Evidence{Key: c.Key, Action: c.Action, Hash: c.Hash, Result: result, Reason: code, ID: uuid.NewString(), At: now, State: out.State}
		if err = audit(ctx, tx, c.OrderID, c.Caller, "CancelTask", reason, map[string]any{}, map[string]any{"state": out.State, "result": result}, now); err != nil {
			return err
		}
		return saveCommand(ctx, tx, c, result, out.State, code, out.TaskID, now, out)
	})
	return
}
