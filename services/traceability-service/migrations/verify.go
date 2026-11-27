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
	for index, name := range versions {
		sql, err := scripts.ReadFile(name + ".up.sql")
		if err != nil {
			return err
		}
		expected := fmt.Sprintf("%x", sha256.Sum256(sql))
		var checksum, storedMode string
		var count int
		err = conn.QueryRow(ctx, `SELECT checksum,(SELECT mode FROM service_metadata WHERE singleton),(SELECT count(*) FROM schema_migrations) FROM schema_migrations WHERE version=$1`, index+1).Scan(&checksum, &storedMode, &count)
		if err != nil {
			return err
		}
		if count != len(versions) || checksum != expected || storedMode != mode {
			return errors.New("database schema checksum, version or mode mismatch")
		}
	}
	return nil
}

func RequireWarehouseWrites(ctx context.Context, conn *pgx.Conn) error {
	_, err := conn.Exec(ctx, `UPDATE service_metadata SET warehouse_writes_required=true WHERE singleton AND NOT warehouse_writes_required`)
	return err
}

func VerifyWarehouseWrites(ctx context.Context, conn *pgx.Conn) error {
	var enabled bool
	if err := conn.QueryRow(ctx, `SELECT warehouse_writes_required FROM service_metadata WHERE singleton`).Scan(&enabled); err != nil {
		return err
	}
	if !enabled {
		return errors.New("enable warehouse writer enforcement before scoped traffic")
	}
	return nil
}
