BEGIN;

DROP INDEX IF EXISTS idx_audit_correlation;
DROP INDEX IF EXISTS idx_outbox_correlation;

ALTER TABLE audit_logs
    DROP COLUMN IF EXISTS correlation_id;

ALTER TABLE outbox_events
    DROP COLUMN IF EXISTS correlation_id;

COMMIT;
