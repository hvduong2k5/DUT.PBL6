-- Stop draft editing first. Never delete command evidence to force rollback.
DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM purchase_order_action_results WHERE action='EDIT_DRAFT') THEN
        RAISE EXCEPTION 'cannot roll back: completed draft edit command evidence exists';
    END IF;
END $$;
ALTER TABLE purchase_order_action_results DROP CONSTRAINT purchase_order_action_results_action_check;
ALTER TABLE purchase_order_action_results ADD CONSTRAINT purchase_order_action_results_action_check
CHECK (action IN ('SUBMIT','APPROVE','REJECT','SEND','ACKNOWLEDGE','REQUEST_CHANGE','DECLINE','CANCEL','CLOSE','REVISION'));
