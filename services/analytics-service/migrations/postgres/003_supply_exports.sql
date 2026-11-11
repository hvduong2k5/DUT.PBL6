SET LOCAL search_path = analytics_config, public;

CREATE TABLE supply_export (
    scope_key text NOT NULL,
    pipeline text NOT NULL,
    sku_code text NOT NULL,
    warehouse_id text NOT NULL,
    kind text NOT NULL CHECK (kind IN ('order_day', 'inventory_day', 'inventory_batches')),
    business_date date NOT NULL,
    revision integer NOT NULL CHECK (revision > 0),
    source_as_of timestamptz NOT NULL,
    approved_by text NOT NULL,
    evidence_ref text NOT NULL,
    payload jsonb NOT NULL CHECK (jsonb_typeof(payload) = 'object'),
    content_digest bytea NOT NULL CHECK (octet_length(content_digest) = 32),
    imported_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY (scope_key, pipeline, sku_code, warehouse_id, kind, business_date, revision)
);

CREATE INDEX supply_export_warehouse_sku_idx
    ON supply_export (scope_key, pipeline, warehouse_id, sku_code);
