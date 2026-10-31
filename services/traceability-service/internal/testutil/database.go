// Package testutil provides disposable PostgreSQL schemas for integration tests.
package testutil

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"testing"
	"time"

	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/migrations"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func Database(t *testing.T) (context.Context, *pgxpool.Pool) {
	t.Helper()
	url := os.Getenv("TRACEABILITY_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("NOT_RUN: set TRACEABILITY_TEST_DATABASE_URL for PostgreSQL tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	admin, err := pgx.Connect(ctx, url)
	if err != nil {
		t.Fatal("cannot connect to test database")
	}
	token := make([]byte, 12)
	if _, err = rand.Read(token); err != nil {
		t.Fatal(err)
	}
	schema := "trace_test_" + hex.EncodeToString(token)
	quoted := pgx.Identifier{schema}.Sanitize()
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+quoted); err != nil {
		admin.Close(ctx)
		t.Fatal(err)
	}
	t.Cleanup(func() {
		clean, done := context.WithTimeout(context.Background(), 5*time.Second)
		defer done()
		// schema derives only from a fixed prefix + cryptographic hex, never input.
		if _, err := admin.Exec(clean, "DROP SCHEMA "+quoted+" CASCADE"); err != nil {
			t.Error(err)
		}
		admin.Close(clean)
	})
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal("invalid test database config")
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	cfg.MaxConns = 8
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal("cannot create test pool")
	}
	t.Cleanup(pool.Close)
	conn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal("cannot acquire migration connection")
	}
	err = migrations.Apply(ctx, conn.Conn(), "sandbox")
	conn.Release()
	if err != nil {
		t.Fatal(err)
	}
	return ctx, pool
}
