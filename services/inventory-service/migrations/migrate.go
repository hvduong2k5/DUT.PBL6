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
		if err = verifyLegacyObjects(ctx, tx); err != nil {
			return err
		}
		// Operator attestation is still required for historical data semantics.
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
	if err = verifyLegacyObjects(ctx, tx); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `SELECT sku,base_unit FROM inventory_procurement_sku_units LIMIT 0;
SELECT source,event_id,payload_hash,hash_version FROM procurement_event_ledger LIMIT 0;
SELECT actor_id,idem_key,batch_id,request,response FROM inventory_qc_releases LIMIT 0;
SELECT event_id,source,payload_hash,topic,partition_id,offset_id FROM procurement_receipt_inbox LIMIT 0;
SELECT supplier_lot_code,version,mfg_date,exp_date FROM batches LIMIT 0;
SELECT source,receipt_id,batch_id,base_unit,data_hash FROM procurement_receipts LIMIT 0;`); err != nil {
		return fmt.Errorf("Inventory schema is incomplete despite migration history; repair using the reviewed migration job: %w", err)
	}
	return tx.Commit()
}

func verifyLegacyObjects(ctx context.Context, tx *sql.Tx) error {
	for _, name := range []string{"inventory_items", "batches", "stock_reservations", "stock_reservation_allocations", "stock_adjustments", "idempotency_keys", "outbox_events", "inventory_terminal", "stock_receipts"} {
		var exists bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname=current_schema() AND c.relname=$1 AND c.relkind='r')`, name).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("Inventory schema incomplete: missing table %s", name)
		}
	}
	for _, signature := range []string{"inventory_cohort_guard()", "inventory_batch_revision()", "inventory_sellable_quantity(text,timestamp with time zone)", "inventory_emit_stock_snapshot(text)", "inventory_stock_snapshot_trigger()"} {
		var exists bool
		if err := tx.QueryRowContext(ctx, `SELECT to_regprocedure(format('%I.%s',current_schema(),$1::text)) IS NOT NULL`, signature).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("Inventory schema incomplete: missing function %s", signature)
		}
	}
	for _, pair := range [][2]string{{"stock_reservations", "inventory_legacy_cohort"}, {"batches", "inventory_batch_revision"}, {"inventory_items", "inventory_item_snapshot"}, {"batches", "inventory_batch_snapshot"}} {
		var exists bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM pg_trigger t WHERE t.tgrelid=to_regclass($1) AND t.tgname=$2 AND t.tgenabled IN ('O','A'))`, pair[0], pair[1]).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("Inventory schema incomplete: missing or disabled trigger %s", pair[1])
		}
	}
	return nil
}
