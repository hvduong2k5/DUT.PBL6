CREATE TABLE IF NOT EXISTS supply_plans (
    id UUID PRIMARY KEY,
    plan_number VARCHAR(48) NOT NULL UNIQUE,
    status VARCHAR(16) NOT NULL CHECK (status IN ('DRAFT')),
    current_revision INTEGER NOT NULL CHECK (current_revision > 0),
    created_by VARCHAR(160) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    version BIGINT NOT NULL DEFAULT 1 CHECK (version > 0)
);

CREATE TABLE IF NOT EXISTS supply_plan_revisions (
    supply_plan_id UUID NOT NULL REFERENCES supply_plans(id) ON DELETE CASCADE,
    revision INTEGER NOT NULL CHECK (revision > 0),
    snapshot JSONB NOT NULL CHECK (jsonb_typeof(snapshot) = 'object'),
    created_by VARCHAR(160) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (supply_plan_id, revision)
);

CREATE INDEX IF NOT EXISTS supply_plans_status_created_idx ON supply_plans (status, created_at DESC);
