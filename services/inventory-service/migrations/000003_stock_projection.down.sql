DROP TRIGGER IF EXISTS inventory_batch_snapshot ON batches;
DROP TRIGGER IF EXISTS inventory_item_snapshot ON inventory_items;
DROP FUNCTION IF EXISTS inventory_stock_snapshot_trigger();
DROP FUNCTION IF EXISTS inventory_emit_stock_snapshot(TEXT);
-- Keep version high-watermarks: redeployment must not emit versions older than
-- those already applied by Catalog. Removing this additive column resets authority.
