CREATE TABLE iam_roles (
    role_id UUID PRIMARY KEY,
    role_code VARCHAR(100) NOT NULL UNIQUE,
    role_name VARCHAR(160) NOT NULL,
    description TEXT,
    is_protected BOOLEAN NOT NULL DEFAULT FALSE,
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    version BIGINT NOT NULL DEFAULT 0 CHECK (version >= 0),
    created_by_principal_id UUID REFERENCES identity_principals(principal_id) ON DELETE RESTRICT,
    updated_by_principal_id UUID REFERENCES identity_principals(principal_id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX ix_iam_roles_enabled ON iam_roles (is_enabled, role_code);
CREATE TRIGGER trg_iam_roles_updated_at
BEFORE UPDATE ON iam_roles
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE iam_permissions (
    permission_code VARCHAR(160) PRIMARY KEY,
    resource_code VARCHAR(100) NOT NULL,
    action_code VARCHAR(100) NOT NULL,
    risk_level VARCHAR(16) NOT NULL DEFAULT 'LOW'
        CHECK (risk_level IN ('LOW', 'MEDIUM', 'HIGH', 'CRITICAL')),
    scope_required BOOLEAN NOT NULL DEFAULT FALSE,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_iam_permission_resource_action UNIQUE (resource_code, action_code)
);

CREATE TABLE iam_role_permissions (
    role_id UUID NOT NULL REFERENCES iam_roles(role_id) ON DELETE RESTRICT,
    permission_code VARCHAR(160) NOT NULL REFERENCES iam_permissions(permission_code) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (role_id, permission_code)
);
CREATE INDEX ix_iam_role_permissions_permission ON iam_role_permissions (permission_code);

CREATE TABLE iam_role_assignments (
    assignment_id UUID PRIMARY KEY,
    principal_id UUID NOT NULL REFERENCES identity_principals(principal_id) ON DELETE RESTRICT,
    role_id UUID NOT NULL REFERENCES iam_roles(role_id) ON DELETE RESTRICT,
    scope_type VARCHAR(32) NOT NULL DEFAULT 'GLOBAL'
        CHECK (scope_type IN ('GLOBAL', 'WAREHOUSE', 'STORE', 'REGION', 'DEPARTMENT', 'BRANCH', 'BUSINESS_UNIT')),
    scope_id VARCHAR(160),
    assignment_status VARCHAR(16) NOT NULL DEFAULT 'ACTIVE'
        CHECK (assignment_status IN ('ACTIVE', 'REVOKED', 'EXPIRED')),
    valid_from TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    valid_until TIMESTAMPTZ,
    assigned_by_principal_id UUID REFERENCES identity_principals(principal_id) ON DELETE RESTRICT,
    revoked_by_principal_id UUID REFERENCES identity_principals(principal_id) ON DELETE RESTRICT,
    reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    version BIGINT NOT NULL DEFAULT 0 CHECK (version >= 0),
    CONSTRAINT ck_iam_assignment_validity CHECK (valid_until IS NULL OR valid_until > valid_from),
    CONSTRAINT ck_iam_assignment_scope CHECK (
        (scope_type = 'GLOBAL' AND scope_id IS NULL)
        OR (scope_type <> 'GLOBAL' AND scope_id IS NOT NULL AND length(trim(scope_id)) > 0)
    )
);
CREATE UNIQUE INDEX uq_iam_active_role_assignment_scope
    ON iam_role_assignments (principal_id, role_id, scope_type, (COALESCE(scope_id, '')))
    WHERE assignment_status = 'ACTIVE';
CREATE INDEX ix_iam_assignment_principal_effective
    ON iam_role_assignments (principal_id, assignment_status, valid_from, valid_until);
CREATE INDEX ix_iam_assignment_role_scope
    ON iam_role_assignments (role_id, scope_type, scope_id, assignment_status);
CREATE TRIGGER trg_iam_role_assignments_updated_at
BEFORE UPDATE ON iam_role_assignments
FOR EACH ROW EXECUTE FUNCTION set_updated_at();
