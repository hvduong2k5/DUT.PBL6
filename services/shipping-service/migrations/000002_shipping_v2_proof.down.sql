ALTER TABLE shipments
    DROP COLUMN order_snapshot_version,
    DROP COLUMN items_digest,
    DROP COLUMN ready_generation,
    DROP COLUMN quote_id;
