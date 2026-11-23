-- F-55: the decide half of the local payment case authority. A FINANCE
-- holder with this grant records the resolution decision for a
-- reconciliation case; a different holder resolves the case with it.
-- order-service enforces the maker-checker rule (a decision is void for
-- its own maker), so the grant joins the FINANCE role next to
-- PAYMENT_CASE_RESOLVE instead of forming a separate role.

INSERT INTO iam_permissions (permission_code, resource_code, action_code, risk_level, scope_required, description)
VALUES
    ('PAYMENT_CASE_DECIDE', 'PAYMENT', 'CASE_DECIDE', 'HIGH', FALSE,
     'Ra quyết định xử lý hồ sơ đối soát thanh toán (người quyết phải khác người xử lý).')
ON CONFLICT (permission_code) DO NOTHING;

INSERT INTO iam_role_permissions (role_id, permission_code)
SELECT r.role_id, p.permission_code
FROM iam_roles r
JOIN iam_permissions p ON p.permission_code IN ('PAYMENT_CASE_DECIDE')
WHERE r.role_code = 'FINANCE' AND r.archived_at IS NULL
ON CONFLICT DO NOTHING;
