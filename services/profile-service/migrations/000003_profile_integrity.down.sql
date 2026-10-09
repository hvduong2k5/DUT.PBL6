BEGIN;
ALTER TABLE shipping_addresses DROP CONSTRAINT IF EXISTS ck_deleted_address_not_default;
ALTER TABLE shipping_addresses DROP COLUMN IF EXISTS version;
-- Keep gender widened: narrowing would fail for existing UNSPECIFIED values.
COMMIT;
