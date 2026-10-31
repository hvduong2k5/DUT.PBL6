package testhelper

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"
)

// SetupPostgresContainer boots a real PostgreSQL 16 container and initializes the DDL schema.
func SetupPostgresContainer(t *testing.T) (*pgxpool.Pool, func()) {
	ctx := context.Background()

	pgContainer, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("profile_db"),
		tcpostgres.WithUsername("omamx_user"),
		tcpostgres.WithPassword("omamx_password"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	require.NoError(t, err, "Failed to start Postgres container")

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err, "Failed to get Postgres connection string")

	poolConfig, err := pgxpool.ParseConfig(connStr)
	require.NoError(t, err, "Failed to parse pgxpool config")
	poolConfig.MaxConns = 50

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	require.NoError(t, err, "Failed to connect to Postgres pool")

	// Execute DDL Schema
	_, source, _, _ := runtime.Caller(0)
	files, err := filepath.Glob(filepath.Join(filepath.Dir(source), "../../migrations/*.up.sql"))
	require.NoError(t, err)
	sort.Strings(files)
	for _, file := range files {
		data, err := os.ReadFile(file)
		require.NoError(t, err)
		_, err = pool.Exec(ctx, string(data))
		require.NoError(t, err, "migration %s", file)
	}

	cleanup := func() {
		pool.Close()
		_ = pgContainer.Terminate(context.Background())
	}

	return pool, cleanup
}

// SetupRedisContainer boots a real Redis 7 container.
func SetupRedisContainer(t *testing.T) (*redis.Client, func()) {
	ctx := context.Background()

	redisContainer, err := tcredis.Run(ctx,
		"redis:7-alpine",
		testcontainers.WithWaitStrategy(
			wait.ForLog("Ready to accept connections").
				WithStartupTimeout(30*time.Second),
		),
	)
	require.NoError(t, err, "Failed to start Redis container")

	endpoint, err := redisContainer.Endpoint(ctx, "")
	require.NoError(t, err, "Failed to get Redis endpoint")

	client := redis.NewClient(&redis.Options{
		Addr: endpoint,
	})

	err = client.Ping(ctx).Err()
	require.NoError(t, err, "Failed to ping Redis container")

	cleanup := func() {
		_ = client.Close()
		_ = redisContainer.Terminate(context.Background())
	}

	return client, cleanup
}

// SeedSampleAdministrativeUnits populates Huế administrative units (post-July 2025: Municipality -> Ward)
func SeedSampleAdministrativeUnits(ctx context.Context, pool *pgxpool.Pool) error {
	queries := []string{
		`INSERT INTO administrative_units (code, name, name_en, full_name, parent_code, level) 
		 VALUES ('75', 'Huế', 'Hue', 'Thành phố Huế', NULL, 'PROVINCE') 
		 ON CONFLICT (code) DO NOTHING;`,
		`INSERT INTO administrative_units (code, name, name_en, full_name, parent_code, level) 
		 VALUES ('WARD-TH-001', 'Thuận Hòa', 'Thuan Hoa', 'Phường Thuận Hòa', '75', 'WARD') 
		 ON CONFLICT (code) DO NOTHING;`,
		`INSERT INTO administrative_units (code, name, name_en, full_name, parent_code, level) 
		 VALUES ('WARD-VL-002', 'Vĩnh Lộc', 'Vinh Loc', 'Xã Vĩnh Lộc', '75', 'WARD') 
		 ON CONFLICT (code) DO NOTHING;`,
		`INSERT INTO administrative_units (code, name, name_en, full_name, parent_code, level) 
		 VALUES ('WARD-PB-003', 'Phú Bài', 'Phu Bai', 'Phường Phú Bài', '75', 'WARD') 
		 ON CONFLICT (code) DO NOTHING;`,
	}
	for _, q := range queries {
		if _, err := pool.Exec(ctx, q); err != nil {
			return fmt.Errorf("failed executing seed unit query: %w", err)
		}
	}
	return nil
}
