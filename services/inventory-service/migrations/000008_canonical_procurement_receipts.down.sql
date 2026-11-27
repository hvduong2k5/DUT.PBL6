-- Preserve evidence: do not discard received stock or truncate audit tables.
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM procurement_receipts) OR EXISTS(SELECT 1 FROM procurement_supplier_mapping)
 OR EXISTS(SELECT 1 FROM inventory_procurement_sku_units)
 OR EXISTS(SELECT 1 FROM batches WHERE mfg_date IS NULL OR exp_date IS NULL)
 OR EXISTS(SELECT 1 FROM procurement_receipt_inbox WHERE source<>'legacy' OR
 event_id !~* '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$')
 THEN RAISE EXCEPTION 'Canonical procurement data exists; rollback refused to preserve evidence'; END IF;
END $$;
DROP FUNCTION inventory_prune_procurement_inbox(timestamptz);
DROP TABLE procurement_receipts;
DROP TABLE procurement_supplier_mapping;
DROP TABLE inventory_procurement_sku_units;
DROP INDEX batches_sku_batch_code_unique;
ALTER TABLE batches DROP CONSTRAINT batches_known_dates_or_quarantine;
ALTER TABLE batches ALTER COLUMN mfg_date SET NOT NULL;
ALTER TABLE batches ALTER COLUMN exp_date SET NOT NULL;
ALTER TABLE procurement_receipt_inbox DROP CONSTRAINT procurement_receipt_inbox_pkey;
ALTER TABLE procurement_receipt_inbox DROP COLUMN payload_hash, DROP COLUMN offset_id,
 DROP COLUMN partition_id, DROP COLUMN topic, DROP COLUMN source;
ALTER TABLE procurement_receipt_inbox ALTER COLUMN event_id TYPE uuid USING event_id::uuid;
ALTER TABLE procurement_receipt_inbox ADD PRIMARY KEY(event_id);
DROP INDEX procurement_receipt_inbox_processed;
