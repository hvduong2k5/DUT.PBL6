CREATE TABLE campaigns (
    id UUID PRIMARY KEY,
    department_id VARCHAR(200) NOT NULL CHECK (btrim(department_id) <> ''),
    current_revision_id UUID NOT NULL,
    next_revision_number BIGINT NOT NULL DEFAULT 2 CHECK (next_revision_number >= 2),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE campaign_revisions (
    id UUID PRIMARY KEY,
    campaign_id UUID NOT NULL REFERENCES campaigns(id),
    revision_no BIGINT NOT NULL CHECK (revision_no > 0),
    base_revision_id UUID,
    version BIGINT NOT NULL DEFAULT 1 CHECK (version > 0),
    state TEXT NOT NULL CHECK (state IN ('DRAFT', 'IN_REVIEW', 'APPROVED', 'REJECTED', 'NEEDS_CHANGES')),
    created_by UUID NOT NULL,
    owner_id UUID NOT NULL,
    name VARCHAR(200) NOT NULL,
    objective VARCHAR(200) NOT NULL,
    starts_at TIMESTAMPTZ NOT NULL,
    ends_at TIMESTAMPTZ NOT NULL CHECK (ends_at > starts_at),
    timezone VARCHAR(200) NOT NULL,
    planned_budget_vnd BIGINT NOT NULL CHECK (planned_budget_vnd BETWEEN 0 AND 9007199254740991),
    UNIQUE (campaign_id, revision_no),
    UNIQUE (campaign_id, id),
    FOREIGN KEY (campaign_id, base_revision_id) REFERENCES campaign_revisions(campaign_id, id)
);

ALTER TABLE campaigns ADD CONSTRAINT campaigns_current_revision_fk
    FOREIGN KEY (id, current_revision_id) REFERENCES campaign_revisions(campaign_id, id)
    DEFERRABLE INITIALLY DEFERRED;

CREATE INDEX campaigns_department_order_idx ON campaigns (department_id, updated_at DESC, id DESC);

CREATE TABLE channel_plans (
    revision_id UUID NOT NULL REFERENCES campaign_revisions(id),
    channel VARCHAR(200) NOT NULL,
    starts_at TIMESTAMPTZ NOT NULL,
    ends_at TIMESTAMPTZ NOT NULL CHECK (ends_at > starts_at),
    brief VARCHAR(4000) NOT NULL,
    PRIMARY KEY (revision_id, channel)
);

CREATE TABLE kpi_targets (
    revision_id UUID NOT NULL REFERENCES campaign_revisions(id),
    metric_code VARCHAR(200) NOT NULL,
    unit VARCHAR(200) NOT NULL,
    target NUMERIC(18, 6) NOT NULL CHECK (target >= 0),
    PRIMARY KEY (revision_id, metric_code)
);

CREATE TABLE product_references (
    revision_id UUID NOT NULL REFERENCES campaign_revisions(id),
    product_id UUID NOT NULL,
    slug VARCHAR(200) NOT NULL,
    PRIMARY KEY (revision_id, product_id)
);

CREATE TABLE approval_requests (
    id UUID PRIMARY KEY,
    campaign_id UUID NOT NULL,
    revision_id UUID NOT NULL,
    submitted_revision_version BIGINT NOT NULL CHECK (submitted_revision_version > 0),
    snapshot JSONB NOT NULL,
    snapshot_hash CHAR(64) NOT NULL,
    policy_version VARCHAR(100) NOT NULL,
    submitted_by UUID NOT NULL,
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    state TEXT NOT NULL CHECK (state IN ('OPEN', 'WITHDRAWN', 'APPROVED', 'REJECTED', 'NEEDS_CHANGES')),
    FOREIGN KEY (campaign_id, revision_id) REFERENCES campaign_revisions(campaign_id, id),
    UNIQUE (campaign_id, revision_id, id)
);

CREATE UNIQUE INDEX approval_requests_one_open_idx ON approval_requests(campaign_id) WHERE state = 'OPEN';
CREATE INDEX approval_requests_revision_idx ON approval_requests(revision_id, submitted_revision_version DESC);

CREATE TABLE approval_decisions (
    request_id UUID PRIMARY KEY REFERENCES approval_requests(id),
    action TEXT NOT NULL CHECK (action IN ('WITHDRAWN', 'APPROVED', 'REJECTED', 'NEEDS_CHANGES')),
    actor_id UUID NOT NULL,
    decided_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    reason VARCHAR(200),
    approved_budget_vnd BIGINT,
    CHECK (
        (action = 'APPROVED' AND approved_budget_vnd IS NOT NULL AND approved_budget_vnd >= 0)
        OR (action <> 'APPROVED' AND approved_budget_vnd IS NULL AND reason IS NOT NULL AND btrim(reason) <> '')
    )
);

CREATE TABLE audit_entries (
    id UUID PRIMARY KEY,
    campaign_id UUID NOT NULL,
    revision_id UUID NOT NULL,
    approval_request_id UUID,
    action TEXT NOT NULL,
    actor_id UUID NOT NULL,
    department_id VARCHAR(200) NOT NULL,
    request_id TEXT NOT NULL,
    from_state TEXT,
    to_state TEXT NOT NULL,
    revision_version BIGINT NOT NULL,
    reason VARCHAR(200),
    changed_fields TEXT[] NOT NULL DEFAULT '{}',
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    FOREIGN KEY (campaign_id, revision_id) REFERENCES campaign_revisions(campaign_id, id),
    FOREIGN KEY (campaign_id, revision_id, approval_request_id)
        REFERENCES approval_requests(campaign_id, revision_id, id)
);

CREATE TABLE idempotency_records (
    actor_id UUID NOT NULL,
    department_id VARCHAR(200) NOT NULL,
    action TEXT NOT NULL,
    idempotency_key VARCHAR(128) NOT NULL CHECK (idempotency_key ~ '^[!-~]{16,128}$'),
    request_hash CHAR(64) NOT NULL,
    response_status INTEGER NOT NULL CHECK (response_status IN (200, 201)),
    response_body JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (actor_id, department_id, action, idempotency_key)
);
