ALTER TABLE goods_receipts
    ADD COLUMN inspection_idempotency_key VARCHAR(128) NOT NULL DEFAULT '',
    ADD COLUMN inspection_request_hash VARCHAR(64) NOT NULL DEFAULT '',
    ADD CONSTRAINT goods_receipts_inspection_idempotency_valid CHECK (
        (inspection_idempotency_key = '' AND inspection_request_hash = '')
        OR (status = 'INSPECTED' AND inspected_by IS NOT NULL
            AND length(inspection_idempotency_key) BETWEEN 16 AND 128
            AND inspection_request_hash ~ '^[0-9a-f]{64}$')
    );

-- One inspection per receipt. The signed inspector + receipt ID + key identify
-- a retry; evidence is retained with the receipt, not in an expiring HTTP cache.
-- Historical inspections remain without replay evidence and are not backfilled.
