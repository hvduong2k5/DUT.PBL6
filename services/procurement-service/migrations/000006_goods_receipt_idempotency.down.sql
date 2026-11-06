ALTER TABLE goods_receipts DROP CONSTRAINT IF EXISTS goods_receipts_idempotency_pair_check;
DROP INDEX IF EXISTS goods_receipts_received_by_idempotency_key_unique;
ALTER TABLE goods_receipts
    DROP COLUMN IF EXISTS idempotency_request_hash,
    DROP COLUMN IF EXISTS idempotency_key;
