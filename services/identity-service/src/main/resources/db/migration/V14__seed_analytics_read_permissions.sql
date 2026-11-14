INSERT INTO iam_permissions (permission_code, resource_code, action_code, risk_level, scope_required, description)
VALUES
 ('ANALYTICS_DASHBOARD_VIEW', 'ANALYTICS', 'DASHBOARD_VIEW', 'MEDIUM', TRUE, 'Read certified Order analytics'),
 ('ANALYTICS_SUPPLY_VIEW', 'ANALYTICS', 'SUPPLY_VIEW', 'MEDIUM', TRUE, 'Read certified Supply analytics')
ON CONFLICT (permission_code) DO NOTHING;

INSERT INTO iam_role_permissions (role_id, permission_code)
SELECT 'a0000000-0000-0000-0000-000000000001', permission_code
FROM iam_permissions WHERE permission_code IN ('ANALYTICS_DASHBOARD_VIEW', 'ANALYTICS_SUPPLY_VIEW')
ON CONFLICT (role_id, permission_code) DO NOTHING;
