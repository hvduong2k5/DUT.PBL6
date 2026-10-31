-- Existing volumes must apply this migration before deploying the new binary.
BEGIN;
-- UNSPECIFIED has 11 characters; the original VARCHAR(10) rejected the enum.
ALTER TABLE customer_profiles ALTER COLUMN gender TYPE VARCHAR(16);
ALTER TABLE shipping_addresses ADD COLUMN IF NOT EXISTS version INTEGER NOT NULL DEFAULT 1 CHECK(version > 0);
UPDATE shipping_addresses SET is_default=false WHERE is_deleted AND is_default;
WITH ranked AS (
 SELECT id,row_number() OVER(PARTITION BY customer_id ORDER BY updated_at DESC,id ASC) AS rn
 FROM shipping_addresses a WHERE NOT is_deleted AND NOT EXISTS(
  SELECT 1 FROM shipping_addresses d WHERE d.customer_id=a.customer_id AND d.is_default AND NOT d.is_deleted)
) UPDATE shipping_addresses SET is_default=true WHERE id IN(SELECT id FROM ranked WHERE rn=1);
ALTER TABLE shipping_addresses ADD CONSTRAINT ck_deleted_address_not_default CHECK(NOT(is_deleted AND is_default));
COMMIT;
