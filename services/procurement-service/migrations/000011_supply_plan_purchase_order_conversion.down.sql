DROP INDEX IF EXISTS purchase_orders_supply_plan_unique_idx;
ALTER TABLE purchase_orders DROP CONSTRAINT IF EXISTS purchase_orders_supply_plan_revision_check;
ALTER TABLE supply_plans DROP CONSTRAINT IF EXISTS supply_plans_status_check;
UPDATE supply_plans SET status='APPROVED', converted_purchase_order_id=NULL, converted_at=NULL WHERE status='CONVERTED';
ALTER TABLE supply_plans ADD CONSTRAINT supply_plans_status_check
    CHECK (status IN ('DRAFT', 'PENDING_APPROVAL', 'APPROVED', 'REJECTED'));
ALTER TABLE purchase_orders DROP COLUMN IF EXISTS supply_plan_id, DROP COLUMN IF EXISTS supply_plan_revision;
ALTER TABLE supply_plans DROP COLUMN IF EXISTS converted_purchase_order_id, DROP COLUMN IF EXISTS converted_at;
