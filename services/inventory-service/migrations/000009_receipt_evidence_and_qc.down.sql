DO $$ BEGIN
IF EXISTS(SELECT 1 FROM inventory_qc_releases) OR EXISTS(SELECT 1 FROM procurement_event_ledger)
THEN RAISE EXCEPTION 'Receipt/QC evidence exists; rollback refused'; END IF; END $$;
CREATE OR REPLACE FUNCTION inventory_prune_procurement_inbox(before_time timestamptz) RETURNS bigint
LANGUAGE plpgsql AS $$ DECLARE removed bigint; BEGIN
 DELETE FROM procurement_receipt_inbox i WHERE processed_at<before_time AND
 EXISTS(SELECT 1 FROM procurement_receipts r WHERE r.source=i.source AND r.event_id=i.event_id);
 GET DIAGNOSTICS removed = ROW_COUNT; RETURN removed; END $$;
DROP TABLE inventory_qc_releases;
DROP TABLE procurement_event_ledger;
