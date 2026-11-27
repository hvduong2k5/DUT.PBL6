ALTER TABLE purchase_order_action_results DROP CONSTRAINT purchase_order_action_results_action_check;
ALTER TABLE purchase_order_action_results ADD CONSTRAINT purchase_order_action_results_action_check
CHECK (action IN ('SUBMIT','APPROVE','REJECT','SEND','ACKNOWLEDGE','REQUEST_CHANGE','DECLINE','CANCEL','CLOSE','REVISION'));

CREATE FUNCTION reject_purchase_order_revision_mutation() RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'purchase order revision snapshots are immutable';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER purchase_order_revisions_immutable
BEFORE UPDATE OR DELETE ON purchase_order_revision_snapshots
FOR EACH ROW EXECUTE FUNCTION reject_purchase_order_revision_mutation();

CREATE TRIGGER purchase_order_revisions_no_truncate
BEFORE TRUNCATE ON purchase_order_revision_snapshots
FOR EACH STATEMENT EXECUTE FUNCTION reject_purchase_order_revision_mutation();
