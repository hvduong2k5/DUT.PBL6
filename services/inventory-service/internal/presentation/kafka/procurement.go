package kafka

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"math/big"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	segmentio "github.com/segmentio/kafka-go"
)

const procurementTopic = "procurement.events.v1"
const goodsReceivedType = "vn.omama.procurement.goods.received.v1"

type invalidReceipt string

func (e invalidReceipt) Error() string { return string(e) }

type goodsReceipt struct {
	ID      uuid.UUID `json:"id"`
	Type    string    `json:"type"`
	Version string    `json:"specversion"`
	Data    struct {
		SKU          string          `json:"sku_code"`
		Warehouse    string          `json:"warehouse_id"`
		Batch        string          `json:"batch_code"`
		Supplier     uuid.UUID       `json:"supplier_id"`
		SupplierLot  string          `json:"supplier_lot_code"`
		Unit         string          `json:"unit"`
		Accepted     json.RawMessage `json:"quantity_accepted"`
		Received     json.RawMessage `json:"quantity_received"`
		Rejected     json.RawMessage `json:"quantity_rejected"`
		Manufactured string          `json:"manufactured_on"`
		Expires      string          `json:"expires_on"`
	} `json:"data"`
}

func quantity(raw json.RawMessage) (*big.Rat, bool) {
	// Bound rational parsing to the producer's decimal format (no exponent).
	s := string(raw)
	if len(s) > 64 || strings.ContainsAny(s, "eE/") {
		return nil, false
	}
	q, ok := new(big.Rat).SetString(s)
	return q, ok && q.Sign() >= 0
}

func decodeReceipt(msg KafkaMessage) (goodsReceipt, int64, error) {
	var e goodsReceipt
	if json.Unmarshal(msg.Value, &e) != nil || e.Type != goodsReceivedType || e.Version != "1.0" || e.ID == uuid.Nil {
		return e, 0, invalidReceipt("invalid GoodsReceived CloudEvent")
	}
	d := e.Data
	if strings.TrimSpace(d.SKU) == "" || utf8.RuneCountInString(d.SKU) > 50 || string(msg.Key) != d.SKU ||
		strings.TrimSpace(d.Warehouse) == "" || strings.TrimSpace(d.Batch) == "" || utf8.RuneCountInString(d.Batch) > 100 ||
		d.Supplier == uuid.Nil || strings.TrimSpace(d.Unit) == "" ||
		strings.ContainsRune(d.SKU+d.Batch+d.SupplierLot, 0) {
		return e, 0, invalidReceipt("invalid receipt identifiers or unit")
	}
	accepted, a := quantity(d.Accepted)
	rejected, b := quantity(d.Rejected)
	received, c := quantity(d.Received)
	if !a || !b || !c || received.Sign() <= 0 || new(big.Rat).Add(accepted, rejected).Cmp(received) != 0 {
		return e, 0, invalidReceipt("invalid or inconsistent receipt quantities")
	}
	if !accepted.IsInt() || !accepted.Num().IsInt64() || accepted.Num().Int64() > math.MaxInt32 {
		return e, 0, invalidReceipt("accepted quantity must fit an Inventory integer exactly")
	}
	count := accepted.Num().Int64()
	if count > 0 || d.Manufactured != "" || d.Expires != "" {
		mfg, errM := time.Parse(time.DateOnly, d.Manufactured)
		exp, errE := time.Parse(time.DateOnly, d.Expires)
		if errM != nil || errE != nil || mfg.Year() == 0 || !mfg.Before(exp) {
			return e, 0, invalidReceipt("receipt requires valid manufacturing and expiry dates in order")
		}
	}
	return e, count, nil
}

type procurementStore struct {
	db         *sql.DB
	warehouses map[string]uuid.UUID
}

func (s procurementStore) handle(ctx context.Context, msg KafkaMessage) error {
	e, count, err := decodeReceipt(msg)
	if err != nil {
		return err
	}
	d := e.Data
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `INSERT INTO procurement_receipt_inbox(event_id) VALUES($1) ON CONFLICT DO NOTHING`, e.ID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 || count == 0 {
		return tx.Commit()
	}

	warehouse, ok := s.warehouses[d.Warehouse]
	if !ok {
		return invalidReceipt("unmapped Procurement warehouse")
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
	if stock+count > math.MaxInt32 {
		return invalidReceipt("receipt would overflow inventory quantity")
	}

	// The item lock serializes receipts for this SKU. Refuse ambiguous legacy lots.
	var batch uuid.UUID
	var matching bool
	var batchStock int64
	var matches int
	err = tx.QueryRowContext(ctx, `SELECT id, supplier_id=$3 AND mfg_date=$4::date AND exp_date=$5::date
 AND (supplier_lot_code IS NULL OR supplier_lot_code IS NOT DISTINCT FROM NULLIF($6,'')),
 physical_qty, count(*) OVER () FROM batches WHERE sku=$1 AND batch_code=$2 LIMIT 1`,
		d.SKU, d.Batch, d.Supplier, d.Manufactured, d.Expires, d.SupplierLot).Scan(&batch, &matching, &batchStock, &matches)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		_, err = tx.ExecContext(ctx, `INSERT INTO batches(sku,batch_code,supplier_id,mfg_date,exp_date,supplier_lot_code,physical_qty,status)
  VALUES($1,$2,$3,$4,$5,NULLIF($6,''),$7,CASE WHEN $5::date<=CURRENT_DATE THEN 'EXPIRED' ELSE 'ACTIVE' END)`,
			d.SKU, d.Batch, d.Supplier, d.Manufactured, d.Expires, d.SupplierLot, count)
	case err != nil:
		return err
	case matches != 1 || !matching || batchStock+count > math.MaxInt32:
		return invalidReceipt("conflicting batch metadata or quantity")
	default:
		_, err = tx.ExecContext(ctx, `UPDATE batches SET physical_qty=physical_qty+$2,
  supplier_lot_code=COALESCE(supplier_lot_code,NULLIF($3,'')) WHERE id=$1`, batch, count, d.SupplierLot)
	}
	if err != nil {
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

func processProcurement(ctx context.Context, r MessageReader, w deadLetterWriter, handle func(context.Context, KafkaMessage) error, msg KafkaMessage) error {
	err := handle(ctx, msg)
	var invalid invalidReceipt
	if errors.As(err, &invalid) {
		err = w.WriteMessages(ctx, segmentio.Message{Topic: procurementTopic + ".DLT", Key: msg.Key, Value: msg.Value,
			Headers: []segmentio.Header{{Key: "inventory-error", Value: []byte(invalid.Error())},
				{Key: "source-position", Value: []byte(fmt.Sprintf("%s:%d:%d", msg.Topic, msg.Partition, msg.Offset))}}})
	}
	if err != nil {
		return err
	}
	return r.CommitMessages(ctx, msg)
}

func ProcurementWarehouseMap(raw string) (map[string]uuid.UUID, error) {
	mappings := make(map[string]uuid.UUID)
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

func RunProcurementConsumer(ctx context.Context, brokers []string, db *sql.DB, warehouses map[string]uuid.UUID) {
	r := &SegmentioReaderAdapter{reader: segmentio.NewReader(segmentio.ReaderConfig{
		Brokers: brokers, Topic: procurementTopic, GroupID: "inventory-service-procurement-goods-received-group",
	})}
	defer r.Close()
	w := &segmentio.Writer{Addr: segmentio.TCP(brokers...), RequiredAcks: segmentio.RequireAll}
	defer w.Close()
	store := procurementStore{db: db, warehouses: warehouses}
	for ctx.Err() == nil {
		msg, err := r.FetchMessage(ctx)
		if err != nil {
			if !procurementRetry(ctx, err) {
				return
			}
			continue
		}
		// Fetch advances the local reader position even without a commit. Retry this
		// message before fetching any later offset.
		for ctx.Err() == nil {
			if err = processProcurement(ctx, r, w, store.handle, msg); err == nil {
				break
			}
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
	log.Printf("[Inventory Procurement consumer] retrying after error: %v", err)
	select {
	case <-ctx.Done():
		return false
	case <-time.After(time.Second):
		return true
	}
}
