package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/omamx/order-service/internal/domain"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/segmentio/kafka-go"
	"log/slog"
	"time"
)

func (s *Service) Registry() *prometheus.Registry { return s.register }
func (s *Service) Start(ctx context.Context) {
	for i := 0; i < 4; i++ {
		s.stop.Add(1)
		go func() {
			defer s.stop.Done()
			ticker := time.NewTicker(s.C.Poll)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					if _, e := s.WorkPlacement(ctx); e != nil && !errors.Is(e, context.Canceled) {
						slog.Error("placement worker", "code", safeCode(s, e))
					}
					if _, e := s.WorkJob(ctx); e != nil && !errors.Is(e, context.Canceled) {
						slog.Error("job worker", "code", safeCode(s, e))
					}
				}
			}
		}()
	}
	s.stop.Add(1)
	go func() {
		defer s.stop.Done()
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				for _, fn := range []func(context.Context) error{s.ReconcilePayments, s.SweepTimeouts, s.ReplayDeferred, s.CompleteOrders} {
					if e := fn(ctx); e != nil && !errors.Is(e, context.Canceled) {
						slog.Error("recovery worker", "code", safeCode(s, e))
					}
				}
			}
		}
	}()
	if s.Writer != nil {
		s.stop.Add(1)
		go func() {
			defer s.stop.Done()
			ticker := time.NewTicker(200 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					for i := 0; i < 20; i++ {
						worked, e := s.PublishOne(ctx)
						if e != nil || !worked {
							break
						}
					}
				}
			}
		}()
		s.stop.Add(1)
		go s.consume(ctx)
	}
}
func safeCode(s *Service, e error) string { code, _ := s.ErrorCode(e); return code }
func (s *Service) PublishOne(ctx context.Context) (bool, error) {
	if s.Writer == nil {
		return false, nil
	}
	var id, aggregate, topic string
	var b []byte
	var epoch int64
	e := s.DB.QueryRow(ctx, `WITH picked AS (SELECT a.id FROM outbox a WHERE a.published_at IS NULL AND a.run_after<=clock_timestamp() AND (a.lease_until IS NULL OR a.lease_until<clock_timestamp()) AND NOT EXISTS(SELECT 1 FROM outbox older WHERE older.aggregate_type=a.aggregate_type AND older.aggregate_id=a.aggregate_id AND (older.version,older.ordinal)<(a.version,a.ordinal) AND older.published_at IS NULL) ORDER BY a.version,a.ordinal FOR UPDATE SKIP LOCKED LIMIT 1) UPDATE outbox o SET lease_owner=$1,lease_until=clock_timestamp()+interval '10 seconds',lease_epoch=lease_epoch+1 FROM picked WHERE o.id=picked.id RETURNING o.id,o.aggregate_id,o.topic,o.payload,o.lease_epoch`, s.workerID).Scan(&id, &aggregate, &topic, &b, &epoch)
	if errors.Is(e, pgx.ErrNoRows) {
		return false, nil
	}
	if e != nil {
		return false, e
	}
	e = s.Writer.WriteMessages(ctx, kafka.Message{Topic: topic, Key: []byte(aggregate), Value: b})
	if e != nil {
		_, _ = s.DB.Exec(ctx, "UPDATE outbox SET run_after=clock_timestamp()+interval '1 second',lease_owner=NULL,lease_until=NULL WHERE id=$1 AND lease_owner=$2 AND lease_epoch=$3", id, s.workerID, epoch)
		return true, e
	}
	_, e = s.DB.Exec(ctx, "UPDATE outbox SET published_at=clock_timestamp(),lease_owner=NULL,lease_until=NULL WHERE id=$1 AND lease_owner=$2 AND lease_epoch=$3", id, s.workerID, epoch)
	return true, e
}
func (s *Service) consume(ctx context.Context) {
	defer s.stop.Done()
	topic := s.C.SourceTopic
	if topic == "" {
		topic = "order.sources.v1"
	}
	reader := kafka.NewReader(kafka.ReaderConfig{Brokers: splitBrokers(s.C.Kafka), Topic: topic, GroupID: consumerGroup(s.C), MinBytes: 1, MaxBytes: 1 << 20, CommitInterval: 0})
	defer reader.Close()
	for {
		m, e := reader.FetchMessage(ctx)
		if e != nil {
			if ctx.Err() != nil {
				return
			}
			continue
		}
		for {
			var event domain.Event
			decoder := json.NewDecoder(bytes.NewReader(m.Value))
			decoder.DisallowUnknownFields()
			e = decoder.Decode(&event)
			if e == nil {
				_, e = s.ProcessEvent(ctx, event)
			}
			if e != nil {
				_, code := s.ErrorCode(e)
				if code >= 500 {
					select {
					case <-ctx.Done():
						return
					case <-time.After(time.Second):
						continue
					}
				}
				if e = s.Quarantine(ctx, fmt.Sprintf("%s:%d:%d", m.Topic, m.Partition, m.Offset), m.Value, "INVALID_SOURCE_EVENT"); e != nil {
					continue
				}
			}
			if e = reader.CommitMessages(ctx, m); e != nil {
				if ctx.Err() != nil {
					return
				}
				continue
			}
			break
		}
	}
}
func splitBrokers(v string) []string {
	out := []string{}
	start := 0
	for i, c := range v {
		if c == ',' {
			out = append(out, v[start:i])
			start = i + 1
		}
	}
	return append(out, v[start:])
}

func consumerGroup(c Config) string {
	if c.ConsumerGroup != "" {
		return c.ConsumerGroup
	}
	return "order-service-source-v3"
}
