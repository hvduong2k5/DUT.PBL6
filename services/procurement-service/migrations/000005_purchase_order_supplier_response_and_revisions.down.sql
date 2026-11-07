DROP TABLE IF EXISTS purchase_order_revision_snapshots;
ALTER TABLE purchase_orders DROP CONSTRAINT IF EXISTS purchase_orders_status_check;
ALTER TABLE purchase_orders ADD CONSTRAINT purchase_orders_status_check CHECK (status IN (
    'DRAFT','PENDING_APPROVAL','APPROVED','REJECTED','SENT','PARTIALLY_RECEIVED','RECEIVED','CANCELLED'
));
ALTER TABLE purchase_orders
    DROP COLUMN IF EXISTS revision,
    DROP COLUMN IF EXISTS supplier_response_note,
    DROP COLUMN IF EXISTS supplier_responded_by,
    DROP COLUMN IF EXISTS supplier_responded_at,
    DROP COLUMN IF EXISTS closed_by,
    DROP COLUMN IF EXISTS closed_at,
    DROP COLUMN IF EXISTS close_reason;
