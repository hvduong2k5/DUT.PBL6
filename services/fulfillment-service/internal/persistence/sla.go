package persistence

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"time"
)

// Local audit intent only. Notification is an asynchronous future consumer, not
// a synchronous dependency of packing or a reason to auto-cancel an overdue task.
func (s *Store) SLA(ctx context.Context) error {
	rows, e := s.DB.Query(ctx, `SELECT task_id::text,order_id::text FROM fulfillment.tasks t WHERE workflow_state<>'COMPLETED' AND packing_due_at<clock_timestamp() AND NOT EXISTS(SELECT 1 FROM fulfillment.jobs j WHERE j.dedupe_key='sla:'||t.task_id::text||':'||t.deadline_revision::text||':breached') ORDER BY packing_due_at LIMIT 25`)
	if e != nil {
		return e
	}
	pairs := [][2]string{}
	for rows.Next() {
		var p [2]string
		if e = rows.Scan(&p[0], &p[1]); e != nil {
			rows.Close()
			return e
		}
		pairs = append(pairs, p)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	for _, p := range pairs {
		e = s.orderTx(ctx, p[1], func(tx pgx.Tx, now time.Time) error {
			t, e := loadTask(ctx, tx, p[0])
			if e != nil {
				return e
			}
			if t.State == "COMPLETED" || t.Due == nil || !now.After(*t.Due) {
				return nil
			}
			key := fmt.Sprintf("sla:%s:%d:breached", t.ID, t.DeadlineRevision)
			result, e := tx.Exec(ctx, `INSERT INTO fulfillment.jobs(job_id,kind,dedupe_key,state,stage,task_id,input,next_run_at,created_at,updated_at) VALUES($1,'SLA_ALERT',$2,'SUCCEEDED','RECORDED',$3,'{}',$4,$4,$4) ON CONFLICT(dedupe_key) DO NOTHING`, uuid.NewString(), key, t.ID, now)
			if e != nil || result.RowsAffected() == 0 {
				return e
			}
			return audit(ctx, tx, t.OrderID, "service:fulfillment-service", "SLA_BREACHED", "deadline passed", map[string]any{}, map[string]any{"task_id": t.ID, "deadline_revision": t.DeadlineRevision, "workflow": t.State}, now)
		})
		if e != nil {
			return e
		}
	}
	return nil
}
