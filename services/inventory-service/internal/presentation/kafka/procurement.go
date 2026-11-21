package kafka

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"math/big"
	"net/url"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	segmentio "github.com/segmentio/kafka-go"
)

const procurementTopic = "procurement.events.v1"
const procurementDLQ = "procurement.events.dlq.v1"
const goodsReceivedType = "vn.omama.procurement.goods.received.v1"

type invalidReceipt string

func (e invalidReceipt) Error() string { return string(e) }

type goodsReceipt struct {
	ID          string      `json:"id"`
	Source      string      `json:"source"`
	Type        string      `json:"type"`
	Version     string      `json:"specversion"`
	Subject     string      `json:"subject"`
	Time        string      `json:"time"`
	ContentType string      `json:"datacontenttype"`
	Traceparent string      `json:"traceparent"`
	Data        receiptData `json:"data"`
}
type receiptData struct {
	Receipt      string          `json:"receipt_id"`
	PO           string          `json:"po_number"`
	Supplier     string          `json:"supplier_id"`
	SupplierName string          `json:"supplier_name,omitempty"`
	SKU          string          `json:"sku_code"`
	Received     json.RawMessage `json:"quantity_received"`
	Unit         string          `json:"unit"`
	Warehouse    string          `json:"warehouse_id"`
	Origin       string          `json:"raw_material_origin_region,omitempty"`
	ReceivedAt   string          `json:"received_at"`
}

func identifier(s string, max int) bool {
	return strings.TrimSpace(s) != "" && utf8.RuneCountInString(s) <= max && !strings.ContainsRune(s, 0)
}

// Decode only the canonical contract. Extra lot/date/accepted fields are never
// interpreted as authority to release quarantine or change the received amount.
func decodeReceipt(msg KafkaMessage) (goodsReceipt, error) {
	var e goodsReceipt
	if len(msg.Value) > 1<<20 || json.Unmarshal(msg.Value, &e) != nil {
		return e, invalidReceipt("invalid GoodsReceived JSON")
	}
	if e.Type != goodsReceivedType || e.Version != "1.0" || e.ContentType != "application/json" {
		return e, invalidReceipt("invalid GoodsReceived CloudEvent type/version/content type")
	}
	u, err := url.Parse(e.Source)
	if err != nil || !u.IsAbs() || !identifier(e.Source, 1000) || !identifier(e.ID, 256) || !identifier(e.Subject, 1000) || !identifier(e.Traceparent, 256) {
		return e, invalidReceipt("invalid CloudEvent envelope")
	}
	if _, err = time.Parse(time.RFC3339Nano, e.Time); err != nil {
		return e, invalidReceipt("invalid CloudEvent time")
	}
	e.Data.Unit = strings.ToUpper(strings.TrimSpace(e.Data.Unit))
	d := e.Data
	if !identifier(d.Receipt, 256) || !identifier(d.PO, 256) || !identifier(d.Supplier, 256) || !identifier(d.SKU, 50) || !identifier(d.Warehouse, 256) || string(msg.Key) != d.SKU {
		return e, invalidReceipt("invalid receipt identifiers or partition key")
	}
	if strings.ContainsRune(d.SupplierName+d.Origin, 0) || len(d.SupplierName) > 2000 || len(d.Origin) > 2000 {
		return e, invalidReceipt("invalid receipt evidence")
	}
	if _, err = time.Parse(time.RFC3339Nano, d.ReceivedAt); err != nil {
		return e, invalidReceipt("invalid receipt received_at")
	}
	q, ok := decimal(d.Received)
	if !ok || q.Cmp(big.NewRat(1, 100)) < 0 {
		return e, invalidReceipt("quantity_received must be at least 0.01")
	}
	return e, nil
}

func decimal(n json.RawMessage) (*big.Rat, bool) {
	s := string(n)
	if len(s) == 0 || s[0] == '"' || !json.Valid(n) {
		return nil, false
	}
	if len(s) > 64 {
		return nil, false
	}
	// big.Rat understands JSON decimal/exponent notation without float rounding.
	if i := strings.IndexAny(s, "eE"); i >= 0 {
		exponent, err := strconv.Atoi(s[i+1:])
		if err != nil || exponent < -9 || exponent > 9 {
			return nil, false
		}
	}
	q, ok := new(big.Rat).SetString(s)
	return q, ok && q.Sign() > 0
}
func receiptQuantity(e goodsReceipt, base string) (int64, error) {
	q, ok := decimal(e.Data.Received)
	if !ok {
		return 0, invalidReceipt("invalid received quantity")
	}
	switch {
	case base == "GRAM" && e.Data.Unit == "KG":
		q.Mul(q, big.NewRat(1000, 1))
	case base == "GRAM" && (e.Data.Unit == "G" || e.Data.Unit == "GRAM"):
	case base == "UNIT" && (e.Data.Unit == "UNIT" || e.Data.Unit == "PIECE"):
	default:
		return 0, invalidReceipt("SKU base unit is missing or incompatible with receipt unit")
	}
	if !q.IsInt() || !q.Num().IsInt64() || q.Num().Int64() > math.MaxInt32 {
		return 0, invalidReceipt("converted quantity must fit an Inventory integer exactly")
	}
	return q.Num().Int64(), nil
}
func payloadHash(v any) string {
	data, _ := json.Marshal(v)
	return fmt.Sprintf("%x", sha256.Sum256(data))
}

type procurementStore struct {
	db         *sql.DB
	warehouses map[string]uuid.UUID
	units      map[string]string
}

func (s procurementStore) handle(ctx context.Context, msg KafkaMessage) error {
	e, err := decodeReceipt(msg)
	if err != nil {
		return err
	}
	count, err := receiptQuantity(e, s.units[e.Data.SKU])
	if err != nil {
		return err
	}
	d := e.Data
	warehouse, ok := s.warehouses[d.Warehouse]
	if !ok {
		return invalidReceipt("unmapped Procurement warehouse")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	hash := canonicalHash(e)
	legacyHash := payloadHash(e)
	result, err := tx.ExecContext(ctx, `INSERT INTO procurement_event_ledger(event_id,source,payload_hash) VALUES($1,$2,$3) ON CONFLICT DO NOTHING`, e.ID, e.Source, hash)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		var original string
		var hashVersion int
		if err = tx.QueryRowContext(ctx, `SELECT payload_hash,hash_version FROM procurement_event_ledger WHERE source=$1 AND event_id=$2`, e.Source, e.ID).Scan(&original, &hashVersion); err != nil {
			return err
		}
		if hashVersion == 1 && original != legacyHash {
			var oldPayload []byte
			if lookupErr := tx.QueryRowContext(ctx, `SELECT payload FROM procurement_receipts WHERE source=$1 AND event_id=$2`, e.Source, e.ID).Scan(&oldPayload); lookupErr == nil {
				if prior, decodeErr := decodeReceipt(KafkaMessage{Key: msg.Key, Value: oldPayload}); decodeErr == nil {
					original = canonicalHash(prior)
					hashVersion = 2
				}
			}
		}
		if original != hash && !(hashVersion == 1 && original == legacyHash) {
			return invalidReceipt("event ID reused with different payload")
		}
		return tx.Commit()
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO procurement_receipt_inbox(event_id,source,topic,partition_id,offset_id,payload_hash) VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT DO NOTHING`, e.ID, e.Source, msg.Topic, msg.Partition, msg.Offset, hash); err != nil {
		return err
	}
	var actual uuid.UUID
	var stock int64
	err = tx.QueryRowContext(ctx, `SELECT warehouse_id, physical_qty FROM inventory_items WHERE sku=$1 FOR UPDATE`, d.SKU).Scan(&actual, &stock)
	if errors.Is(err, sql.ErrNoRows) {
		return invalidReceipt("SKU is not provisioned in Inventory")
	}
	if err != nil {
		return err
	}
	if actual != warehouse {
		return invalidReceipt("SKU warehouse does not match mapped warehouse")
	}
	dataHash := canonicalHash(d)
	var existing string
	err = tx.QueryRowContext(ctx, `SELECT data_hash FROM procurement_receipts WHERE source=$1 AND receipt_id=$2 AND sku=$3`, e.Source, d.Receipt, d.SKU).Scan(&existing)
	if err == nil {
		if existing != dataHash && existing != payloadHash(d) {
			var previous []byte
			if err = tx.QueryRowContext(ctx, `SELECT payload FROM procurement_receipts WHERE source=$1 AND receipt_id=$2 AND sku=$3`, e.Source, d.Receipt, d.SKU).Scan(&previous); err != nil {
				return err
			}
			prior, decodeErr := decodeReceipt(KafkaMessage{Key: msg.Key, Value: previous})
			if decodeErr != nil || canonicalHash(prior.Data) != dataHash {
				return invalidReceipt("receipt ID reused with different receipt data")
			}
		}
		return tx.Commit()
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if stock+count > math.MaxInt32 {
		return invalidReceipt("receipt would overflow inventory quantity")
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO inventory_procurement_sku_units(sku,base_unit) VALUES($1,$2) ON CONFLICT DO NOTHING`, d.SKU, s.units[d.SKU]); err != nil {
		return err
	}
	var boundUnit string
	if err = tx.QueryRowContext(ctx, `SELECT base_unit FROM inventory_procurement_sku_units WHERE sku=$1`, d.SKU).Scan(&boundUnit); err != nil {
		return err
	}
	if boundUnit != s.units[d.SKU] {
		return invalidReceipt("SKU base unit differs from its persisted receiving unit")
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO procurement_supplier_mapping(supplier_code,supplier_id) VALUES($1,$2) ON CONFLICT DO NOTHING`, d.Supplier, uuid.NewString()); err != nil {
		return err
	}
	var supplier uuid.UUID
	if err = tx.QueryRowContext(ctx, `SELECT supplier_id FROM procurement_supplier_mapping WHERE supplier_code=$1`, d.Supplier).Scan(&supplier); err != nil {
		return err
	}
	// One internal batch per immutable receipt line: never merge supplier lots.
	batch := uuid.NewString()
	batchCode := "GR-" + uuid.NewSHA1(uuid.NameSpaceURL, []byte(e.Source+"\x00"+d.Receipt+"\x00"+d.SKU)).String()
	if _, err = tx.ExecContext(ctx, `INSERT INTO batches(id,sku,batch_code,supplier_id,physical_qty,status) VALUES($1,$2,$3,$4,$5,'QUARANTINE')`, batch, d.SKU, batchCode, supplier, count); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO procurement_receipts(source,receipt_id,sku,event_id,po_number,supplier_code,supplier_name,origin_region,warehouse_code,received_at,unit,base_unit,quantity_received,stock_quantity,batch_id,data_hash,payload)
 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)`, e.Source, d.Receipt, d.SKU, e.ID, d.PO, d.Supplier, d.SupplierName, d.Origin, d.Warehouse, d.ReceivedAt, d.Unit, s.units[d.SKU], string(d.Received), count, batch, dataHash, string(msg.Value)); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE inventory_items SET physical_qty=physical_qty+$2,updated_at=clock_timestamp() WHERE sku=$1`, d.SKU, count); err != nil {
		return err
	}
	return tx.Commit()
}

type deadLetterWriter interface {
	WriteMessages(context.Context, ...segmentio.Message) error
}

type processingPanic struct{ stack string }

func (e processingPanic) Error() string { return "procurement handler panic" }
func safelyHandle(ctx context.Context, handle func(context.Context, KafkaMessage) error, msg KafkaMessage) (err error) {
	defer func() {
		if recover() != nil {
			err = processingPanic{string(debug.Stack())}
		}
	}()
	return handle(ctx, msg)
}

// Persist to DLQ before committing the source offset. DB retries are bounded;
// failed DLQ writes/commits keep this message in place and signal unhealthy state.
func processProcurement(ctx context.Context, r MessageReader, w deadLetterWriter, handle func(context.Context, KafkaMessage) error, msg KafkaMessage) error {
	return processWithRetry(ctx, r, w, handle, msg, time.Second)
}
func processWithRetry(ctx context.Context, r MessageReader, w deadLetterWriter, handle func(context.Context, KafkaMessage) error, msg KafkaMessage, backoff time.Duration) error {
	var err error
	attempts := 0
	for attempts < 5 {
		attempts++
		var envelope struct {
			Type    string `json:"type"`
			Version string `json:"specversion"`
			ID      string `json:"id"`
			Source  string `json:"source"`
		}
		if json.Unmarshal(msg.Value, &envelope) == nil && envelope.Type != goodsReceivedType && strings.HasPrefix(envelope.Type, "vn.omama.procurement.") && envelope.Version == "1.0" && identifier(envelope.ID, 256) && identifier(envelope.Source, 1000) {
			return r.CommitMessages(ctx, msg)
		}
		attemptCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		err = safelyHandle(attemptCtx, handle, msg)
		cancel()
		if err == nil {
			return r.CommitMessages(ctx, msg)
		}
		var invalid invalidReceipt
		if errors.As(err, &invalid) || ctx.Err() != nil {
			break
		}
		if attempts < 5 {
			procurementRetries.Add(1)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff * time.Duration(1<<(attempts-1))):
			}
		}
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	stack := string(debug.Stack())
	var panicErr processingPanic
	if errors.As(err, &panicErr) {
		stack = panicErr.stack
	}
	if err = w.WriteMessages(ctx, segmentio.Message{Topic: procurementDLQ, Key: msg.Key, Value: msg.Value,
		Headers: []segmentio.Header{
			{Key: "x-exception-message", Value: []byte(err.Error())}, {Key: "x-exception-stacktrace", Value: []byte(stack)},
			{Key: "x-original-topic", Value: []byte(msg.Topic)}, {Key: "x-original-partition", Value: []byte(fmt.Sprint(msg.Partition))},
			{Key: "x-original-offset", Value: []byte(fmt.Sprint(msg.Offset))}, {Key: "x-attempts", Value: []byte(fmt.Sprint(attempts))},
		}}); err != nil {
		return err
	}
	procurementDeadLetters.Add(1)
	return r.CommitMessages(ctx, msg)
}

func ProcurementWarehouseMap(raw string) (map[string]uuid.UUID, error) {
	mappings := map[string]uuid.UUID{}
	for _, entry := range strings.Split(raw, ",") {
		code, value, ok := strings.Cut(entry, "=")
		code = strings.TrimSpace(code)
		id, err := uuid.Parse(strings.TrimSpace(value))
		if !ok || code == "" || err != nil || id == uuid.Nil || mappings[code] != uuid.Nil {
			return nil, errors.New("expected unique Procurement warehouse mappings in CODE=UUID format")
		}
		mappings[code] = id
	}
	return mappings, nil
}
func ProcurementUnitMap(raw string) (map[string]string, error) {
	mappings := map[string]string{}
	for _, entry := range strings.Split(raw, ",") {
		sku, unit, ok := strings.Cut(entry, "=")
		sku = strings.TrimSpace(sku)
		unit = strings.TrimSpace(unit)
		if !ok || !identifier(sku, 50) || (unit != "GRAM" && unit != "UNIT") || mappings[sku] != "" {
			return nil, errors.New("expected unique Procurement SKU unit mappings in SKU=GRAM or SKU=UNIT format")
		}
		mappings[sku] = unit
	}
	return mappings, nil
}

func RunProcurementConsumer(ctx context.Context, brokers []string, db *sql.DB, warehouses map[string]uuid.UUID, units map[string]string) {
	procurementHealthy.Store(true)
	defer procurementHealthy.Store(false)
	defer func() {
		if recover() != nil {
			log.Printf("[Inventory Procurement] consumer panic recovered; unhealthy\n%s", debug.Stack())
		}
	}()
	r := &SegmentioReaderAdapter{reader: segmentio.NewReader(segmentio.ReaderConfig{Brokers: brokers, Topic: procurementTopic, GroupID: procurementGroup, StartOffset: segmentio.LastOffset, MaxBytes: 1 << 20})}
	defer r.Close()
	w := &segmentio.Writer{Addr: segmentio.TCP(brokers...), RequiredAcks: segmentio.RequireAll, WriteTimeout: 10 * time.Second}
	defer w.Close()
	monitorCtx, cancelMonitor := context.WithCancel(ctx)
	defer cancelMonitor()
	go monitorProcurementLag(monitorCtx, brokers)
	store := procurementStore{db: db, warehouses: warehouses, units: units}
	for ctx.Err() == nil {
		msg, err := r.FetchMessage(ctx)
		if err != nil {
			procurementHealthy.Store(false)
			if !procurementRetry(ctx, err) {
				return
			}
			continue
		}
		for ctx.Err() == nil {
			if err = processProcurement(ctx, r, w, store.handle, msg); err == nil {
				procurementHealthy.Store(true)
				break
			}
			procurementHealthy.Store(false)
			if !procurementRetry(ctx, err) {
				return
			}
		}
	}
}
func procurementRetry(ctx context.Context, err error) bool {
	if ctx.Err() != nil {
		return false
	}
	log.Printf("[Inventory Procurement] offset blocked (DLQ/commit/read failure): %v", err)
	select {
	case <-ctx.Done():
		return false
	case <-time.After(5 * time.Second):
		return true
	}
}

// Canonical hashes compare JSON numbers by exact rational value rather than spelling.
func canonicalHash(v any) string {
	raw, _ := json.Marshal(v)
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.UseNumber()
	var value any
	_ = dec.Decode(&value)
	var normalize func(any) any
	normalize = func(v any) any {
		switch x := v.(type) {
		case json.Number:
			q, ok := new(big.Rat).SetString(string(x))
			if ok {
				return map[string]string{"$number": q.RatString()}
			}
		case map[string]any:
			for k, v := range x {
				x[k] = normalize(v)
			}
		case []any:
			for i, v := range x {
				x[i] = normalize(v)
			}
		}
		return v
	}
	return payloadHash(normalize(value))
}
