-- Disable Fulfillment reads before rolling back this additive schema capability.
DROP TRIGGER IF EXISTS inventory_batch_revision ON batches;
DROP FUNCTION IF EXISTS inventory_batch_revision();
ALTER TABLE batches DROP COLUMN IF EXISTS version;
