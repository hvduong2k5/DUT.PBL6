package migrations

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

func seedDatabase(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("INVENTORY_TEST_DB_URL")
	if dsn == "" {
		t.Skip("requires isolated INVENTORY_TEST_DB_URL")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	schema := "inventory_seed_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err = db.Exec("CREATE SCHEMA " + schema + "; SET search_path TO " + schema + ",public"); err != nil {
		db.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec("SET search_path TO public; DROP SCHEMA " + schema + " CASCADE")
		_ = db.Close()
	})
	files, err := filepath.Glob("*.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		if strings.HasPrefix(file, "000007_") {
			continue
		}
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = db.Exec(string(data)); err != nil {
			t.Fatalf("apply %s: %v", file, err)
		}
	}
	return db
}

func rollbackSeed(db *sql.DB) error {
	data, err := os.ReadFile("000007_seed_pbl6_demo.down.sql")
	if err != nil {
		return err
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(string(data)); err != nil {
		return err
	}
	return tx.Commit()
}

func TestDemoSeedStartupPreservesStockAndRefusesUsedRollback(t *testing.T) {
	db := seedDatabase(t)
	ctx := context.Background()
	for range 2 {
		if err := ApplyDemoSeed(ctx, db); err != nil {
			t.Fatal(err)
		}
	}
	var count, stock int
	if err := db.QueryRow("SELECT count(*),sum(physical_qty) FROM inventory_items").Scan(&count, &stock); err != nil || count != 40 || stock != 2600 {
		t.Fatalf("unexpected seed count/stock %d/%d: %v", count, stock, err)
	}
	if _, err := db.Exec("UPDATE batches SET physical_qty=physical_qty-1 WHERE sku='OM-MX-01'; UPDATE inventory_items SET physical_qty=physical_qty-1 WHERE sku='OM-MX-01'"); err != nil {
		t.Fatal(err)
	}
	if err := ApplyDemoSeed(ctx, db); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow("SELECT physical_qty FROM inventory_items WHERE sku='OM-MX-01'").Scan(&stock); err != nil || stock != 79 {
		t.Fatalf("startup reset used stock to %d: %v", stock, err)
	}
	if err := rollbackSeed(db); err == nil {
		t.Fatal("rollback accepted stock used after migration")
	}
	if err := db.QueryRow("SELECT count(*) FROM inventory_demo_seed_000007").Scan(&count); err != nil || count != 40 {
		t.Fatalf("failed rollback changed marker: %d, %v", count, err)
	}
}

func TestDemoRollbackPreservesRowsThatPredateMigration(t *testing.T) {
	db := seedDatabase(t)
	_, err := db.Exec(`
INSERT INTO inventory_items(sku,warehouse_id,physical_qty) VALUES
('OM-MX-01','9b200006-2026-4020-8020-000000000001',80),
('EXISTING-SKU','00000000-0000-0000-0000-000000000001',12);
INSERT INTO batches(id,sku,batch_code,supplier_id,mfg_date,exp_date,physical_qty,status)
VALUES ('9b200005-2026-4020-8020-000000000001','OM-MX-01','PREEXISTING-LOT',
'9b200007-2026-4020-8020-000000000001',CURRENT_DATE-7,CURRENT_DATE+90,80,'ACTIVE');
`)
	if err != nil {
		t.Fatal(err)
	}
	if err = ApplyDemoSeed(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	if err = rollbackSeed(db); err != nil {
		t.Fatal(err)
	}
	var count int
	if err = db.QueryRow("SELECT count(*) FROM inventory_items").Scan(&count); err != nil || count != 2 {
		t.Fatalf("rollback removed pre-existing stock: %d, %v", count, err)
	}
	if err = db.QueryRow("SELECT count(*) FROM batches WHERE batch_code='PREEXISTING-LOT'").Scan(&count); err != nil || count != 1 {
		t.Fatalf("rollback removed pre-existing batch: %d, %v", count, err)
	}
}
