package migrations

import (
	"context"
	"database/sql"
	_ "embed"
)

//go:embed 000007_seed_pbl6_demo.up.sql
var demoSeed string

// ApplyDemoSeed installs the packaged demo data once, before background workers start.
// Existing schema migrations must already be applied.
func ApplyDemoSeed(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock(2026112041)"); err != nil {
		return err
	}
	var applied bool
	if err = tx.QueryRowContext(ctx, "SELECT to_regclass('inventory_demo_seed_000007') IS NOT NULL").Scan(&applied); err != nil {
		return err
	}
	if !applied {
		if _, err = tx.ExecContext(ctx, demoSeed); err != nil {
			return err
		}
	}
	return tx.Commit()
}
