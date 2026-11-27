CREATE TABLE master_data_create_results (
    actor_id TEXT NOT NULL CHECK(length(actor_id)>0),
    idempotency_key VARCHAR(128) NOT NULL CHECK(length(idempotency_key) BETWEEN 16 AND 128 AND idempotency_key !~ '[^!-~]'),
    kind VARCHAR(16) NOT NULL CHECK(kind IN ('SUPPLIER','MATERIAL','OFFER')),
    resource_id UUID NOT NULL,
    request_hash VARCHAR(64) NOT NULL CHECK(request_hash ~ '^[0-9a-f]{64}$'),
    result_snapshot JSONB NOT NULL CHECK(jsonb_typeof(result_snapshot)='object' AND result_snapshot->>'id'=resource_id::text),
    completed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY(actor_id,idempotency_key)
);
-- Frozen completed resources, including actor attribution. No independent pruning.
