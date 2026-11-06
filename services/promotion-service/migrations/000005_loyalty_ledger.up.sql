CREATE TABLE loyalty_ledger_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id TEXT NOT NULL CHECK (length(btrim(customer_id)) > 0),
    entry_type TEXT NOT NULL CHECK (entry_type IN (
        'PURCHASE_EARN',
        'REFUND_CONVERSION',
        'REDEMPTION_DEBIT',
        'REVERSAL',
        'EXPIRATION',
        'MANUAL_ADJUSTMENT'
    )),
    points_delta BIGINT NOT NULL CHECK (points_delta <> 0),
    basis_amount_vnd BIGINT CHECK (basis_amount_vnd IS NULL OR basis_amount_vnd >= 0),
    source_type TEXT NOT NULL CHECK (source_type IN (
        'ORDER_COMPLETION',
        'REFUND_DECISION',
        'CHECKOUT_REDEMPTION',
        'ORDER_OR_RETURN_REVERSAL',
        'POINTS_EXPIRY',
        'ADMIN_ADJUSTMENT'
    )),
    source_id TEXT NOT NULL CHECK (length(btrim(source_id)) > 0),
    source_decision_id TEXT,
    idempotency_key TEXT NOT NULL UNIQUE,
    request_hash CHAR(64) NOT NULL CHECK (request_hash ~ '^[0-9a-f]{64}$'),
    policy_version TEXT NOT NULL CHECK (length(btrim(policy_version)) > 0),
    reference_entry_id UUID REFERENCES loyalty_ledger_entries(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    CHECK (
        (entry_type IN ('PURCHASE_EARN', 'REFUND_CONVERSION') AND points_delta > 0)
        OR (entry_type IN ('REDEMPTION_DEBIT', 'REVERSAL', 'EXPIRATION') AND points_delta < 0)
        OR entry_type = 'MANUAL_ADJUSTMENT'
    ),
    UNIQUE (source_type, source_id, entry_type)
);

CREATE INDEX loyalty_ledger_customer_history_idx
    ON loyalty_ledger_entries(customer_id, created_at DESC, id DESC);

CREATE INDEX loyalty_ledger_reference_idx
    ON loyalty_ledger_entries(reference_entry_id)
    WHERE reference_entry_id IS NOT NULL;

CREATE FUNCTION reject_loyalty_ledger_mutation() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'loyalty ledger entries are append-only';
END;
$$;

CREATE TRIGGER loyalty_ledger_entries_append_only
    BEFORE UPDATE OR DELETE ON loyalty_ledger_entries
    FOR EACH ROW EXECUTE FUNCTION reject_loyalty_ledger_mutation();
