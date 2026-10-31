// Package migrations installs the dedicated service schema without resetting data.
package migrations

import (
	"context"
	"crypto/sha256"
	"embed"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

//go:embed *.sql
var scripts embed.FS

func Apply(ctx context.Context, conn *pgx.Conn, mode string) error {
	if mode != "sandbox" && mode != "integration" {
		return errors.New("invalid migration mode")
	}
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(80030001)`); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations(version integer PRIMARY KEY, checksum text NOT NULL, applied_at timestamptz NOT NULL DEFAULT now())`); err != nil {
		return err
	}
	sql, err := scripts.ReadFile("000001_init.up.sql")
	if err != nil {
		return err
	}
	checksum := fmt.Sprintf("%x", sha256.Sum256(sql))
	var actual string
	err = tx.QueryRow(ctx, `SELECT checksum FROM schema_migrations WHERE version=1`).Scan(&actual)
	if err == nil {
		if actual != checksum {
			return errors.New("migration checksum mismatch")
		}
		var storedMode string
		if err = tx.QueryRow(ctx, `SELECT mode FROM service_metadata WHERE singleton`).Scan(&storedMode); err != nil {
			return err
		}
		if storedMode != mode {
			return errors.New("database mode mismatch")
		}
		return tx.Commit(ctx)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	if _, err = tx.Exec(ctx, string(sql)); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO service_metadata(singleton,mode) VALUES(true,$1)`, mode); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO schema_migrations(version,checksum) VALUES(1,$1)`, checksum); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// DownForTest is intentionally unavailable outside a disposable trace_test_* schema.
func DownForTest(ctx context.Context, conn *pgx.Conn) error {
	var schema string
	if err := conn.QueryRow(ctx, `SELECT current_schema()`).Scan(&schema); err != nil {
		return err
	}
	if !strings.HasPrefix(schema, "trace_test_") {
		return errors.New("down migration requires an isolated trace_test_ schema")
	}
	sql, err := scripts.ReadFile("000001_init.down.sql")
	if err != nil {
		return err
	}
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, string(sql)); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM schema_migrations WHERE version=1`); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
