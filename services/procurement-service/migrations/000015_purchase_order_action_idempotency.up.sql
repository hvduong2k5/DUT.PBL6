CREATE TABLE purchase_order_action_results (
    actor_id TEXT NOT NULL CHECK (length(actor_id) > 0),
    idempotency_key VARCHAR(128) NOT NULL CHECK (length(idempotency_key) BETWEEN 16 AND 128 AND idempotency_key !~ '[^!-~]'),
    purchase_order_id UUID NOT NULL REFERENCES purchase_orders(id) ON DELETE RESTRICT,
    revision INTEGER NOT NULL CHECK (revision > 0),
    action VARCHAR(16) NOT NULL CHECK (action IN ('SUBMIT','APPROVE','REJECT','SEND')),
    expected_version BIGINT NOT NULL CHECK (expected_version > 0),
    request_hash VARCHAR(64) NOT NULL CHECK (request_hash ~ '^[0-9a-f]{64}$'),
    result_snapshot JSONB NOT NULL CHECK (jsonb_typeof(result_snapshot) = 'object'),
    completed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (actor_id, idempotency_key)
);

-- Only completed commands are recorded, in the PO/history transaction. No TTL:
-- deleting evidence would allow a previous key to be applied to a later revision.
