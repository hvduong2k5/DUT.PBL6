package kafka

import (
	"context"
	"dut-pbl6/inventory-service/internal/testdb"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestProcurementFailureRollsBackStockAndEvidence(t *testing.T) {
	db := testdb.Open(t)
	provision(t, db)
	_, err := db.Exec(`CREATE FUNCTION reject_stock_write() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected failure'; END $$;
CREATE TRIGGER reject_stock_write BEFORE UPDATE ON inventory_items FOR EACH ROW EXECUTE FUNCTION reject_stock_write();`)
	require.NoError(t, err)
	require.ErrorContains(t, receiptStore(t, db).handle(context.Background(), receiptMessage(t)), "injected failure")
	for _, table := range []string{"procurement_receipt_inbox", "procurement_receipts", "procurement_supplier_mapping", "batches"} {
		var count int
		require.NoError(t, db.QueryRow("SELECT count(*) FROM "+table).Scan(&count))
		require.Zero(t, count, table)
	}
	var stock int
	require.NoError(t, db.QueryRow("SELECT physical_qty FROM inventory_items WHERE sku='SKU'").Scan(&stock))
	require.Zero(t, stock)
}
