-- Promotion coupon administration is explicitly granted through IAM, not a
-- realm-role bypass. V19-V21 are already allocated; never backfill V18.
INSERT INTO iam_permissions (permission_code, resource_code, action_code, risk_level, scope_required, description)
VALUES
    ('PROMOTION_VIEW', 'PROMOTION', 'VIEW', 'LOW', TRUE, 'Read coupon administration with GLOBAL scope'),
    ('PROMOTION_MANAGE', 'PROMOTION', 'MANAGE', 'HIGH', TRUE, 'Create, edit, publish and pause coupons with GLOBAL scope')
ON CONFLICT (permission_code) DO NOTHING;

-- Assign permissions to the existing enabled ADMIN role only. Do not create
-- principals/assignments or reactivate archived roles. A scoped employee role
-- assignment does not become GLOBAL: Promotion still checks the signed scope.
INSERT INTO iam_role_permissions (role_id, permission_code)
SELECT r.role_id, p.permission_code
FROM iam_roles r
JOIN iam_permissions p ON p.permission_code IN ('PROMOTION_VIEW', 'PROMOTION_MANAGE')
WHERE r.role_code = 'ADMIN' AND r.is_enabled = TRUE AND r.archived_at IS NULL
ON CONFLICT (role_id, permission_code) DO NOTHING;

-- After deploying, sync affected principals through PermissionSyncService
-- (forceRelogin=false for additive grants) and refresh tokens. SQL seed data
-- cannot rewrite already-issued JWTs or Keycloak permission attributes.
