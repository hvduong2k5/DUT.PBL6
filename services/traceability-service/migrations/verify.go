package migrations

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// Verify is read-only: runtime startup must never create or repair a schema.
func Verify(ctx context.Context, conn *pgx.Conn, mode string) error {
	if mode != "sandbox" && mode != "integration" {
		return errors.New("unsupported database mode")
	}
	sql, err := scripts.ReadFile("000001_init.up.sql")
	if err != nil {
		return err
	}
	expected := fmt.Sprintf("%x", sha256.Sum256(sql))
	var checksum, storedMode string
	var versions int
	err = conn.QueryRow(ctx, `SELECT checksum,(SELECT mode FROM service_metadata WHERE singleton),(SELECT count(*) FROM schema_migrations) FROM schema_migrations WHERE version=1`).Scan(&checksum, &storedMode, &versions)
	if err != nil {
		return err
	}
	if versions != 1 || checksum != expected || storedMode != mode {
		return errors.New("database schema checksum, version or mode mismatch")
	}
	return nil
}
