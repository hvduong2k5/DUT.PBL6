DROP TABLE IF EXISTS supply_plan_status_history;

UPDATE supply_plans SET status='DRAFT', submitted_by=NULL, submitted_at=NULL,
    approved_by=NULL, approved_at=NULL, rejected_by=NULL, rejected_at=NULL, rejection_reason=''
WHERE status <> 'DRAFT';

ALTER TABLE supply_plans DROP CONSTRAINT IF EXISTS supply_plans_status_check;
ALTER TABLE supply_plans ADD CONSTRAINT supply_plans_status_check CHECK (status IN ('DRAFT'));
ALTER TABLE supply_plans
    DROP COLUMN IF EXISTS submitted_by,
    DROP COLUMN IF EXISTS submitted_at,
    DROP COLUMN IF EXISTS approved_by,
    DROP COLUMN IF EXISTS approved_at,
    DROP COLUMN IF EXISTS rejected_by,
    DROP COLUMN IF EXISTS rejected_at,
    DROP COLUMN IF EXISTS rejection_reason;
