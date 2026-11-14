CREATE TABLE campaign_executions (
    id UUID PRIMARY KEY,
    campaign_id UUID NOT NULL,
    revision_id UUID NOT NULL,
    approval_id UUID NOT NULL,
    channel VARCHAR(200) NOT NULL,
    starts_at TIMESTAMPTZ NOT NULL,
    ends_at TIMESTAMPTZ NOT NULL CHECK (ends_at > starts_at),
    state TEXT NOT NULL CHECK (state IN ('SCHEDULED','ACTIVE','PAUSED','COMPLETED','CANCELLED')),
    version BIGINT NOT NULL CHECK (version > 0),
    UNIQUE (revision_id, channel),
    FOREIGN KEY (campaign_id, revision_id, approval_id)
        REFERENCES approval_requests(campaign_id, revision_id, id)
);

CREATE TABLE campaign_execution_history (
    execution_id UUID NOT NULL REFERENCES campaign_executions(id),
    version BIGINT NOT NULL,
    actor_id UUID NOT NULL,
    request_id TEXT NOT NULL,
    from_state TEXT,
    to_state TEXT NOT NULL,
    reason VARCHAR(200),
    evidence_ref VARCHAR(200),
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY (execution_id, version)
);
