ALTER TABLE shipments
    ADD COLUMN IF NOT EXISTS event_version SMALLINT NOT NULL DEFAULT 1;

ALTER TABLE shipments
    ADD CONSTRAINT shipments_event_version_check CHECK (event_version IN (1, 2));
