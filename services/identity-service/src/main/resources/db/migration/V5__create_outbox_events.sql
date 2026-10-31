CREATE TABLE outbox_events (
    event_id UUID PRIMARY KEY,
    aggregate_type VARCHAR(120) NOT NULL,
    aggregate_id VARCHAR(200) NOT NULL,
    event_type VARCHAR(160) NOT NULL,
    schema_version INTEGER NOT NULL CHECK (schema_version > 0),
    partition_key VARCHAR(200) NOT NULL,
    payload JSONB NOT NULL CHECK (jsonb_typeof(payload) = 'object'),
    delivery_status VARCHAR(16) NOT NULL DEFAULT 'PENDING'
        CHECK (delivery_status IN ('PENDING', 'PROCESSING', 'PUBLISHED', 'DEAD')),
    attempt_count INTEGER NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    locked_by VARCHAR(160),
    locked_until TIMESTAMPTZ,
    last_error VARCHAR(1000),
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    published_at TIMESTAMPTZ,
    CONSTRAINT ck_published_outbox_has_timestamp
        CHECK (delivery_status <> 'PUBLISHED' OR published_at IS NOT NULL)
);
CREATE INDEX ix_outbox_due_events
    ON outbox_events (next_attempt_at, created_at)
    WHERE delivery_status IN ('PENDING', 'PROCESSING');
CREATE INDEX ix_outbox_aggregate
    ON outbox_events (aggregate_type, aggregate_id, occurred_at);
CREATE INDEX ix_outbox_event_type_time
    ON outbox_events (event_type, occurred_at DESC);
