ALTER TABLE supply_plans DROP CONSTRAINT IF EXISTS supply_plans_status_check;
ALTER TABLE supply_plans ADD CONSTRAINT supply_plans_status_check
    CHECK (status IN ('DRAFT', 'PENDING_APPROVAL', 'APPROVED', 'REJECTED', 'CONVERTED'));
ALTER TABLE supply_plans
    ADD COLUMN converted_purchase_order_id UUID REFERENCES purchase_orders(id),
    ADD COLUMN converted_at TIMESTAMPTZ;

ALTER TABLE purchase_orders
    ADD COLUMN supply_plan_id UUID REFERENCES supply_plans(id),
    ADD COLUMN supply_plan_revision INTEGER;
ALTER TABLE purchase_orders ADD CONSTRAINT purchase_orders_supply_plan_revision_check
    CHECK ((supply_plan_id IS NULL AND supply_plan_revision IS NULL) OR (supply_plan_id IS NOT NULL AND supply_plan_revision IS NOT NULL AND supply_plan_revision > 0));
CREATE UNIQUE INDEX purchase_orders_supply_plan_unique_idx ON purchase_orders (supply_plan_id) WHERE supply_plan_id IS NOT NULL;
