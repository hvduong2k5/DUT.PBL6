-- Stop revision writes first. Retain every completed command and archived snapshot.
DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM purchase_order_action_results WHERE action='REVISION') THEN
        RAISE EXCEPTION 'cannot roll back: completed revision command evidence exists';
    END IF;
END $$;
DROP TRIGGER purchase_order_revisions_no_truncate ON purchase_order_revision_snapshots;
DROP TRIGGER purchase_order_revisions_immutable ON purchase_order_revision_snapshots;
DROP FUNCTION reject_purchase_order_revision_mutation();
ALTER TABLE purchase_order_action_results DROP CONSTRAINT purchase_order_action_results_action_check;
ALTER TABLE purchase_order_action_results ADD CONSTRAINT purchase_order_action_results_action_check
CHECK (action IN ('SUBMIT','APPROVE','REJECT','SEND','ACKNOWLEDGE','REQUEST_CHANGE','DECLINE','CANCEL','CLOSE'));
