CREATE TABLE loyalty_policies (
    policy_version TEXT PRIMARY KEY CHECK (length(btrim(policy_version)) > 0),
    status TEXT NOT NULL CHECK (status IN ('DRAFT', 'ACTIVE', 'RETIRED')),
    earn_rate_basis_points INTEGER NOT NULL CHECK (earn_rate_basis_points BETWEEN 0 AND 10000),
    points_per_vnd_numerator BIGINT NOT NULL CHECK (points_per_vnd_numerator > 0),
    points_per_vnd_denominator BIGINT NOT NULL CHECK (points_per_vnd_denominator > 0),
    base_tier_code TEXT NOT NULL DEFAULT 'BRONZE' CHECK (base_tier_code IN ('BRONZE','SILVER','GOLD','DIAMOND','VIP_OCOP')),
    rounding_mode TEXT NOT NULL CHECK (rounding_mode IN ('FLOOR')),
    hold_ttl_minutes INTEGER NOT NULL CHECK (hold_ttl_minutes BETWEEN 1 AND 60),
    points_expire_days INTEGER CHECK (points_expire_days IS NULL OR points_expire_days > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);

CREATE UNIQUE INDEX loyalty_one_active_policy_idx ON loyalty_policies(status) WHERE status='ACTIVE';

-- Provisional development policy. Replace/retire before production acceptance.
INSERT INTO loyalty_policies(policy_version,status,earn_rate_basis_points,points_per_vnd_numerator,points_per_vnd_denominator,base_tier_code,rounding_mode,hold_ttl_minutes,points_expire_days)
VALUES('loyalty-dev-v1','ACTIVE',100,1,1,'BRONZE','FLOOR',15,NULL);

CREATE TABLE loyalty_redemption_holds (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id TEXT NOT NULL UNIQUE CHECK (length(btrim(order_id)) > 0),
    customer_id TEXT NOT NULL,
    points BIGINT NOT NULL CHECK (points >= 0),
    discount_amount_vnd BIGINT NOT NULL CHECK (discount_amount_vnd >= 0),
    currency_code CHAR(3) NOT NULL DEFAULT 'VND' CHECK (currency_code='VND'),
    state TEXT NOT NULL CHECK (state IN ('HELD','REDEEMED','RELEASED','EXPIRED')),
    resource_version BIGINT NOT NULL DEFAULT 1 CHECK (resource_version > 0),
    expires_at TIMESTAMPTZ NOT NULL,
    policy_version TEXT NOT NULL REFERENCES loyalty_policies(policy_version),
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    finalized_at TIMESTAMPTZ,
    released_at TIMESTAMPTZ,
    CHECK (
        (points > 0 AND customer_id <> '' AND discount_amount_vnd > 0)
        OR (state='RELEASED' AND points=0 AND customer_id='' AND discount_amount_vnd=0 AND finalized_at IS NULL)
    ),
    CHECK (state <> 'REDEEMED' OR finalized_at IS NOT NULL),
    CHECK (state <> 'RELEASED' OR released_at IS NOT NULL)
);

CREATE INDEX loyalty_redemption_holds_customer_idx ON loyalty_redemption_holds(customer_id,state,expires_at);
CREATE INDEX loyalty_redemption_holds_expiry_idx ON loyalty_redemption_holds(expires_at) WHERE state='HELD';

CREATE TABLE loyalty_command_idempotency (
    idempotency_key TEXT PRIMARY KEY,
    command_type TEXT NOT NULL,
    request_hash CHAR(64) NOT NULL CHECK (request_hash ~ '^[0-9a-f]{64}$'),
    resource_id UUID NOT NULL,
    response_json JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);

ALTER TABLE loyalty_ledger_entries
    ADD COLUMN redemption_hold_id UUID REFERENCES loyalty_redemption_holds(id);

CREATE UNIQUE INDEX loyalty_redemption_debit_hold_idx
    ON loyalty_ledger_entries(redemption_hold_id)
    WHERE entry_type='REDEMPTION_DEBIT' AND redemption_hold_id IS NOT NULL;
