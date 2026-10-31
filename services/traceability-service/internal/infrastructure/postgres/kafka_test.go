package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/application"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/domain"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/infrastructure/broker"
	"github.com/segmentio/kafka-go"
)

func isolatedBroker(t *testing.T, ctx context.Context) []string {
	t.Helper()
	raw := os.Getenv("TRACEABILITY_TEST_KAFKA_BROKERS")
	if raw == "" {
		t.Skip("NOT_RUN: TRACEABILITY_TEST_KAFKA_BROKERS must target the isolated test broker")
	}
	brokers := strings.Split(raw, ",")
	dialer := kafka.Dialer{Timeout: 3 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", brokers[0])
	if err != nil {
		t.Fatal("test broker unavailable")
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	controller, err := conn.Controller()
	if err != nil {
		t.Fatal("cannot find test broker controller")
	}
	admin, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port)))
	if err != nil {
		t.Fatal("cannot connect to test broker controller")
	}
	defer admin.Close()
	_ = admin.SetDeadline(time.Now().Add(5 * time.Second))
	err = admin.CreateTopics(kafka.TopicConfig{Topic: "traceability.events.v1", NumPartitions: 1, ReplicationFactor: 1})
	if err != nil && !errors.Is(err, kafka.TopicAlreadyExists) {
		t.Fatal("cannot provision isolated test topic", err)
	}
	return brokers
}

func readMatchingEvents(t *testing.T, ctx context.Context, brokers []string, ids map[string]int) []domain.CloudEvent {
	t.Helper()
	reader := kafka.NewReader(kafka.ReaderConfig{Brokers: brokers, Topic: "traceability.events.v1", Partition: 0, MinBytes: 1, MaxBytes: 1 << 20, MaxWait: 100 * time.Millisecond, ReadLagInterval: -1})
	defer reader.Close()
	if err := reader.SetOffset(kafka.FirstOffset); err != nil {
		t.Fatal(err)
	}
	read, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	wanted := 0
	for _, count := range ids {
		wanted += count
	}
	result := []domain.CloudEvent{}
	for len(result) < wanted {
		message, err := reader.ReadMessage(read)
		if err != nil {
			t.Fatal("expected events not consumed", err)
		}
		var event domain.CloudEvent
		if json.Unmarshal(message.Value, &event) != nil {
			t.Fatal("broker event invalid JSON")
		}
		if count := ids[event.ID]; count > 0 {
			if string(message.Key) != event.Data.ProfileID || !domain.ValidTraceParent(event.TraceParent) {
				t.Fatal("broker key/trace invalid")
			}
			ids[event.ID]--
			result = append(result, event)
		}
	}
	return result
}

func TestKafkaRelayUnavailableThenAck(t *testing.T) {
	if os.Getenv("TRACEABILITY_TEST_KAFKA_BROKERS") == "" {
		t.Skip("NOT_RUN: isolated Kafka environment required")
	}
	ctx, store, p, queue := pendingSetup(t)
	brokers := isolatedBroker(t, ctx)
	unavailable, err := broker.NewKafka("sandbox", []string{"127.0.0.1:1"})
	if err != nil {
		t.Fatal(err)
	}
	relay := &application.Relay{Queue: queue, Publisher: unavailable, PublishTimeout: 500 * time.Millisecond}
	processed, err := relay.Tick(ctx)
	if err == nil || !processed {
		t.Fatal("unavailable broker reported success")
	}
	_ = unavailable.Close()
	stats, err := queue.Stats(ctx)
	if err != nil || stats.Pending != 1 || stats.Failed != 0 {
		t.Fatal("broker failure lost pending event", err)
	}
	if _, err = store.Public(ctx, p.QRCode); err != nil {
		t.Fatal("public read depended on broker", err)
	}
	if _, err = store.DB.Exec(ctx, `UPDATE outbox_events SET retry_at=clock_timestamp()`); err != nil {
		t.Fatal(err)
	}
	publisher, err := broker.NewKafka("sandbox", brokers)
	if err != nil {
		t.Fatal(err)
	}
	defer publisher.Close()
	relay.Publisher = publisher
	relay.PublishTimeout = 5 * time.Second
	if processed, err = relay.Tick(ctx); err != nil || !processed {
		t.Fatal("broker did not acknowledge delivery", err)
	}
	var id, status string
	var attempts int
	if err = store.DB.QueryRow(ctx, `SELECT id::text,status,attempts FROM outbox_events`).Scan(&id, &status, &attempts); err != nil || status != "PUBLISHED" || attempts != 2 {
		t.Fatal("ack not persisted", err)
	}
	events := readMatchingEvents(t, ctx, brokers, map[string]int{id: 1})
	if events[0].Data.AggregateVersion != 4 {
		t.Fatal("wrong aggregate version delivered")
	}
}

func TestKafkaCrashWindowReplaysSameEvent(t *testing.T) {
	if os.Getenv("TRACEABILITY_TEST_KAFKA_BROKERS") == "" {
		t.Skip("NOT_RUN: isolated Kafka environment required")
	}
	ctx, store, _, queue := pendingSetup(t)
	brokers := isolatedBroker(t, ctx)
	publisher, err := broker.NewKafka("sandbox", brokers)
	if err != nil {
		t.Fatal(err)
	}
	defer publisher.Close()
	first, found, err := queue.Claim(ctx)
	if err != nil || !found {
		t.Fatal(err)
	}
	if err = publisher.Publish(ctx, first); err != nil {
		t.Fatal(err)
	}
	// Simulate process death precisely after broker ack and before local Ack.
	// Advance only the lease deadline in the disposable schema, without changing
	// immutable event payload or ID; a second process can then reclaim it.
	if _, err = store.DB.Exec(ctx, `UPDATE outbox_events SET lease_until=clock_timestamp()-interval '1 second' WHERE id=$1`, first.ID); err != nil {
		t.Fatal(err)
	}
	second, found, err := queue.Claim(ctx)
	if err != nil || !found || first.ID != second.ID || string(first.Payload) != string(second.Payload) {
		t.Fatal("replay changed identity/payload", err)
	}
	if err = publisher.Publish(ctx, second); err != nil {
		t.Fatal(err)
	}
	if err = queue.Ack(ctx, second); err != nil {
		t.Fatal(err)
	}
	if err = queue.Ack(ctx, first); !errors.Is(err, domain.ErrLeaseLost) {
		t.Fatal("dead owner acknowledged replay")
	}
	events := readMatchingEvents(t, ctx, brokers, map[string]int{first.ID: 2})
	if events[0].ID != events[1].ID || events[0].Data.AggregateVersion != events[1].Data.AggregateVersion {
		t.Fatal("duplicate event not stable")
	}
}

func TestKafkaSameProfileOrder(t *testing.T) {
	if os.Getenv("TRACEABILITY_TEST_KAFKA_BROKERS") == "" {
		t.Skip("NOT_RUN: isolated Kafka environment required")
	}
	ctx, store, p, queue := pendingSetup(t)
	brokers := isolatedBroker(t, ctx)
	if _, err := store.SetStatus(ctx, reviewMeta("suspend"), p.ID, "SUSPENDED", domain.RevisionCommand{ExpectedVersion: 4, Reason: "Test"}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.DB.Exec(ctx, `UPDATE outbox_events SET retry_at=clock_timestamp()`); err != nil {
		t.Fatal(err)
	}
	publisher, err := broker.NewKafka("sandbox", brokers)
	if err != nil {
		t.Fatal(err)
	}
	defer publisher.Close()
	relay := application.Relay{Queue: queue, Publisher: publisher, PublishTimeout: 5 * time.Second}
	for range 2 {
		if found, err := relay.Tick(ctx); err != nil || !found {
			t.Fatal("relay order failed", err)
		}
	}
	rows, err := store.DB.Query(ctx, `SELECT id::text FROM outbox_events`)
	if err != nil {
		t.Fatal(err)
	}
	ids := map[string]int{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		ids[id] = 1
	}
	rows.Close()
	events := readMatchingEvents(t, ctx, brokers, ids)
	if len(events) != 2 || events[0].Data.AggregateVersion != 4 || events[1].Data.AggregateVersion != 5 {
		t.Fatal("same-profile events reordered")
	}
}
