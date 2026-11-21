package seed

import (
	"context"
	"database/sql"
	"dut-pbl6/inventory-service/internal/testdb"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestDisabledDoesNotOpenDatabase(t *testing.T) {
	require.NoError(t, Apply(context.Background(), nil, false))
}

func TestSeedJobsAndCompensatingRollback(t *testing.T) {
	db := testdb.Open(t)
	other := testdb.Second(t, db)
	ctx := context.Background()
	done := make(chan error, 2)
	for _, conn := range []*sql.DB{db, other} {
		go func(conn *sql.DB) { done <- Apply(ctx, conn, true) }(conn)
	}
	for range 2 {
		require.NoError(t, <-done)
	}
	var count, stock int
	require.NoError(t, db.QueryRow("SELECT count(*),sum(physical_qty) FROM inventory_items").Scan(&count, &stock))
	require.Equal(t, 40, count)
	require.Equal(t, 2600, stock)
	require.NoError(t, db.QueryRow("SELECT count(*) FROM outbox_events WHERE event_type='inventory.stock-changed.v1'").Scan(&count))
	require.Equal(t, 80, count)
	var version int64
	require.NoError(t, db.QueryRow("SELECT max((payload->>'inventory_version')::bigint) FROM outbox_events WHERE aggregate_id='OM-MX-01'").Scan(&version))
	require.NoError(t, Rollback(ctx, db))
	var zero int64
	require.NoError(t, db.QueryRow("SELECT (payload->>'sellable_quantity')::bigint,(payload->>'inventory_version')::bigint FROM outbox_events WHERE aggregate_id='OM-MX-01' ORDER BY (payload->>'inventory_version')::bigint DESC LIMIT 1").Scan(&stock, &zero))
	require.Zero(t, stock)
	require.Greater(t, zero, version)
	require.NoError(t, Apply(ctx, db, true))
	require.NoError(t, db.QueryRow("SELECT inventory_version FROM inventory_items WHERE sku='OM-MX-01'").Scan(&version))
	require.Greater(t, version, zero)
}

func TestExistingZeroStockIsSkipped(t *testing.T) {
	db := testdb.Open(t)
	ctx := context.Background()
	_, err := db.Exec("INSERT INTO inventory_items(sku,physical_qty) VALUES('OM-MX-01',0),('EXISTING',12)")
	require.NoError(t, err)
	require.NoError(t, Apply(ctx, db, true))
	require.NoError(t, Apply(ctx, db, true))
	var qty, count int
	require.NoError(t, db.QueryRow("SELECT physical_qty FROM inventory_items WHERE sku='OM-MX-01'").Scan(&qty))
	require.Zero(t, qty)
	require.NoError(t, db.QueryRow("SELECT count(*) FROM batches WHERE sku='OM-MX-01'").Scan(&count))
	require.Zero(t, count)
	require.NoError(t, Rollback(ctx, db))
	require.NoError(t, db.QueryRow("SELECT count(*) FROM inventory_items").Scan(&count))
	require.Equal(t, 2, count)
}

func TestRollbackRefusesBusinessChanges(t *testing.T) {
	for _, change := range []string{
		"UPDATE inventory_items SET physical_qty=79 WHERE sku='OM-MX-01'",
		"UPDATE inventory_items SET status='INACTIVE' WHERE sku='OM-MX-01'",
		"INSERT INTO stock_receipts(actor_id,idem_key,sku,source_reference,request,response) VALUES(gen_random_uuid(),'test','OM-MX-01','R','{}','{}')",
		"INSERT INTO stock_adjustments(adjustment_code,sku,adjustment_type,quantity_delta,adjusted_by) VALUES('A','OM-MX-01','TEST',0,gen_random_uuid())",
		"INSERT INTO batches(batch_code,sku,supplier_id,mfg_date,exp_date) VALUES('EXTRA','OM-MX-01',gen_random_uuid(),CURRENT_DATE-1,CURRENT_DATE+30)",
	} {
		t.Run(change, func(t *testing.T) {
			db := testdb.Open(t)
			ctx := context.Background()
			require.NoError(t, Apply(ctx, db, true))
			_, err := db.Exec(change)
			require.NoError(t, err)
			require.ErrorContains(t, Rollback(ctx, db), "rollback refused")
			var count int
			require.NoError(t, db.QueryRow("SELECT count(*) FROM inventory_demo_seed_000007").Scan(&count))
			require.Equal(t, 40, count)
		})
	}
}
