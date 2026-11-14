package kafka

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	segmentio "github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/require"
)

const warehouseID = "00000000-0000-0000-0000-000000000001"
const receiptJSON = `{"specversion":"1.0","id":"00000000-0000-0000-0000-000000000002","type":"vn.omama.procurement.goods.received.v1","data":{"sku_code":"SKU","warehouse_id":"WH-HUE","batch_code":"GR-TEST","supplier_id":"00000000-0000-0000-0000-000000000003","supplier_lot_code":"LOT","unit":"KG","quantity_accepted":2,"quantity_rejected":0.5,"quantity_received":2.5,"manufactured_on":"2026-01-01","expires_on":"2027-01-01"}}`

func receiptMessage(body string) KafkaMessage {
	return KafkaMessage{Key: []byte("SKU"), Value: []byte(body)}
}

func TestDecodeProcurementReceipt(t *testing.T) {
	for _, tt := range []struct {
		name, body string
		count      int64
		invalid    bool
	}{
		{"accepted only", receiptJSON, 2, false},
		{"wrong type", strings.Replace(receiptJSON, goodsReceivedType, "other", 1), 0, true},
		{"missing accepted", strings.Replace(receiptJSON, `"quantity_accepted":2,`, "", 1), 0, true},
		{"negative", strings.Replace(receiptJSON, `"quantity_accepted":2`, `"quantity_accepted":-2`, 1), 0, true},
		{"non-number", strings.Replace(receiptJSON, `"quantity_accepted":2`, `"quantity_accepted":"2"`, 1), 0, true},
		{"fractional", strings.NewReplacer(`"quantity_accepted":2`, `"quantity_accepted":1.5`, `"quantity_received":2.5`, `"quantity_received":2`).Replace(receiptJSON), 0, true},
		{"bad date", strings.Replace(receiptJSON, "2026-01-01", "2026-02-30", 1), 0, true},
		{"missing date", strings.Replace(receiptJSON, `,"expires_on":"2027-01-01"`, "", 1), 0, true},
		{"zero without dates", zeroReceipt(), 0, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, count, err := decodeReceipt(receiptMessage(tt.body))
			if tt.invalid {
				var invalid invalidReceipt
				require.ErrorAs(t, err, &invalid)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.count, count)
		})
	}
}

func zeroReceipt() string {
	return strings.NewReplacer(`"quantity_accepted":2`, `"quantity_accepted":0`, `"quantity_received":2.5`, `"quantity_received":0.5`, `,"manufactured_on":"2026-01-01","expires_on":"2027-01-01"`, "").Replace(receiptJSON)
}

func TestProcurementStore(t *testing.T) {
	ctx := context.Background()
	for _, name := range []string{"accepted then replay", "zero", "unknown warehouse", "stock write fails"} {
		t.Run(name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()
			mappings, err := ProcurementWarehouseMap("WH-HUE=" + warehouseID)
			require.NoError(t, err)
			store := procurementStore{db: db, warehouses: mappings}
			msg := receiptMessage(receiptJSON)
			mock.ExpectBegin()
			mock.ExpectExec("INSERT INTO procurement_receipt_inbox").WillReturnResult(sqlmock.NewResult(0, 1))
			switch name {
			case "zero":
				msg = receiptMessage(zeroReceipt())
				mock.ExpectCommit()
			case "unknown warehouse":
				store.warehouses = nil
				mock.ExpectRollback()
			default:
				mock.ExpectQuery("SELECT warehouse_id, physical_qty").WithArgs("SKU").
					WillReturnRows(sqlmock.NewRows([]string{"warehouse_id", "physical_qty"}).AddRow(warehouseID, 10))
				mock.ExpectQuery("SELECT id, supplier_id").WillReturnError(sql.ErrNoRows)
				mock.ExpectExec("INSERT INTO batches").WithArgs("SKU", "GR-TEST", "00000000-0000-0000-0000-000000000003", "2026-01-01", "2027-01-01", "LOT", int64(2)).
					WillReturnResult(sqlmock.NewResult(0, 1))
				update := mock.ExpectExec("UPDATE inventory_items").WithArgs("SKU", int64(2))
				if name == "stock write fails" {
					update.WillReturnError(errors.New("database unavailable"))
					mock.ExpectRollback()
				} else {
					update.WillReturnResult(sqlmock.NewResult(0, 1))
					mock.ExpectCommit()
					mock.ExpectBegin()
					mock.ExpectExec("INSERT INTO procurement_receipt_inbox").WillReturnResult(sqlmock.NewResult(0, 0))
					mock.ExpectCommit()
				}
			}
			err = store.handle(ctx, msg)
			switch name {
			case "unknown warehouse":
				require.EqualError(t, err, "unmapped Procurement warehouse")
			case "stock write fails":
				require.EqualError(t, err, "database unavailable")
			default:
				require.NoError(t, err)
			}
			if name == "accepted then replay" {
				require.NoError(t, store.handle(ctx, msg))
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

type procurementReaderStub struct {
	MessageReader
	commits int
}

func (r *procurementReaderStub) CommitMessages(context.Context, ...KafkaMessage) error {
	r.commits++
	return nil
}

type procurementDLTStub struct {
	writes int
	err    error
}

func (w *procurementDLTStub) WriteMessages(_ context.Context, msgs ...segmentio.Message) error {
	w.writes += len(msgs)
	return w.err
}

func TestProcurementOffsetPolicy(t *testing.T) {
	for _, tt := range []struct {
		name            string
		processing, dlt error
		commits, writes int
	}{
		{"success", nil, nil, 1, 0},
		{"DB failure", errors.New("DB error"), nil, 0, 0},
		{"invalid and published", invalidReceipt("bad event"), nil, 1, 1},
		{"DLT unavailable", invalidReceipt("bad event"), errors.New("broker error"), 0, 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := &procurementReaderStub{}
			w := &procurementDLTStub{err: tt.dlt}
			err := processProcurement(context.Background(), r, w, func(context.Context, KafkaMessage) error { return tt.processing }, KafkaMessage{})
			if tt.commits == 0 {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tt.commits, r.commits)
			require.Equal(t, tt.writes, w.writes)
		})
	}
}
