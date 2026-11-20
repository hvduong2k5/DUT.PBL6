-- Stop Inventory before rollback. Startup reapplies this demo seed when its marker is absent.
-- Only delete rows recorded as newly created by this migration.
SELECT pg_advisory_xact_lock(2026112041);
DO $$ BEGIN
IF EXISTS (
    SELECT 1 FROM batches b JOIN inventory_demo_seed_000007 s ON b.id=s.batch_id
    WHERE s.batch_created AND
        (b.sku<>s.sku OR b.version<>1 OR b.physical_qty<>s.initial_quantity OR b.reserved_qty<>0)
) OR EXISTS (
    SELECT 1 FROM inventory_items i JOIN inventory_demo_seed_000007 s ON i.sku=s.sku
    WHERE s.item_created AND
        (i.warehouse_id<>'9b200006-2026-4020-8020-000000000001'
         OR i.physical_qty<>s.initial_quantity OR i.reserved_qty<>0)
) THEN
    RAISE EXCEPTION 'Demo stock has changed; inspect its usage before rollback';
END IF;
END $$;
DELETE FROM batches b USING inventory_demo_seed_000007 s
WHERE b.id=s.batch_id AND s.batch_created;
DELETE FROM inventory_items i USING inventory_demo_seed_000007 s
WHERE i.sku=s.sku AND s.item_created;
DROP TABLE inventory_demo_seed_000007;
