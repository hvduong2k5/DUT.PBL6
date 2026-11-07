ALTER TABLE purchase_order_lines
    ADD COLUMN IF NOT EXISTS accepted_quantity NUMERIC(18, 6) NOT NULL DEFAULT 0
    CHECK (accepted_quantity >= 0 AND accepted_quantity <= received_quantity AND accepted_quantity <= quantity);
ALTER TABLE purchase_order_lines DROP CONSTRAINT IF EXISTS purchase_order_lines_received_lte_ordered;

CREATE TABLE IF NOT EXISTS goods_receipts (
    id UUID PRIMARY KEY,
    receipt_number VARCHAR(48) NOT NULL UNIQUE,
    purchase_order_id UUID NOT NULL REFERENCES purchase_orders(id),
    purchase_order_number VARCHAR(48) NOT NULL,
    supplier_id UUID NOT NULL REFERENCES suppliers(id),
    supplier_name VARCHAR(240) NOT NULL,
    warehouse_code VARCHAR(80) NOT NULL,
    status VARCHAR(16) NOT NULL CHECK (status IN ('DRAFT', 'INSPECTED')),
    lines JSONB NOT NULL CHECK (jsonb_typeof(lines) = 'array' AND jsonb_array_length(lines) > 0),
    received_by VARCHAR(160) NOT NULL,
    inspected_by VARCHAR(160) NULL,
    inspected_at TIMESTAMPTZ NULL,
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    version BIGINT NOT NULL DEFAULT 1 CHECK (version > 0),
    CONSTRAINT goods_receipts_inspection_state CHECK ((status = 'DRAFT' AND inspected_by IS NULL AND inspected_at IS NULL) OR (status = 'INSPECTED' AND inspected_by IS NOT NULL AND inspected_at IS NOT NULL))
);

CREATE INDEX IF NOT EXISTS goods_receipts_po_created_idx ON goods_receipts (purchase_order_id, created_at DESC);
CREATE INDEX IF NOT EXISTS goods_receipts_status_created_idx ON goods_receipts (status, created_at DESC);

CREATE TABLE IF NOT EXISTS goods_receipt_status_history (
    id UUID PRIMARY KEY,
    goods_receipt_id UUID NOT NULL REFERENCES goods_receipts(id) ON DELETE CASCADE,
    from_status VARCHAR(16) NULL,
    to_status VARCHAR(16) NOT NULL,
    action VARCHAR(32) NOT NULL,
    actor_id VARCHAR(160) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS goods_receipt_history_timeline_idx ON goods_receipt_status_history (goods_receipt_id, created_at, id);
