ALTER TABLE batches ADD COLUMN version bigint NOT NULL DEFAULT 1 CHECK(version>0);
CREATE FUNCTION inventory_batch_revision() RETURNS trigger AS $$
BEGIN
 NEW.version := OLD.version+1;
 RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE TRIGGER inventory_batch_revision BEFORE UPDATE ON batches
FOR EACH ROW EXECUTE FUNCTION inventory_batch_revision();
