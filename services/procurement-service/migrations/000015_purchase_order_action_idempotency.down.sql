-- Stop the command endpoints before rollback. Dropping this table removes retry
-- evidence; PO business state and status-history records are not rolled back.
DROP TABLE purchase_order_action_results;
