package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/promotion-service/internal/repository/postgres"
	"github.com/segmentio/kafka-go"
)

type eventRepo struct {
	calls   int
	invalid bool
	seen    []string
}

func (r *eventRepo) Pool() *pgxpool.Pool { return nil }
func (r *eventRepo) ProcessOrderPaidV3(_ context.Context, b []byte, _ string) (bool, error) {
	r.calls++
	r.seen = append(r.seen, string(b))
	if r.invalid {
		return false, postgres.ErrInvalidOrderEvent
	}
	if r.calls == 1 {
		return false, errors.New("temporary database outage")
	}
	return true, nil
}

type eventReader struct{ commits int }

func (r *eventReader) FetchMessage(context.Context) (kafka.Message, error) {
	panic("must not fetch ahead during processing")
}
func (r *eventReader) Close() error { return nil }
func (r *eventReader) CommitMessages(context.Context, ...kafka.Message) error {
	r.commits++
	return nil
}

type flakyWriter struct{ calls int }

func (w *flakyWriter) Close() error { return nil }
func (w *flakyWriter) WriteMessages(context.Context, ...kafka.Message) error {
	w.calls++
	if w.calls == 1 {
		return errors.New("DLQ unavailable")
	}
	return nil
}

func TestOrderPaidRetryKeepsCurrentRecordUntilDurable(t *testing.T) {
	for _, invalid := range []bool{false, true} {
		r, reader, writer := &eventRepo{invalid: invalid}, &eventReader{}, &flakyWriter{}
		w := &EventWorkers{repo: r, reader: reader, writer: writer, outputTopic: "promotion.events.v1", dlqTopic: "promotion.order.v3.dlq"}
		ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
		if !w.processOrderMessage(ctx, kafka.Message{Topic: "order.events.v3", Value: []byte("original-event"), Offset: 7}) {
			t.Fatal("record was not settled")
		}
		cancel()
		if r.calls != 2 || reader.commits != 1 || r.seen[0] != r.seen[1] {
			t.Fatal("skipped record or committed before durable processing", r.calls, reader.commits)
		}
		if invalid && writer.calls != 2 {
			t.Fatal("DLQ must retry before source offset commit")
		}
	}
}
