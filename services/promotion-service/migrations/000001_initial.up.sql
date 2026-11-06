CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE promotions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('DRAFT', 'SCHEDULED', 'ACTIVE', 'PAUSED', 'ENDED', 'ARCHIVED')),
    starts_at TIMESTAMPTZ NOT NULL,
    ends_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (ends_at > starts_at)
);

CREATE TABLE coupons (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    promotion_id UUID REFERENCES promotions(id),
    code TEXT NOT NULL UNIQUE,
    discount_type TEXT NOT NULL CHECK (discount_type IN ('FIXED_AMOUNT', 'PERCENTAGE', 'FREE_SHIPPING')),
    discount_value BIGINT NOT NULL CHECK (discount_value >= 0),
    max_discount_amount BIGINT,
    minimum_order_amount BIGINT NOT NULL DEFAULT 0 CHECK (minimum_order_amount >= 0),
    max_redemptions BIGINT CHECK (max_redemptions IS NULL OR max_redemptions > 0),
    max_redemptions_per_customer BIGINT CHECK (max_redemptions_per_customer IS NULL OR max_redemptions_per_customer > 0),
    starts_at TIMESTAMPTZ NOT NULL,
    ends_at TIMESTAMPTZ NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('DRAFT', 'SCHEDULED', 'ACTIVE', 'PAUSED', 'ENDED', 'ARCHIVED')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (ends_at > starts_at),
    CHECK (max_discount_amount IS NULL OR max_discount_amount >= 0),
    CHECK (discount_type <> 'PERCENTAGE' OR discount_value <= 10000)
);
CREATE INDEX coupons_status_window_idx ON coupons(status, starts_at, ends_at);

CREATE TABLE voucher_holds (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    coupon_id UUID NOT NULL REFERENCES coupons(id),
    order_id TEXT NOT NULL,
    customer_id TEXT NOT NULL,
    idempotency_key TEXT NOT NULL,
    request_hash TEXT NOT NULL,
    contract_version TEXT NOT NULL,
    discount_amount BIGINT NOT NULL CHECK (discount_amount >= 0),
    currency_code CHAR(3) NOT NULL DEFAULT 'VND',
    state TEXT NOT NULL CHECK (state IN ('HELD', 'FINALIZED', 'RELEASED', 'EXPIRED')),
    resource_version BIGINT NOT NULL DEFAULT 1 CHECK (resource_version > 0),
    expires_at TIMESTAMPTZ NOT NULL,
    finalized_at TIMESTAMPTZ,
    released_at TIMESTAMPTZ,
    release_reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (order_id),
    UNIQUE (idempotency_key),
    CHECK ((state <> 'FINALIZED') OR finalized_at IS NOT NULL),
    CHECK ((state <> 'RELEASED') OR released_at IS NOT NULL)
);
CREATE INDEX voucher_holds_expiry_idx ON voucher_holds(expires_at) WHERE state = 'HELD';
CREATE INDEX voucher_holds_coupon_customer_idx ON voucher_holds(coupon_id, customer_id, state);

CREATE TABLE voucher_usage (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    hold_id UUID NOT NULL UNIQUE REFERENCES voucher_holds(id),
    coupon_id UUID NOT NULL REFERENCES coupons(id),
    order_id TEXT NOT NULL UNIQUE,
    customer_id TEXT NOT NULL,
    discount_amount BIGINT NOT NULL CHECK (discount_amount >= 0),
    finalized_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX voucher_usage_coupon_customer_idx ON voucher_usage(coupon_id, customer_id);

CREATE TABLE command_idempotency (
    idempotency_key TEXT PRIMARY KEY,
    command_type TEXT NOT NULL,
    request_hash TEXT NOT NULL,
    resource_id UUID,
    response_json JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX command_idempotency_expiry_idx ON command_idempotency(expires_at);

CREATE TABLE outbox_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_type TEXT NOT NULL,
    aggregate_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    event_version INTEGER NOT NULL CHECK (event_version > 0),
    payload JSONB NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    published_at TIMESTAMPTZ,
    attempts INTEGER NOT NULL DEFAULT 0,
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_error TEXT
);
CREATE INDEX outbox_events_pending_idx ON outbox_events(next_attempt_at, occurred_at) WHERE published_at IS NULL;
