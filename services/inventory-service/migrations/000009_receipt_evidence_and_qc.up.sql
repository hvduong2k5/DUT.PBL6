CREATE TABLE procurement_event_ledger (
 source text NOT NULL,event_id text NOT NULL,payload_hash text NOT NULL,
 hash_version integer NOT NULL DEFAULT 2, receipt_id text, sku text,
 PRIMARY KEY(source,event_id)
);
INSERT INTO procurement_event_ledger(source,event_id,payload_hash,hash_version,receipt_id,sku)
SELECT i.source,i.event_id,i.payload_hash,1,r.receipt_id,r.sku
FROM procurement_receipt_inbox i LEFT JOIN procurement_receipts r ON r.source=i.source AND r.event_id=i.event_id
WHERE i.payload_hash IS NOT NULL;
CREATE TABLE inventory_qc_releases (
 actor_id uuid NOT NULL,idem_key text NOT NULL, batch_id uuid NOT NULL REFERENCES batches(id),
 request jsonb NOT NULL, response jsonb NOT NULL, reason text NOT NULL,warehouse_id uuid NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now(),PRIMARY KEY(actor_id,idem_key)
);
CREATE OR REPLACE FUNCTION inventory_prune_procurement_inbox(before_time timestamptz) RETURNS bigint
LANGUAGE plpgsql AS $$ DECLARE removed bigint; BEGIN
 DELETE FROM procurement_receipt_inbox i WHERE processed_at<before_time AND
 EXISTS(SELECT 1 FROM procurement_event_ledger e WHERE e.source=i.source AND e.event_id=i.event_id);
 GET DIAGNOSTICS removed = ROW_COUNT; RETURN removed;
END $$;
