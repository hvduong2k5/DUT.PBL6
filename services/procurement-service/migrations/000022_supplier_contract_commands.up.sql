ALTER TABLE supplier_contract_status_history ADD COLUMN note TEXT NOT NULL DEFAULT '';
CREATE TABLE supplier_contract_command_results (
 actor_id TEXT NOT NULL CHECK(length(actor_id)>0),
 idempotency_key VARCHAR(128) NOT NULL CHECK(length(idempotency_key) BETWEEN 16 AND 128 AND idempotency_key !~ '[^!-~]'),
 action VARCHAR(16) NOT NULL CHECK(action IN ('CREATE','ACTIVATE','TERMINATE')),
 supplier_contract_id UUID NOT NULL REFERENCES supplier_contracts(id) ON DELETE RESTRICT,
 expected_updated_at TIMESTAMPTZ,
 reason TEXT NOT NULL DEFAULT '',
 request_hash VARCHAR(64) NOT NULL CHECK(request_hash ~ '^[0-9a-f]{64}$'),
 result_snapshot JSONB NOT NULL CHECK(jsonb_typeof(result_snapshot)='object'),
 completed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 PRIMARY KEY(actor_id,idempotency_key)
);
CREATE TRIGGER supplier_contracts_updated_at_token BEFORE UPDATE ON supplier_contracts FOR EACH ROW EXECUTE FUNCTION advance_master_data_updated_at();
CREATE FUNCTION protect_supplier_contract_terms() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP<>'UPDATE' THEN RAISE EXCEPTION 'supplier contract evidence cannot be deleted'; END IF;
 IF ROW(NEW.id,NEW.supplier_id,NEW.contract_number,NEW.revision,NEW.scope,NEW.payment_terms,NEW.document_file_id,NEW.effective_from,NEW.effective_to,NEW.created_by,NEW.created_at)
 IS DISTINCT FROM ROW(OLD.id,OLD.supplier_id,OLD.contract_number,OLD.revision,OLD.scope,OLD.payment_terms,OLD.document_file_id,OLD.effective_from,OLD.effective_to,OLD.created_by,OLD.created_at) THEN
  RAISE EXCEPTION 'supplier contract terms are immutable; create a new revision';
 END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER supplier_contracts_immutable_terms BEFORE UPDATE OR DELETE ON supplier_contracts FOR EACH ROW EXECUTE FUNCTION protect_supplier_contract_terms();
CREATE TRIGGER supplier_contracts_no_truncate BEFORE TRUNCATE ON supplier_contracts FOR EACH STATEMENT EXECUTE FUNCTION protect_supplier_contract_terms();
-- Commands, contract and history commit atomically. No TTL/independent pruning.
