package worker_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/omamx/procurement-service/internal/domain"
	"github.com/omamx/procurement-service/internal/worker"
)

type fakeOutbox struct {
	message       domain.OutboxMessage
	found         bool
	claimErr      error
	markErr       error
	retryErr      error
	marked        []string
	retried       []string
	retryFailures []string
}

func (f *fakeOutbox) ClaimGoodsReceivedEvent(context.Context) (domain.OutboxMessage, bool, error) {
	return f.message, f.found, f.claimErr
}
func (f *fakeOutbox) MarkGoodsReceivedEventPublished(_ context.Context, id string) error {
	f.marked = append(f.marked, id)
	return f.markErr
}
func (f *fakeOutbox) RetryGoodsReceivedEvent(_ context.Context, id, failure string) error {
	f.retried = append(f.retried, id)
	f.retryFailures = append(f.retryFailures, failure)
	return f.retryErr
}

type fakePublisher struct {
	topic, key string
	payload    []byte
	err        error
}

func (f *fakePublisher) Publish(_ context.Context, topic, key string, payload []byte) error {
	f.topic, f.key, f.payload = topic, key, append([]byte(nil), payload...)
	return f.err
}

func TestDispatchOnePublishesThenMarksOutboxMessage(t *testing.T) {
	store := &fakeOutbox{found: true, message: domain.OutboxMessage{ID: "stable-event-id", Topic: domain.GoodsReceivedEventTopic, Key: "MAT-1", Payload: []byte(`{"id":"stable-event-id"}`)}}
	publisher := &fakePublisher{}
	worker := worker.NewGoodsReceivedOutboxWorker(store, publisher, slog.New(slog.NewTextHandler(io.Discard, nil)))

	found, err := worker.DispatchOne(context.Background())
	if err != nil || !found {
		t.Fatalf("DispatchOne() = (%v, %v), want (true, nil)", found, err)
	}
	if publisher.topic != domain.GoodsReceivedEventTopic || publisher.key != "MAT-1" || string(publisher.payload) != `{"id":"stable-event-id"}` {
		t.Fatalf("unexpected Kafka message: topic=%q key=%q payload=%s", publisher.topic, publisher.key, publisher.payload)
	}
	if len(store.marked) != 1 || store.marked[0] != "stable-event-id" || len(store.retried) != 0 {
		t.Fatalf("unexpected outbox updates: marked=%v retried=%v", store.marked, store.retried)
	}
}

func TestDispatchOneSchedulesRetryWhenPublishFails(t *testing.T) {
	store := &fakeOutbox{found: true, message: domain.OutboxMessage{ID: "stable-event-id"}}
	publisher := &fakePublisher{err: errors.New("broker unavailable")}
	worker := worker.NewGoodsReceivedOutboxWorker(store, publisher, slog.New(slog.NewTextHandler(io.Discard, nil)))

	found, err := worker.DispatchOne(context.Background())
	if !found || err == nil {
		t.Fatalf("DispatchOne() = (%v, %v), want (true, error)", found, err)
	}
	if len(store.retried) != 1 || store.retried[0] != "stable-event-id" || store.retryFailures[0] != "broker unavailable" {
		t.Fatalf("publish failure was not scheduled for retry: %+v", store)
	}
	if len(store.marked) != 0 {
		t.Fatalf("failed publish must not be marked successful: %v", store.marked)
	}
}

func TestDispatchOneSurfacesOutboxClaimFailure(t *testing.T) {
	wantErr := errors.New("database unavailable")
	store := &fakeOutbox{claimErr: wantErr}
	worker := worker.NewGoodsReceivedOutboxWorker(store, &fakePublisher{}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	found, err := worker.DispatchOne(context.Background())
	if found || !errors.Is(err, wantErr) {
		t.Fatalf("DispatchOne() = (%v, %v), want (false, %v)", found, err, wantErr)
	}
}

func TestDispatchOneReturnsWhenOutboxIsEmpty(t *testing.T) {
	worker := worker.NewGoodsReceivedOutboxWorker(&fakeOutbox{}, &fakePublisher{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	found, err := worker.DispatchOne(context.Background())
	if found || err != nil {
		t.Fatalf("DispatchOne() = (%v, %v), want (false, nil)", found, err)
	}
}
