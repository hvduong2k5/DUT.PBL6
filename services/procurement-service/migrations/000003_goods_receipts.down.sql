DROP TABLE IF EXISTS goods_receipt_status_history;
DROP TABLE IF EXISTS goods_receipts;
ALTER TABLE purchase_order_lines DROP COLUMN IF EXISTS accepted_quantity;
