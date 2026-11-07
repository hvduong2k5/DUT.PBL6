package worker

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/care-service/internal/domain"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/rs/zerolog/log"
)

var (
	mediaFilesByStatus = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "care_media_files", Help: "Current Care media records by lifecycle status.",
	}, []string{"status"})
	mediaFilesByScanStatus = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "care_media_scan_files", Help: "Current Care media records by scan status.",
	}, []string{"scan_status"})
	mediaCleanupObjects = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "care_media_cleanup_objects_total", Help: "Media object cleanup attempts by outcome.",
	}, []string{"outcome"})
)

type expiredMedia struct {
	ID        uuid.UUID
	ObjectKey string
}

type MediaCleanup struct {
	pool         *pgxpool.Pool
	storage      domain.ObjectStorage
	pollInterval time.Duration
	batchSize    int
	now          func() time.Time
	stop         chan struct{}
	stopOnce     sync.Once
	wg           sync.WaitGroup
}

func NewMediaCleanup(pool *pgxpool.Pool, storage domain.ObjectStorage, pollInterval time.Duration, batchSize int) *MediaCleanup {
	mediaCleanupObjects.WithLabelValues("deleted")
	mediaCleanupObjects.WithLabelValues("failed")
	return &MediaCleanup{
		pool: pool, storage: storage, pollInterval: pollInterval, batchSize: batchSize,
		now: time.Now, stop: make(chan struct{}),
	}
}

func (w *MediaCleanup) Start(ctx context.Context) {
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

func (w *MediaCleanup) run(ctx context.Context) {
	batchCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	err := w.processBatch(batchCtx)
	w.reportMetrics(batchCtx)
	if err != nil && ctx.Err() == nil {
		log.Error().Err(err).Msg("media cleanup batch failed; expired objects remain available for retry")
	}
}

func (w *MediaCleanup) Stop() {
	w.stopOnce.Do(func() { close(w.stop) })
	w.wg.Wait()
}

func (w *MediaCleanup) processBatch(ctx context.Context) error {
	now := w.now().UTC()
	if err := w.expireCandidates(ctx, now); err != nil {
		return err
	}
	return w.deleteExpired(ctx, now)
}

func (w *MediaCleanup) expireCandidates(ctx context.Context, now time.Time) error {
	tx, err := w.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		UPDATE media_files
		SET status = 'EXPIRED', updated_at = $1, version = version + 1
		WHERE id IN (
			SELECT id
			FROM media_files
			WHERE status IN ('PENDING_UPLOAD', 'UPLOADED', 'SCANNING', 'READY', 'REJECTED')
			  AND expires_at <= $1
			ORDER BY expires_at, id
			LIMIT $2
			FOR UPDATE SKIP LOCKED
		)
	`, now, w.batchSize); err != nil {
		return fmt.Errorf("expire media cleanup candidates: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit expired media candidates: %w", err)
	}
	return nil
}

func (w *MediaCleanup) deleteExpired(ctx context.Context, now time.Time) error {
	tx, err := w.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `
		SELECT id, object_key
		FROM media_files
		WHERE status = 'EXPIRED'
		ORDER BY expires_at, id
		LIMIT $1
		FOR UPDATE SKIP LOCKED
	`, w.batchSize)
	if err != nil {
		return fmt.Errorf("select expired media: %w", err)
	}
	items := make([]expiredMedia, 0, w.batchSize)
	for rows.Next() {
		var item expiredMedia
		if err := rows.Scan(&item.ID, &item.ObjectKey); err != nil {
			rows.Close()
			return err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()

	var cleanupErrors []error
	deletedCount := 0
	for _, item := range items {
		if err := w.storage.Remove(ctx, item.ObjectKey); err != nil {
			mediaCleanupObjects.WithLabelValues("failed").Inc()
			cleanupErrors = append(cleanupErrors, fmt.Errorf("remove media object %s: %w", item.ID, err))
			continue
		}
		if _, err := tx.Exec(ctx, `
			UPDATE media_files
			SET status = 'DELETED', deleted_at = $2, updated_at = $2, version = version + 1
			WHERE id = $1 AND status = 'EXPIRED'
		`, item.ID, now); err != nil {
			return fmt.Errorf("mark expired media deleted: %w", err)
		}
		deletedCount++
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit expired media cleanup: %w", err)
	}
	mediaCleanupObjects.WithLabelValues("deleted").Add(float64(deletedCount))
	return errors.Join(cleanupErrors...)
}

func (w *MediaCleanup) reportMetrics(ctx context.Context) {
	statuses := []string{"PENDING_UPLOAD", "UPLOADED", "SCANNING", "READY", "ATTACHED", "REJECTED", "EXPIRED", "DELETED"}
	for _, status := range statuses {
		mediaFilesByStatus.WithLabelValues(status).Set(0)
	}
	rows, err := w.pool.Query(ctx, `SELECT status, COUNT(*) FROM media_files GROUP BY status`)
	if err == nil {
		for rows.Next() {
			var status string
			var count int64
			if rows.Scan(&status, &count) == nil {
				mediaFilesByStatus.WithLabelValues(status).Set(float64(count))
			}
		}
		rows.Close()
	}

	scanStatuses := []string{"PENDING", "CLEAN", "INFECTED", "FAILED"}
	for _, status := range scanStatuses {
		mediaFilesByScanStatus.WithLabelValues(status).Set(0)
	}
	rows, err = w.pool.Query(ctx, `SELECT scan_status, COUNT(*) FROM media_files GROUP BY scan_status`)
	if err == nil {
		for rows.Next() {
			var status string
			var count int64
			if rows.Scan(&status, &count) == nil {
				mediaFilesByScanStatus.WithLabelValues(status).Set(float64(count))
			}
		}
		rows.Close()
	}
}
