INSERT INTO iam_permissions (permission_code, resource_code, action_code, risk_level, scope_required, description)
VALUES ('MARKETING_EXECUTION_MANAGE', 'MARKETING', 'EXECUTION_MANAGE', 'HIGH', FALSE,
        'Schedule campaigns and record manual channel execution')
ON CONFLICT (permission_code) DO NOTHING;
