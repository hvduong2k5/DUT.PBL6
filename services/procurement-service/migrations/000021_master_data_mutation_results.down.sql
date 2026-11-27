DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM master_data_mutation_results) THEN
  RAISE EXCEPTION 'cannot roll back: completed master data mutations exist';
 END IF;
END $$;
DROP TRIGGER offers_updated_at_token ON supplier_material_offers;
DROP TRIGGER materials_updated_at_token ON raw_materials;
DROP TRIGGER suppliers_updated_at_token ON suppliers;
DROP FUNCTION advance_master_data_updated_at();
DROP TABLE master_data_mutation_results;
