-- Enable pgcrypto for UUID generation if needed
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS inventory_items (
    sku VARCHAR(50) PRIMARY KEY,
    warehouse_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001',
    physical_qty INT NOT NULL DEFAULT 0 CHECK (physical_qty >= 0),
    reserved_qty INT NOT NULL DEFAULT 0 CHECK (reserved_qty >= 0),
    available_qty INT GENERATED ALWAYS AS (physical_qty - reserved_qty) STORED,
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT check_avail_qty CHECK (physical_qty >= reserved_qty)
);

CREATE TABLE IF NOT EXISTS batches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    batch_code VARCHAR(100) NOT NULL,
    sku VARCHAR(50) NOT NULL REFERENCES inventory_items(sku),
    supplier_id UUID NOT NULL,
    mfg_date DATE NOT NULL,
    exp_date DATE NOT NULL,
    physical_qty INT NOT NULL DEFAULT 0 CHECK (physical_qty >= 0),
    reserved_qty INT NOT NULL DEFAULT 0 CHECK (reserved_qty >= 0),
    status VARCHAR(20) NOT NULL DEFAULT 'QUARANTINE',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT check_dates CHECK (mfg_date < exp_date),
    CONSTRAINT check_batch_avail_qty CHECK (physical_qty >= reserved_qty)
);

CREATE TABLE IF NOT EXISTS stock_reservations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id VARCHAR(50) NOT NULL UNIQUE,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS stock_reservation_allocations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reservation_id UUID NOT NULL REFERENCES stock_reservations(id) ON DELETE CASCADE,
    sku VARCHAR(50) NOT NULL,
    batch_id UUID NOT NULL REFERENCES batches(id),
    allocated_qty INT NOT NULL CHECK (allocated_qty > 0)
);

CREATE TABLE IF NOT EXISTS stock_adjustments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    adjustment_code VARCHAR(50) NOT NULL UNIQUE,
    sku VARCHAR(50) NOT NULL REFERENCES inventory_items(sku),
    batch_id UUID REFERENCES batches(id),
    adjustment_type VARCHAR(30) NOT NULL,
    quantity_delta INT NOT NULL,
    reason TEXT,
    adjusted_by UUID NOT NULL,
    approved_by UUID,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS idempotency_keys (
    key_name VARCHAR(100) PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS outbox_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_type VARCHAR(50) NOT NULL,
    aggregate_id VARCHAR(100) NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    retry_count INT NOT NULL DEFAULT 0 CHECK (retry_count >= 0),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    processed_at TIMESTAMP WITH TIME ZONE,
    error_message TEXT
);

-- Indexes for FEFO and worker cleanup
CREATE INDEX IF NOT EXISTS idx_batches_fefo ON batches(sku, exp_date ASC) WHERE status IN ('ACTIVE', 'NEAR_EXPIRY');
CREATE INDEX IF NOT EXISTS idx_reservations_pending_expire ON stock_reservations(expires_at) WHERE status = 'PENDING';
CREATE INDEX IF NOT EXISTS idx_batches_expiry ON batches(exp_date) WHERE status IN ('ACTIVE', 'NEAR_EXPIRY');
CREATE INDEX IF NOT EXISTS idx_outbox_pending ON outbox_events(created_at ASC) WHERE status = 'PENDING';

