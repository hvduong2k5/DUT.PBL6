// Package testdb provides isolated PostgreSQL schemas for integration tests.
package testdb

import (
	"context"
	"database/sql"
	"dut-pbl6/inventory-service/migrations"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"os"
	"strings"
	"testing"
)

func Open(t *testing.T) *sql.DB {
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
	schema := "inv_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err = db.Exec("CREATE SCHEMA " + schema + "; SET search_path TO " + schema + ",public"); err != nil {
		db.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Exec("SET search_path TO public; DROP SCHEMA " + schema + " CASCADE"); db.Close() })
	if err = migrations.Apply(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	return db
}

func Second(t *testing.T, db *sql.DB) *sql.DB {
	t.Helper()
	var schema string
	if err := db.QueryRow("SELECT current_schema()").Scan(&schema); err != nil {
		t.Fatal(err)
	}
	other, err := sql.Open("postgres", os.Getenv("INVENTORY_TEST_DB_URL"))
	if err != nil {
		t.Fatal(err)
	}
	other.SetMaxOpenConns(1)
	if _, err = other.Exec("SET search_path TO " + schema + ",public"); err != nil {
		other.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() { other.Close() })
	return other
}
