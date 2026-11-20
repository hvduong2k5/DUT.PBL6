CREATE TABLE IF NOT EXISTS return_shipments (
    id UUID PRIMARY KEY,
    shipment_number TEXT NOT NULL UNIQUE,
    return_case_id TEXT NOT NULL,
    order_id TEXT NOT NULL,
    package_id TEXT NOT NULL,
    idempotency_key TEXT NOT NULL UNIQUE,
    request_hash TEXT NOT NULL,
    carrier_code TEXT NOT NULL,
    service_code TEXT NOT NULL,
    pickup_address JSONB NOT NULL,
    destination JSONB NOT NULL,
    parcel JSONB NOT NULL,
    reason_code TEXT NOT NULL,
    shipping_notes TEXT NOT NULL DEFAULT '',
    carrier_order_id TEXT NOT NULL DEFAULT '',
    tracking_code TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL CHECK (status IN ('CREATING', 'CREATED', 'CREATION_FAILED')),
    cod_amount_vnd BIGINT NOT NULL DEFAULT 0 CHECK (cod_amount_vnd = 0),
    fee_payer TEXT NOT NULL DEFAULT 'SHOP' CHECK (fee_payer = 'SHOP'),
    provider_estimated_fee_vnd BIGINT NOT NULL DEFAULT 0 CHECK (provider_estimated_fee_vnd >= 0),
    failure_code TEXT NOT NULL DEFAULT '',
    failure_reason TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    revision BIGINT NOT NULL CHECK (revision > 0),
    CONSTRAINT return_shipments_case_package_unique UNIQUE (return_case_id, package_id)
);

CREATE UNIQUE INDEX IF NOT EXISTS return_shipments_carrier_tracking_unique
    ON return_shipments (carrier_code, tracking_code)
    WHERE tracking_code <> '';

CREATE INDEX IF NOT EXISTS return_shipments_order_idx
    ON return_shipments (order_id, created_at);

CREATE INDEX IF NOT EXISTS return_shipments_case_idx
    ON return_shipments (return_case_id, created_at);
