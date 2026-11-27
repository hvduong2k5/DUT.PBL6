DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM supplier_contract_command_results) OR EXISTS(SELECT 1 FROM supplier_contract_status_history WHERE note<>'') THEN
  RAISE EXCEPTION 'cannot roll back: completed supplier contract evidence exists';
 END IF;
END $$;
DROP TRIGGER supplier_contracts_no_truncate ON supplier_contracts;
DROP TRIGGER supplier_contracts_immutable_terms ON supplier_contracts;
DROP TRIGGER supplier_contracts_updated_at_token ON supplier_contracts;
DROP FUNCTION protect_supplier_contract_terms();
DROP TABLE supplier_contract_command_results;
ALTER TABLE supplier_contract_status_history DROP COLUMN note;
