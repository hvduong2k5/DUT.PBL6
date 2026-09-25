package testhelper

import (
	"context"
	"fmt"
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

// DDLSchema defines the full schema for MS-15 profile_db according to the design specification
const DDLSchema = `
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- 1. ADMINISTRATIVE UNITS (2-TIER MODEL POST-JULY 2025: PROVINCE/MUNICIPALITY -> WARD/COMMUNE)
CREATE TABLE IF NOT EXISTS administrative_units (
    code VARCHAR(20) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    name_en VARCHAR(100),
    full_name VARCHAR(150) NOT NULL,
    parent_code VARCHAR(20) REFERENCES administrative_units(code),
    level VARCHAR(20) NOT NULL CHECK (level IN ('PROVINCE', 'WARD')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_admin_units_parent ON administrative_units(parent_code);
CREATE INDEX IF NOT EXISTS idx_admin_units_name_trgm ON administrative_units USING GIN (name gin_trgm_ops);

-- 2. CUSTOMER PROFILES
CREATE TABLE IF NOT EXISTS customer_profiles (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL UNIQUE,
    full_name VARCHAR(150) NOT NULL,
    phone_number VARCHAR(20) NOT NULL,
    email VARCHAR(255),
    date_of_birth DATE,
    gender VARCHAR(10) CHECK (gender IN ('MALE', 'FEMALE', 'OTHER', 'UNSPECIFIED')),
    avatar_url VARCHAR(500),
    preferences JSONB DEFAULT '{"favorite_products": [], "dietary_preference": "NORMAL", "allergy_alert": []}'::jsonb,
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'SUSPENDED', 'PENDING_VERIFICATION')),
    version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_customer_phone ON customer_profiles(phone_number);
CREATE INDEX IF NOT EXISTS idx_customer_email ON customer_profiles(email);
CREATE INDEX IF NOT EXISTS idx_customer_user_id ON customer_profiles(user_id);
CREATE INDEX IF NOT EXISTS idx_customer_preferences ON customer_profiles USING GIN (preferences);

-- 3. SHIPPING ADDRESSES (2-TIER STREAMLINED)
CREATE TABLE IF NOT EXISTS shipping_addresses (
    id UUID PRIMARY KEY,
    customer_id UUID NOT NULL REFERENCES customer_profiles(id) ON DELETE CASCADE,
    recipient_name VARCHAR(150) NOT NULL,
    phone_number VARCHAR(20) NOT NULL,
    street_address VARCHAR(255) NOT NULL,
    ward_code VARCHAR(20) NOT NULL REFERENCES administrative_units(code),
    ward_name VARCHAR(100) NOT NULL,
    province_code VARCHAR(20) NOT NULL REFERENCES administrative_units(code),
    province_name VARCHAR(100) NOT NULL,
    latitude DECIMAL(10, 7),
    longitude DECIMAL(10, 7),
    label VARCHAR(50) DEFAULT 'HOME' CHECK (label IN ('HOME', 'OFFICE', 'GIFT_RECIPIENT', 'OTHER')),
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_customer_default_address 
ON shipping_addresses (customer_id) 
WHERE is_default = TRUE AND is_deleted = FALSE;

CREATE INDEX IF NOT EXISTS idx_shipping_addresses_customer ON shipping_addresses(customer_id) WHERE is_deleted = FALSE;

-- 4. GUEST ORDER CLAIMS
CREATE TABLE IF NOT EXISTS guest_order_claims (
    id UUID PRIMARY KEY,
    customer_id UUID NOT NULL REFERENCES customer_profiles(id) ON DELETE CASCADE,
    order_id VARCHAR(50) NOT NULL,
    phone_number VARCHAR(20) NOT NULL,
    claimed_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    claim_status VARCHAR(20) NOT NULL DEFAULT 'VERIFIED' CHECK (claim_status IN ('VERIFIED', 'REVOKED'))
);

CREATE INDEX IF NOT EXISTS idx_guest_claims_phone ON guest_order_claims(phone_number);

CREATE UNIQUE INDEX IF NOT EXISTS uq_order_claim_active 
ON guest_order_claims(order_id) 
WHERE claim_status = 'VERIFIED';

-- 5. DEPARTMENTS
CREATE TABLE IF NOT EXISTS departments (
    id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(150) NOT NULL,
    description TEXT,
    manager_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 6. EMPLOYEE PROFILES (HR)
CREATE TABLE IF NOT EXISTS employee_profiles (
    id UUID PRIMARY KEY,
    user_id UUID UNIQUE,
    employee_code VARCHAR(50) NOT NULL UNIQUE,
    full_name VARCHAR(150) NOT NULL,
    phone_number VARCHAR(20) NOT NULL,
    id_card_encrypted BYTEA NOT NULL,
    id_card_nonce BYTEA NOT NULL,
    encrypted_dek BYTEA NOT NULL,
    kek_version INT NOT NULL DEFAULT 1,
    department_id VARCHAR(50) NOT NULL REFERENCES departments(id),
    position VARCHAR(100) NOT NULL,
    contract_type VARCHAR(20) NOT NULL CHECK (contract_type IN ('FULLTIME', 'PARTTIME', 'SEASONAL')),
    contract_start_date DATE NOT NULL,
    contract_end_date DATE,
    food_safety_cert_no VARCHAR(100),
    food_safety_cert_expiry DATE,
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'ON_LEAVE', 'TERMINATED')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 7. OUTBOX EVENTS
CREATE TABLE IF NOT EXISTS outbox_events (
    id UUID PRIMARY KEY,
    aggregate_type VARCHAR(50) NOT NULL,
    aggregate_id VARCHAR(100) NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    topic VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    published_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_outbox_unpublished ON outbox_events(created_at) WHERE published_at IS NULL;
`

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
	_, err = pool.Exec(ctx, DDLSchema)
	require.NoError(t, err, "Failed to execute DDL schema migrations")

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
