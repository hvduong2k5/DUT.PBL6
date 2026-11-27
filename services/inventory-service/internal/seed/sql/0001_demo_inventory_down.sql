-- Stop Inventory and all stock writers before rollback. Keep outbox/audit history.
SELECT pg_advisory_xact_lock(2026112041);
LOCK TABLE inventory_items, batches, stock_reservation_allocations, stock_adjustments, stock_receipts IN SHARE ROW EXCLUSIVE MODE;
DO $$ BEGIN
IF EXISTS (
 SELECT 1 FROM inventory_demo_seed_000007 s JOIN inventory_items i USING(sku)
 WHERE s.item_created AND (i.status<>'ACTIVE' OR i.physical_qty<>s.initial_quantity OR i.reserved_qty<>0
 OR i.warehouse_id<>'9b200006-2026-4020-8020-000000000001'
 OR s.initial_inventory_version IS NULL OR i.inventory_version<>s.initial_inventory_version)
) OR EXISTS (
 SELECT 1 FROM inventory_demo_seed_000007 s JOIN batches b ON b.sku=s.sku
 WHERE s.item_created AND (b.id<>s.batch_id OR b.version<>1 OR b.status<>'ACTIVE'
 OR b.physical_qty<>s.initial_quantity OR b.reserved_qty<>0)
) OR EXISTS (
 SELECT 1 FROM inventory_demo_seed_000007 s WHERE s.item_created AND (
 EXISTS(SELECT 1 FROM stock_adjustments a WHERE a.sku=s.sku OR a.batch_id=s.batch_id) OR
 EXISTS(SELECT 1 FROM stock_receipts r WHERE r.sku=s.sku) OR
 EXISTS(SELECT 1 FROM stock_reservation_allocations a WHERE a.batch_id=s.batch_id) OR
 EXISTS(SELECT 1 FROM procurement_receipts r WHERE r.batch_id=s.batch_id) OR
 NOT EXISTS(SELECT 1 FROM batches b WHERE b.id=s.batch_id AND b.sku=s.sku)))
THEN RAISE EXCEPTION 'Demo stock has changed or has business references; rollback refused'; END IF;
END $$;
-- Emit a higher-version zero snapshot while the item still exists. Deferred
-- DELETE triggers alone cannot emit after both the item and batch are removed.
UPDATE inventory_items SET status='INACTIVE'
WHERE sku IN (SELECT sku FROM inventory_demo_seed_000007 WHERE item_created);
SET CONSTRAINTS ALL IMMEDIATE;
DELETE FROM batches b USING inventory_demo_seed_000007 s WHERE b.id=s.batch_id AND s.batch_created;
DELETE FROM inventory_items i USING inventory_demo_seed_000007 s WHERE i.sku=s.sku AND s.item_created;
DROP TABLE inventory_demo_seed_000007;
