ALTER TABLE supply_plans DROP CONSTRAINT IF EXISTS supply_plans_status_check;
ALTER TABLE supply_plans ADD CONSTRAINT supply_plans_status_check
    CHECK (status IN ('DRAFT', 'PENDING_APPROVAL', 'APPROVED', 'REJECTED'));

ALTER TABLE supply_plans
    ADD COLUMN submitted_by VARCHAR(160),
    ADD COLUMN submitted_at TIMESTAMPTZ,
    ADD COLUMN approved_by VARCHAR(160),
    ADD COLUMN approved_at TIMESTAMPTZ,
    ADD COLUMN rejected_by VARCHAR(160),
    ADD COLUMN rejected_at TIMESTAMPTZ,
    ADD COLUMN rejection_reason TEXT NOT NULL DEFAULT '';

CREATE TABLE supply_plan_status_history (
    id UUID PRIMARY KEY,
    supply_plan_id UUID NOT NULL REFERENCES supply_plans(id),
    from_status VARCHAR(24) NOT NULL,
    to_status VARCHAR(24) NOT NULL,
    action VARCHAR(32) NOT NULL,
    actor VARCHAR(160) NOT NULL,
    reason TEXT NOT NULL DEFAULT '',
    version BIGINT NOT NULL CHECK (version > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (supply_plan_id, version)
);

CREATE INDEX supply_plan_status_history_plan_created_idx
    ON supply_plan_status_history (supply_plan_id, created_at, id);
