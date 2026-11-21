package kafka

import (
	"context"
	"dut-pbl6/inventory-service/internal/testdb"
	"errors"
	segmentio "github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/require"
	"os"
	"strings"
	"testing"
	"time"
)

// Requires an isolated broker: uses the actual production topic/group names.
func TestProcurementRealKafka(t *testing.T) {
	raw := os.Getenv("INVENTORY_TEST_KAFKA_BROKERS")
	if raw == "" {
		t.Skip("requires isolated INVENTORY_TEST_KAFKA_BROKERS")
	}
	brokers := strings.Split(raw, ",")
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	var conn *segmentio.Conn
	var err error
	require.Eventually(t, func() bool { conn, err = segmentio.DialContext(ctx, "tcp", brokers[0]); return err == nil }, 20*time.Second, 200*time.Millisecond)
	defer conn.Close()
	for _, topic := range []string{procurementTopic, procurementDLQ} {
		err = conn.CreateTopics(segmentio.TopicConfig{Topic: topic, NumPartitions: 1, ReplicationFactor: 1})
		require.True(t, err == nil || errors.Is(err, segmentio.TopicAlreadyExists), "%v", err)
	}
	db := testdb.Open(t)
	provision(t, db)
	store := receiptStore(t, db)
	consumerCtx, stop := context.WithCancel(ctx)
	defer stop()
	done := make(chan struct{})
	go func() {
		defer close(done)
		RunProcurementConsumer(consumerCtx, brokers, db, store.warehouses, store.units)
	}()
	defer func() {
		stop()
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			t.Error("consumer did not stop")
		}
	}()
	w := &segmentio.Writer{Addr: segmentio.TCP(brokers...), Topic: procurementTopic, RequiredAcks: segmentio.RequireAll, BatchTimeout: 10 * time.Millisecond}
	defer w.Close()
	msg := receiptMessage(t)
	// LastOffset is deliberate: repeatedly publishing the same immutable event also
	// verifies inbox dedupe while the new group finishes assignment.
	require.Eventually(t, func() bool {
		err = w.WriteMessages(ctx, segmentio.Message{Key: msg.Key, Value: msg.Value})
		if err != nil {
			return false
		}
		var qty int
		return db.QueryRow("SELECT physical_qty FROM inventory_items WHERE sku='SKU'").Scan(&qty) == nil && qty == 2500
	}, 25*time.Second, 300*time.Millisecond)
	client := &segmentio.Client{Addr: segmentio.TCP(brokers...), Timeout: 3 * time.Second}
	require.Eventually(t, func() bool { lag, err := readProcurementLag(ctx, client); return err == nil && lag == 0 }, 10*time.Second, 200*time.Millisecond)
	dlqLeader, err := segmentio.DialLeader(ctx, "tcp", brokers[0], procurementDLQ, 0)
	require.NoError(t, err)
	endOffset, err := dlqLeader.ReadLastOffset()
	require.NoError(t, err)
	dlqLeader.Close()
	invalid := changed(t, msg, "data.receipt_id", `null`)
	require.NoError(t, w.WriteMessages(ctx, segmentio.Message{Key: invalid.Key, Value: invalid.Value}))
	dlq := segmentio.NewReader(segmentio.ReaderConfig{Brokers: brokers, Topic: procurementDLQ, Partition: 0, MaxWait: time.Second})
	defer dlq.Close()
	require.NoError(t, dlq.SetOffset(endOffset))
	event, err := dlq.ReadMessage(ctx)
	require.NoError(t, err)
	require.JSONEq(t, string(invalid.Value), string(event.Value))
	headers := map[string]string{}
	for _, h := range event.Headers {
		headers[h.Key] = string(h.Value)
	}
	require.Equal(t, procurementTopic, headers["x-original-topic"])
	require.Equal(t, "1", headers["x-attempts"])
	require.NotEmpty(t, headers["x-exception-message"])
	var qty int
	require.NoError(t, db.QueryRow("SELECT physical_qty FROM inventory_items WHERE sku='SKU'").Scan(&qty))
	require.Equal(t, 2500, qty)
}
