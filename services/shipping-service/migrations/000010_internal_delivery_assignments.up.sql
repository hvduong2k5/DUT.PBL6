CREATE TABLE internal_deliveries (
    shipment_id UUID PRIMARY KEY REFERENCES shipments(id),
    driver_id UUID NOT NULL CHECK (driver_id <> '00000000-0000-0000-0000-000000000000'),
    assigned_by UUID NOT NULL CHECK (assigned_by <> '00000000-0000-0000-0000-000000000000'),
    version BIGINT NOT NULL CHECK (version > 0),
    reason TEXT NOT NULL CHECK (char_length(reason) BETWEEN 1 AND 500),
    assigned_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX internal_deliveries_driver_idx ON internal_deliveries(driver_id, shipment_id);

-- Append-only assignment snapshots: retries return the original result, not current state.
CREATE TABLE internal_delivery_assignment_events (
    id UUID PRIMARY KEY,
    shipment_id UUID NOT NULL REFERENCES shipments(id),
    driver_id UUID NOT NULL CHECK (driver_id <> '00000000-0000-0000-0000-000000000000'),
    previous_driver_id UUID,
    assigned_by UUID NOT NULL CHECK (assigned_by <> '00000000-0000-0000-0000-000000000000'),
    version BIGINT NOT NULL CHECK (version > 0),
    reason TEXT NOT NULL CHECK (char_length(reason) BETWEEN 1 AND 500),
    assigned_at TIMESTAMPTZ NOT NULL,
    idempotency_key TEXT NOT NULL CHECK (char_length(idempotency_key) BETWEEN 8 AND 128),
    request_hash TEXT NOT NULL CHECK (request_hash ~ '^[0-9a-f]{64}$'),
    UNIQUE(assigned_by, idempotency_key),
    UNIQUE(shipment_id, version)
);
