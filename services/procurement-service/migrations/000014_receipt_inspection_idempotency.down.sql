-- Removes retry evidence only; existing inspection/PO/outbox records remain.
ALTER TABLE goods_receipts
    DROP CONSTRAINT goods_receipts_inspection_idempotency_valid,
    DROP COLUMN inspection_idempotency_key,
    DROP COLUMN inspection_request_hash;
