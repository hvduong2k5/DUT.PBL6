CREATE TABLE shipment_action_idempotency (
    idempotency_key TEXT PRIMARY KEY,
    shipment_id UUID NOT NULL REFERENCES shipments(id) ON DELETE CASCADE,
    action_type TEXT NOT NULL CHECK (action_type IN ('CANCEL', 'HANDOVER')),
    request_hash TEXT NOT NULL CHECK (request_hash ~ '^[0-9a-f]{64}$'),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT shipment_action_idempotency_once_per_action UNIQUE (shipment_id, action_type)
);
