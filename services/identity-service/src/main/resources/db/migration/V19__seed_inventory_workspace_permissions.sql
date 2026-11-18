INSERT INTO iam_permissions (permission_code, resource_code, action_code, risk_level, scope_required, description)
VALUES
 ('INVENTORY_MANAGEMENT_VIEW', 'INVENTORY', 'MANAGEMENT_VIEW', 'MEDIUM', TRUE, 'View inventory management workspace'),
 ('BATCH_VIEW', 'BATCH', 'VIEW', 'MEDIUM', TRUE, 'View inventory batches')
ON CONFLICT (permission_code) DO NOTHING;

INSERT INTO iam_role_permissions (role_id, permission_code)
SELECT 'a0000000-0000-0000-0000-000000000001', permission_code
FROM iam_permissions WHERE permission_code IN ('INVENTORY_MANAGEMENT_VIEW', 'BATCH_VIEW')
ON CONFLICT (role_id, permission_code) DO NOTHING;
