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

var versions = []string{"000001_init", "000002_warehouse_scope"}

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
	for index, name := range versions {
		version := index + 1
		sql, err := scripts.ReadFile(name + ".up.sql")
		if err != nil {
			return err
		}
		checksum := fmt.Sprintf("%x", sha256.Sum256(sql))
		var actual string
		err = tx.QueryRow(ctx, `SELECT checksum FROM schema_migrations WHERE version=$1`, version).Scan(&actual)
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
			continue
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		if _, err = tx.Exec(ctx, string(sql)); err != nil {
			return err
		}
		if version == 1 {
			if _, err = tx.Exec(ctx, `INSERT INTO service_metadata(singleton,mode) VALUES(true,$1)`, mode); err != nil {
				return err
			}
		}
		if _, err = tx.Exec(ctx, `INSERT INTO schema_migrations(version,checksum) VALUES($1,$2)`, version, checksum); err != nil {
			return err
		}
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
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())
	for index := len(versions) - 1; index >= 0; index-- {
		var exists bool
		if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=$1)`, index+1).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			continue
		}
		sql, err := scripts.ReadFile(versions[index] + ".down.sql")
		if err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, string(sql)); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `DELETE FROM schema_migrations WHERE version=$1`, index+1); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
