CREATE TABLE IF NOT EXISTS goods_received_outbox (
    event_id TEXT PRIMARY KEY,
    topic VARCHAR(128) NOT NULL,
    message_key VARCHAR(160) NOT NULL,
    payload JSONB NOT NULL CHECK (jsonb_typeof(payload) = 'object'),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    claimed_until TIMESTAMPTZ NULL,
    attempts INTEGER NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    last_error TEXT NOT NULL DEFAULT '',
    published_at TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS goods_received_outbox_pending_idx
    ON goods_received_outbox (next_attempt_at, created_at)
    WHERE published_at IS NULL;
