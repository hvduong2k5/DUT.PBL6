ALTER TABLE shipments DROP CONSTRAINT IF EXISTS shipments_packing_proof_check;
ALTER TABLE shipments
    DROP COLUMN IF EXISTS packed_at,
    DROP COLUMN IF EXISTS packing_task_version,
    DROP COLUMN IF EXISTS packing_task_id;
