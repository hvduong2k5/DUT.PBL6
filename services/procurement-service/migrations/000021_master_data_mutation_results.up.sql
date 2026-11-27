CREATE TABLE master_data_mutation_results (
 actor_id TEXT NOT NULL CHECK(length(actor_id)>0),
 idempotency_key VARCHAR(128) NOT NULL CHECK(length(idempotency_key) BETWEEN 16 AND 128 AND idempotency_key !~ '[^!-~]'),
 action VARCHAR(24) NOT NULL CHECK(action IN ('EDIT_SUPPLIER','EDIT_MATERIAL','SUPPLIER_STATUS','MATERIAL_STATUS','OFFER_STATUS','PREFERRED_SUPPLIER')),
 resource_id UUID NOT NULL,
 parent_supplier_id UUID,
 expected_updated_at TIMESTAMPTZ NOT NULL,
 request_hash VARCHAR(64) NOT NULL CHECK(request_hash ~ '^[0-9a-f]{64}$'),
 result_snapshot JSONB NOT NULL CHECK(jsonb_typeof(result_snapshot)='object' AND result_snapshot->>'id'=resource_id::text),
 completed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 PRIMARY KEY(actor_id,idempotency_key)
);
-- Every write, including implicit preference cleanup and price-window closing,
-- advances the optimistic token even for no-op updates or clock rollback.
CREATE FUNCTION advance_master_data_updated_at() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 NEW.updated_at:=GREATEST(clock_timestamp(),OLD.updated_at+interval '1 microsecond');
 RETURN NEW;
END $$;
CREATE TRIGGER suppliers_updated_at_token BEFORE UPDATE ON suppliers FOR EACH ROW EXECUTE FUNCTION advance_master_data_updated_at();
CREATE TRIGGER materials_updated_at_token BEFORE UPDATE ON raw_materials FOR EACH ROW EXECUTE FUNCTION advance_master_data_updated_at();
CREATE TRIGGER offers_updated_at_token BEFORE UPDATE ON supplier_material_offers FOR EACH ROW EXECUTE FUNCTION advance_master_data_updated_at();
-- Completed command snapshots have no TTL or independent pruning.
