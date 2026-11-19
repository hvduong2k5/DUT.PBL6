package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	d "github.com/omamx/fulfillment-service/internal/domain"
	"github.com/segmentio/kafka-go"
	"os"
	"strings"
	"testing"
	"time"
)

func TestInboxDedupCollisionAndInvalidQuarantine(t *testing.T) {
	s, in := fixture(t)
	raw, e := os.ReadFile("../../../packages/events/examples/order/v3/fulfillment_ready.json")
	if e != nil {
		// Build the exact ready contract; no assumption that every checkout contains
		// the integration branch's example files.
		raw = []byte(`{}`)
	}
	var event map[string]any
	_ = json.Unmarshal(raw, &event)
	ordinal := int64(0)
	ready := d.Ready{Spec: "1.0", ID: uuid.NewString(), Source: "https://omama.vn/services/order-service", Type: "vn.omama.order.fulfillment_ready.v3", Subject: "order:" + in.Inputs.OrderID, Time: time.Now().UTC(), Content: "application/json"}
	ready.Data.Order = in.Inputs.OrderID
	ready.Data.Version = 8
	ready.Data.Ordinal = &ordinal
	ready.Data.Status = "PAID"
	ready.Data.Payment = "CONFIRMED"
	ready.Data.Stock = "COMMITTED"
	ready.Data.Digest = in.Inputs.Digest
	ready.Data.Warehouse = in.Inputs.Warehouse
	ready.Data.Reservation = in.Inputs.ReservationID
	ready.Data.StockVersion = 3
	ready.Data.Generation = 1
	ready.Data.Policy = in.Inputs.Policy.Version
	b, _ := json.Marshal(ready)
	ctx := context.Background()
	for n := 0; n < 2; n++ {
		if e = s.IngestReady(ctx, b, []byte(ready.Data.Order), "order.events.v3", 0, int64(n)); e != nil {
			t.Fatal(e)
		}
	}
	var count int
	if e = pool.QueryRow(ctx, `SELECT count(*) FROM fulfillment.jobs WHERE order_id=$1 AND kind='READY_ADMISSION'`, in.Inputs.OrderID).Scan(&count); e != nil || count != 1 {
		t.Fatal("duplicate jobs", e, count)
	}
	ready.Data.Digest = d.Hash("changed")
	changed, _ := json.Marshal(ready)
	if e = s.IngestReady(ctx, changed, []byte(ready.Data.Order), "order.events.v3", 0, 3); e != nil {
		t.Fatal(e)
	}
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM fulfillment.inbox_conflicts WHERE event_id=$1`, ready.ID).Scan(&count)
	if count != 1 {
		t.Fatal("collision not retained")
	}
	ready.ID = uuid.NewString()
	ready.Source = "https://forged.invalid"
	changed, _ = json.Marshal(ready)
	if e = s.IngestReady(ctx, changed, []byte(ready.Data.Order), "order.events.v3", 0, 4); e != nil {
		t.Fatal(e)
	}
	var disposition string
	_ = pool.QueryRow(ctx, `SELECT disposition FROM fulfillment.inbox WHERE event_id=$1`, ready.ID).Scan(&disposition)
	if disposition != "QUARANTINED" {
		t.Fatal("invalid source not quarantined")
	}
}
func TestKafkaOutboxReplayAndLeaseFencing(t *testing.T) {
	s, in := fixture(t)
	task := mustTask(t, s, in)
	broker := os.Getenv("FULFILLMENT_TEST_KAFKA")
	if broker == "" {
		t.Skip("real Kafka not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	topic := "fulfillment.test." + strings.ReplaceAll(uuid.NewString(), "-", "")
	client := &kafka.Client{Addr: kafka.TCP(broker), Timeout: 10 * time.Second}
	created, e := client.CreateTopics(ctx, &kafka.CreateTopicsRequest{Topics: []kafka.TopicConfig{{Topic: topic, NumPartitions: 1, ReplicationFactor: 1}}})
	if e != nil {
		t.Fatal(e)
	}
	for _, err := range created.Errors {
		if err != nil {
			t.Fatal(err)
		}
	}

	var b []byte
	if e = pool.QueryRow(ctx, `SELECT envelope FROM fulfillment.outbox WHERE stream_id=$1 AND topic='fulfillment.task.events.v1'`, task.OrderID).Scan(&b); e != nil {
		t.Fatal(e)
	}
	id := uuid.NewString()
	var envelope map[string]any
	_ = json.Unmarshal(b, &envelope)
	envelope["id"] = id
	b, _ = json.Marshal(envelope)
	_, e = pool.Exec(ctx, `INSERT INTO fulfillment.outbox(event_id,stream_id,stream_version,ordinal,topic,partition_key,envelope,occurred_at,next_run_at) VALUES($1,$2,1,0,$3,$4,$5,clock_timestamp()-interval '1 day',clock_timestamp()-interval '1 day')`, id, "test:"+id, topic, task.OrderID, b)
	if e != nil {
		t.Fatal(e)
	}
	first, ok, e := s.ClaimDelivery(ctx, "owner-a")
	if e != nil || !ok || first.ID != id {
		t.Fatal("claim", first.ID, e)
	}
	_, e = pool.Exec(ctx, `UPDATE fulfillment.outbox SET lease_until=clock_timestamp()-interval '1 second' WHERE event_id=$1`, id)
	if e != nil {
		t.Fatal(e)
	}
	second, ok, e := s.ClaimDelivery(ctx, "owner-b")
	if e != nil || !ok || second.ID != id || second.Epoch <= first.Epoch || !bytes.Equal(first.Payload, second.Payload) {
		t.Fatal("replay changed immutable payload", e)
	}
	writer := &kafka.Writer{Addr: kafka.TCP(broker), RequiredAcks: kafka.RequireAll}
	defer writer.Close()
	for n := 0; n < 2; n++ {
		if e = writer.WriteMessages(ctx, kafka.Message{Topic: topic, Key: []byte(second.Key), Value: second.Payload}); e != nil {
			t.Fatal(e)
		}
	}
	reader := kafka.NewReader(kafka.ReaderConfig{Brokers: []string{broker}, Topic: topic, Partition: 0})
	defer reader.Close()
	for n := 0; n < 2; n++ {
		msg, e := reader.ReadMessage(ctx)
		if e != nil || !bytes.Equal(msg.Value, second.Payload) || string(msg.Key) != task.OrderID {
			t.Fatal("actual broker payload differs", e)
		}
	}
	if e = s.DeliveryResult(ctx, "owner-a", first, true); e != nil {
		t.Fatal(e)
	}
	var at *time.Time
	_ = pool.QueryRow(ctx, `SELECT published_at FROM fulfillment.outbox WHERE event_id=$1`, id).Scan(&at)
	if at != nil {
		t.Fatal("stale publisher acknowledged current lease")
	}
	if e = s.DeliveryResult(ctx, "owner-b", second, true); e != nil {
		t.Fatal(e)
	}
	_ = os.MkdirAll("results", 0755)
	_ = os.WriteFile("results/accepted-event.json", second.Payload, 0644)
}
func TestSLARecordedOnceWithoutChangingTask(t *testing.T) {
	s, in := fixture(t)
	due := time.Now().Add(-time.Hour)
	in.Inputs.Due = &due
	task := mustTask(t, s, in)
	ctx := context.Background()
	for n := 0; n < 2; n++ {
		if e := s.SLA(ctx); e != nil {
			t.Fatal(e)
		}
	}
	var count int
	e := pool.QueryRow(ctx, `SELECT count(*) FROM fulfillment.jobs WHERE dedupe_key=$1`, fmt.Sprintf("sla:%s:1:breached", task.ID)).Scan(&count)
	if e != nil || count != 1 {
		t.Fatal("SLA alert not deduplicated", e, count)
	}
	current, e := s.Task(ctx, task.ID)
	if e != nil || current.State != task.State || current.Version != task.Version {
		t.Fatal("SLA mutated task", e)
	}
}

func TestOutboxQuarantinesRepeatedFailuresAndAllowsReplay(t *testing.T) {
	s, _ := fixture(t)
	ctx := context.Background()
	id := uuid.NewString()
	if _, err := pool.Exec(ctx, `INSERT INTO fulfillment.outbox
		(event_id,stream_id,stream_version,ordinal,topic,partition_key,envelope,occurred_at,next_run_at)
		VALUES ($1,$2,1,0,'audit.events.v1',$2,'{}',now()-interval '1 day',now())`, id, id); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 10; i++ {
		delivery, found, err := s.ClaimDelivery(ctx, "outbox-test")
		if err != nil || !found || delivery.ID != id {
			t.Fatalf("claim %d: %v %v %s", i, err, found, delivery.ID)
		}
		if err = s.DeliveryResult(ctx, "outbox-test", delivery, false); err != nil {
			t.Fatal(err)
		}
		if _, err = pool.Exec(ctx, "UPDATE fulfillment.outbox SET next_run_at=now() WHERE event_id=$1", id); err != nil {
			t.Fatal(err)
		}
	}
	var dead bool
	var attempts int
	if err := pool.QueryRow(ctx, "SELECT dead_lettered_at IS NOT NULL,attempts FROM fulfillment.outbox WHERE event_id=$1", id).Scan(&dead, &attempts); err != nil || !dead || attempts != 10 {
		t.Fatalf("dead-letter state: %v %d %v", dead, attempts, err)
	}
	if delivery, found, err := s.ClaimDelivery(ctx, "outbox-test"); err != nil || (found && delivery.ID == id) {
		t.Fatalf("dead letter was eligible again: %v %v %s", err, found, delivery.ID)
	}
	if _, err := pool.Exec(ctx, `UPDATE fulfillment.outbox SET dead_lettered_at=NULL,
		last_error_code=NULL,consecutive_failures=0,next_run_at=now() WHERE event_id=$1`, id); err != nil {
		t.Fatal(err)
	}
	delivery, found, err := s.ClaimDelivery(ctx, "outbox-test")
	if err != nil || !found || delivery.ID != id {
		t.Fatalf("replay: %v %v %s", err, found, delivery.ID)
	}
	if err := s.DeliveryResult(ctx, "outbox-test", delivery, true); err != nil {
		t.Fatal(err)
	}
}
