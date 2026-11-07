CREATE TABLE IF NOT EXISTS purchase_orders (
    id UUID PRIMARY KEY,
    po_number VARCHAR(48) NOT NULL UNIQUE,
    supplier_id UUID NOT NULL REFERENCES suppliers(id),
    supplier_name VARCHAR(240) NOT NULL,
    status VARCHAR(24) NOT NULL CHECK (status IN (
        'DRAFT', 'PENDING_APPROVAL', 'APPROVED', 'REJECTED', 'SENT',
        'PARTIALLY_RECEIVED', 'RECEIVED', 'CANCELLED'
    )),
    currency CHAR(3) NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    total_amount_minor BIGINT NOT NULL CHECK (total_amount_minor > 0),
    notes TEXT NOT NULL DEFAULT '',
    expected_delivery_date DATE NULL,
    created_by VARCHAR(160) NOT NULL,
    submitted_by VARCHAR(160) NULL,
    submitted_at TIMESTAMPTZ NULL,
    approved_by VARCHAR(160) NULL,
    approved_at TIMESTAMPTZ NULL,
    rejected_by VARCHAR(160) NULL,
    rejected_at TIMESTAMPTZ NULL,
    rejection_reason TEXT NOT NULL DEFAULT '',
    sent_by VARCHAR(160) NULL,
    sent_at TIMESTAMPTZ NULL,
    cancelled_by VARCHAR(160) NULL,
    cancelled_at TIMESTAMPTZ NULL,
    cancellation_reason TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    version BIGINT NOT NULL DEFAULT 1 CHECK (version > 0)
);

CREATE INDEX IF NOT EXISTS purchase_orders_supplier_created_idx ON purchase_orders (supplier_id, created_at DESC);
CREATE INDEX IF NOT EXISTS purchase_orders_status_created_idx ON purchase_orders (status, created_at DESC);

CREATE TABLE IF NOT EXISTS purchase_order_lines (
    id UUID PRIMARY KEY,
    purchase_order_id UUID NOT NULL REFERENCES purchase_orders(id) ON DELETE CASCADE,
    line_number INTEGER NOT NULL CHECK (line_number > 0),
    material_id UUID NOT NULL REFERENCES raw_materials(id),
    material_code VARCHAR(80) NOT NULL,
    material_name VARCHAR(240) NOT NULL,
    quantity NUMERIC(18, 6) NOT NULL CHECK (quantity > 0),
    received_quantity NUMERIC(18, 6) NOT NULL DEFAULT 0 CHECK (received_quantity >= 0),
    unit VARCHAR(24) NOT NULL,
    unit_price_minor BIGINT NOT NULL CHECK (unit_price_minor > 0),
    line_amount_minor BIGINT NOT NULL CHECK (line_amount_minor > 0),
    CONSTRAINT purchase_order_lines_number_unique UNIQUE (purchase_order_id, line_number),
    CONSTRAINT purchase_order_lines_material_unique UNIQUE (purchase_order_id, material_id),
    CONSTRAINT purchase_order_lines_received_lte_ordered CHECK (received_quantity <= quantity)
);

CREATE TABLE IF NOT EXISTS purchase_order_status_history (
    id UUID PRIMARY KEY,
    purchase_order_id UUID NOT NULL REFERENCES purchase_orders(id) ON DELETE CASCADE,
    from_status VARCHAR(24) NULL,
    to_status VARCHAR(24) NOT NULL,
    action VARCHAR(32) NOT NULL,
    actor_id VARCHAR(160) NOT NULL,
    reason TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS purchase_order_history_timeline_idx ON purchase_order_status_history (purchase_order_id, created_at, id);
