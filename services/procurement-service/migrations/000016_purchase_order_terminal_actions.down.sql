-- Stop the new endpoints first. Never delete completed command evidence to roll back.
DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM purchase_order_action_results WHERE action IN ('ACKNOWLEDGE','REQUEST_CHANGE','DECLINE','CANCEL','CLOSE')) THEN
        RAISE EXCEPTION 'cannot roll back: terminal PO action evidence exists';
    END IF;
END $$;
ALTER TABLE purchase_order_action_results DROP CONSTRAINT purchase_order_action_results_action_check;
ALTER TABLE purchase_order_action_results ADD CONSTRAINT purchase_order_action_results_action_check
CHECK (action IN ('SUBMIT','APPROVE','REJECT','SEND'));
