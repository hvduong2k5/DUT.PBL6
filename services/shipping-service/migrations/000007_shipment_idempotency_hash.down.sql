ALTER TABLE shipments
    DROP CONSTRAINT IF EXISTS shipments_request_hash_check;

ALTER TABLE shipments
    DROP COLUMN IF EXISTS request_hash;
