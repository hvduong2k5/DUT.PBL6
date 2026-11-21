// Package seed contains optional demo data, independent of schema migrations.
package seed

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"

	"dut-pbl6/inventory-service/migrations"
)

//go:embed sql/0001_demo_inventory.sql
var demo string

//go:embed sql/0001_demo_inventory_down.sql
var down string

// Apply is a no-op unless explicitly enabled. Existing SKUs are never replenished.
func Apply(ctx context.Context, db *sql.DB, enabled bool) error {
	if !enabled {
		return nil
	}
	if err := migrations.Verify(ctx, db); err != nil {
		return err
	}
	return execute(ctx, db, demo)
}

func Rollback(ctx context.Context, db *sql.DB) error { return execute(ctx, db, down) }

func execute(ctx context.Context, db *sql.DB, script string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, script); err != nil {
		return fmt.Errorf("demo seed: %w", err)
	}
	return tx.Commit()
}
