INSERT INTO iam_permissions (permission_code, resource_code, action_code, risk_level, scope_required, description)
VALUES
    ('MARKETING_CAMPAIGN_VIEW', 'MARKETING', 'CAMPAIGN_VIEW', 'LOW', FALSE, 'Read campaigns and approval records'),
    ('MARKETING_PLAN_EDIT', 'MARKETING', 'PLAN_EDIT', 'MEDIUM', FALSE, 'Create campaigns and edit draft revisions'),
    ('MARKETING_BUDGET_EDIT', 'MARKETING', 'BUDGET_EDIT', 'HIGH', FALSE, 'Set the planned campaign budget'),
    ('MARKETING_PLAN_SUBMIT', 'MARKETING', 'PLAN_SUBMIT', 'MEDIUM', FALSE, 'Submit or withdraw a campaign review'),
    ('MARKETING_PLAN_APPROVE', 'MARKETING', 'PLAN_APPROVE', 'HIGH', FALSE, 'Decide a campaign review')
ON CONFLICT (permission_code) DO NOTHING;
