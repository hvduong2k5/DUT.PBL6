ALTER TABLE purchase_order_action_results DROP CONSTRAINT purchase_order_action_results_action_check;
ALTER TABLE purchase_order_action_results ADD CONSTRAINT purchase_order_action_results_action_check
CHECK (action IN ('SUBMIT','APPROVE','REJECT','SEND','ACKNOWLEDGE','REQUEST_CHANGE','DECLINE','CANCEL','CLOSE','REVISION','EDIT_DRAFT'));
