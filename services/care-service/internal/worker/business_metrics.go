package worker

import (
	"context"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/rs/zerolog/log"
)

var (
	ticketsByState = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "care_tickets", Help: "Current Care tickets by status, priority, and assignment state.",
	}, []string{"status", "priority", "assignment"})
	slaMilestonesByState = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "care_sla_milestones", Help: "Current Care SLA milestones by type and state.",
	}, []string{"milestone_type", "state"})
	businessMetricsCollections = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "care_business_metrics_collections_total", Help: "Business metric collection runs by outcome.",
	}, []string{"outcome"})
)

type BusinessMetrics struct {
	pool         *pgxpool.Pool
	pollInterval time.Duration
	stop         chan struct{}
	stopOnce     sync.Once
	wg           sync.WaitGroup
}

func NewBusinessMetrics(pool *pgxpool.Pool, pollInterval time.Duration) *BusinessMetrics {
	businessMetricsCollections.WithLabelValues("success")
	businessMetricsCollections.WithLabelValues("failed")
	return &BusinessMetrics{pool: pool, pollInterval: pollInterval, stop: make(chan struct{})}
}

func (w *BusinessMetrics) Start(ctx context.Context) {
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

func (w *BusinessMetrics) Stop() {
	w.stopOnce.Do(func() { close(w.stop) })
	w.wg.Wait()
}

func (w *BusinessMetrics) run(ctx context.Context) {
	queryCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := w.collect(queryCtx); err != nil {
		businessMetricsCollections.WithLabelValues("failed").Inc()
		if ctx.Err() == nil {
			log.Error().Err(err).Msg("business metrics collection failed")
		}
		return
	}
	businessMetricsCollections.WithLabelValues("success").Inc()
}

func (w *BusinessMetrics) collect(ctx context.Context) error {
	statuses := []string{"NEW", "TRIAGED", "ASSIGNED", "IN_PROGRESS", "WAITING_CUSTOMER", "WAITING_INTERNAL", "RESOLVED", "CLOSED"}
	priorities := []string{"LOW", "NORMAL", "HIGH", "URGENT"}
	assignments := []string{"assigned", "unassigned"}
	for _, status := range statuses {
		for _, priority := range priorities {
			for _, assignment := range assignments {
				ticketsByState.WithLabelValues(status, priority, assignment).Set(0)
			}
		}
	}

	rows, err := w.pool.Query(ctx, `
		SELECT status, priority,
		       CASE WHEN assigned_staff_id IS NULL THEN 'unassigned' ELSE 'assigned' END,
		       COUNT(*)
		FROM support_tickets
		GROUP BY status, priority, assigned_staff_id IS NULL
	`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var status, priority, assignment string
		var count int64
		if err := rows.Scan(&status, &priority, &assignment, &count); err != nil {
			rows.Close()
			return err
		}
		ticketsByState.WithLabelValues(status, priority, assignment).Set(float64(count))
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()

	types := []string{"FIRST_RESPONSE", "RESOLUTION"}
	states := []string{"active", "paused", "breached", "completed", "completed_breached"}
	for _, milestoneType := range types {
		for _, state := range states {
			slaMilestonesByState.WithLabelValues(milestoneType, state).Set(0)
		}
	}
	rows, err = w.pool.Query(ctx, `
		SELECT milestone_type,
		       CASE
		           WHEN completed_at IS NOT NULL AND breached_at IS NOT NULL THEN 'completed_breached'
		           WHEN completed_at IS NOT NULL THEN 'completed'
		           WHEN paused_at IS NOT NULL THEN 'paused'
		           WHEN due_at <= NOW() THEN 'breached'
		           ELSE 'active'
		       END AS state,
		       COUNT(*)
		FROM ticket_sla_milestones
		GROUP BY milestone_type, state
	`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var milestoneType, state string
		var count int64
		if err := rows.Scan(&milestoneType, &state, &count); err != nil {
			return err
		}
		slaMilestonesByState.WithLabelValues(milestoneType, state).Set(float64(count))
	}
	return rows.Err()
}
