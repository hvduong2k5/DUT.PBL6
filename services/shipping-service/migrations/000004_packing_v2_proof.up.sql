ALTER TABLE shipments
    ADD COLUMN IF NOT EXISTS packing_task_id TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS packing_task_version BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS packed_at TIMESTAMPTZ NULL;

ALTER TABLE shipments
    ADD CONSTRAINT shipments_packing_proof_check CHECK (
        (packing_task_id = '' AND packing_task_version = 0 AND packed_at IS NULL)
        OR
        (packing_task_id <> '' AND packing_task_version > 0 AND packed_at IS NOT NULL)
    );
