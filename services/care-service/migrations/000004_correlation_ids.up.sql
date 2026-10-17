BEGIN;

ALTER TABLE outbox_events
    ADD COLUMN correlation_id VARCHAR(100);

ALTER TABLE audit_logs
    ADD COLUMN correlation_id VARCHAR(100);

CREATE INDEX idx_outbox_correlation
    ON outbox_events (correlation_id, created_at)
    WHERE correlation_id IS NOT NULL;

CREATE INDEX idx_audit_correlation
    ON audit_logs (correlation_id, created_at)
    WHERE correlation_id IS NOT NULL;

COMMIT;
