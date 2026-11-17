INSERT INTO iam_permissions (permission_code, resource_code, action_code, risk_level, scope_required, description)
VALUES
    ('MARKETING_KPI_VIEW', 'MARKETING', 'KPI_VIEW', 'LOW', FALSE, 'Read channel execution metrics'),
    ('MARKETING_CHANNEL_MANAGE', 'MARKETING', 'CHANNEL_MANAGE', 'MEDIUM', FALSE, 'Record and correct manual channel metrics')
ON CONFLICT (permission_code) DO NOTHING;
