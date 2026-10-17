BEGIN;

ALTER TABLE ticket_sla_milestones
    ADD COLUMN near_breach_alerted_at TIMESTAMPTZ,
    ADD COLUMN breach_alerted_at TIMESTAMPTZ;

CREATE INDEX idx_ticket_sla_alert_candidates
    ON ticket_sla_milestones (due_at, id)
    WHERE completed_at IS NULL AND paused_at IS NULL;

COMMIT;
