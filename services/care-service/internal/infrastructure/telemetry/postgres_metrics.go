package telemetry

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
)

type postgresPoolCollector struct {
	pool            *pgxpool.Pool
	connections     *prometheus.Desc
	acquires        *prometheus.Desc
	acquireDuration *prometheus.Desc
	emptyAcquires   *prometheus.Desc
}

func RegisterPostgresPoolMetrics(pool *pgxpool.Pool) error {
	return prometheus.Register(&postgresPoolCollector{
		pool: pool,
		connections: prometheus.NewDesc(
			"care_db_pool_connections", "PostgreSQL pool connections by state.",
			[]string{"state"}, nil,
		),
		acquires: prometheus.NewDesc(
			"care_db_pool_acquires_total", "Total successful PostgreSQL pool acquisitions.", nil, nil,
		),
		acquireDuration: prometheus.NewDesc(
			"care_db_pool_acquire_duration_seconds_total", "Cumulative time waiting for PostgreSQL pool connections.", nil, nil,
		),
		emptyAcquires: prometheus.NewDesc(
			"care_db_pool_empty_acquires_total", "PostgreSQL acquisitions that waited for a connection.", nil, nil,
		),
	})
}

func (c *postgresPoolCollector) Describe(channel chan<- *prometheus.Desc) {
	channel <- c.connections
	channel <- c.acquires
	channel <- c.acquireDuration
	channel <- c.emptyAcquires
}

func (c *postgresPoolCollector) Collect(channel chan<- prometheus.Metric) {
	stats := c.pool.Stat()
	channel <- prometheus.MustNewConstMetric(c.connections, prometheus.GaugeValue, float64(stats.TotalConns()), "total")
	channel <- prometheus.MustNewConstMetric(c.connections, prometheus.GaugeValue, float64(stats.MaxConns()), "max")
	channel <- prometheus.MustNewConstMetric(c.connections, prometheus.GaugeValue, float64(stats.AcquiredConns()), "acquired")
	channel <- prometheus.MustNewConstMetric(c.connections, prometheus.GaugeValue, float64(stats.IdleConns()), "idle")
	channel <- prometheus.MustNewConstMetric(c.acquires, prometheus.CounterValue, float64(stats.AcquireCount()))
	channel <- prometheus.MustNewConstMetric(c.acquireDuration, prometheus.CounterValue, stats.AcquireDuration().Seconds())
	channel <- prometheus.MustNewConstMetric(c.emptyAcquires, prometheus.CounterValue, float64(stats.EmptyAcquireCount()))
}
