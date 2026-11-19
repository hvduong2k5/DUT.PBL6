package migrations

import (
	"context"
	"embed"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed *.up.sql
var files embed.FS

func Apply(ctx context.Context, pool *pgxpool.Pool) ([]string, error) {
	entries, err := files.ReadDir(".")
	if err != nil {
		return nil, err
	}
	var names []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".up.sql") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	var applied []string
	for _, name := range names {
		sqlBytes, err := files.ReadFile(name)
		if err != nil {
			return applied, err
		}
		tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
		if err != nil {
			return applied, err
		}
		if _, err = tx.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT now())`); err != nil {
			_ = tx.Rollback(ctx)
			return applied, err
		}
		var exists bool
		if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=$1)`, name).Scan(&exists); err != nil {
			_ = tx.Rollback(ctx)
			return applied, err
		}
		if exists {
			_ = tx.Rollback(ctx)
			continue
		}
		if _, err = tx.Exec(ctx, string(sqlBytes)); err != nil {
			_ = tx.Rollback(ctx)
			return applied, fmt.Errorf("apply migration %s: %w", name, err)
		}
		if _, err = tx.Exec(ctx, `INSERT INTO schema_migrations(version) VALUES ($1)`, name); err != nil {
			_ = tx.Rollback(ctx)
			return applied, err
		}
		if err = tx.Commit(ctx); err != nil {
			return applied, err
		}
		applied = append(applied, name)
	}
	return applied, nil
}
