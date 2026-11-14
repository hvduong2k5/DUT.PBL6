CREATE TABLE procurement_receipt_inbox (
    event_id uuid PRIMARY KEY,
    processed_at timestamptz NOT NULL DEFAULT now()
);
ALTER TABLE batches ADD COLUMN supplier_lot_code text;
