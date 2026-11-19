package migrations

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

//go:embed 000001_order.up.sql
var Schema string

//go:embed 000002_observability.up.sql
var Observability string

//go:embed 000003_provider_cursor.up.sql
var ProviderCursor string

//go:embed 000004_core_foundation.up.sql
var CoreFoundation string

//go:embed 000005_sepay_payment.up.sql
var SepayPayment string

//go:embed 000006_connected_runtime.up.sql
var ConnectedRuntime string

//go:embed 000007_customer_projection.up.sql
var CustomerProjection string

//go:embed 000008_fulfillment_integration.up.sql
var FulfillmentIntegration string

//go:embed 000012_promotion_checkout.up.sql
var PromotionCheckout string

// Discover by filename so merging F-42 migrations 9-11 cannot renumber Promotion.
// Never infer persistent migration versions from a slice index.
//
//go:embed *.up.sql
var migrationFiles embed.FS

type migration struct {
	version int
	name    string
	ddl     string
}

func migrationPlan(source fs.FS) ([]migration, error) {
	names, err := fs.Glob(source, "*.up.sql")
	if err != nil {
		return nil, err
	}
	plan := make([]migration, 0, len(names))
	seen := map[int]bool{}
	for _, name := range names {
		prefix, _, ok := strings.Cut(name, "_")
		version, err := strconv.Atoi(prefix)
		if !ok || len(prefix) != 6 || err != nil || version < 1 || fmt.Sprintf("%06d", version) != prefix || seen[version] {
			return nil, fmt.Errorf("invalid or duplicate Order migration version: %s", name)
		}
		seen[version] = true
		ddl, err := fs.ReadFile(source, name)
		if err != nil {
			return nil, err
		}
		plan = append(plan, migration{version: version, name: name, ddl: string(ddl)})
	}
	return plan, nil
}

type migrationDB interface {
	Begin(context.Context) (pgx.Tx, error)
}

func Apply(ctx context.Context, db migrationDB) error {
	plan, e := migrationPlan(migrationFiles)
	if e != nil {
		return e
	}
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
	for _, step := range plan {
		version := step.version
		var done bool
		if e = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM order_migrations WHERE version=$1)", version).Scan(&done); e != nil {
			return e
		}
		if !done {
			if _, e = tx.Exec(ctx, step.ddl); e != nil {
				return fmt.Errorf("apply Order migration %s: %w", step.name, e)
			}
			if _, e = tx.Exec(ctx, "INSERT INTO order_migrations(version) VALUES($1)", version); e != nil {
				return e
			}
		}
	}
	return tx.Commit(ctx)
}
