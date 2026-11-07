package migrations

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed *.up.sql
var files embed.FS

type Applied struct {
	Version int64
	Name    string
}

type migration struct {
	version  int64
	name     string
	checksum string
	body     string
}

// Apply executes every unapplied up migration in filename order. The advisory
// transaction lock prevents two deployment jobs from migrating the same database
// concurrently. A checksum mismatch fails closed instead of silently accepting a
// modified migration that has already run.
func Apply(ctx context.Context, pool *pgxpool.Pool) ([]Applied, error) {
	migrations, err := load()
	if err != nil {
		return nil, err
	}

	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin migration transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('care-service-schema-migrations'))`); err != nil {
		return nil, fmt.Errorf("lock Care schema migrations: %w", err)
	}
	var ledgerExists, ownedSchemaExists bool
	if err := tx.QueryRow(ctx, `
		SELECT to_regclass(current_schema() || '.care_schema_migrations') IS NOT NULL,
		       to_regclass(current_schema() || '.support_tickets') IS NOT NULL
	`).Scan(&ledgerExists, &ownedSchemaExists); err != nil {
		return nil, fmt.Errorf("inspect existing Care schema: %w", err)
	}
	if !ledgerExists && ownedSchemaExists {
		return nil, fmt.Errorf("existing Care tables have no migration ledger; verify and baseline them explicitly before using the runner")
	}
	if _, err := tx.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS care_schema_migrations (
			version BIGINT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			checksum_sha256 CHAR(64) NOT NULL,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`); err != nil {
		return nil, fmt.Errorf("create migration ledger: %w", err)
	}

	applied := make([]Applied, 0, len(migrations))
	for _, item := range migrations {
		var storedName, storedChecksum string
		err := tx.QueryRow(ctx, `
			SELECT name, checksum_sha256
			FROM care_schema_migrations
			WHERE version = $1
		`, item.version).Scan(&storedName, &storedChecksum)
		switch {
		case err == nil:
			if storedName != item.name || storedChecksum != item.checksum {
				return nil, fmt.Errorf("migration %06d checksum or name differs from the applied version", item.version)
			}
			continue
		case err != pgx.ErrNoRows:
			return nil, fmt.Errorf("read migration %06d state: %w", item.version, err)
		}

		if _, err := tx.Exec(ctx, item.body); err != nil {
			return nil, fmt.Errorf("apply migration %s: %w", item.name, err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO care_schema_migrations (version, name, checksum_sha256)
			VALUES ($1, $2, $3)
		`, item.version, item.name, item.checksum); err != nil {
			return nil, fmt.Errorf("record migration %s: %w", item.name, err)
		}
		applied = append(applied, Applied{Version: item.version, Name: item.name})
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit Care schema migrations: %w", err)
	}
	return applied, nil
}

func load() ([]migration, error) {
	entries, err := files.ReadDir(".")
	if err != nil {
		return nil, fmt.Errorf("list embedded migrations: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })

	result := make([]migration, 0, len(entries))
	var previous int64
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".up.sql") {
			continue
		}
		separator := strings.IndexByte(name, '_')
		if separator < 1 {
			return nil, fmt.Errorf("invalid migration filename %q", name)
		}
		version, err := strconv.ParseInt(name[:separator], 10, 64)
		if err != nil || version <= previous {
			return nil, fmt.Errorf("invalid or duplicate migration version in %q", name)
		}
		contents, err := files.ReadFile(name)
		if err != nil {
			return nil, fmt.Errorf("read embedded migration %s: %w", name, err)
		}
		body, err := stripTransactionWrapper(string(contents))
		if err != nil {
			return nil, fmt.Errorf("migration %s: %w", name, err)
		}
		digest := sha256.Sum256(contents)
		result = append(result, migration{
			version: version, name: name, body: body, checksum: hex.EncodeToString(digest[:]),
		})
		previous = version
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("no embedded up migrations found")
	}
	return result, nil
}

func stripTransactionWrapper(sql string) (string, error) {
	trimmed := strings.TrimSpace(sql)
	if len(trimmed) < len("BEGIN;")+len("COMMIT;") || !strings.EqualFold(trimmed[:len("BEGIN;")], "BEGIN;") {
		return "", fmt.Errorf("expected leading BEGIN statement")
	}
	trimmed = strings.TrimSpace(trimmed[len("BEGIN;"):])
	if len(trimmed) < len("COMMIT;") || !strings.EqualFold(trimmed[len(trimmed)-len("COMMIT;"):], "COMMIT;") {
		return "", fmt.Errorf("expected trailing COMMIT statement")
	}
	trimmed = strings.TrimSpace(trimmed[:len(trimmed)-len("COMMIT;")])
	if trimmed == "" {
		return "", fmt.Errorf("migration body is empty")
	}
	return trimmed, nil
}
