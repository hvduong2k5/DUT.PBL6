ALTER TABLE goods_receipts
    ADD COLUMN IF NOT EXISTS idempotency_key VARCHAR(128) NULL,
    ADD COLUMN IF NOT EXISTS idempotency_request_hash CHAR(64) NULL;

CREATE UNIQUE INDEX IF NOT EXISTS goods_receipts_received_by_idempotency_key_unique
    ON goods_receipts (received_by, idempotency_key);

ALTER TABLE goods_receipts
    ADD CONSTRAINT goods_receipts_idempotency_pair_check
    CHECK ((idempotency_key IS NULL AND idempotency_request_hash IS NULL) OR
           (idempotency_key IS NOT NULL AND idempotency_request_hash IS NOT NULL AND idempotency_request_hash ~ '^[0-9a-f]{64}$'));
