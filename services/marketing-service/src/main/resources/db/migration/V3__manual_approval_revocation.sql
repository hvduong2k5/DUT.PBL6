CREATE TABLE campaign_approval_revocations (
    approval_id UUID PRIMARY KEY REFERENCES approval_requests(id),
    actor_id UUID NOT NULL,
    request_id TEXT NOT NULL,
    reason VARCHAR(200) NOT NULL CHECK (btrim(reason) <> ''),
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);

ALTER TABLE campaign_executions DROP CONSTRAINT campaign_executions_state_check;
ALTER TABLE campaign_executions ADD CHECK (
    state IN ('SCHEDULED','ACTIVE','PAUSED','STOP_PENDING','COMPLETED','CANCELLED')
);
