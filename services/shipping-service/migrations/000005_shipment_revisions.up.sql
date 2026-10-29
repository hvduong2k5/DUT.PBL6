CREATE TABLE IF NOT EXISTS shipment_revisions (
    shipment_id UUID NOT NULL REFERENCES shipments(id),
    resource_version BIGINT NOT NULL CHECK (resource_version > 0),
    order_id TEXT NOT NULL,
    package_id TEXT NOT NULL,
    ready_generation BIGINT NOT NULL CHECK (ready_generation > 0),
    kind TEXT NOT NULL,
    seal_code TEXT NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL,
    settlement_verified BOOLEAN NOT NULL DEFAULT FALSE,
    PRIMARY KEY (shipment_id, resource_version),
    CONSTRAINT shipment_revisions_kind_check CHECK (kind IN (
        'SHIPMENT_CREATED', 'SHIPMENT_DISPATCHED', 'SHIPMENT_DELIVERED', 'SHIPMENT_FAILED'
    )),
    CONSTRAINT shipment_revisions_seal_check CHECK (seal_code <> '')
);

CREATE INDEX IF NOT EXISTS shipment_revisions_order_idx
    ON shipment_revisions (order_id, occurred_at);
