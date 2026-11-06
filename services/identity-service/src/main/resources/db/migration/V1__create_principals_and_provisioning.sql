-- Principal metadata only. Credential/session state stays in Keycloak.
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TABLE identity_principals (
    principal_id UUID PRIMARY KEY,
    idp_issuer VARCHAR(512) NOT NULL,
    idp_subject VARCHAR(255),
    principal_type VARCHAR(20) NOT NULL
        CHECK (principal_type IN ('CUSTOMER', 'EMPLOYEE')),
    provisioning_status VARCHAR(20) NOT NULL DEFAULT 'PENDING'
        CHECK (provisioning_status IN ('PENDING', 'ACTIVE', 'FAILED')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    version BIGINT NOT NULL DEFAULT 0 CHECK (version >= 0),
    CONSTRAINT ck_active_principal_has_subject
        CHECK (provisioning_status <> 'ACTIVE' OR idp_subject IS NOT NULL)
);

CREATE UNIQUE INDEX uq_identity_principals_issuer_subject
    ON identity_principals (idp_issuer, idp_subject)
    WHERE idp_subject IS NOT NULL;
CREATE INDEX ix_identity_principals_type_status
    ON identity_principals (principal_type, provisioning_status);

CREATE TRIGGER trg_identity_principals_updated_at
BEFORE UPDATE ON identity_principals
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE registration_operations (
    operation_id UUID PRIMARY KEY,
    operation_type VARCHAR(40) NOT NULL DEFAULT 'CUSTOMER_REGISTER'
        CHECK (operation_type IN ('CUSTOMER_REGISTER')),
    idempotency_key VARCHAR(200) NOT NULL,
    request_fingerprint CHAR(64) NOT NULL,
    principal_id UUID NOT NULL REFERENCES identity_principals(principal_id) ON DELETE RESTRICT,
    operation_status VARCHAR(24) NOT NULL DEFAULT 'PENDING'
        CHECK (operation_status IN ('PENDING', 'IDP_CREATED', 'EVENT_QUEUED', 'COMPLETED', 'FAILED')),
    error_code VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMPTZ,
    CONSTRAINT uq_registration_idempotency UNIQUE (operation_type, idempotency_key)
);
CREATE INDEX ix_registration_operations_status_created
    ON registration_operations (operation_status, created_at);
CREATE TRIGGER trg_registration_operations_updated_at
BEFORE UPDATE ON registration_operations
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE employee_invitations (
    invitation_id UUID PRIMARY KEY,
    principal_id UUID NOT NULL REFERENCES identity_principals(principal_id) ON DELETE RESTRICT,
    invited_by_principal_id UUID REFERENCES identity_principals(principal_id) ON DELETE RESTRICT,
    invitation_status VARCHAR(20) NOT NULL DEFAULT 'PENDING'
        CHECK (invitation_status IN ('PENDING', 'ACCEPTED', 'REVOKED', 'EXPIRED')),
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    accepted_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    CONSTRAINT ck_invitation_expiry_after_creation CHECK (expires_at > created_at)
);
CREATE UNIQUE INDEX uq_employee_invitation_pending_principal
    ON employee_invitations (principal_id)
    WHERE invitation_status = 'PENDING';
CREATE INDEX ix_employee_invitations_status_expiry
    ON employee_invitations (invitation_status, expires_at);
CREATE TRIGGER trg_employee_invitations_updated_at
BEFORE UPDATE ON employee_invitations
FOR EACH ROW EXECUTE FUNCTION set_updated_at();
