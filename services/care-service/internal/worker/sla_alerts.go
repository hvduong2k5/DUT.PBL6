package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/care-service/internal/idgen"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/rs/zerolog/log"
)

const (
	slaNearBreachEvent = "vn.omama.care.ticket.sla-near-breach.v1"
	slaBreachedEvent   = "vn.omama.care.ticket.sla-breached.v1"
)

var (
	slaAlertsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "care_sla_alerts_total", Help: "Care SLA alerts created by alert type.",
	}, []string{"alert_type"})
	slaAlertBatchTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "care_sla_alert_batches_total", Help: "Care SLA alert batches by outcome.",
	}, []string{"outcome"})
)

type slaMilestone struct {
	ID              uuid.UUID
	TicketID        uuid.UUID
	MilestoneType   string
	DueAt           time.Time
	TicketCode      string
	Priority        string
	AssignedStaffID uuid.UUID
}

type SLAAlerts struct {
	pool         *pgxpool.Pool
	topic        string
	pollInterval time.Duration
	nearWindow   time.Duration
	batchSize    int
	now          func() time.Time
	stop         chan struct{}
	stopOnce     sync.Once
	wg           sync.WaitGroup
}

func NewSLAAlerts(pool *pgxpool.Pool, topic string, pollInterval, nearWindow time.Duration, batchSize int) *SLAAlerts {
	slaAlertsTotal.WithLabelValues("near_breach")
	slaAlertsTotal.WithLabelValues("breached")
	slaAlertBatchTotal.WithLabelValues("success")
	slaAlertBatchTotal.WithLabelValues("failed")
	return &SLAAlerts{
		pool: pool, topic: topic, pollInterval: pollInterval, nearWindow: nearWindow,
		batchSize: batchSize, now: time.Now, stop: make(chan struct{}),
	}
}

func (w *SLAAlerts) Start(ctx context.Context) {
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		ticker := time.NewTicker(w.pollInterval)
		defer ticker.Stop()
		w.run(ctx)
		for {
			select {
			case <-ctx.Done():
				return
			case <-w.stop:
				return
			case <-ticker.C:
				w.run(ctx)
			}
		}
	}()
}

func (w *SLAAlerts) run(ctx context.Context) {
	batchCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	if err := w.processBatch(batchCtx); err != nil {
		slaAlertBatchTotal.WithLabelValues("failed").Inc()
		if ctx.Err() != nil {
			return
		}
		log.Error().Err(err).Msg("SLA alert batch failed; milestones remain available for retry")
		return
	}
	slaAlertBatchTotal.WithLabelValues("success").Inc()
}

func (w *SLAAlerts) Stop() {
	w.stopOnce.Do(func() { close(w.stop) })
	w.wg.Wait()
}

func (w *SLAAlerts) processBatch(ctx context.Context) error {
	now := w.now().UTC()
	tx, err := w.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `
		SELECT m.id, m.ticket_id, m.milestone_type, m.due_at,
		       t.code, t.priority,
		       COALESCE(t.assigned_staff_id, '00000000-0000-0000-0000-000000000000'::uuid)
		FROM ticket_sla_milestones m
		JOIN support_tickets t ON t.id = m.ticket_id
		WHERE m.completed_at IS NULL
		  AND m.paused_at IS NULL
		  AND t.status NOT IN ('RESOLVED', 'CLOSED')
		  AND (
		      (m.breach_alerted_at IS NULL AND m.due_at <= $1)
		      OR
		      (m.near_breach_alerted_at IS NULL AND m.due_at > $1 AND m.due_at <= $2)
		  )
		ORDER BY m.due_at, m.id
		LIMIT $3
		FOR UPDATE OF m SKIP LOCKED
	`, now, now.Add(w.nearWindow), w.batchSize)
	if err != nil {
		return fmt.Errorf("select SLA alert candidates: %w", err)
	}

	milestones := make([]slaMilestone, 0, w.batchSize)
	for rows.Next() {
		var milestone slaMilestone
		if err := rows.Scan(
			&milestone.ID, &milestone.TicketID, &milestone.MilestoneType, &milestone.DueAt,
			&milestone.TicketCode, &milestone.Priority, &milestone.AssignedStaffID,
		); err != nil {
			rows.Close()
			return err
		}
		milestones = append(milestones, milestone)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()

	nearBreachCount := 0
	breachedCount := 0
	for _, milestone := range milestones {
		eventType := classifySLAAlert(milestone.DueAt, now, w.nearWindow)
		if eventType == "" {
			continue
		}
		var commandTag pgconn.CommandTag
		if eventType == slaBreachedEvent {
			commandTag, err = tx.Exec(ctx, `
				UPDATE ticket_sla_milestones
				SET breached_at = COALESCE(breached_at, $2), breach_alerted_at = $2, updated_at = $2
				WHERE id = $1 AND completed_at IS NULL AND paused_at IS NULL AND breach_alerted_at IS NULL
			`, milestone.ID, now)
		} else {
			commandTag, err = tx.Exec(ctx, `
				UPDATE ticket_sla_milestones
				SET near_breach_alerted_at = $2, updated_at = $2
				WHERE id = $1 AND completed_at IS NULL AND paused_at IS NULL AND near_breach_alerted_at IS NULL
			`, milestone.ID, now)
		}
		if err != nil {
			return fmt.Errorf("mark SLA alert: %w", err)
		}
		if commandTag.RowsAffected() != 1 {
			continue
		}
		if eventType == slaBreachedEvent {
			breachedCount++
		} else {
			nearBreachCount++
		}
		payload, _ := json.Marshal(slaAlertPayload(milestone, now))
		correlationID := idgen.New().String()
		if _, err := tx.Exec(ctx, `
			INSERT INTO outbox_events (
				id, aggregate_type, aggregate_id, event_type, topic, partition_key, payload, correlation_id
			) VALUES ($1,'TICKET_SLA_MILESTONE',$2,$3,$4,$5,$6,$7)
		`, idgen.New(), milestone.ID.String(), eventType, w.topic, milestone.TicketID.String(), payload, correlationID); err != nil {
			return fmt.Errorf("insert SLA alert outbox event: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit SLA alert batch: %w", err)
	}
	slaAlertsTotal.WithLabelValues("near_breach").Add(float64(nearBreachCount))
	slaAlertsTotal.WithLabelValues("breached").Add(float64(breachedCount))
	return nil
}

func classifySLAAlert(dueAt, now time.Time, nearWindow time.Duration) string {
	if !dueAt.After(now) {
		return slaBreachedEvent
	}
	if !dueAt.After(now.Add(nearWindow)) {
		return slaNearBreachEvent
	}
	return ""
}

func slaAlertPayload(milestone slaMilestone, observedAt time.Time) map[string]any {
	payload := map[string]any{
		"ticket_id": milestone.TicketID, "ticket_code": milestone.TicketCode,
		"milestone_id": milestone.ID, "milestone_type": milestone.MilestoneType,
		"priority": milestone.Priority, "due_at": milestone.DueAt, "observed_at": observedAt,
	}
	if milestone.AssignedStaffID != uuid.Nil {
		payload["assigned_staff_id"] = milestone.AssignedStaffID
	} else {
		payload["assigned_staff_id"] = nil
	}
	return payload
}
