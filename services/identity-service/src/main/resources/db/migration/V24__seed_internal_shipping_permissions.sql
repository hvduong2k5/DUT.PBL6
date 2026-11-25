-- Additive catalog/bundle seed only: never create principals or GLOBAL assignments.
INSERT INTO iam_permissions (permission_code, resource_code, action_code, risk_level, scope_required, description)
VALUES ('SHIPMENT_ASSIGN', 'SHIPMENT', 'ASSIGN', 'MEDIUM', TRUE, 'Assign active company drivers with GLOBAL scope'),
       ('SHIPMENT_EXECUTE', 'SHIPMENT', 'EXECUTE', 'MEDIUM', TRUE, 'Read and act on own assigned company shipments; GLOBAL grant still requires ownership')
ON CONFLICT (permission_code) DO NOTHING;

INSERT INTO iam_role_permissions (role_id, permission_code)
SELECT r.role_id, p.permission_code FROM iam_roles r
JOIN iam_permissions p ON p.permission_code IN ('SHIPMENT_ASSIGN', 'SHIPMENT_EXECUTE')
WHERE r.role_code = 'ADMIN' AND r.is_enabled = TRUE AND r.archived_at IS NULL
ON CONFLICT (role_id, permission_code) DO NOTHING;

-- Deployment handoff: call PermissionSyncService through the existing administrative
-- sync-permissions endpoint for affected holders (forceRelogin=false), then refresh
-- sessions/JWTs. This SQL cannot mutate already-issued tokens or Keycloak attributes.
-- USER_VIEW already exists; dispatchers need its GLOBAL grant for employee lookup.
