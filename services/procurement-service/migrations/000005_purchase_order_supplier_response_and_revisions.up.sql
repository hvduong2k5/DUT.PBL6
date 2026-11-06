ALTER TABLE purchase_orders DROP CONSTRAINT IF EXISTS purchase_orders_status_check;
ALTER TABLE purchase_orders ADD CONSTRAINT purchase_orders_status_check CHECK (status IN (
    'DRAFT','PENDING_APPROVAL','APPROVED','REJECTED','SENT','ACKNOWLEDGED',
    'CHANGE_REQUESTED','SUPPLIER_DECLINED','PARTIALLY_RECEIVED','RECEIVED','CANCELLED','CLOSED'
));

ALTER TABLE purchase_orders
    ADD COLUMN IF NOT EXISTS revision INTEGER NOT NULL DEFAULT 1 CHECK (revision > 0),
    ADD COLUMN IF NOT EXISTS supplier_response_note TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS supplier_responded_by VARCHAR(160) NULL,
    ADD COLUMN IF NOT EXISTS supplier_responded_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS closed_by VARCHAR(160) NULL,
    ADD COLUMN IF NOT EXISTS closed_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS close_reason TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS purchase_order_revision_snapshots (
    purchase_order_id UUID NOT NULL REFERENCES purchase_orders(id) ON DELETE CASCADE,
    revision INTEGER NOT NULL CHECK (revision > 0),
    snapshot JSONB NOT NULL CHECK (jsonb_typeof(snapshot) = 'object'),
    changed_by VARCHAR(160) NOT NULL,
    change_reason TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (purchase_order_id, revision)
);
