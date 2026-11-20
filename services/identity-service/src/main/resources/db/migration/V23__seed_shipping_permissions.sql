-- V22 is reserved by the Promotion integration; do not renumber published seeds.
-- Deploy the reserved V22 first. If release order changes, allocate the unreleased
-- later seed above V23 instead of silently skipping it with Flyway outOfOrder=false.
INSERT INTO iam_permissions (permission_code, resource_code, action_code, risk_level, scope_required, description)
VALUES ('SHIPMENT_VIEW', 'SHIPMENT', 'VIEW', 'LOW', TRUE, 'Read shipping administration with GLOBAL scope')
ON CONFLICT (permission_code) DO NOTHING;

INSERT INTO iam_role_permissions (role_id, permission_code)
SELECT r.role_id, p.permission_code
FROM iam_roles r
JOIN iam_permissions p ON p.permission_code = 'SHIPMENT_VIEW'
WHERE r.role_code = 'ADMIN' AND r.is_enabled = TRUE AND r.archived_at IS NULL
ON CONFLICT (role_id, permission_code) DO NOTHING;

-- Additive grant: sync affected employee principals through PermissionSyncService
-- with forceRelogin=false, then refresh JWTs. Existing scoped assignments retain
-- their scope; this migration does not create principals or GLOBAL assignments.
