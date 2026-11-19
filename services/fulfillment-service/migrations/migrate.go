package migrations

import (
	"context"
	"crypto/sha256"
	"embed"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"strings"
)

//go:embed *.sql
var files embed.FS

func Apply(ctx context.Context, pool *pgxpool.Pool) error {
	tx, e := pool.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(context.Background())
	if _, e = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(7268002)`); e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, `CREATE TABLE IF NOT EXISTS public.fulfillment_migrations(name text PRIMARY KEY, sha256 text NOT NULL, applied_at timestamptz NOT NULL DEFAULT clock_timestamp())`); e != nil {
		return e
	}
	entries, e := files.ReadDir(".")
	if e != nil {
		return e
	}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		raw, _ := files.ReadFile(entry.Name())
		hash := fmt.Sprintf("%x", sha256.Sum256(raw))
		var prior string
		rows, e := tx.Query(ctx, `SELECT sha256 FROM public.fulfillment_migrations WHERE name=$1`, entry.Name())
		if e != nil {
			return e
		}
		found := rows.Next()
		if found {
			e = rows.Scan(&prior)
		}
		rows.Close()
		if e != nil {
			return e
		}
		if found {
			if prior != hash {
				return fmt.Errorf("migration hash mismatch: %s", entry.Name())
			}
			continue
		}
		sql := strings.Replace(string(raw), "BEGIN;", "", 1)
		index := strings.LastIndex(sql, "COMMIT;")
		if index >= 0 {
			sql = sql[:index] + sql[index+7:]
		}
		if _, e = tx.Exec(ctx, sql); e != nil {
			return fmt.Errorf("migration %s: %w", entry.Name(), e)
		}
		if _, e = tx.Exec(ctx, `INSERT INTO public.fulfillment_migrations(name,sha256) VALUES($1,$2)`, entry.Name(), hash); e != nil {
			return e
		}
	}
	return tx.Commit(ctx)
}
