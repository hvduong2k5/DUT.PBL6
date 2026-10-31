ALTER TABLE shipments
    ADD COLUMN quote_id TEXT NOT NULL DEFAULT '',
    ADD COLUMN ready_generation BIGINT NOT NULL DEFAULT 0 CHECK (ready_generation >= 0),
    ADD COLUMN items_digest TEXT NOT NULL DEFAULT '',
    ADD COLUMN order_snapshot_version BIGINT NOT NULL DEFAULT 0 CHECK (order_snapshot_version >= 0);
