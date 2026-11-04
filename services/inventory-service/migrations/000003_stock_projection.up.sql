-- Emit absolute stock snapshots atomically for both legacy and terminal cohorts.
ALTER TABLE inventory_items ADD COLUMN IF NOT EXISTS inventory_version BIGINT NOT NULL DEFAULT 0;

CREATE FUNCTION inventory_emit_stock_snapshot(target_sku TEXT) RETURNS void
LANGUAGE plpgsql AS $$
DECLARE
 item inventory_items%ROWTYPE;
 sellable BIGINT;
 observed TIMESTAMPTZ;
BEGIN
 -- Serialize version allocation per SKU. Trigger recursion is avoided because
 -- inventory_version is not a watched UPDATE column.
 UPDATE inventory_items SET inventory_version = inventory_version + 1
 WHERE sku = target_sku RETURNING * INTO item;
 IF NOT FOUND THEN RETURN; END IF;
 observed := clock_timestamp();
 SELECT COALESCE(SUM(GREATEST(physical_qty::BIGINT - reserved_qty, 0)), 0)
 INTO sellable FROM batches
 WHERE sku = target_sku AND status IN ('ACTIVE', 'NEAR_EXPIRY') AND exp_date > observed;
 IF item.status <> 'ACTIVE' THEN sellable := 0; END IF;
 sellable := GREATEST(0, LEAST(sellable, item.physical_qty::BIGINT - item.reserved_qty));
 INSERT INTO outbox_events(id, aggregate_type, aggregate_id, event_type, payload, status, retry_count, created_at)
 VALUES(gen_random_uuid(), 'InventoryItem', target_sku, 'inventory.stock-changed.v1',
 jsonb_build_object('sku_code', target_sku, 'sellable_quantity', sellable,
 'inventory_version', item.inventory_version, 'source_updated_at', observed), 'PENDING', 0, observed);
END;
$$;

CREATE FUNCTION inventory_stock_snapshot_trigger() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP = 'DELETE' THEN
  PERFORM inventory_emit_stock_snapshot(OLD.sku);
 ELSE
  PERFORM inventory_emit_stock_snapshot(NEW.sku);
  IF TG_OP = 'UPDATE' AND OLD.sku <> NEW.sku THEN
   PERFORM inventory_emit_stock_snapshot(OLD.sku);
  END IF;
 END IF;
 RETURN NULL;
END;
$$;

-- Deferred events observe final item AND batch balances, not partial FEFO writes.
-- A transaction may emit multiple versions for a SKU; all are absolute snapshots.
CREATE CONSTRAINT TRIGGER inventory_item_snapshot
AFTER INSERT OR UPDATE OF physical_qty, reserved_qty, status ON inventory_items
DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION inventory_stock_snapshot_trigger();
CREATE CONSTRAINT TRIGGER inventory_batch_snapshot
AFTER INSERT OR DELETE OR UPDATE OF physical_qty, reserved_qty, status, exp_date, sku ON batches
DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION inventory_stock_snapshot_trigger();

-- Bootstrap existing SKUs; also call this function per SKU to rebuild projection.
SELECT inventory_emit_stock_snapshot(sku) FROM inventory_items ORDER BY sku;
