-- Policy rows are versioned; create a new policy_version rather than mutating rule_config in place.
CREATE TABLE specialized_permission_policies (
    policy_id UUID PRIMARY KEY,
    policy_code VARCHAR(120) NOT NULL,
    policy_version INTEGER NOT NULL CHECK (policy_version > 0),
    action_code VARCHAR(160) NOT NULL,
    policy_name VARCHAR(200) NOT NULL,
    policy_type VARCHAR(32) NOT NULL
        CHECK (policy_type IN ('AMOUNT_THRESHOLD', 'PERCENT_THRESHOLD', 'CONTEXT_MATCH')),
    scope_type VARCHAR(32) NOT NULL DEFAULT 'GLOBAL'
        CHECK (scope_type IN ('GLOBAL', 'WAREHOUSE', 'STORE', 'REGION', 'DEPARTMENT', 'BUSINESS_UNIT')),
    scope_id VARCHAR(160),
    rule_config JSONB NOT NULL CHECK (jsonb_typeof(rule_config) = 'object'),
    required_approval_level SMALLINT CHECK (required_approval_level IS NULL OR required_approval_level >= 0),
    fail_closed BOOLEAN NOT NULL DEFAULT TRUE,
    lifecycle_status VARCHAR(16) NOT NULL DEFAULT 'DRAFT'
        CHECK (lifecycle_status IN ('DRAFT', 'ACTIVE', 'RETIRED')),
    valid_from TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    valid_until TIMESTAMPTZ,
    created_by_principal_id UUID REFERENCES identity_principals(principal_id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_special_policy_version UNIQUE (policy_code, policy_version),
    CONSTRAINT ck_special_policy_validity CHECK (valid_until IS NULL OR valid_until > valid_from),
    CONSTRAINT ck_special_policy_scope CHECK (
        (scope_type = 'GLOBAL' AND scope_id IS NULL)
        OR (scope_type <> 'GLOBAL' AND scope_id IS NOT NULL AND length(trim(scope_id)) > 0)
    )
);
CREATE UNIQUE INDEX uq_special_policy_active_version
    ON specialized_permission_policies (policy_code)
    WHERE lifecycle_status = 'ACTIVE';
CREATE INDEX ix_special_policy_action_status
    ON specialized_permission_policies (action_code, lifecycle_status, scope_type, scope_id);

CREATE TABLE authorization_decisions (
    decision_id UUID PRIMARY KEY,
    actor_kind VARCHAR(16) NOT NULL
        CHECK (actor_kind IN ('CUSTOMER', 'EMPLOYEE', 'SERVICE', 'SYSTEM')),
    actor_principal_id UUID REFERENCES identity_principals(principal_id) ON DELETE RESTRICT,
    actor_client_id VARCHAR(200),
    action_code VARCHAR(160) NOT NULL,
    decision VARCHAR(24) NOT NULL
        CHECK (decision IN ('ALLOW', 'DENY', 'REQUIRES_APPROVAL', 'INDETERMINATE')),
    reason_code VARCHAR(120) NOT NULL,
    policy_id UUID REFERENCES specialized_permission_policies(policy_id) ON DELETE RESTRICT,
    policy_version INTEGER,
    correlation_id VARCHAR(160),
    context_summary JSONB NOT NULL DEFAULT '{}'::jsonb
        CHECK (jsonb_typeof(context_summary) = 'object'),
    evaluated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT ck_authorization_actor CHECK (
        (actor_kind IN ('CUSTOMER', 'EMPLOYEE') AND actor_principal_id IS NOT NULL)
        OR (actor_kind = 'SERVICE' AND actor_client_id IS NOT NULL)
        OR actor_kind = 'SYSTEM'
    )
);
CREATE INDEX ix_authorization_decisions_actor_time
    ON authorization_decisions (actor_principal_id, evaluated_at DESC);
CREATE INDEX ix_authorization_decisions_action_time
    ON authorization_decisions (action_code, evaluated_at DESC);
CREATE INDEX ix_authorization_decisions_correlation
    ON authorization_decisions (correlation_id)
    WHERE correlation_id IS NOT NULL;
