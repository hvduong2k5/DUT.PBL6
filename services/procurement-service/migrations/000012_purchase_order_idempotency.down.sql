DROP INDEX IF EXISTS purchase_orders_created_by_idempotency_key_unique;
ALTER TABLE purchase_orders DROP CONSTRAINT IF EXISTS purchase_orders_idempotency_fields_check;
ALTER TABLE purchase_orders DROP COLUMN IF EXISTS idempotency_key, DROP COLUMN IF EXISTS idempotency_request_hash;
