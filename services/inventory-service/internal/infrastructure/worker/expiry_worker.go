package worker

import (
	"context"
	"errors"
	"log"
	"sync/atomic"
	"time"

	"dut-pbl6/inventory-service/internal/application/port"
	"dut-pbl6/inventory-service/internal/domain/entity"
)

// ExpiryCheckWorkerConfig defines parameters for ExpiryCheckWorker.
type ExpiryCheckWorkerConfig struct {
	Interval      time.Duration
	ThresholdDays int
	BatchSize     int
}

// DefaultExpiryCheckWorkerConfig returns default settings (daily run, 45-day threshold, batch size 50).
func DefaultExpiryCheckWorkerConfig() ExpiryCheckWorkerConfig {
	return ExpiryCheckWorkerConfig{
		Interval:      24 * time.Hour,
		ThresholdDays: 45,
		BatchSize:     50,
	}
}

// ExpiryCheckWorker scans batches and marks near-expiry (< 45 days) or expired batches.
type ExpiryCheckWorker struct {
	repo       port.InventoryRepository
	txManager  port.TransactionManager
	outboxRepo port.OutboxRepository
	config     ExpiryCheckWorkerConfig
	isRunning  atomic.Bool
	stopChan   chan struct{}
	doneChan   chan struct{}
}

// NewExpiryCheckWorker creates a new ExpiryCheckWorker instance.
func NewExpiryCheckWorker(
	repo port.InventoryRepository,
	txManager port.TransactionManager,
	outboxRepo port.OutboxRepository,
	cfg ...ExpiryCheckWorkerConfig,
) *ExpiryCheckWorker {
	c := DefaultExpiryCheckWorkerConfig()
	if len(cfg) > 0 {
		c = cfg[0]
		if c.Interval <= 0 {
			c.Interval = 24 * time.Hour
		}
		if c.ThresholdDays <= 0 {
			c.ThresholdDays = 45
		}
		if c.BatchSize <= 0 {
			c.BatchSize = 50
		}
	}

	return &ExpiryCheckWorker{
		repo:       repo,
		txManager:  txManager,
		outboxRepo: outboxRepo,
		config:     c,
		stopChan:   make(chan struct{}),
		doneChan:   make(chan struct{}),
	}
}

// Start launches the periodic expiry check background goroutine.
func (w *ExpiryCheckWorker) Start(ctx context.Context) error {
	if !w.isRunning.CompareAndSwap(false, true) {
		return ErrWorkerAlreadyRunning
	}

	w.stopChan = make(chan struct{})
	w.doneChan = make(chan struct{})

	go w.run(ctx)
	return nil
}

// Stop gracefully terminates the expiry check worker.
func (w *ExpiryCheckWorker) Stop(timeout ...time.Duration) error {
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
		return errors.New("timeout waiting for expiry worker to stop")
	}
}

func (w *ExpiryCheckWorker) run(ctx context.Context) {
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

// RunOnce scans batches against the threshold date and transitions their statuses.
func (w *ExpiryCheckWorker) RunOnce(ctx context.Context, now time.Time) (int, error) {
	if w.repo == nil || w.txManager == nil {
		return 0, nil
	}

	thresholdDate := now.Add(time.Duration(w.config.ThresholdDays) * 24 * time.Hour)
	candidateBatches, err := w.repo.GetBatchesNearExpiry(ctx, thresholdDate, w.config.BatchSize)
	if err != nil {
		log.Printf("[ExpiryCheckWorker] Error querying near-expiry batches: %v", err)
		return 0, err
	}

	if len(candidateBatches) == 0 {
		return 0, nil
	}

	processedCount := 0

	for _, cand := range candidateBatches {
		txErr := w.txManager.ExecuteInTransaction(ctx, func(tx port.Transaction) error {
			batch, err := w.repo.GetBatchByID(ctx, tx, cand.ID)
			if err != nil {
				return err
			}
			if batch == nil {
				return nil
			}

			// If already expired or quarantined, nothing to do
			if batch.Status == entity.BatchStatusExpired || batch.Status == entity.BatchStatusQuarantine {
				return nil
			}

			// Case 1: Already expired (exp_date <= now)
			if !now.Before(batch.ExpDate) {
				if batch.ReservedQty > 0 {
					batch.Status = entity.BatchStatusQuarantine
				} else {
					batch.Status = entity.BatchStatusExpired
				}
				if err := w.repo.UpdateBatch(ctx, tx, batch); err != nil {
					return err
				}
				processedCount++
				return nil
			}

			// Case 2: Near expiry (< 45 days)
			if batch.ExpDate.Before(thresholdDate) || batch.ExpDate.Equal(thresholdDate) {
				if batch.Status == entity.BatchStatusActive {
					batch.Status = entity.BatchStatusNearExpiry
					if err := w.repo.UpdateBatch(ctx, tx, batch); err != nil {
						return err
					}

					daysUntil := int(batch.ExpDate.Sub(now).Hours() / 24)
					if daysUntil < 0 {
						daysUntil = 0
					}

					if w.outboxRepo != nil && batch.AvailableQty() > 0 {
						ce, err := entity.NewExpiryWarningCloudEvent(
							batch.BatchCode,
							batch.SKU,
							entity.DefaultWarehouseID.String(),
							batch.AvailableQty(),
							daysUntil,
							batch.ExpDate,
							"FLASH_SALE_PROMOTION",
							"",
						)
						if err != nil {
							return err
						}

						outboxEv, err := entity.NewOutboxEvent(
							"Batch",
							batch.ID.String(),
							entity.EventTypeExpiryWarning,
							ce,
						)
						if err != nil {
							return err
						}

						if err := w.outboxRepo.SaveEvent(ctx, tx, outboxEv); err != nil {
							return err
						}
					}
					processedCount++
				}
			}

			return nil
		})

		if txErr != nil {
			log.Printf("[ExpiryCheckWorker] Error processing batch %s (%s): %v", cand.ID, cand.BatchCode, txErr)
		}
	}

	return processedCount, nil
}
