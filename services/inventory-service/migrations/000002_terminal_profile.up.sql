-- Separate cohort ledger; shared physical stock is still locked via inventory_items/batches.
CREATE TABLE inventory_terminal (
 order_id TEXT PRIMARY KEY,
 outcome BYTEA NOT NULL,
 allocations JSONB NOT NULL DEFAULT '[]',
 commands JSONB NOT NULL DEFAULT '{}'
);

-- Both legacy and new reserve use the same per-order lock, including absent resources.
CREATE FUNCTION inventory_cohort_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 PERFORM pg_advisory_xact_lock(hashtextextended(NEW.order_id, 731));
 IF EXISTS (SELECT 1 FROM inventory_terminal WHERE order_id=NEW.order_id) THEN
  RAISE EXCEPTION 'order belongs to terminal cohort';
 END IF;
 RETURN NEW;
END;
$$;
CREATE TRIGGER inventory_legacy_cohort BEFORE INSERT ON stock_reservations
FOR EACH ROW EXECUTE FUNCTION inventory_cohort_guard();
