ALTER TABLE shipments
    ADD COLUMN request_hash TEXT NULL;

ALTER TABLE shipments
    ADD CONSTRAINT shipments_request_hash_check
    CHECK (request_hash ~ '^[0-9a-f]{64}$');
