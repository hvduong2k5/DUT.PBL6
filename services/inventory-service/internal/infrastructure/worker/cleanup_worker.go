package worker

import (
	"context"
	"errors"
	"log"
	"sync/atomic"
	"time"

	"dut-pbl6/inventory-service/internal/application/port"
	"dut-pbl6/inventory-service/internal/application/usecase"
)

// CleanupWorkerConfig defines configuration for the TTLReservationCleanupWorker.
type CleanupWorkerConfig struct {
	Interval  time.Duration
	BatchSize int
}

// DefaultCleanupWorkerConfig provides production defaults (every 60 seconds, batch size 50).
func DefaultCleanupWorkerConfig() CleanupWorkerConfig {
	return CleanupWorkerConfig{
		Interval:  60 * time.Second,
		BatchSize: 50,
	}
}

// TTLReservationCleanupWorker periodically scans and releases expired pending stock reservations.
type TTLReservationCleanupWorker struct {
	repo      port.InventoryRepository
	releaseUC *usecase.ReleaseReservationUseCase
	config    CleanupWorkerConfig
	isRunning atomic.Bool
	stopChan  chan struct{}
	doneChan  chan struct{}
}

// NewTTLReservationCleanupWorker creates a new TTLReservationCleanupWorker.
func NewTTLReservationCleanupWorker(
	repo port.InventoryRepository,
	releaseUC *usecase.ReleaseReservationUseCase,
	cfg ...CleanupWorkerConfig,
) *TTLReservationCleanupWorker {
	c := DefaultCleanupWorkerConfig()
	if len(cfg) > 0 {
		c = cfg[0]
		if c.Interval <= 0 {
			c.Interval = 60 * time.Second
		}
		if c.BatchSize <= 0 {
			c.BatchSize = 50
		}
	}

	return &TTLReservationCleanupWorker{
		repo:      repo,
		releaseUC: releaseUC,
		config:    c,
		stopChan:  make(chan struct{}),
		doneChan:  make(chan struct{}),
	}
}

// Start launches the periodic cleanup loop.
func (w *TTLReservationCleanupWorker) Start(ctx context.Context) error {
	if !w.isRunning.CompareAndSwap(false, true) {
		return ErrWorkerAlreadyRunning
	}

	w.stopChan = make(chan struct{})
	w.doneChan = make(chan struct{})

	go w.run(ctx)
	return nil
}

// Stop gracefully stops the worker.
func (w *TTLReservationCleanupWorker) Stop(timeout ...time.Duration) error {
	if !w.isRunning.CompareAndSwap(true, false) {
		return ErrWorkerNotRunning
	}

	close(w.stopChan)

	to := 5 * time.Second
	if len(timeout) > 0 && timeout[0] > 0 {
		to = timeout[0]
	}

	select {
	case <-w.doneChan:
		return nil
	case <-time.After(to):
		return errors.New("timeout waiting for cleanup worker to stop")
	}
}

func (w *TTLReservationCleanupWorker) run(ctx context.Context) {
	defer close(w.doneChan)

	ticker := time.NewTicker(w.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopChan:
			return
		case <-ticker.C:
			_, _ = w.RunOnce(ctx, time.Now().UTC())
		}
	}
}

// RunOnce performs a single pass of scanning and releasing expired reservations.
func (w *TTLReservationCleanupWorker) RunOnce(ctx context.Context, now time.Time) (int, error) {
	if w.repo == nil || w.releaseUC == nil {
		return 0, nil
	}

	expiredReservations, err := w.repo.GetExpiredPendingReservations(ctx, now, w.config.BatchSize)
	if err != nil {
		log.Printf("[TTLReservationCleanupWorker] Error querying expired reservations: %v", err)
		return 0, err
	}

	if len(expiredReservations) == 0 {
		return 0, nil
	}

	releasedCount := 0
	for _, res := range expiredReservations {
		itemsRestored, err := w.releaseUC.ExecuteWithResult(ctx, res.OrderID, res.ID.String())
		if err != nil {
			log.Printf("[TTLReservationCleanupWorker] Failed to release expired reservation %s (order: %s): %v",
				res.ID, res.OrderID, err)
			continue
		}
		log.Printf("[TTLReservationCleanupWorker] Successfully released expired reservation %s (order: %s), restored %d items",
			res.ID, res.OrderID, itemsRestored)
		releasedCount++
	}

	return releasedCount, nil
}
