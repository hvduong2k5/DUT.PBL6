package qc

import (
	"context"
	"dut-pbl6/inventory-service/internal/testdb"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
	"time"
)

func TestReleaseAuditAndReplay(t *testing.T) {
	db := testdb.Open(t)
	ctx := context.Background()
	batch := uuid.NewString()
	_, err := db.Exec(`DO $$ BEGIN CREATE ROLE inventory_qc_operator NOLOGIN; EXCEPTION WHEN duplicate_object THEN NULL; END $$;`)
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO inventory_items(sku,physical_qty) VALUES('QC',10)")
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO batches(id,sku,batch_code,supplier_id,physical_qty,status) VALUES($1,'QC','QC',gen_random_uuid(),10,'QUARANTINE')", batch)
	require.NoError(t, err)
	r := Request{Actor: uuid.NewString(), Key: "qc-release-test-key", Batch: batch, Warehouse: "00000000-0000-0000-0000-000000000001", Manufactured: time.Now().UTC().AddDate(0, 0, -1).Format(time.DateOnly), Expires: time.Now().UTC().AddDate(0, 1, 0).Format(time.DateOnly), Reason: "QC certificate 001", Version: 1}
	wrong := r
	wrong.Warehouse = uuid.NewString()
	_, err = Release(ctx, db, wrong)
	require.ErrorContains(t, err, "warehouse mismatch")
	wrong = r
	wrong.Version = 99
	_, err = Release(ctx, db, wrong)
	require.ErrorContains(t, err, "state conflict")
	first, err := Release(ctx, db, r)
	require.NoError(t, err)
	second, err := Release(ctx, db, r)
	require.NoError(t, err)
	require.Equal(t, first, second)
	var qty int
	require.NoError(t, db.QueryRow("SELECT inventory_sellable_quantity('QC',now())").Scan(&qty))
	require.Equal(t, 10, qty)
	r.Reason = "different"
	_, err = Release(ctx, db, r)
	require.ErrorContains(t, err, "idempotency conflict")
	r.Key = "new-qc-release-key"
	_, err = Release(ctx, db, r)
	require.ErrorContains(t, err, "state conflict")
	var count int
	require.NoError(t, db.QueryRow("SELECT count(*) FROM inventory_qc_releases").Scan(&count))
	require.Equal(t, 1, count)
}

func TestReleaseRequiresOperatorRole(t *testing.T) {
	db := testdb.Open(t)
	role := "qc_denied_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	_, err := db.Exec("CREATE ROLE " + role + " NOLOGIN")
	require.NoError(t, err)
	defer func() { db.Exec("RESET ROLE"); db.Exec("DROP ROLE " + role) }()
	_, err = db.Exec("SET ROLE " + role)
	require.NoError(t, err)
	r := Request{Actor: uuid.NewString(), Batch: uuid.NewString(), Warehouse: uuid.NewString(), Key: "qc-release-no-grant", Version: 1, Reason: "test", Manufactured: time.Now().UTC().AddDate(0, 0, -1).Format(time.DateOnly), Expires: time.Now().UTC().AddDate(0, 1, 0).Format(time.DateOnly)}
	_, err = Release(context.Background(), db, r)
	require.ErrorContains(t, err, "database role membership")
}

func TestReleaseWithRestrictedDatabaseRole(t *testing.T) {
	db := testdb.Open(t)
	var schema string
	require.NoError(t, db.QueryRow("SELECT current_schema()").Scan(&schema))
	_, err := db.Exec(`DO $$ BEGIN CREATE ROLE inventory_qc_operator NOLOGIN; EXCEPTION WHEN duplicate_object THEN NULL; END $$;`)
	require.NoError(t, err)
	role := "qc_allowed_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	_, err = db.Exec("CREATE ROLE " + role + " NOLOGIN; GRANT inventory_qc_operator TO " + role)
	require.NoError(t, err)
	defer func() { db.Exec("RESET ROLE"); db.Exec("DROP ROLE " + role) }()
	_, err = db.Exec("GRANT USAGE ON SCHEMA " + schema + " TO inventory_qc_operator; GRANT SELECT ON ALL TABLES IN SCHEMA " + schema + " TO inventory_qc_operator; GRANT UPDATE(mfg_date,exp_date,status) ON batches TO inventory_qc_operator; GRANT UPDATE(inventory_version) ON inventory_items TO inventory_qc_operator; GRANT INSERT ON inventory_qc_releases,outbox_events TO inventory_qc_operator;")
	require.NoError(t, err)
	batch := uuid.NewString()
	_, err = db.Exec("INSERT INTO inventory_items(sku,physical_qty) VALUES('ROLE',7)")
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO batches(id,sku,batch_code,supplier_id,physical_qty,status) VALUES($1,'ROLE','ROLE',gen_random_uuid(),7,'QUARANTINE')", batch)
	require.NoError(t, err)
	_, err = db.Exec("SET ROLE " + role)
	require.NoError(t, err)
	r := Request{Actor: uuid.NewString(), Key: "restricted-role-key", Batch: batch, Warehouse: "00000000-0000-0000-0000-000000000001", Manufactured: time.Now().UTC().AddDate(0, 0, -1).Format(time.DateOnly), Expires: time.Now().UTC().AddDate(0, 1, 0).Format(time.DateOnly), Reason: "verified", Version: 1}
	_, err = Release(context.Background(), db, r)
	require.NoError(t, err)
}
