CREATE TABLE identity_audit_log (
    audit_id UUID PRIMARY KEY,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actor_kind VARCHAR(16) NOT NULL
        CHECK (actor_kind IN ('CUSTOMER', 'EMPLOYEE', 'SERVICE', 'SYSTEM', 'ANONYMOUS')),
    actor_principal_id UUID REFERENCES identity_principals(principal_id) ON DELETE RESTRICT,
    actor_client_id VARCHAR(200),
    action_code VARCHAR(160) NOT NULL,
    target_type VARCHAR(120),
    target_id VARCHAR(200),
    outcome VARCHAR(16) NOT NULL CHECK (outcome IN ('SUCCESS', 'DENIED', 'FAILURE')),
    reason_code VARCHAR(120),
    request_id VARCHAR(160),
    correlation_id VARCHAR(160),
    source_ip INET,
    details JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(details) = 'object')
);
CREATE INDEX ix_identity_audit_actor_time
    ON identity_audit_log (actor_principal_id, occurred_at DESC);
CREATE INDEX ix_identity_audit_action_time
    ON identity_audit_log (action_code, occurred_at DESC);
CREATE INDEX ix_identity_audit_target
    ON identity_audit_log (target_type, target_id, occurred_at DESC);
CREATE INDEX ix_identity_audit_correlation
    ON identity_audit_log (correlation_id)
    WHERE correlation_id IS NOT NULL;

-- Treat this table as append-only. Restrict UPDATE/DELETE in database grants for the runtime DB role.
