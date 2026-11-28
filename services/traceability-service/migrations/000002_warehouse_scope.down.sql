DROP TRIGGER warehouse_binding_protection ON traceability_profiles;
DROP FUNCTION protect_warehouse_binding();
DROP TRIGGER metadata_immutable ON service_metadata;
DROP FUNCTION protect_scoped_metadata();
CREATE TRIGGER metadata_immutable BEFORE UPDATE OR DELETE ON service_metadata FOR EACH ROW EXECUTE FUNCTION reject_history_mutation();
ALTER TABLE service_metadata DROP COLUMN warehouse_writes_required;
DROP INDEX profile_warehouse_pagination;
ALTER TABLE traceability_profiles DROP COLUMN warehouse_id;
