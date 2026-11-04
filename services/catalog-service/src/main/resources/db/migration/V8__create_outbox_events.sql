-- Catalog Service — om_catalog_db
-- Transactional Outbox for events emitted by Catalog (e.g. ProductApproved).
-- Inventory has its own outbox in its own database; do not share that table.

CREATE TABLE outbox_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_type  VARCHAR(100) NOT NULL,
    aggregate_id    VARCHAR(100) NOT NULL,
    event_type      VARCHAR(150) NOT NULL,
    schema_version  SMALLINT NOT NULL DEFAULT 1
                    CHECK (schema_version > 0),
    payload         JSONB NOT NULL,
    occurred_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at    TIMESTAMPTZ NULL,
    attempts        INT NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    claim_token     UUID NULL,
    claimed_until   TIMESTAMPTZ NULL,
    last_error      TEXT NULL,

    CONSTRAINT chk_outbox_claim_pair
        CHECK ((claim_token IS NULL) = (claimed_until IS NULL))
);

CREATE INDEX idx_outbox_unpublished_due
    ON outbox_events(next_attempt_at, occurred_at)
    WHERE published_at IS NULL;

CREATE INDEX idx_outbox_aggregate_history
    ON outbox_events(aggregate_type, aggregate_id, occurred_at DESC);

COMMENT ON TABLE outbox_events IS
    'Catalog events persisted in the same DB transaction as the domain change, then published to Kafka';
COMMENT ON COLUMN outbox_events.aggregate_id IS
    'String form of the domain aggregate ID; kept flexible across event types';
COMMENT ON COLUMN outbox_events.claimed_until IS
    'Lease expiry for concurrent outbox publishers; NULL means not claimed';
COMMENT ON COLUMN outbox_events.next_attempt_at IS
    'Earliest timestamp at which the publisher may retry delivery';
