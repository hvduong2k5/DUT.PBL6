-- Migration V21: Seed bootstrap admin principal and grant traceability permissions to ADMIN role (Baseline 7-B)

-- 1. Grant all traceability permissions from V20 to the supreme ADMIN role
INSERT INTO iam_role_permissions (role_id, permission_code)
SELECT 'a0000000-0000-0000-0000-000000000001', permission_code
FROM iam_permissions
WHERE permission_code LIKE 'traceability.%'
ON CONFLICT (role_id, permission_code) DO NOTHING;

-- 2. Bootstrap the default administrator principal for employee access
INSERT INTO identity_principals (
    principal_id, idp_issuer, idp_subject, principal_type, provisioning_status, version
) VALUES (
    'a0000000-0000-0000-0000-000000000002',
    'omama',
    'a0000000-0000-0000-0000-000000000002',
    'EMPLOYEE',
    'ACTIVE',
    0
) ON CONFLICT (principal_id) DO NOTHING;

-- 3. Assign the supreme ADMIN role with GLOBAL scope to the bootstrap admin principal
INSERT INTO iam_role_assignments (
    assignment_id, principal_id, role_id, scope_type, scope_id, assignment_status, valid_from, valid_until, reason, version
) VALUES (
    'a0000000-0000-0000-0000-000000000002',
    'a0000000-0000-0000-0000-000000000002',
    'a0000000-0000-0000-0000-000000000001',
    'GLOBAL',
    NULL,
    'ACTIVE',
    CURRENT_TIMESTAMP,
    NULL,
    'Initial bootstrap system administrator (Baseline 7-B)',
    0
) ON CONFLICT (assignment_id) DO NOTHING;
