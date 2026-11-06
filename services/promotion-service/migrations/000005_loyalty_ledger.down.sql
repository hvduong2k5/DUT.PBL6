DROP TRIGGER IF EXISTS loyalty_ledger_entries_append_only ON loyalty_ledger_entries;
DROP FUNCTION IF EXISTS reject_loyalty_ledger_mutation();
DROP TABLE IF EXISTS loyalty_ledger_entries;
