-- 000007 was an unreleased demo seed; demo data is now an explicit optional job.
ALTER TABLE procurement_receipt_inbox ALTER COLUMN event_id TYPE text;
ALTER TABLE procurement_receipt_inbox ADD COLUMN source text NOT NULL DEFAULT 'legacy';
ALTER TABLE procurement_receipt_inbox ADD COLUMN topic text;
ALTER TABLE procurement_receipt_inbox ADD COLUMN partition_id integer;
ALTER TABLE procurement_receipt_inbox ADD COLUMN offset_id bigint;
ALTER TABLE procurement_receipt_inbox ADD COLUMN payload_hash text;
ALTER TABLE procurement_receipt_inbox DROP CONSTRAINT procurement_receipt_inbox_pkey;
ALTER TABLE procurement_receipt_inbox ADD PRIMARY KEY(source,event_id);

CREATE TABLE procurement_supplier_mapping (
 supplier_code text PRIMARY KEY, supplier_id uuid NOT NULL UNIQUE
);
CREATE TABLE inventory_procurement_sku_units (
 sku text PRIMARY KEY REFERENCES inventory_items(sku),
 base_unit text NOT NULL CHECK(base_unit IN ('GRAM','UNIT'))
);
CREATE UNIQUE INDEX batches_sku_batch_code_unique ON batches(sku,batch_code);
-- Canonical receipts contain no manufacturing/expiry dates. Unknown dates are
-- represented honestly and are allowed only for quarantined, unsellable batches.
ALTER TABLE batches ALTER COLUMN mfg_date DROP NOT NULL;
ALTER TABLE batches ALTER COLUMN exp_date DROP NOT NULL;
ALTER TABLE batches ADD CONSTRAINT batches_known_dates_or_quarantine CHECK (
 (mfg_date IS NOT NULL AND exp_date IS NOT NULL) OR status='QUARANTINE');

CREATE TABLE procurement_receipts (
 source text NOT NULL, receipt_id text NOT NULL, sku text NOT NULL REFERENCES inventory_items(sku),
 event_id text NOT NULL, po_number text NOT NULL,
 supplier_code text NOT NULL REFERENCES procurement_supplier_mapping(supplier_code),
 supplier_name text NOT NULL, origin_region text NOT NULL,
 warehouse_code text NOT NULL, received_at timestamptz NOT NULL,
 unit text NOT NULL, base_unit text NOT NULL CHECK(base_unit IN ('GRAM','UNIT')),
 quantity_received numeric NOT NULL CHECK(quantity_received>0),
 stock_quantity integer NOT NULL CHECK(stock_quantity>0),
 batch_id uuid NOT NULL UNIQUE REFERENCES batches(id), data_hash text NOT NULL,
 payload jsonb NOT NULL, created_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(source,receipt_id,sku)
);
CREATE INDEX procurement_receipt_inbox_processed ON procurement_receipt_inbox(processed_at);

-- Inbox retention never removes the receipt-level dedupe/evidence record.
CREATE FUNCTION inventory_prune_procurement_inbox(before_time timestamptz) RETURNS bigint
LANGUAGE plpgsql AS $$ DECLARE removed bigint; BEGIN
 DELETE FROM procurement_receipt_inbox i WHERE processed_at<before_time AND
 EXISTS(SELECT 1 FROM procurement_receipts r WHERE r.source=i.source AND r.event_id=i.event_id);
 GET DIAGNOSTICS removed = ROW_COUNT;
 RETURN removed;
END $$;
