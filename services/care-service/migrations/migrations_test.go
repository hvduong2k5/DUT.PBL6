package migrations

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func TestStripTransactionWrapper(t *testing.T) {
	body, err := stripTransactionWrapper("\nBEGIN;\nCREATE TABLE example (id INT);\nCOMMIT;\n")
	require.NoError(t, err)
	require.Equal(t, "CREATE TABLE example (id INT);", body)
}

func TestStripTransactionWrapperRejectsUnwrappedMigration(t *testing.T) {
	_, err := stripTransactionWrapper("CREATE TABLE example (id INT);")
	require.Error(t, err)
}

func TestIntegrationMigrationRunnerIsIdempotent(t *testing.T) {
	databaseURL := os.Getenv("CARE_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set CARE_TEST_DATABASE_URL to run PostgreSQL integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	adminPool, err := pgxpool.New(ctx, databaseURL)
	require.NoError(t, err)
	t.Cleanup(adminPool.Close)
	require.NoError(t, adminPool.Ping(ctx))

	schema := "care_migrate_it_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	_, err = adminPool.Exec(ctx, `CREATE SCHEMA `+schema)
	require.NoError(t, err)
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, _ = adminPool.Exec(cleanupCtx, `DROP SCHEMA IF EXISTS `+schema+` CASCADE`)
	})

	config, err := pgxpool.ParseConfig(databaseURL)
	require.NoError(t, err)
	config.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, config)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	type result struct {
		applied []Applied
		err     error
	}
	results := make(chan result, 2)
	for range 2 {
		go func() {
			applied, applyErr := Apply(ctx, pool)
			results <- result{applied: applied, err: applyErr}
		}()
	}
	totalApplied := 0
	for range 2 {
		result := <-results
		require.NoError(t, result.err)
		totalApplied += len(result.applied)
	}
	require.Equal(t, 4, totalApplied)

	applied, err := Apply(ctx, pool)
	require.NoError(t, err)
	require.Empty(t, applied)

	var migrationCount int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM care_schema_migrations`).Scan(&migrationCount)
	require.NoError(t, err)
	require.Equal(t, 4, migrationCount)

	var alertColumns int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM information_schema.columns
		WHERE table_schema = $1 AND table_name = 'ticket_sla_milestones'
		  AND column_name IN ('near_breach_alerted_at', 'breach_alerted_at')
	`, schema).Scan(&alertColumns)
	require.NoError(t, err)
	require.Equal(t, 2, alertColumns)

	_, err = pool.Exec(ctx, `UPDATE care_schema_migrations SET checksum_sha256 = repeat('0', 64) WHERE version = 1`)
	require.NoError(t, err)
	_, err = Apply(ctx, pool)
	require.ErrorContains(t, err, "checksum or name differs")
}
