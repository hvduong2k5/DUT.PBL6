-- The public operation projection must expose a durable update timestamp,
-- including asynchronous worker transitions. Existing records retain the
-- latest known decision time rather than pretending migration time was it.
ALTER TABLE checkout_operations ADD COLUMN updated_at timestamptz;
UPDATE checkout_operations SET updated_at=COALESCE(completed_at,accepted_at);
ALTER TABLE checkout_operations ALTER COLUMN updated_at SET NOT NULL;
ALTER TABLE checkout_operations ALTER COLUMN updated_at SET DEFAULT clock_timestamp();
CREATE FUNCTION order_checkout_touch_updated_at() RETURNS trigger AS $$
BEGIN
 NEW.updated_at := clock_timestamp();
 RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE TRIGGER checkout_operation_updated_at BEFORE UPDATE ON checkout_operations
FOR EACH ROW EXECUTE FUNCTION order_checkout_touch_updated_at();
