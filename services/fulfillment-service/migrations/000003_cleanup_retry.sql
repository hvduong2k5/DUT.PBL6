BEGIN;
ALTER TABLE fulfillment.media_files ADD COLUMN cleanup_lease_until timestamptz;
COMMIT;
