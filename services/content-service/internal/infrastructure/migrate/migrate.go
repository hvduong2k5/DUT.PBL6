// Package migrate applies the service-owned SQL migrations in order and records
// them in schema_migrations. Files must be named NNNNNN_name.{up,down}.sql and
// be wrapped in BEGIN; ... COMMIT; so they also run unchanged through psql. The
// runner strips that wrapper and applies each file plus its bookkeeping row in
// one transaction.
package migrate

import (
	"context"
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Arbitrary constant that serializes concurrent runners (several replicas).
const lockID int64 = 7010_000_010

var name = regexp.MustCompile(`^(\d{6})_[a-z0-9_]+\.(up|down)\.sql$`)

type file struct {
	version int64
	sql     string
}

func load(fsys fs.FS, direction string) ([]file, error) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, err
	}
	var out []file
	for _, e := range entries {
		m := name.FindStringSubmatch(e.Name())
		if m == nil || m[2] != direction {
			continue
		}
		raw, err := fs.ReadFile(fsys, e.Name())
		if err != nil {
			return nil, err
		}
		body := strings.TrimSpace(string(raw))
		if !strings.HasPrefix(body, "BEGIN;") || !strings.HasSuffix(body, "COMMIT;") {
			return nil, fmt.Errorf("%s must be wrapped in BEGIN; and COMMIT;", e.Name())
		}
		body = strings.TrimSuffix(strings.TrimPrefix(body, "BEGIN;"), "COMMIT;")
		v, _ := strconv.ParseInt(m[1], 10, 64)
		out = append(out, file{version: v, sql: body})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].version < out[j].version })
	return out, nil
}

// Up applies every pending migration and returns the versions it applied.
func Up(ctx context.Context, pool *pgxpool.Pool, fsys fs.FS) ([]int64, error) {
	files, err := load(fsys, "up")
	if err != nil {
		return nil, err
	}
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Release()
	if _, err = conn.Exec(ctx, `SELECT pg_advisory_lock($1)`, lockID); err != nil {
		return nil, err
	}
	defer func() { _, _ = conn.Exec(context.Background(), `SELECT pg_advisory_unlock($1)`, lockID) }()
	if _, err = conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations(version BIGINT PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT now())`); err != nil {
		return nil, err
	}
	// A database created by psql or the old initdb mount already has version 1.
	if _, err = conn.Exec(ctx, `INSERT INTO schema_migrations(version) SELECT 1 WHERE to_regclass('articles') IS NOT NULL AND NOT EXISTS (SELECT 1 FROM schema_migrations) ON CONFLICT DO NOTHING`); err != nil {
		return nil, err
	}
	applied := map[int64]bool{}
	rows, err := conn.Query(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var v int64
		if err = rows.Scan(&v); err != nil {
			rows.Close()
			return nil, err
		}
		applied[v] = true
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, err
	}
	var done []int64
	for _, f := range files {
		if applied[f.version] {
			continue
		}
		tx, err := conn.Begin(ctx)
		if err != nil {
			return done, err
		}
		if _, err = tx.Exec(ctx, f.sql); err == nil {
			_, err = tx.Exec(ctx, `INSERT INTO schema_migrations(version) VALUES($1)`, f.version)
		}
		if err != nil {
			_ = tx.Rollback(ctx)
			return done, fmt.Errorf("migration %06d failed: %w", f.version, err)
		}
		if err = tx.Commit(ctx); err != nil {
			return done, err
		}
		done = append(done, f.version)
	}
	return done, nil
}

// Down reverts the most recently applied migration and returns its version, or
// 0 when nothing is applied. It destroys data: use it only on disposable
// databases or with an approved backup.
func Down(ctx context.Context, pool *pgxpool.Pool, fsys fs.FS) (int64, error) {
	files, err := load(fsys, "down")
	if err != nil {
		return 0, err
	}
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return 0, err
	}
	defer conn.Release()
	if _, err = conn.Exec(ctx, `SELECT pg_advisory_lock($1)`, lockID); err != nil {
		return 0, err
	}
	defer func() { _, _ = conn.Exec(context.Background(), `SELECT pg_advisory_unlock($1)`, lockID) }()
	var latest *int64
	if err = conn.QueryRow(ctx, `SELECT max(version) FROM schema_migrations`).Scan(&latest); err != nil || latest == nil {
		return 0, err
	}
	for _, f := range files {
		if f.version != *latest {
			continue
		}
		tx, err := conn.Begin(ctx)
		if err != nil {
			return 0, err
		}
		if _, err = tx.Exec(ctx, f.sql); err == nil {
			_, err = tx.Exec(ctx, `DELETE FROM schema_migrations WHERE version=$1`, f.version)
		}
		if err != nil {
			_ = tx.Rollback(ctx)
			return 0, fmt.Errorf("rollback of %06d failed: %w", f.version, err)
		}
		return f.version, tx.Commit(ctx)
	}
	return 0, fmt.Errorf("no down migration for version %06d", *latest)
}
