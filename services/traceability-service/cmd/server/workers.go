package main

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/application"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/config"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/infrastructure/broker"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/infrastructure/postgres"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/observability"
)

func background(ctx context.Context, options config.Operations, queue *postgres.Outbox, store *postgres.Profiles, metrics *observability.Metrics, logger *slog.Logger) (func(), error) {
	if logger == nil {
		logger = slog.Default()
	}
	var publisher *broker.Kafka
	var err error
	if options.RelayEnabled {
		publisher, err = broker.NewKafka("sandbox", options.Brokers)
		if err != nil {
			return nil, err
		}
	}
	workerCtx, stop := context.WithCancel(ctx)
	var wg sync.WaitGroup
	if publisher != nil {
		relay := &application.Relay{Queue: queue, Publisher: publisher, PublishTimeout: 5 * time.Second}
		startLoop(workerCtx, &wg, 500*time.Millisecond, func(ctx context.Context) {
			processed, err := relay.Tick(ctx)
			metrics.Relay(processed, err)
			if err != nil && workerCtx.Err() == nil {
				logger.Error("outbox relay tick failed", "error_code", "RELAY_TICK_FAILED")
			}
		})
	}
	if options.ExpiryEnabled {
		expiry := &application.ExpiryWorker{Store: store, PageSize: 100}
		startLoop(workerCtx, &wg, 30*time.Second, func(ctx context.Context) {
			count, err := expiry.Tick(ctx)
			metrics.Expiry(count, err)
			if err != nil && workerCtx.Err() == nil {
				logger.Error("expiry worker tick failed", "error_code", "EXPIRY_TICK_FAILED")
			}
		})
	}
	logger.Info("background capabilities configured", "relay_enabled", options.RelayEnabled, "expiry_worker_enabled", options.ExpiryEnabled)
	return func() {
		stop()
		wg.Wait()
		if publisher != nil {
			if err := publisher.Close(); err != nil {
				logger.Error("broker close failed", "error_code", "BROKER_CLOSE_FAILED")
			}
		}
	}, nil
}

func startLoop(ctx context.Context, wg *sync.WaitGroup, interval time.Duration, tick func(context.Context)) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		timer := time.NewTicker(interval)
		defer timer.Stop()
		for {
			if ctx.Err() != nil {
				return
			}
			run, cancel := context.WithTimeout(ctx, 8*time.Second)
			tick(run)
			cancel()
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
			}
		}
	}()
}
