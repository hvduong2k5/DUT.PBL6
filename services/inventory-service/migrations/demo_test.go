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

func seedDatabase(t *testing.T, empty ...bool) *sql.DB {
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
	if _, err := db.Exec(`BEGIN; SELECT pg_advisory_xact_lock(2026112102); CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public; COMMIT`); err != nil {
		db.Close()
		t.Fatal(err)
	}
	schema := "inventory_seed_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err = db.Exec("CREATE SCHEMA " + schema + "; SET search_path TO " + schema); err != nil {
		db.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec("SET search_path TO public; DROP SCHEMA " + schema + " CASCADE")
		_ = db.Close()
	})
	if len(empty) > 0 && empty[0] {
		return db
	}
	files, err := filepath.Glob("*.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		if file >= "000006" {
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
	if err := Baseline(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestRunnerFreshDatabaseAndChecksum(t *testing.T) {
	db := seedDatabase(t, true)
	ctx := context.Background()
	if err := Verify(ctx, db); err == nil {
		t.Fatal("verify accepted missing history")
	}
	var exists bool
	if err := db.QueryRow("SELECT to_regclass('inventory_schema_migrations') IS NOT NULL").Scan(&exists); err != nil || exists {
		t.Fatal("verify wrote history", err)
	}
	if err := Apply(ctx, db); err != nil {
		t.Fatal(err)
	}
	if err := Apply(ctx, db); err != nil {
		t.Fatal(err)
	}
	if err := Verify(ctx, db); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := db.QueryRow("SELECT count(*) FROM inventory_schema_migrations").Scan(&count); err != nil || count != 7 {
		t.Fatal(count, err)
	}
	if _, err := db.Exec("UPDATE inventory_schema_migrations SET checksum='tampered' WHERE version='000003'"); err != nil {
		t.Fatal(err)
	}
	for _, fn := range []func(context.Context, *sql.DB) error{Apply, Verify} {
		if err := fn(ctx, db); err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
			t.Fatal("checksum was not rejected", err)
		}
	}
}

func TestRunnerRequiresLegacyBaseline(t *testing.T) {
	db := seedDatabase(t)
	ctx := context.Background()
	if _, err := db.Exec("DROP TABLE inventory_schema_migrations"); err != nil {
		t.Fatal(err)
	}
	if err := Apply(ctx, db); err == nil || !strings.Contains(err.Error(), "baseline") {
		t.Fatal("legacy database was not refused", err)
	}
	if err := Baseline(ctx, db); err != nil {
		t.Fatal(err)
	}
	if err := Verify(ctx, db); err == nil || !strings.Contains(err.Error(), "000006") {
		t.Fatal("missing migration not detected", err)
	}
	if err := Apply(ctx, db); err != nil {
		t.Fatal(err)
	}
	if err := Verify(ctx, db); err != nil {
		t.Fatal(err)
	}
	if err := Baseline(ctx, db); err == nil {
		t.Fatal("baseline accepted tracked database")
	}
}

func TestRunnerRollsBackPendingBatch(t *testing.T) {
	db := seedDatabase(t)
	ctx := context.Background()
	if _, err := db.Exec("CREATE TABLE procurement_receipts (wrong_column text)"); err != nil {
		t.Fatal(err)
	}
	if err := Apply(ctx, db); err == nil {
		t.Fatal("invalid seed schema accepted")
	}
	var exists bool
	if err := db.QueryRow("SELECT to_regclass('procurement_receipt_inbox') IS NOT NULL").Scan(&exists); err != nil || exists {
		t.Fatal("000006 was not rolled back", err)
	}
	var count int
	if err := db.QueryRow("SELECT count(*) FROM inventory_schema_migrations").Scan(&count); err != nil || count != 5 {
		t.Fatal("history partially committed", count, err)
	}
}

func TestConcurrentMigrationJobs(t *testing.T) {
	db := seedDatabase(t, true)
	var schema string
	if err := db.QueryRow("SELECT current_schema()").Scan(&schema); err != nil {
		t.Fatal(err)
	}
	other, err := sql.Open("postgres", os.Getenv("INVENTORY_TEST_DB_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	other.SetMaxOpenConns(1)
	if _, err := other.Exec("SET search_path TO " + schema); err != nil {
		t.Fatal(err)
	}
	results := make(chan error, 2)
	for _, connection := range []*sql.DB{db, other} {
		go func(connection *sql.DB) { results <- Apply(context.Background(), connection) }(connection)
	}
	for range 2 {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
	if err := Verify(context.Background(), db); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyRejectsPartialSchemaDespiteHistory(t *testing.T) {
	db := seedDatabase(t, true)
	ctx := context.Background()
	if err := Apply(ctx, db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("ALTER TABLE batches DROP COLUMN supplier_lot_code"); err != nil {
		t.Fatal(err)
	}
	if err := Verify(ctx, db); err == nil || !strings.Contains(err.Error(), "incomplete") {
		t.Fatal("partial schema accepted", err)
	}
}

func TestCanonicalMigrationRoundTripWithoutBusinessData(t *testing.T) {
	db := seedDatabase(t, true)
	ctx := context.Background()
	if err := Apply(ctx, db); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile("000008_canonical_procurement_receipts.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err = tx.Exec(string(data)); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec("DELETE FROM inventory_schema_migrations WHERE version='000008'"); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err = Apply(ctx, db); err != nil {
		t.Fatal(err)
	}
	if err = Verify(ctx, db); err != nil {
		t.Fatal(err)
	}
}
