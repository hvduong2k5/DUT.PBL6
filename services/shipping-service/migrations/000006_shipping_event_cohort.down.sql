ALTER TABLE shipments DROP CONSTRAINT IF EXISTS shipments_event_version_check;
ALTER TABLE shipments DROP COLUMN IF EXISTS event_version;
