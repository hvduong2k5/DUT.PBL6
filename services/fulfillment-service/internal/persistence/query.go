package persistence

import (
	"context"
	"encoding/base64"
	"encoding/json"
	d "github.com/omamx/fulfillment-service/internal/domain"
	"time"
)

type Filter struct {
	Warehouse, State, Assignee, Cursor string
	Limit                              int
}
type cursor struct {
	Scope    string     `json:"scope"`
	Due      *time.Time `json:"due"`
	Admitted time.Time  `json:"admitted"`
	ID       string     `json:"id"`
	AsOf     time.Time  `json:"as_of"`
}
type Page struct {
	Items  []d.Task  `json:"items"`
	Cursor *string   `json:"next_cursor"`
	AsOf   time.Time `json:"queue_as_of"`
}

func (s *Store) List(ctx context.Context, p d.Principal, f Filter) (out Page, e error) {
	if p.Service != "" || f.Limit < 1 || f.Limit > 100 {
		return out, d.Fail("INVALID_QUERY", 400)
	}
	allowed := []string{}
	for _, w := range p.Warehouses {
		if p.Can("fulfillment.task.read", w) && (f.Warehouse == "" || f.Warehouse == w) {
			allowed = append(allowed, w)
		}
	}
	if len(allowed) == 0 {
		return out, d.Fail("FORBIDDEN", 403)
	}
	if f.Assignee != "" && !d.UUID(f.Assignee) {
		return out, d.Fail("INVALID_ASSIGNEE", 400)
	}
	scope := d.Hash(map[string]any{"actor": p.ID, "warehouses": allowed, "state": f.State, "assignee": f.Assignee})
	c := cursor{AsOf: time.Now().UTC()}
	if f.Cursor != "" {
		b, err := base64.RawURLEncoding.DecodeString(f.Cursor)
		if err != nil || json.Unmarshal(b, &c) != nil || c.Scope != scope || !d.UUID(c.ID) {
			return out, d.Fail("INVALID_CURSOR", 400)
		}
	}
	rows, e := s.DB.Query(ctx, `SELECT task_id::text FROM fulfillment.tasks WHERE warehouse_id=ANY($1) AND (($2='' AND workflow_state IN ('READY','ASSIGNED','IN_PROGRESS')) OR workflow_state=$2) AND ($3='' OR assignee_id::text=$3) AND admitted_at<=$4 AND ($5='' OR (COALESCE(packing_due_at,'infinity'::timestamptz),admitted_at,task_id)>(COALESCE($6::timestamptz,'infinity'::timestamptz),$7::timestamptz,NULLIF($5,'')::uuid)) ORDER BY packing_due_at NULLS LAST,admitted_at,task_id LIMIT $8`, allowed, f.State, f.Assignee, c.AsOf, c.ID, c.Due, c.Admitted, f.Limit+1)
	if e != nil {
		return out, e
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if e = rows.Scan(&id); e != nil {
			rows.Close()
			return
		}
		ids = append(ids, id)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return
	}
	more := len(ids) > f.Limit
	if more {
		ids = ids[:f.Limit]
	}
	out.Items = []d.Task{}
	out.AsOf = c.AsOf
	for _, id := range ids {
		t, err := s.Task(ctx, id)
		if err != nil {
			return out, err
		}
		out.Items = append(out.Items, t)
	}
	if more && len(out.Items) > 0 {
		last := out.Items[len(out.Items)-1]
		b := marshal(cursor{Scope: scope, Due: last.Due, Admitted: last.Admitted, ID: last.ID, AsOf: c.AsOf})
		v := base64.RawURLEncoding.EncodeToString(b)
		out.Cursor = &v
	}
	return
}

type Revision struct {
	Task     d.Task
	Kind     string
	Occurred time.Time
}

func (s *Store) Revision(ctx context.Context, id string, version int64) (out Revision, e error) {
	if !d.UUID(id) || version < 1 {
		return out, d.Fail("INVALID_REVISION", 400)
	}
	var b []byte
	e = s.DB.QueryRow(ctx, `SELECT kind,occurred_at,snapshot FROM fulfillment.task_revisions WHERE task_id=$1 AND resource_version=$2`, id, version).Scan(&out.Kind, &out.Occurred, &b)
	if e != nil {
		return out, mapped(e)
	}
	e = json.Unmarshal(b, &out.Task)
	return
}
func (s *Store) PackageRevision(ctx context.Context, id string, version int64) (Revision, error) {
	if !d.UUID(id) {
		return Revision{}, d.Fail("INVALID_PACKAGE", 400)
	}
	var task string
	e := s.DB.QueryRow(ctx, `SELECT task_id::text FROM fulfillment.packages WHERE package_id=$1`, id).Scan(&task)
	if e != nil {
		return Revision{}, mapped(e)
	}
	r, e := s.Revision(ctx, task, version)
	if e == nil && (r.Kind != "PACKING_SEALED" || r.Task.Package.ID != id) {
		e = d.Fail("REVISION_KIND_MISMATCH", 412)
	}
	return r, e
}
func (s *Store) History(ctx context.Context, p d.Principal, id string, after int64, limit int) ([]map[string]any, error) {
	t, e := s.Task(ctx, id)
	if e != nil {
		return nil, e
	}
	if !p.Can("fulfillment.task.read_history", t.Warehouse) {
		return nil, d.Fail("FORBIDDEN", 403)
	}
	if limit < 1 || limit > 100 || after < 0 {
		return nil, d.Fail("INVALID_QUERY", 400)
	}
	rows, e := s.DB.Query(ctx, `SELECT history_id::text,task_version,action,actor_id,reason,occurred_at FROM fulfillment.task_history WHERE task_id=$1 AND task_version>$2 ORDER BY task_version LIMIT $3`, id, after, limit)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var hid, action, actor string
		var version int64
		var reason *string
		var at time.Time
		if e = rows.Scan(&hid, &version, &action, &actor, &reason, &at); e != nil {
			return nil, e
		}
		out = append(out, map[string]any{"history_id": hid, "task_version": version, "action": action, "actor_id": actor, "reason": reason, "occurred_at": at})
	}
	return out, rows.Err()
}
func (s *Store) Command(ctx context.Context, p d.Principal, order, action, key string) (map[string]any, error) {
	if !d.UUID(order) || !d.Key(key) {
		return nil, d.Fail("INVALID_QUERY", 400)
	}
	var warehouse string
	e := s.DB.QueryRow(ctx, `SELECT warehouse_id FROM fulfillment.tasks WHERE order_id=$1`, order).Scan(&warehouse)
	if e != nil {
		return nil, mapped(e)
	}
	if !p.Can("fulfillment.task.read", warehouse) {
		return nil, d.Fail("FORBIDDEN", 403)
	}
	var result, state, id string
	var at time.Time
	var task *string
	var response []byte
	e = s.DB.QueryRow(ctx, `SELECT result,resulting_state,decision_id::text,decision_at,task_id::text,response FROM fulfillment.command_registry WHERE caller_id=$1 AND action=$2 AND order_id=$3 AND command_key=$4`, p.Caller(), action, order, key).Scan(&result, &state, &id, &at, &task, &response)
	if e != nil {
		return nil, mapped(e)
	}
	out := map[string]any{"action": action, "order_id": order, "command_key": key, "result": result, "resulting_state": state, "decision_id": id, "decision_at": at}
	if task != nil {
		out["task_id"] = *task
	}
	var t struct {
		Version int64 `json:"version"`
	}
	if json.Unmarshal(response, &t) == nil && t.Version > 0 {
		out["version"] = t.Version
	}
	return out, nil
}
