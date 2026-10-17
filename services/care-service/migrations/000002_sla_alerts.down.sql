BEGIN;

DROP INDEX IF EXISTS idx_ticket_sla_alert_candidates;

ALTER TABLE ticket_sla_milestones
    DROP COLUMN IF EXISTS breach_alerted_at,
    DROP COLUMN IF EXISTS near_breach_alerted_at;

COMMIT;
