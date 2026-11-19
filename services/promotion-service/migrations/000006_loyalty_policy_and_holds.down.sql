DROP INDEX IF EXISTS loyalty_redemption_debit_hold_idx;
ALTER TABLE loyalty_ledger_entries DROP COLUMN IF EXISTS redemption_hold_id;
DROP TABLE IF EXISTS loyalty_command_idempotency;
DROP TABLE IF EXISTS loyalty_redemption_holds;
DROP TABLE IF EXISTS loyalty_policies;
