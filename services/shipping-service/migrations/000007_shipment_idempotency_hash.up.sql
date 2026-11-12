ALTER TABLE shipments
    ADD COLUMN IF NOT EXISTS request_hash TEXT NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'shipments_request_hash_check'
          AND conrelid = 'shipments'::regclass
    ) THEN
        ALTER TABLE shipments
            ADD CONSTRAINT shipments_request_hash_check
            CHECK (request_hash IS NULL OR request_hash ~ '^[0-9a-f]{64}$');
    END IF;
END
$$;
