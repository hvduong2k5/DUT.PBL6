ALTER TABLE purchase_orders
    ADD COLUMN idempotency_key VARCHAR(128),
    ADD COLUMN idempotency_request_hash VARCHAR(64);

ALTER TABLE purchase_orders ADD CONSTRAINT purchase_orders_idempotency_fields_check
    CHECK ((idempotency_key IS NULL AND idempotency_request_hash IS NULL) OR
           (idempotency_key IS NOT NULL AND idempotency_request_hash IS NOT NULL AND length(idempotency_request_hash)=64));

CREATE UNIQUE INDEX purchase_orders_created_by_idempotency_key_unique
    ON purchase_orders (created_by, idempotency_key)
    WHERE idempotency_key IS NOT NULL;
