ALTER TABLE outbox_events
    ADD COLUMN topic TEXT NOT NULL DEFAULT 'promotion.events.v1',
    ADD COLUMN partition_key TEXT NOT NULL DEFAULT '',
    ADD COLUMN locked_until TIMESTAMPTZ;

CREATE TABLE promotion_consumed_events (
    event_id TEXT PRIMARY KEY,
    event_type TEXT NOT NULL,
    aggregate_id TEXT NOT NULL,
    payload_hash TEXT NOT NULL,
    consumed_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);

