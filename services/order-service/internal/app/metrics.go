package app

import (
	"context"
	"github.com/prometheus/client_golang/prometheus"
	"time"
)

type backlogCollector struct {
	S          *Service
	Age, Count *prometheus.Desc
}

func (c *backlogCollector) Describe(out chan<- *prometheus.Desc) { out <- c.Age; out <- c.Count }
func (c *backlogCollector) Collect(out chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	queries := map[string]string{
		"checkout":       "SELECT count(*),COALESCE(EXTRACT(EPOCH FROM clock_timestamp()-min(accepted_at)),0) FROM checkout_operations WHERE completed_at IS NULL",
		"outbox":         "SELECT count(*),COALESCE(EXTRACT(EPOCH FROM clock_timestamp()-min(created_at)),0) FROM outbox WHERE published_at IS NULL",
		"manual_review":  "SELECT count(*),COALESCE(EXTRACT(EPOCH FROM clock_timestamp()-min(created_at)),0) FROM jobs WHERE status='MANUAL_REVIEW'",
		"deferred_inbox": "SELECT count(*),COALESCE(EXTRACT(EPOCH FROM clock_timestamp()-min(received_at)),0) FROM inbox WHERE status='DEFERRED'",
		"refund_unknown": "SELECT count(*),COALESCE(EXTRACT(EPOCH FROM clock_timestamp()-min(created_at)),0) FROM refunds WHERE status IN ('UNKNOWN','MANUAL_REVIEW')",
		"reconciliation": "SELECT count(*),COALESCE(EXTRACT(EPOCH FROM clock_timestamp()-min(created_at)),0) FROM reconciliation_cases WHERE status='OPEN'",
		"sla_overdue":    "SELECT count(*),COALESCE(EXTRACT(EPOCH FROM clock_timestamp()-min(sla_due_at)),0) FROM orders WHERE sla_due_at<clock_timestamp() AND status IN ('PAID','CONFIRMED_COD','PROCESSING','PACKED','SHIPPED','DELIVERY_FAILED')",
	}
	for phase, query := range queries {
		var count, age float64
		if e := c.S.DB.QueryRow(ctx, query).Scan(&count, &age); e != nil {
			continue
		}
		if age < 0 {
			age = 0
		}
		out <- prometheus.MustNewConstMetric(c.Count, prometheus.GaugeValue, count, phase)
		out <- prometheus.MustNewConstMetric(c.Age, prometheus.GaugeValue, age, phase)
	}
}
