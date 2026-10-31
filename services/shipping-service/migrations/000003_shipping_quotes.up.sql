CREATE SEQUENCE IF NOT EXISTS shipping_quote_version_seq;

CREATE TABLE IF NOT EXISTS shipping_quotes (
    quote_id UUID PRIMARY KEY,
    idempotency_key TEXT NOT NULL UNIQUE,
    contract_version TEXT NOT NULL,
    request_hash TEXT NOT NULL,
    destination JSONB NOT NULL,
    parcel JSONB NOT NULL,
    preferred_carrier TEXT NOT NULL DEFAULT '',
    requested_service TEXT NOT NULL DEFAULT '',
    origin_warehouse_id TEXT NOT NULL DEFAULT '',
    carrier_code TEXT NOT NULL,
    service_code TEXT NOT NULL,
    service_name TEXT NOT NULL,
    fee_vnd BIGINT NOT NULL CHECK (fee_vnd >= 0),
    estimated_delivery_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    quote_version BIGINT NOT NULL DEFAULT nextval('shipping_quote_version_seq') CHECK (quote_version > 0),
    CONSTRAINT shipping_quotes_contract_version_check CHECK (contract_version = 'order-3.2'),
    CONSTRAINT shipping_quotes_request_hash_check CHECK (request_hash ~ '^[0-9a-f]{64}$'),
    CONSTRAINT shipping_quotes_expiry_check CHECK (expires_at > created_at)
);

CREATE UNIQUE INDEX IF NOT EXISTS shipping_quotes_version_unique
    ON shipping_quotes (quote_version);
CREATE INDEX IF NOT EXISTS shipping_quotes_expiry_idx
    ON shipping_quotes (expires_at);
