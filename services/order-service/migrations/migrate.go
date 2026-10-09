package migrations

import (
	"context"
	_ "embed"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed 000001_order.up.sql
var Schema string

//go:embed 000002_observability.up.sql
var Observability string

//go:embed 000003_provider_cursor.up.sql
var ProviderCursor string

func Apply(ctx context.Context, db *pgxpool.Pool) error {
	tx, e := db.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	if _, e = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(440030)"); e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, "CREATE TABLE IF NOT EXISTS order_migrations(version integer PRIMARY KEY, applied_at timestamptz NOT NULL DEFAULT clock_timestamp())"); e != nil {
		return e
	}
	for index, ddl := range []string{Schema, Observability, ProviderCursor} {
		version := index + 1
		var done bool
		if e = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM order_migrations WHERE version=$1)", version).Scan(&done); e != nil {
			return e
		}
		if !done {
			if _, e = tx.Exec(ctx, ddl); e != nil {
				return e
			}
			if _, e = tx.Exec(ctx, "INSERT INTO order_migrations(version) VALUES($1)", version); e != nil {
				return e
			}
		}
	}
	return tx.Commit(ctx)
}
