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
		sqlBytes, readErr := files.ReadFile(name)
		if readErr != nil {
			return applied, readErr
		}
		tx, beginErr := pool.BeginTx(ctx, pgx.TxOptions{})
		if beginErr != nil {
			return applied, beginErr
		}
		if _, execErr := tx.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT now())`); execErr != nil {
			_ = tx.Rollback(ctx)
			return applied, execErr
		}
		var exists bool
		if queryErr := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=$1)`, name).Scan(&exists); queryErr != nil {
			_ = tx.Rollback(ctx)
			return applied, queryErr
		}
		if exists {
			_ = tx.Rollback(ctx)
			continue
		}
		if _, execErr := tx.Exec(ctx, string(sqlBytes)); execErr != nil {
			_ = tx.Rollback(ctx)
			return applied, fmt.Errorf("apply migration %s: %w", name, execErr)
		}
		if _, execErr := tx.Exec(ctx, `INSERT INTO schema_migrations(version) VALUES ($1)`, name); execErr != nil {
			_ = tx.Rollback(ctx)
			return applied, execErr
		}
		if commitErr := tx.Commit(ctx); commitErr != nil {
			return applied, commitErr
		}
		applied = append(applied, name)
	}
	return applied, nil
}
