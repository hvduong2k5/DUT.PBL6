CREATE TABLE internal_delivery_progress_events (
    id UUID PRIMARY KEY,
    shipment_id UUID NOT NULL REFERENCES shipments(id),
    actor_id UUID NOT NULL CHECK (actor_id <> '00000000-0000-0000-0000-000000000000'),
    assignment_version BIGINT NOT NULL,
    revision BIGINT NOT NULL CHECK (revision > 0),
    status TEXT NOT NULL CHECK (status IN ('IN_TRANSIT','OUT_FOR_DELIVERY','DELIVERED','DELIVERY_FAILED','RETURNING','RETURNED')),
    reason TEXT NOT NULL CHECK (char_length(reason) BETWEEN 1 AND 500),
    occurred_at TIMESTAMPTZ NOT NULL,
    idempotency_key TEXT NOT NULL CHECK (char_length(idempotency_key) BETWEEN 8 AND 128),
    request_hash TEXT NOT NULL CHECK (request_hash ~ '^[0-9a-f]{64}$'),
    UNIQUE (actor_id, idempotency_key),
    UNIQUE (shipment_id, revision),
    FOREIGN KEY (shipment_id, assignment_version)
        REFERENCES internal_delivery_acceptances(shipment_id, assignment_version)
);
