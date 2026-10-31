CREATE TABLE IF NOT EXISTS shipments (
    id UUID PRIMARY KEY,
    shipment_number TEXT NOT NULL UNIQUE,
    idempotency_key TEXT NOT NULL UNIQUE,
    order_id TEXT NOT NULL,
    package_id TEXT NOT NULL,
    seal_code TEXT NOT NULL DEFAULT '',
    carrier_code TEXT NOT NULL,
    service_code TEXT NOT NULL,
    carrier_order_id TEXT NOT NULL DEFAULT '',
    tracking_code TEXT NOT NULL DEFAULT '',
    label_reference TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL,
    recipient JSONB NOT NULL,
    parcel JSONB NOT NULL,
    cod_amount_vnd BIGINT NOT NULL DEFAULT 0 CHECK (cod_amount_vnd >= 0),
    checkout_fee_vnd BIGINT NOT NULL DEFAULT 0 CHECK (checkout_fee_vnd >= 0),
    provider_estimated_fee_vnd BIGINT NOT NULL DEFAULT 0 CHECK (provider_estimated_fee_vnd >= 0),
    actual_fee_vnd BIGINT NULL CHECK (actual_fee_vnd IS NULL OR actual_fee_vnd >= 0),
    failure_code TEXT NOT NULL DEFAULT '',
    failure_reason TEXT NOT NULL DEFAULT '',
    estimated_pickup_at TIMESTAMPTZ NULL,
    estimated_delivery_at TIMESTAMPTZ NULL,
    handed_over_at TIMESTAMPTZ NULL,
    delivered_at TIMESTAMPTZ NULL,
    last_event_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    revision BIGINT NOT NULL CHECK (revision > 0),
    CONSTRAINT shipments_order_package_unique UNIQUE (order_id, package_id),
    CONSTRAINT shipments_status_check CHECK (status IN (
        'CREATING', 'CREATION_FAILED', 'CREATED', 'READY_FOR_HANDOVER',
        'HANDED_OVER', 'IN_TRANSIT', 'OUT_FOR_DELIVERY', 'DELIVERED',
        'DELIVERY_FAILED', 'RETURNING', 'RETURNED', 'CANCELLED'
    ))
);

CREATE UNIQUE INDEX IF NOT EXISTS shipments_carrier_tracking_unique
    ON shipments (carrier_code, tracking_code)
    WHERE tracking_code <> '';

CREATE INDEX IF NOT EXISTS shipments_order_id_idx ON shipments (order_id, created_at);
CREATE INDEX IF NOT EXISTS shipments_status_idx ON shipments (status, updated_at);

CREATE TABLE IF NOT EXISTS tracking_events (
    id UUID PRIMARY KEY,
    shipment_id UUID NOT NULL REFERENCES shipments(id),
    external_event_id TEXT NOT NULL,
    carrier_code TEXT NOT NULL,
    provider_status_code TEXT NOT NULL,
    provider_status_label TEXT NOT NULL DEFAULT '',
    mapped_status TEXT NOT NULL,
    mapping_result TEXT NOT NULL,
    location TEXT NOT NULL DEFAULT '',
    public_description TEXT NOT NULL DEFAULT '',
    internal_note TEXT NOT NULL DEFAULT '',
    event_at TIMESTAMPTZ NOT NULL,
    received_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT tracking_events_source_unique UNIQUE (carrier_code, external_event_id),
    CONSTRAINT tracking_events_result_check CHECK (mapping_result IN ('APPLIED', 'DUPLICATE', 'STALE', 'REJECTED', 'UNMAPPED'))
);

CREATE INDEX IF NOT EXISTS tracking_events_shipment_timeline_idx
    ON tracking_events (shipment_id, event_at, received_at);

CREATE TABLE IF NOT EXISTS outbox_events (
    id UUID PRIMARY KEY,
    aggregate_id UUID NOT NULL,
    aggregate_type TEXT NOT NULL,
    event_type TEXT NOT NULL,
    topic TEXT NOT NULL,
    partition_key TEXT NOT NULL,
    payload JSONB NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL,
    published_at TIMESTAMPTZ NULL,
    attempts INTEGER NOT NULL DEFAULT 0,
    last_error TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS outbox_events_pending_idx
    ON outbox_events (occurred_at)
    WHERE published_at IS NULL;

CREATE TABLE IF NOT EXISTS schema_migrations (
    version TEXT PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
