package kafka

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

// INVENTORY_TEST_DB_URL must point to a PostgreSQL test database with permission
// to create schemas and run the Inventory migrations.
func TestProcurementPostgresIntegration(t *testing.T) {
	dsn := os.Getenv("INVENTORY_TEST_DB_URL")
	if dsn == "" {
		t.Skip("set INVENTORY_TEST_DB_URL to run against PostgreSQL")
	}
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	db.SetMaxOpenConns(1)
	schema := "inventory_test_" + uuid.New().String()[:8]
	_, err = db.Exec("CREATE SCHEMA " + schema)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err := db.Exec("DROP SCHEMA " + schema + " CASCADE")
		if err != nil {
			t.Error(err)
		}
	})
	_, err = db.Exec("SET search_path TO " + schema + ", public")
	require.NoError(t, err)
	for _, name := range []string{"000001_init_inventory_schema.up.sql", "000006_procurement_inbox.up.sql"} {
		contents, err := os.ReadFile(filepath.Join("..", "..", "..", "migrations", name))
		require.NoError(t, err)
		_, err = db.Exec(string(contents))
		require.NoError(t, err)
	}
	_, err = db.Exec(`INSERT INTO inventory_items(sku, warehouse_id) VALUES('SKU',$1)`, warehouseID)
	require.NoError(t, err)

	mappings, err := ProcurementWarehouseMap("WH-HUE=" + warehouseID)
	require.NoError(t, err)
	store := procurementStore{db: db, warehouses: mappings}
	msg := receiptMessage(receiptJSON)
	require.NoError(t, store.handle(context.Background(), msg))
	require.NoError(t, store.handle(context.Background(), msg))

	var stock, batchStock, lots, inbox int
	var supplierLot string
	require.NoError(t, db.QueryRow("SELECT physical_qty FROM inventory_items WHERE sku='SKU'").Scan(&stock))
	require.NoError(t, db.QueryRow("SELECT physical_qty,supplier_lot_code FROM batches WHERE batch_code='GR-TEST'").Scan(&batchStock, &supplierLot))
	require.NoError(t, db.QueryRow("SELECT count(*) FROM batches").Scan(&lots))
	require.NoError(t, db.QueryRow("SELECT count(*) FROM procurement_receipt_inbox").Scan(&inbox))
	require.Equal(t, 2, stock)
	require.Equal(t, 2, batchStock)
	require.Equal(t, "LOT", supplierLot)
	require.Equal(t, 1, lots)
	require.Equal(t, 1, inbox)
}
