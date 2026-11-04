CREATE TABLE checkout_snapshots (
    id UUID PRIMARY KEY,
    payload JSONB NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_checkout_snapshots_expiry ON checkout_snapshots(expires_at);
