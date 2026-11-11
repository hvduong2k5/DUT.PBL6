package persistence

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	d "github.com/omamx/fulfillment-service/internal/domain"
	"time"
)

func insertTask(ctx context.Context, tx pgx.Tx, t d.Task) error {
	_, e := tx.Exec(ctx, `INSERT INTO fulfillment.tasks(task_id,order_id,ready_generation,workflow_state,version,items_digest,warehouse_id,channel,cohort,input_snapshot_id,input_snapshot_version,input_snapshot,policy_version,policy_snapshot,reservation_id,allocation_version,allocation_snapshot,evidence_mode,packing_due_at,deadline_revision,admitted_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$21)`, t.ID, t.OrderID, t.Generation, t.State, t.Version, t.Digest, t.Warehouse, t.Channel, t.Cohort, t.Inputs.SnapshotID, t.Inputs.Version, marshal(t.Inputs), t.PolicyVersion, marshal(t.Inputs.Policy), t.Inputs.ReservationID, t.Stock.Version, marshal(t.Stock), t.Evidence, t.Due, t.DeadlineRevision, t.Admitted)
	if e != nil {
		return e
	}
	ids := map[string]string{}
	for _, l := range t.Lines {
		lid := uuid.NewString()
		ids[l.ID] = lid
		if _, e = tx.Exec(ctx, `INSERT INTO fulfillment.task_lines(line_id,task_id,order_line_id,sku_code,product_name,packaging_text,required_quantity,fragile) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, lid, t.ID, l.ID, l.SKU, l.Name, l.Packaging, l.Quantity, l.Fragile); e != nil {
			return e
		}
	}
	for _, a := range t.Allocations {
		if _, e = tx.Exec(ctx, `INSERT INTO fulfillment.pick_allocations(allocation_id,task_id,line_id,batch_id,batch_code,allocated_quantity,allocation_version,batch_valid_until,eligibility_revision) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, a.ID, t.ID, ids[a.LineID], a.BatchID, a.BatchCode, a.Allocated, t.Stock.Version, a.Until, a.EligibilityVersion); e != nil {
			return e
		}
	}
	for _, c := range t.Checklist {
		if _, e = tx.Exec(ctx, `INSERT INTO fulfillment.checklist_items(task_id,item_code,definition,policy_version,required) VALUES($1,$2,$3,$4,$5)`, t.ID, c.Code, c.Definition, t.PolicyVersion, c.Required); e != nil {
			return e
		}
	}
	_, e = tx.Exec(ctx, `INSERT INTO fulfillment.packages(package_id,task_id,state,created_at) VALUES($1,$2,'DRAFT',$3)`, t.Package.ID, t.ID, t.Admitted)
	return e
}
func (s *Store) Task(ctx context.Context, id string) (d.Task, error) {
	if !d.UUID(id) {
		return d.Task{}, d.Fail("INVALID_TASK_ID", 400)
	}
	tx, e := s.DB.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if e != nil {
		return d.Task{}, e
	}
	defer tx.Rollback(context.Background())
	t, e := loadTask(ctx, tx, id)
	if e != nil {
		return t, mapped(e)
	}
	e = tx.Commit(ctx)
	return t, e
}
func loadTask(ctx context.Context, q query, id string) (t d.Task, e error) {
	var inputs, stock []byte
	var resume *string
	e = q.QueryRow(ctx, `SELECT task_id::text,order_id::text,ready_generation,workflow_state,version,items_digest,warehouse_id,channel,cohort,assignee_id::text,input_snapshot,allocation_snapshot,packing_due_at,deadline_revision,admitted_at,started_at,completed_at,resume_state,clock_timestamp() FROM fulfillment.tasks WHERE task_id=$1`, id).Scan(&t.ID, &t.OrderID, &t.Generation, &t.State, &t.Version, &t.Digest, &t.Warehouse, &t.Channel, &t.Cohort, &t.Assignee, &inputs, &stock, &t.Due, &t.DeadlineRevision, &t.Admitted, &t.Started, &t.Completed, &resume, &t.AsOf)
	if e != nil {
		return
	}
	if resume != nil {
		t.Resume = *resume
	}
	if e = json.Unmarshal(inputs, &t.Inputs); e != nil {
		return
	}
	if e = json.Unmarshal(stock, &t.Stock); e != nil {
		return
	}
	t.Lines = t.Inputs.Lines
	t.PolicyVersion = t.Inputs.Policy.Version
	t.Evidence = t.Inputs.Policy.Evidence
	rows, err := q.Query(ctx, `SELECT a.allocation_id::text,l.order_line_id::text,l.sku_code,a.batch_id::text,a.batch_code,a.allocated_quantity,a.counted_quantity,a.allocation_version,a.batch_valid_until,a.eligibility_revision,a.last_validated_at FROM fulfillment.pick_allocations a JOIN fulfillment.task_lines l ON l.line_id=a.line_id WHERE a.task_id=$1 ORDER BY a.allocation_id`, id)
	if err != nil {
		return t, err
	}
	t.Allocations = []d.Allocation{}
	for rows.Next() {
		var a d.Allocation
		if e = rows.Scan(&a.ID, &a.LineID, &a.SKU, &a.BatchID, &a.BatchCode, &a.Allocated, &a.Counted, &a.Version, &a.Until, &a.EligibilityVersion, &a.ValidatedAt); e != nil {
			rows.Close()
			return
		}
		a.Eligible = true
		t.Allocations = append(t.Allocations, a)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return
	}
	rows, err = q.Query(ctx, `SELECT item_code,definition,required,result,confirmed_by::text,confirmed_at,note FROM fulfillment.checklist_items WHERE task_id=$1 ORDER BY item_code`, id)
	if err != nil {
		return t, err
	}
	t.Checklist = []d.Checklist{}
	for rows.Next() {
		var c d.Checklist
		if e = rows.Scan(&c.Code, &c.Definition, &c.Required, &c.Result, &c.By, &c.At, &c.Note); e != nil {
			rows.Close()
			return
		}
		t.Checklist = append(t.Checklist, c)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return
	}
	rows, err = q.Query(ctx, `SELECT block_id::text,source,reason_code,status FROM fulfillment.task_blocks WHERE task_id=$1 AND status='ACTIVE' ORDER BY block_id`, id)
	if err != nil {
		return t, err
	}
	t.Blocks = []d.Block{}
	for rows.Next() {
		var b d.Block
		if e = rows.Scan(&b.ID, &b.Source, &b.Reason, &b.Status); e != nil {
			rows.Close()
			return
		}
		t.Blocks = append(t.Blocks, b)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return
	}
	var weight, length, width, height *int32
	var fragile *bool
	e = q.QueryRow(ctx, `SELECT package_id::text,state,seal_code,weight_grams,length_cm,width_cm,height_cm,fragile,primary_evidence_id::text FROM fulfillment.packages WHERE task_id=$1`, id).Scan(&t.Package.ID, &t.Package.State, &t.Package.Seal, &weight, &length, &width, &height, &fragile, &t.Package.EvidenceID)
	if e != nil {
		return
	}
	if weight != nil && length != nil && width != nil && height != nil && fragile != nil {
		t.Package.Parcel = &d.Parcel{Weight: *weight, Length: *length, Width: *width, Height: *height, Fragile: *fragile}
	}
	t.Project(t.AsOf)
	return
}
func (s *Store) Mutate(ctx context.Context, p d.Principal, id, key, hash string, m d.Mutation, v *d.Validation) (out d.Task, e error) {
	if !d.UUID(id) || !d.Key(key) {
		return out, d.Fail("INVALID_COMMAND", 400)
	}
	initial, e := s.Task(ctx, id)
	if e != nil {
		return
	}
	c := d.Command{Caller: p.Caller(), Action: m.Action, OrderID: initial.OrderID, Key: key, Hash: hash, Generation: initial.Generation}
	e = s.orderTx(ctx, c.OrderID, func(tx pgx.Tx, now time.Time) error {
		t, err := loadTask(ctx, tx, id)
		if err != nil {
			return err
		}
		permission := t.Permission(m.Action)
		if (m.Action == "AssignTask" && m.AssigneeID != "" && m.AssigneeID != p.ID) || (m.Action == "UnassignTask" && (t.Assignee == nil || *t.Assignee != p.ID)) {
			permission = "FULFILLMENT_TASK_ASSIGN_OTHER"
		}
		if !p.Can(permission, t.Warehouse) {
			return d.Fail("FORBIDDEN", 403)
		}
		if b, found, err := registry(ctx, tx, c); err != nil {
			return err
		} else if found {
			return json.Unmarshal(b, &out)
		}
		before := map[string]any{"state": t.State, "version": t.Version, "assignee": t.Assignee}
		if m.Action == "SealTask" {
			if err = validateStock(t, v, now); err != nil {
				return err
			}
			if err = validateMedia(ctx, tx, t, m, now); err != nil {
				return err
			}
		}
		oldBlocks := append([]d.Block{}, t.Blocks...)
		if err = t.Apply(p, m, now); err != nil {
			return err
		}
		if err = saveTask(ctx, tx, t, p, m, oldBlocks, now); err != nil {
			return err
		}
		kind := "TASK_UPDATED"
		if m.Action == "SealTask" {
			kind = "PACKING_SEALED"
		}
		if err = checkpoint(ctx, tx, t, kind, p.Caller(), m.Action, m.Reason, now); err != nil {
			return err
		}
		if err = audit(ctx, tx, t.OrderID, p.Caller(), m.Action, m.Reason, before, map[string]any{"state": t.State, "version": t.Version, "assignee": t.Assignee}, now); err != nil {
			return err
		}
		out = t
		return saveCommand(ctx, tx, c, "APPLIED", t.State, "", &t.ID, now, t)
	})
	return
}
func validateStock(t d.Task, v *d.Validation, now time.Time) error {
	if v == nil || !v.Allowed || v.ID == "" || v.Contract != d.Contract || v.OrderID != t.OrderID || v.TaskID != t.ID || v.ReservationID != t.Inputs.ReservationID || v.Version != t.Stock.Version || v.Generation != t.Generation || v.Digest != t.Digest || v.Warehouse != t.Warehouse || v.Observed.After(now.Add(2*time.Second)) || now.Sub(v.Observed) > 5*time.Second || !now.Before(v.Until) {
		return d.Fail("ALLOCATION_AUTHORITY_REQUIRED", 503)
	}
	slots := map[string]d.Allocation{}
	for _, a := range v.Slots {
		if _, dup := slots[a.ID]; dup {
			return d.Fail("INVALID_VALIDATION", 412)
		}
		slots[a.ID] = a
	}
	if len(slots) != len(t.Allocations) {
		return d.Fail("VALIDATION_COVERAGE", 412)
	}
	for _, a := range t.Allocations {
		b, ok := slots[a.ID]
		if !ok || b.LineID != a.LineID || b.BatchID != a.BatchID || b.Allocated != a.Allocated || !b.Eligible || b.EligibilityVersion < a.EligibilityVersion || !now.Before(b.Until) {
			return d.Fail("BATCH_INVALID", 409)
		}
	}
	return nil
}
func saveTask(ctx context.Context, tx pgx.Tx, t d.Task, p d.Principal, m d.Mutation, oldBlocks []d.Block, now time.Time) error {
	var resume *string
	if t.Resume != "" {
		resume = &t.Resume
	}
	_, e := tx.Exec(ctx, `UPDATE fulfillment.tasks SET workflow_state=$2,version=$3,assignee_id=$4,resume_state=$5,started_at=$6,completed_at=$7,updated_at=$8 WHERE task_id=$1`, t.ID, t.State, t.Version, t.Assignee, resume, t.Started, t.Completed, now)
	if e != nil {
		return e
	}
	if m.Action == "SetPickedQuantity" {
		_, e = tx.Exec(ctx, `UPDATE fulfillment.pick_allocations SET counted_quantity=$3,counted_by=$4,counted_at=$5 WHERE task_id=$1 AND allocation_id=$2`, t.ID, m.Child, m.Counted, p.ID, now)
	}
	if m.Action == "ConfirmChecklist" {
		_, e = tx.Exec(ctx, `UPDATE fulfillment.checklist_items SET result=$3,note=$4,confirmed_by=$5,confirmed_at=$6 WHERE task_id=$1 AND item_code=$2`, t.ID, m.Child, m.Result, m.Note, p.ID, now)
	}
	if e != nil {
		return e
	}
	if m.Action == "BlockTask" {
		b := t.Blocks[len(t.Blocks)-1]
		_, e = tx.Exec(ctx, `INSERT INTO fulfillment.task_blocks(block_id,task_id,source,source_reference,reason_code,status,created_by,created_at) VALUES($1::uuid,$2::uuid,'OPERATOR',$1::uuid::text,$3,'ACTIVE',$4,$5)`, b.ID, t.ID, b.Reason, p.ID, now)
	}
	if m.Action == "ResolveTaskBlock" {
		_, e = tx.Exec(ctx, `UPDATE fulfillment.task_blocks SET status='RESOLVED',resolved_by=$3,resolved_at=$4,resolution_reason=$5 WHERE task_id=$1 AND block_id=$2`, t.ID, m.Child, p.ID, now, m.Reason)
	}
	if e != nil {
		return e
	}
	if m.Action == "SealTask" {
		x := t.Package.Parcel
		_, e = tx.Exec(ctx, `UPDATE fulfillment.packages SET state='SEALED',seal_code=$2,weight_grams=$3,length_cm=$4,width_cm=$5,height_cm=$6,fragile=$7,primary_evidence_id=$8,sealed_task_version=$9,sealed_at=$10 WHERE package_id=$1`, t.Package.ID, t.Package.Seal, x.Weight, x.Length, x.Width, x.Height, x.Fragile, t.Package.EvidenceID, t.Version, now)
		if e == nil && t.Package.EvidenceID != nil {
			_, e = tx.Exec(ctx, `UPDATE fulfillment.media_files SET attached_at=$2,legal_hold=true WHERE file_id=$1`, *t.Package.EvidenceID, now)
		}
	}
	return e
}
func checkpoint(ctx context.Context, tx pgx.Tx, t d.Task, kind, actor, action, reason string, now time.Time) error {
	_, e := tx.Exec(ctx, `INSERT INTO fulfillment.task_revisions(task_id,resource_version,kind,order_id,ready_generation,items_digest,warehouse_id,package_id,seal_code,occurred_at,snapshot) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, t.ID, t.Version, kind, t.OrderID, t.Generation, t.Digest, t.Warehouse, t.Package.ID, t.Package.Seal, now, marshal(t))
	if e != nil {
		return e
	}
	_, e = tx.Exec(ctx, `INSERT INTO fulfillment.task_history(history_id,task_id,task_version,action,actor_id,reason,before_state,after_state,occurred_at) VALUES($1,$2,$3,$4,$5,$6,'{}',$7,$8)`, uuid.NewString(), t.ID, t.Version, action, actor, reason, marshal(map[string]any{"state": t.State, "version": t.Version}), now)
	if e != nil {
		return e
	}
	if kind == "PACKING_ACCEPTED" {
		if e = audit(ctx, tx, t.OrderID, actor, action, reason, map[string]any{}, map[string]any{"state": t.State, "task_id": t.ID}, now); e != nil {
			return e
		}
		return outbox(ctx, tx, t, "fulfillment.task.events.v1", "vn.omama.fulfillment.task.accepted.v1", map[string]any{"order_id": t.OrderID, "task_id": t.ID, "resource_version": t.Version, "ready_generation": t.Generation, "kind": kind, "items_digest": t.Digest, "warehouse_id": t.Warehouse, "occurred_at": now}, now)
	}
	if kind == "PACKING_SEALED" && t.Cohort == "INTEGRATED_V2" {
		var key, hash string
		if e = tx.QueryRow(ctx, `SELECT final_object_key,verified_sha256 FROM fulfillment.media_files WHERE file_id=$1`, t.Package.EvidenceID).Scan(&key, &hash); e != nil {
			return e
		}
		batches := []string{}
		seen := map[string]bool{}
		for _, a := range t.Allocations {
			if !seen[a.BatchCode] {
				batches = append(batches, a.BatchCode)
				seen[a.BatchCode] = true
			}
		}
		return outbox(ctx, tx, t, "fulfillment.events.v2", "vn.omama.fulfillment.packing.completed.v2", map[string]any{"order_id": t.OrderID, "task_id": t.ID, "resource_version": t.Version, "ready_generation": t.Generation, "package_id": t.Package.ID, "packer_staff_id": *t.Assignee, "seal_code": *t.Package.Seal, "parcel": t.Package.Parcel, "s3_video_object_key": key, "video_checksum_sha256": hash, "scanned_batch_codes": batches, "packed_at": now}, now)
	}
	return nil
}
func outbox(ctx context.Context, tx pgx.Tx, t d.Task, topic, kind string, data any, now time.Time) error {
	event := d.NewEvent(kind, t.OrderID, now, data)
	event.ID = uuid.NewString()
	event.Trace = "00-" + hexID() + "-" + hexID()[:16] + "-01"
	_, e := tx.Exec(ctx, `INSERT INTO fulfillment.outbox(event_id,stream_id,stream_version,ordinal,topic,partition_key,envelope,occurred_at,next_run_at) VALUES($1,$2,$3,0,$4,$5,$6,$7,$7)`, event.ID, t.OrderID, t.Version, topic, t.OrderID, marshal(event), now)
	return e
}
func hexID() string { v := uuid.New(); return fmtHex(v[:]) }
func fmtHex(b []byte) string {
	const chars = "0123456789abcdef"
	out := make([]byte, len(b)*2)
	for n, x := range b {
		out[n*2] = chars[x>>4]
		out[n*2+1] = chars[x&15]
	}
	return string(out)
}
