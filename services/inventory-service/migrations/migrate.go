package migrations

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"embed"
	"fmt"
	"sort"
	"strings"
)

//go:embed *.up.sql
var scripts embed.FS

const migrationLock int64 = 2026112101

type migration struct{ version, name, checksum, sql string }

func load() ([]migration, error) {
	entries, err := scripts.ReadDir(".")
	if err != nil {
		return nil, err
	}
	var result []migration
	seen := map[string]bool{}
	for _, entry := range entries {
		name := entry.Name()
		version, _, ok := strings.Cut(name, "_")
		if !ok || len(version) != 6 || strings.Trim(version, "0123456789") != "" {
			return nil, fmt.Errorf("invalid migration filename %s", name)
		}
		if seen[version] {
			return nil, fmt.Errorf("duplicate migration version %s", version)
		}
		seen[version] = true
		data, err := scripts.ReadFile(name)
		if err != nil {
			return nil, err
		}
		normalized := strings.ReplaceAll(string(data), "\r\n", "\n")
		result = append(result, migration{version, name, fmt.Sprintf("%x", sha256.Sum256([]byte(normalized))), normalized})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].name < result[j].name })
	return result, nil
}

func history(ctx context.Context, tx *sql.Tx) (map[string]string, error) {
	rows, err := tx.QueryContext(ctx, "SELECT version, checksum FROM inventory_schema_migrations ORDER BY version")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	applied := map[string]string{}
	for rows.Next() {
		var version, checksum string
		if err := rows.Scan(&version, &checksum); err != nil {
			return nil, err
		}
		applied[version] = checksum
	}
	return applied, rows.Err()
}

func checkKnown(all []migration, applied map[string]string) error {
	known := map[string]string{}
	for _, m := range all {
		known[m.version] = m.checksum
	}
	for version, checksum := range applied {
		expected, ok := known[version]
		if !ok {
			return fmt.Errorf("unknown applied migration %s; use the matching release binary", version)
		}
		if checksum != expected {
			return fmt.Errorf("migration %s checksum mismatch: database=%s binary=%s; restore the original migration file", version, checksum, expected)
		}
	}
	return nil
}

// Apply runs pending migrations atomically. Legacy databases require explicit Baseline first.
func Apply(ctx context.Context, db *sql.DB) error { return run(ctx, db, false) }

// Baseline records 000001-000005 without executing their SQL. Operators must
// confirm these exact migrations have already been applied to the legacy database.
func Baseline(ctx context.Context, db *sql.DB) error { return run(ctx, db, true) }

func run(ctx context.Context, db *sql.DB, baseline bool) error {
	all, err := load()
	if err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock($1)", migrationLock); err != nil {
		return err
	}
	var tracked, legacy bool
	if err = tx.QueryRowContext(ctx, "SELECT to_regclass('inventory_schema_migrations') IS NOT NULL, to_regclass('inventory_items') IS NOT NULL").Scan(&tracked, &legacy); err != nil {
		return err
	}
	if baseline && (tracked || !legacy) {
		return fmt.Errorf("baseline requires an existing legacy inventory schema without inventory_schema_migrations")
	}
	if !baseline && legacy && !tracked {
		return fmt.Errorf("legacy Inventory database has no migration history; confirm migrations 000001-000005, then run inventory-migrate baseline before inventory-migrate")
	}
	if _, err = tx.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS inventory_schema_migrations (
 version varchar(6) PRIMARY KEY, checksum varchar(64) NOT NULL, applied_at timestamptz NOT NULL DEFAULT now())`); err != nil {
		return err
	}
	applied, err := history(ctx, tx)
	if err != nil {
		return err
	}
	if err = checkKnown(all, applied); err != nil {
		return err
	}
	if baseline {
		// Presence checks cannot prove historical SQL contents; operator attestation is required.
		if _, err = tx.ExecContext(ctx, `SELECT sku, physical_qty FROM inventory_items LIMIT 0;
SELECT version FROM batches LIMIT 0; SELECT order_id FROM inventory_terminal LIMIT 0;
SELECT actor_id, idem_key FROM stock_receipts LIMIT 0;
SELECT inventory_sellable_quantity(NULL,statement_timestamp());`); err != nil {
			return fmt.Errorf("legacy baseline schema validation failed: %w", err)
		}
	}
	pendingSeen := false
	for _, m := range all {
		if baseline && m.version > "000005" {
			break
		}
		if _, ok := applied[m.version]; ok {
			if pendingSeen {
				return fmt.Errorf("migration history has a gap before %s", m.version)
			}
			continue
		}
		pendingSeen = true
		if !baseline {
			if _, err = tx.ExecContext(ctx, m.sql); err != nil {
				return fmt.Errorf("apply %s: %w", m.name, err)
			}
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO inventory_schema_migrations(version,checksum) VALUES($1,$2)", m.version, m.checksum); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// Verify is read-only: the server never creates history or migrates the database.
func Verify(ctx context.Context, db *sql.DB) error {
	all, err := load()
	if err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var exists bool
	if err = tx.QueryRowContext(ctx, "SELECT to_regclass('inventory_schema_migrations') IS NOT NULL").Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("Inventory migration history missing; run inventory-migrate (legacy databases require baseline first)")
	}
	applied, err := history(ctx, tx)
	if err != nil {
		return err
	}
	if err = checkKnown(all, applied); err != nil {
		return err
	}
	for _, m := range all {
		if _, ok := applied[m.version]; !ok {
			return fmt.Errorf("Inventory schema missing migration %s; run inventory-migrate before starting the server", m.name)
		}
	}
	return tx.Commit()
}
