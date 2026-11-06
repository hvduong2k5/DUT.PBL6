package worker

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/omamx/promotion-service/internal/repository/postgres"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
)

type EventWorkers struct {
	reader      *kafka.Reader
	writer      *kafka.Writer
	repo        *postgres.VoucherRepository
	outputTopic string
	dlqTopic    string
	wg          sync.WaitGroup
}

func NewEventWorkers(brokers, inputTopic, group, outputTopic, dlqTopic string, repo *postgres.VoucherRepository) *EventWorkers {
	addresses := strings.Split(brokers, ",")
	return &EventWorkers{
		reader: kafka.NewReader(kafka.ReaderConfig{Brokers: addresses, Topic: inputTopic, GroupID: group, StartOffset: kafka.FirstOffset, MinBytes: 1, MaxBytes: 1 << 20, CommitInterval: 0}),
		writer: &kafka.Writer{Addr: kafka.TCP(addresses...), Balancer: &kafka.Hash{}, RequiredAcks: kafka.RequireAll, WriteTimeout: 5 * time.Second, ReadTimeout: 5 * time.Second, AllowAutoTopicCreation: true},
		repo:   repo, outputTopic: outputTopic, dlqTopic: dlqTopic,
	}
}

func (w *EventWorkers) Run(ctx context.Context) {
	w.wg.Add(2)
	go func() { defer w.wg.Done(); w.consumeOrderPaid(ctx) }()
	go func() { defer w.wg.Done(); w.publishOutbox(ctx) }()
}

func (w *EventWorkers) Close() error {
	readerErr := w.reader.Close()
	w.wg.Wait()
	writerErr := w.writer.Close()
	return errors.Join(readerErr, writerErr)
}

func (w *EventWorkers) consumeOrderPaid(ctx context.Context) {
	for ctx.Err() == nil {
		message, err := w.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Warn().Err(err).Msg("fetch Order event")
			continue
		}
		_, err = w.repo.ProcessOrderPaidV3(ctx, message.Value, w.outputTopic)
		if err != nil {
			if !errors.Is(err, postgres.ErrInvalidOrderEvent) {
				log.Warn().Err(err).Str("topic", message.Topic).Int64("offset", message.Offset).Msg("Order event processing failed; retrying without committing offset")
				if !pause(ctx, time.Second) {
					return
				}
				continue
			}
			log.Error().Err(err).Str("topic", message.Topic).Int64("offset", message.Offset).Msg("route invalid Order paid event to DLQ")
			dlqMessage := kafka.Message{Topic: w.dlqTopic, Key: message.Key, Value: message.Value, Headers: []kafka.Header{
				{Key: "source_topic", Value: []byte(message.Topic)},
				{Key: "source_partition", Value: []byte(fmt.Sprint(message.Partition))},
				{Key: "source_offset", Value: []byte(fmt.Sprint(message.Offset))},
				{Key: "error", Value: []byte(err.Error())},
			}}
			dlqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			dlqErr := w.writer.WriteMessages(dlqCtx, dlqMessage)
			cancel()
			if dlqErr != nil {
				log.Error().Err(dlqErr).Msg("publish Order event to DLQ; source offset not committed")
				if !pause(ctx, time.Second) {
					return
				}
				continue
			}
		}
		if err := w.reader.CommitMessages(ctx, message); err != nil {
			log.Error().Err(err).Str("topic", message.Topic).Int64("offset", message.Offset).Msg("commit Order event offset")
		}
	}
}

func (w *EventWorkers) publishOutbox(ctx context.Context) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for ctx.Err() == nil {
		published, err := w.publishOne(ctx)
		if err != nil {
			log.Warn().Err(err).Msg("publish Promotion outbox event")
			if !pause(ctx, time.Second) {
				return
			}
			continue
		}
		if published {
			continue
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (w *EventWorkers) publishOne(ctx context.Context) (bool, error) {
	var id, topic, key string
	var payload []byte
	claimCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	tx, err := w.repo.Pool().Begin(claimCtx)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback(claimCtx) }()
	err = tx.QueryRow(claimCtx, `WITH candidate AS (
		SELECT id FROM outbox_events WHERE published_at IS NULL AND next_attempt_at <= clock_timestamp()
		AND (locked_until IS NULL OR locked_until < clock_timestamp())
		ORDER BY occurred_at,id FOR UPDATE SKIP LOCKED LIMIT 1
	) UPDATE outbox_events e SET locked_until=clock_timestamp()+interval '30 seconds'
	FROM candidate c WHERE e.id=c.id RETURNING e.id::text,e.topic,e.partition_key,e.payload`).Scan(&id, &topic, &key, &payload)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := tx.Commit(claimCtx); err != nil {
		return false, err
	}
	writeCtx, writeCancel := context.WithTimeout(ctx, 10*time.Second)
	err = w.writer.WriteMessages(writeCtx, kafka.Message{Topic: topic, Key: []byte(key), Value: payload})
	writeCancel()
	if err != nil {
		return true, w.recordPublishFailure(ctx, id, 0, err)
	}
	_, err = w.repo.Pool().Exec(ctx, `UPDATE outbox_events SET published_at=clock_timestamp(),locked_until=NULL,last_error=NULL WHERE id=$1::uuid`, id)
	return true, err
}

func (w *EventWorkers) recordPublishFailure(ctx context.Context, id string, _ int, cause error) error {
	_, err := w.repo.Pool().Exec(ctx, `UPDATE outbox_events SET attempts=attempts+1,locked_until=NULL,
		next_attempt_at=clock_timestamp()+(LEAST(POWER(2,LEAST(attempts+1,8)),300) * interval '1 second'),last_error=$2
		WHERE id=$1::uuid`, id, cause.Error())
	return errors.Join(cause, err)
}

func pause(ctx context.Context, duration time.Duration) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
