package migrations

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed 001_foundation.sql
var foundation string
var checksum = fmt.Sprintf("%x", sha256.Sum256([]byte(foundation)))

func Check(ctx context.Context, p *pgxpool.Pool) error {
	var got string
	if err := p.QueryRow(ctx, "SELECT checksum FROM notification_schema_migrations WHERE version=1").Scan(&got); err != nil {
		return err
	}
	if got != checksum {
		return errors.New("migration checksum mismatch")
	}
	return nil
}
func Apply(ctx context.Context, p *pgxpool.Pool) error {
	tx, err := p.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtext(current_database()),hashtext(current_schema()))"); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, "CREATE TABLE IF NOT EXISTS notification_schema_migrations(version integer PRIMARY KEY,checksum text NOT NULL)"); err != nil {
		return err
	}
	var got string
	err = tx.QueryRow(ctx, "SELECT checksum FROM notification_schema_migrations WHERE version=1").Scan(&got)
	if errors.Is(err, pgx.ErrNoRows) {
		if _, err = tx.Exec(ctx, foundation); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, "INSERT INTO notification_schema_migrations VALUES(1,$1)", checksum); err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else if got != checksum {
		return errors.New("migration checksum mismatch")
	}
	return tx.Commit(ctx)
}
