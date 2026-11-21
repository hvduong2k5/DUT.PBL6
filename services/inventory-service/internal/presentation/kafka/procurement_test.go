package kafka

import (
	"context"
	"database/sql"
	postgresrepo "dut-pbl6/inventory-service/internal/infrastructure/postgres"
	"dut-pbl6/inventory-service/internal/testdb"
	"encoding/json"
	"errors"
	segmentio "github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/require"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

const warehouseID = "00000000-0000-0000-0000-000000000001"

func receiptMessage(t *testing.T) KafkaMessage {
	t.Helper()
	data, err := os.ReadFile("testdata/goods_received.json")
	require.NoError(t, err)
	return KafkaMessage{Topic: procurementTopic, Key: []byte("SKU"), Value: data, Partition: 1, Offset: 12}
}
func changed(t *testing.T, msg KafkaMessage, path, value string) KafkaMessage {
	t.Helper()
	var e map[string]any
	require.NoError(t, json.Unmarshal(msg.Value, &e))
	if strings.HasPrefix(path, "data.") {
		e["data"].(map[string]any)[strings.TrimPrefix(path, "data.")] = json.RawMessage(value)
	} else {
		e[path] = json.RawMessage(value)
	}
	var err error
	msg.Value, err = json.Marshal(e)
	require.NoError(t, err)
	return msg
}
func TestCanonicalReceiptValidation(t *testing.T) {
	msg := receiptMessage(t)
	e, err := decodeReceipt(msg)
	require.NoError(t, err)
	require.Equal(t, "receipt-event-01", e.ID)
	require.Equal(t, "SUP-HTX-QUANG-THO", e.Data.Supplier)
	n, err := receiptQuantity(e, "GRAM")
	require.NoError(t, err)
	require.Equal(t, int64(2500), n)
	for _, tc := range []struct{ field, value string }{
		{"id", `""`}, {"source", `"not-a-uri"`}, {"subject", `null`}, {"time", `"bad"`}, {"datacontenttype", `"text/plain"`}, {"traceparent", `null`},
		{"data.receipt_id", `null`}, {"data.po_number", `null`}, {"data.supplier_id", `123`}, {"data.received_at", `"bad"`}, {"data.quantity_received", `0`}, {"data.quantity_received", `"2"`},
	} {
		_, err := decodeReceipt(changed(t, msg, tc.field, tc.value))
		require.Error(t, err, tc.field)
	}
	for _, tc := range []struct {
		unit, base, qty string
		want            int64
		bad             bool
	}{
		{"KG", "GRAM", "0.125", 125, false}, {"G", "GRAM", "125", 125, false}, {"KG", "GRAM", "1e-2", 10, false},
		{"UNIT", "UNIT", "2", 2, false}, {"KG", "UNIT", "2.5", 0, true}, {"UNIT", "UNIT", "2.5", 0, true},
		{"KG", "GRAM", "0.0101", 0, true}, {"KG", "GRAM", "2147484", 0, true}, {"KG", "", "1", 0, true},
	} {
		e.Data.Unit = tc.unit
		e.Data.Received = json.RawMessage(tc.qty)
		n, err := receiptQuantity(e, tc.base)
		if tc.bad {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			require.Equal(t, tc.want, n)
		}
	}
}
func receiptStore(t *testing.T, db *sql.DB) procurementStore {
	t.Helper()
	w, err := ProcurementWarehouseMap("WH-HUE=" + warehouseID)
	require.NoError(t, err)
	return procurementStore{db: db, warehouses: w, units: map[string]string{"SKU": "GRAM"}}
}
func provision(t *testing.T, db *sql.DB) {
	_, err := db.Exec("INSERT INTO inventory_items(sku,physical_qty) VALUES('SKU',0)")
	require.NoError(t, err)
}

func TestCanonicalReceiptTransactionAndReplay(t *testing.T) {
	db := testdb.Open(t)
	provision(t, db)
	store := receiptStore(t, db)
	ctx := context.Background()
	msg := receiptMessage(t)
	require.NoError(t, store.handle(ctx, msg))
	require.NoError(t, store.handle(ctx, msg))
	// Same receipt with a new event ID must not receive stock twice.
	require.NoError(t, store.handle(ctx, changed(t, msg, "id", `"new-event-id"`)))
	var qty, sellable, receipts, batches int
	require.NoError(t, db.QueryRow("SELECT physical_qty,inventory_sellable_quantity(sku,now()) FROM inventory_items WHERE sku='SKU'").Scan(&qty, &sellable))
	require.Equal(t, 2500, qty)
	require.Zero(t, sellable)
	require.NoError(t, db.QueryRow("SELECT count(*) FROM procurement_receipts").Scan(&receipts))
	require.Equal(t, 1, receipts)
	require.NoError(t, db.QueryRow("SELECT count(*) FROM batches WHERE status='QUARANTINE' AND mfg_date IS NULL AND exp_date IS NULL").Scan(&batches))
	require.Equal(t, 1, batches)
	var po, supplier, origin, base string
	require.NoError(t, db.QueryRow("SELECT po_number,supplier_code,origin_region,base_unit FROM procurement_receipts").Scan(&po, &supplier, &origin, &base))
	require.Equal(t, "PO-2026-001", po)
	require.Equal(t, "SUP-HTX-QUANG-THO", supplier)
	require.Equal(t, "Quang Dien, Hue", origin)
	require.Equal(t, "GRAM", base)
	repo := postgresrepo.NewPostgresInventoryRepository(db)
	found, err := repo.GetBatchesBySKU(ctx, "SKU")
	require.NoError(t, err)
	require.Len(t, found, 1)
	require.True(t, found[0].MfgDate.IsZero())
	require.ErrorContains(t, store.handle(ctx, changed(t, msg, "data.quantity_received", `3`)), "event ID reused")
	altered := changed(t, msg, "id", `"other-id"`)
	altered = changed(t, altered, "data.quantity_received", `3`)
	require.ErrorContains(t, store.handle(ctx, altered), "receipt ID reused")
	var missing int
	require.NoError(t, db.QueryRow("SELECT count(*) FROM procurement_receipt_inbox WHERE event_id='other-id'").Scan(&missing))
	require.Zero(t, missing)
	_, err = db.Exec("UPDATE batches SET status='ACTIVE'")
	require.Error(t, err) // QC/date gate is enforced by DB.
	var snapshots int
	require.NoError(t, db.QueryRow("SELECT count(*) FROM outbox_events WHERE event_type='inventory.stock-changed.v1' AND (payload->>'inventory_version')::bigint>0").Scan(&snapshots))
	require.Equal(t, 3, snapshots)
	// Inbox cleanup preserves receipt-level dedupe on replay.
	_, err = db.Exec("SELECT inventory_prune_procurement_inbox(now()+interval '1 day')")
	require.NoError(t, err)
	require.NoError(t, store.handle(ctx, msg))
	require.NoError(t, db.QueryRow("SELECT physical_qty FROM inventory_items WHERE sku='SKU'").Scan(&qty))
	require.Equal(t, 2500, qty)
}
func TestConcurrentDistinctReceipts(t *testing.T) {
	db := testdb.Open(t)
	other := testdb.Second(t, db)
	provision(t, db)
	a, b := receiptStore(t, db), receiptStore(t, other)
	msg := receiptMessage(t)
	second := changed(t, msg, "id", `"second"`)
	second = changed(t, second, "data.receipt_id", `"GR-SECOND"`)
	done := make(chan error, 2)
	go func() { done <- a.handle(context.Background(), msg) }()
	go func() { done <- b.handle(context.Background(), second) }()
	for range 2 {
		require.NoError(t, <-done)
	}
	var qty, count int
	require.NoError(t, db.QueryRow("SELECT physical_qty FROM inventory_items WHERE sku='SKU'").Scan(&qty))
	require.Equal(t, 5000, qty)
	require.NoError(t, db.QueryRow("SELECT count(DISTINCT batch_code) FROM batches").Scan(&count))
	require.Equal(t, 2, count)
}
func TestReceiptRejectsMismatchedWarehouseAndOverflow(t *testing.T) {
	for _, setup := range []string{"UPDATE inventory_items SET warehouse_id=gen_random_uuid()", "UPDATE inventory_items SET physical_qty=2147483647"} {
		db := testdb.Open(t)
		provision(t, db)
		_, err := db.Exec(setup)
		require.NoError(t, err)
		require.Error(t, receiptStore(t, db).handle(context.Background(), receiptMessage(t)))
		var count int
		require.NoError(t, db.QueryRow("SELECT count(*) FROM procurement_receipt_inbox").Scan(&count))
		require.Zero(t, count)
	}
}

type procurementReaderStub struct {
	MessageReader
	commits int
	err     error
}

func (r *procurementReaderStub) CommitMessages(context.Context, ...KafkaMessage) error {
	r.commits++
	return r.err
}

type procurementDLTStub struct {
	writes   int
	err      error
	messages []segmentio.Message
}

func (w *procurementDLTStub) WriteMessages(_ context.Context, msgs ...segmentio.Message) error {
	w.writes += len(msgs)
	w.messages = append(w.messages, msgs...)
	return w.err
}
func TestProcurementRetryOffsetAndPanicPolicy(t *testing.T) {
	for _, tc := range []struct {
		name                      string
		failure, errorDLQ         error
		panic                     bool
		attempts, commits, writes int
	}{
		{"success", nil, nil, false, 1, 1, 0}, {"DB exhausted", errors.New("DB unavailable"), nil, false, 5, 1, 1},
		{"invalid", invalidReceipt("bad"), nil, false, 1, 1, 1}, {"DLQ unavailable", invalidReceipt("bad"), errors.New("broker down"), false, 1, 0, 1},
		{"panic", nil, nil, true, 5, 1, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := &procurementReaderStub{}
			w := &procurementDLTStub{err: tc.errorDLQ}
			attempts := 0
			err := processWithRetry(context.Background(), r, w, func(context.Context, KafkaMessage) error {
				attempts++
				if tc.panic {
					panic("test panic")
				}
				return tc.failure
			}, KafkaMessage{Topic: procurementTopic}, 0)
			if tc.errorDLQ != nil {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tc.attempts, attempts)
			require.Equal(t, tc.commits, r.commits)
			require.Equal(t, tc.writes, w.writes)
			if w.writes > 0 {
				require.Equal(t, procurementDLQ, w.messages[0].Topic)
				require.Equal(t, "x-exception-message", w.messages[0].Headers[0].Key)
				if tc.panic {
					require.NotEmpty(t, w.messages[0].Headers[1].Value)
				}
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r := &procurementReaderStub{}
	w := &procurementDLTStub{}
	require.Error(t, processWithRetry(ctx, r, w, func(context.Context, KafkaMessage) error { return errors.New("DB") }, KafkaMessage{}, time.Second))
	require.Zero(t, r.commits)
}
func TestProcurementReadiness(t *testing.T) {
	procurementHealthy.Store(false)
	r := httptest.NewRecorder()
	ProcurementMonitor(true).ServeHTTP(r, httptest.NewRequest("GET", "/readyz", nil))
	require.Equal(t, 503, r.Code)
	var failure map[string]string
	require.NoError(t, json.Unmarshal(r.Body.Bytes(), &failure))
	require.Equal(t, "PROCUREMENT_CONSUMER_UNHEALTHY", failure["code"])
	require.Equal(t, r.Header().Get("X-Request-Id"), failure["requestId"])
	require.NotEmpty(t, failure["requestId"])
	r = httptest.NewRecorder()
	ProcurementMonitor(false).ServeHTTP(r, httptest.NewRequest("GET", "/readyz", nil))
	require.Equal(t, 200, r.Code)
}

func TestReceiptBaseUnitCannotChangeAfterReceiving(t *testing.T) {
	db := testdb.Open(t)
	provision(t, db)
	ctx := context.Background()
	store := receiptStore(t, db)
	msg := receiptMessage(t)
	require.NoError(t, store.handle(ctx, msg))
	store.units["SKU"] = "UNIT"
	next := changed(t, msg, "id", `"unit-changed"`)
	next = changed(t, next, "data.receipt_id", `"GR-NEW"`)
	next = changed(t, next, "data.unit", `"UNIT"`)
	next = changed(t, next, "data.quantity_received", `2`)
	require.ErrorContains(t, store.handle(ctx, next), "persisted receiving unit")
	var qty int
	require.NoError(t, db.QueryRow("SELECT physical_qty FROM inventory_items WHERE sku='SKU'").Scan(&qty))
	require.Equal(t, 2500, qty)
}
