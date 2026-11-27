-- V23/V24 belong to Shipping; V27 belongs to Promotion. Coordinate the complete
-- migration chain before deployment: applying V28 first does not backfill them.
-- Add permissions only; never create principals, roles or GLOBAL assignments.
-- After migration, sync affected ADMIN principals through PermissionSyncService
-- (forceRelogin=false), then refresh their tokens. SQL cannot update signed JWTs.
INSERT INTO iam_permissions (permission_code, resource_code, action_code, risk_level, scope_required, description)
VALUES
    ('PROCUREMENT_VIEW', 'PROCUREMENT', 'VIEW', 'LOW', TRUE,
     'Read supplier, material, offer, purchase order and supply plan data with GLOBAL scope.'),
    ('PROCUREMENT_MANAGE', 'PROCUREMENT', 'MANAGE', 'HIGH', TRUE,
     'Manage procurement master data, contracts, purchase orders and supply plans with GLOBAL scope.'),
    ('PROCUREMENT_APPROVE', 'PROCUREMENT', 'APPROVE', 'HIGH', TRUE,
     'Approve or reject purchase orders and supply plans with GLOBAL scope; maker-checker remains mandatory.'),
    ('PROCUREMENT_RECEIVE', 'PROCUREMENT', 'RECEIVE', 'HIGH', TRUE,
     'Create, read and inspect goods receipts in the assigned warehouse or GLOBAL scope.'),
    ('PROCUREMENT_VARIANCE_RESOLVE', 'PROCUREMENT', 'VARIANCE_RESOLVE', 'HIGH', TRUE,
     'Resolve receipt variances in the assigned warehouse or GLOBAL scope.')
ON CONFLICT (permission_code) DO NOTHING;

INSERT INTO iam_role_permissions (role_id, permission_code)
SELECT r.role_id, p.permission_code
FROM iam_roles r
JOIN iam_permissions p ON p.permission_code IN (
    'PROCUREMENT_VIEW', 'PROCUREMENT_MANAGE', 'PROCUREMENT_APPROVE',
    'PROCUREMENT_RECEIVE', 'PROCUREMENT_VARIANCE_RESOLVE'
)
WHERE r.role_code = 'ADMIN' AND r.is_enabled = TRUE AND r.archived_at IS NULL
ON CONFLICT (role_id, permission_code) DO NOTHING;
