package migrations

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPromotionMigrationPostgresAfterF42History(t *testing.T) {
	dsn := os.Getenv("ORDER_CORE_TEST_DATABASE")
	if dsn == "" {
		t.Skip("ORDER_CORE_TEST_DATABASE is required for PostgreSQL migration coverage")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := "promotion_migration_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	name := pgx.Identifier{schema}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+name); err != nil {
		t.Fatal(err)
	}
	defer func() {
		cleanup, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()
		if _, err := admin.Exec(cleanup, "DROP SCHEMA "+name+" CASCADE"); err != nil {
			t.Error(err)
		}
	}()
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema
	db, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, ddl := range []string{Schema, Observability, ProviderCursor, CoreFoundation, SepayPayment, ConnectedRuntime, CustomerProjection, FulfillmentIntegration, CaseDecisions, RefundAuthority, EvidenceVault, GuestCartMerge} {
		if _, err := db.Exec(ctx, ddl); err != nil {
			t.Fatal(err)
		}
	}
	// Reproduce the updated dev database, including real F-42 and guest-cart DDL.
	if _, err := db.Exec(ctx, "CREATE TABLE order_migrations(version integer PRIMARY KEY,applied_at timestamptz NOT NULL DEFAULT clock_timestamp()); INSERT INTO order_migrations(version) SELECT generate_series(1,12)"); err != nil {
		t.Fatal(err)
	}
	for _, oldPromotionApplied := range []bool{false, true} {
		if oldPromotionApplied {
			if _, err := db.Exec(ctx, "DELETE FROM order_migrations WHERE version=13"); err != nil {
				t.Fatal(err)
			}
		}
		if err := Apply(ctx, db); err != nil {
			t.Fatal(err)
		}
		var exists, recorded bool
		if err := db.QueryRow(ctx, "SELECT to_regclass('order_voucher_finalizations') IS NOT NULL, EXISTS(SELECT 1 FROM order_migrations WHERE version=13)").Scan(&exists, &recorded); err != nil || !exists || !recorded {
			t.Fatal("Promotion DDL was silently skipped", err)
		}
		if err := Apply(ctx, db); err != nil {
			t.Fatal("restart must be idempotent", err)
		}
	}
}
