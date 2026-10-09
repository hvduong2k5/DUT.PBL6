package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/omamx/order-service/internal/domain"
	"github.com/segmentio/kafka-go"
	"testing"
	"time"
)

func TestV10OutboxCrashWindowPublishesStableEventID(t *testing.T) {
	h := newHarness(t)
	o := h.place("VIETQR")
	if h.Config.Kafka == "" {
		t.Fatal("Kafka is required for this release gate")
	}
	reader := kafka.NewReader(kafka.ReaderConfig{Brokers: []string{h.Config.Kafka}, Topic: "order.events.v2", GroupID: "order-test-facts-" + uuid.NewString(), MinBytes: 1, MaxBytes: 1 << 20, StartOffset: kafka.FirstOffset})
	defer reader.Close()
	if worked, e := h.S.PublishOne(h.ctx); e != nil || !worked {
		t.Fatalf("publish %v %v", worked, e)
	}
	var eventID string
	if e := h.S.DB.QueryRow(h.ctx, "SELECT id FROM outbox WHERE aggregate_id=$1 AND event_type='vn.omama.order.placed.v2'", o.ID).Scan(&eventID); e != nil {
		t.Fatal(e)
	}
	h.exec("UPDATE outbox SET published_at=NULL,lease_owner=NULL,lease_until=NULL WHERE id=$1", eventID)
	if worked, e := h.S.PublishOne(h.ctx); e != nil || !worked {
		t.Fatal("crash replay did not publish")
	}
	ctx, cancel := context.WithTimeout(h.ctx, 20*time.Second)
	defer cancel()
	seen := 0
	for seen < 2 {
		m, e := reader.ReadMessage(ctx)
		if e != nil {
			t.Fatal(e)
		}
		var envelope struct {
			ID string `json:"id"`
		}
		if e = json.Unmarshal(m.Value, &envelope); e != nil {
			t.Fatal(e)
		}
		if envelope.ID == eventID {
			seen++
			if string(m.Key) != o.ID {
				t.Fatal("partition key not aggregate")
			}
			if bytes.Contains(m.Value, []byte("Synthetic customer")) || bytes.Contains(m.Value, []byte("+84905")) {
				t.Fatal("PII leaked into event")
			}
		}
	}
	if h.count("SELECT count(*) FROM outbox WHERE id=$1", eventID) != 1 {
		t.Fatal("republish generated another ID")
	}
}
func TestV11KafkaConsumerDeduplicatesRealSourceMessages(t *testing.T) {
	h := newHarness(t)
	topic := "order-test-source-" + uuid.NewString()
	client := &kafka.Client{Addr: kafka.TCP(h.Config.Kafka), Timeout: 10 * time.Second}
	_, err := client.CreateTopics(h.ctx, &kafka.CreateTopicsRequest{Topics: []kafka.TopicConfig{{Topic: topic, NumPartitions: 1, ReplicationFactor: 1}}})
	if err != nil {
		t.Fatal(err)
	}
	h.S.C.SourceTopic = topic
	t.Cleanup(func() {
		_, _ = client.DeleteTopics(context.Background(), &kafka.DeleteTopicsRequest{Topics: []string{topic}})
	})
	o := h.paid()
	h.acceptTask(o)
	event := h.source(o, "PACKING_ACCEPTED")
	ctx, cancel := context.WithCancel(h.ctx)
	h.cancel = cancel
	h.S.Start(ctx)
	writer := &kafka.Writer{Addr: kafka.TCP(h.Config.Kafka), Balancer: &kafka.Hash{}, RequiredAcks: kafka.RequireAll, AllowAutoTopicCreation: true}
	defer writer.Close()
	b, e := json.Marshal(event)
	if e != nil {
		t.Fatal(e)
	}
	if e = writer.WriteMessages(h.ctx, kafka.Message{Topic: topic, Key: []byte(o.ID), Value: b}, kafka.Message{Topic: topic, Key: []byte(o.ID), Value: b}); e != nil {
		t.Fatal(e)
	}
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		if h.order(o.ID).Status == "PROCESSING" {
			if h.count("SELECT count(*) FROM inbox WHERE event_id=$1 AND status='APPLIED'", event.ID) != 1 {
				t.Fatal("business effect not tied to inbox commit")
			}
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatal("source consumer did not apply event")
}
func TestV12AuditIsAppendOnlyAndTamperEvident(t *testing.T) {
	h := newHarness(t)
	o := h.paid()
	if e := h.S.VerifyAudit(h.ctx); e != nil {
		t.Fatal(e)
	}
	if _, e := h.S.DB.Exec(h.ctx, "UPDATE audit_records SET action='FORGED' WHERE order_id=$1", o.ID); e == nil {
		t.Fatal("audit UPDATE allowed")
	}
	h.exec("ALTER TABLE audit_records DISABLE TRIGGER immutable_audit")
	h.exec("UPDATE audit_records SET action='FORGED' WHERE sequence=(SELECT min(sequence) FROM audit_records)")
	if e := h.S.VerifyAudit(h.ctx); e == nil {
		t.Fatal("privileged tampering not detected")
	}
}
func TestV10CODReceiptDuringCancelIsReconciled(t *testing.T) {
	h := newHarness(t)
	o := h.place("COD")
	h.jobs()
	o = h.order(o.ID)
	_, e := h.S.Cancel(h.ctx, h.Principal, o.ID, uuid.NewString(), o.Version, "test", false)
	if e != nil {
		t.Fatal(e)
	}
	r := h.pay(o, o.Final)
	if e = h.S.RecordReceipt(h.ctx, r); e != nil {
		t.Fatal(e)
	}
	h.jobs()
	if h.order(o.ID).Status != "CANCELLED_BY_USER" || h.count("SELECT count(*) FROM allocations") != 0 || h.count("SELECT count(*) FROM reconciliation_cases") != 1 {
		t.Fatal("money/cancellation race not reconciled")
	}
}
func TestV10SettledCODRequiresCancellationReview(t *testing.T) {
	h := newHarness(t)
	o := h.place("COD")
	h.jobs()
	r := h.pay(o, o.Final)
	if e := h.S.RecordReceipt(h.ctx, r); e != nil {
		t.Fatal(e)
	}
	o = h.order(o.ID)
	_, e := h.S.Cancel(h.ctx, h.Principal, o.ID, uuid.NewString(), o.Version, "test", false)
	requireCode(t, e, "CANCELLATION_REVIEW_REQUIRED")
	if _, e = h.S.Cancel(h.ctx, h.Principal, o.ID, uuid.NewString(), o.Version, "test", true); e != nil {
		t.Fatal(e)
	}
}
func TestV11HoldReplayAndAuthorizedResume(t *testing.T) {
	h := newHarness(t)
	o := h.paid()
	manager := domain.Principal{ID: uuid.NewString(), Role: "SALES_MANAGER", Warehouse: "HUE"}
	key := uuid.NewString()
	if e := h.S.SetHold(h.ctx, manager, o.ID, o.Version, true, key); e != nil {
		t.Fatal(e)
	}
	held := h.order(o.ID)
	if e := h.S.SetHold(h.ctx, manager, o.ID, o.Version, true, key); e != nil || h.order(o.ID).Version != held.Version {
		t.Fatal("hold replay changed version")
	}
	if e := h.S.SetHold(h.ctx, manager, o.ID, held.Version, false, uuid.NewString()); e != nil {
		t.Fatal(e)
	}
	h.jobs()
	current := h.order(o.ID)
	if current.Hold || !current.Ready || current.Generation <= o.Generation {
		t.Fatal("hold release did not renew ready authorization")
	}
}
