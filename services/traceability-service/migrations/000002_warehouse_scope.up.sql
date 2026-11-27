ALTER TABLE traceability_profiles ADD COLUMN warehouse_id uuid;
CREATE INDEX profile_warehouse_pagination ON traceability_profiles(warehouse_id, created_at, id);
ALTER TABLE service_metadata ADD COLUMN warehouse_writes_required boolean NOT NULL DEFAULT false;

-- Enable once every writer understands warehouse metadata, before v2 traffic.
-- This switch is monotonic; disabling the API must not re-enable old writers.
DROP TRIGGER metadata_immutable ON service_metadata;
CREATE FUNCTION protect_scoped_metadata() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP <> 'UPDATE' THEN
        RAISE EXCEPTION 'metadata is immutable' USING ERRCODE='23514';
    END IF;
    IF ROW(NEW.singleton,NEW.mode) IS DISTINCT FROM ROW(OLD.singleton,OLD.mode)
       OR OLD.warehouse_writes_required OR NOT NEW.warehouse_writes_required
       OR current_user <> pg_get_userbyid((SELECT relowner FROM pg_class WHERE oid=TG_RELID)) THEN
        RAISE EXCEPTION 'warehouse writer enforcement requires schema owner and cannot be reversed' USING ERRCODE='23514';
    END IF;
    RETURN NEW;
END $$;
CREATE TRIGGER metadata_immutable BEFORE UPDATE OR DELETE ON service_metadata FOR EACH ROW EXECUTE FUNCTION protect_scoped_metadata();

CREATE FUNCTION protect_warehouse_binding() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.warehouse_id = '00000000-0000-0000-0000-000000000000'::uuid THEN
        RAISE EXCEPTION 'warehouse identity is invalid' USING ERRCODE='23514';
    END IF;
    IF TG_OP = 'INSERT' THEN
        IF NEW.warehouse_id IS NULL AND (SELECT warehouse_writes_required FROM service_metadata WHERE singleton) THEN
            RAISE EXCEPTION 'writer must supply authoritative warehouse' USING ERRCODE='23514';
        END IF;
    ELSIF NEW.warehouse_id IS DISTINCT FROM OLD.warehouse_id THEN
        IF OLD.warehouse_id IS NOT NULL OR NEW.warehouse_id IS NULL
           OR current_user <> pg_get_userbyid((SELECT relowner FROM pg_class WHERE oid=TG_RELID)) THEN
            RAISE EXCEPTION 'warehouse binding is immutable outside controlled legacy backfill' USING ERRCODE='23514';
        END IF;
    END IF;
    RETURN NEW;
END $$;
CREATE TRIGGER warehouse_binding_protection BEFORE INSERT OR UPDATE ON traceability_profiles FOR EACH ROW EXECUTE FUNCTION protect_warehouse_binding();
