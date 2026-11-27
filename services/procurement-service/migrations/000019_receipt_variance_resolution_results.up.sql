CREATE TABLE receipt_variance_resolution_results (
    actor_id TEXT NOT NULL CHECK (length(actor_id)>0),
    idempotency_key VARCHAR(128) NOT NULL CHECK (length(idempotency_key) BETWEEN 16 AND 128 AND idempotency_key !~ '[^!-~]'),
    receipt_id UUID NOT NULL REFERENCES goods_receipts(id) ON DELETE RESTRICT,
    variance_id UUID NOT NULL,
    expected_version BIGINT NOT NULL CHECK (expected_version>0),
    request_hash VARCHAR(64) NOT NULL CHECK (request_hash ~ '^[0-9a-f]{64}$'),
    result_snapshot JSONB NOT NULL CHECK (jsonb_typeof(result_snapshot)='object'),
    completed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY(actor_id,idempotency_key)
);
-- Completed results commit with receipt/PO/history. No TTL or independent pruning.
