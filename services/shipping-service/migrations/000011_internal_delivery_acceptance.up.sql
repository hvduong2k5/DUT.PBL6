-- Acceptance is bound to an immutable assignment version; reassignment invalidates it.
CREATE TABLE internal_delivery_acceptances (
    shipment_id UUID NOT NULL REFERENCES shipments(id),
    assignment_version BIGINT NOT NULL CHECK (assignment_version > 0),
    driver_id UUID NOT NULL CHECK (driver_id <> '00000000-0000-0000-0000-000000000000'),
    accepted_at TIMESTAMPTZ NOT NULL,
    idempotency_key TEXT NOT NULL CHECK (char_length(idempotency_key) BETWEEN 8 AND 128),
    request_hash TEXT NOT NULL CHECK (request_hash ~ '^[0-9a-f]{64}$'),
    PRIMARY KEY (shipment_id, assignment_version),
    UNIQUE (driver_id, idempotency_key),
    FOREIGN KEY (shipment_id, assignment_version)
        REFERENCES internal_delivery_assignment_events(shipment_id, version)
);
