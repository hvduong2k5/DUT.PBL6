-- V22 brings the dev Flyway chain to parity with release/v0.1.0-rc.1 for
-- the FINANCE payment grants. On the rc.1 chain these statements shipped
-- as V16 (PAYMENT_CASE_DECIDE) and V17 (the three PAYMENT_REFUND_*
-- action grants); the dev chain never received them (its V16-V18 slots
-- stayed empty and V18's analytics seed landed here as V14), so a fresh
-- dev-chain database ends up with a FINANCE role missing exactly these
-- four grants while order-service expects them for the F-42 step-up
-- payment workspace.
--
-- The statements below are the rc.1 V16+V17 bodies unchanged. Every
-- insert is idempotent (ON CONFLICT DO NOTHING): the production
-- identity_db already carries these rows (applied directly during the
-- 2026-11-20 auth rebuild, GĐ1), so this migration is a no-op there and
-- only takes effect on fresh chains. The next free version after this
-- one is V23 (reserved for the promotion-service seed).

-- From rc.1 V16 — F-55: the decide half of the local payment case
-- authority. A FINANCE holder with this grant records the resolution
-- decision for a reconciliation case; a different holder resolves the
-- case with it. order-service enforces the maker-checker rule (a
-- decision is void for its own maker), so the grant joins the FINANCE
-- role next to PAYMENT_CASE_RESOLVE instead of forming a separate role.

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

-- From rc.1 V17 — F-42 slice 2: the refund half of the local payment
-- authorities. Three grants join the FINANCE role next to the read,
-- resolve and decide grants (V15, V22): APPROVE records a manual-refund
-- approval (and accepts approvals made by others into obligations),
-- EXECUTE carries an accepted refund out and attaches its evidence,
-- VERIFY issues the bank-statement certificate and settles the refund
-- against it. order-service separates the duties per person (the
-- approver never executes, the verifier is neither approver nor
-- executor, the evidence issuer is neither), so the grants share one
-- role instead of forming three roles nobody could staff.

INSERT INTO iam_permissions (permission_code, resource_code, action_code, risk_level, scope_required, description)
VALUES
    ('PAYMENT_REFUND_APPROVE', 'PAYMENT', 'REFUND_APPROVE', 'HIGH', FALSE,
     'Phê duyệt hoàn tiền thủ công và tiếp nhận phê duyệt của người khác thành nghĩa vụ hoàn tiền.'),
    ('PAYMENT_REFUND_EXECUTE', 'PAYMENT', 'REFUND_EXECUTE', 'HIGH', FALSE,
     'Thực thi hoàn tiền thủ công đã được duyệt và đính kèm chứng từ sao kê.'),
    ('PAYMENT_REFUND_VERIFY', 'PAYMENT', 'REFUND_VERIFY', 'HIGH', FALSE,
     'Cấp chứng nhận sao kê và xác minh hoàn tiền (phải khác người duyệt và người thực thi).')
ON CONFLICT (permission_code) DO NOTHING;

INSERT INTO iam_role_permissions (role_id, permission_code)
SELECT r.role_id, p.permission_code
FROM iam_roles r
JOIN iam_permissions p ON p.permission_code IN ('PAYMENT_REFUND_APPROVE', 'PAYMENT_REFUND_EXECUTE', 'PAYMENT_REFUND_VERIFY')
WHERE r.role_code = 'FINANCE' AND r.archived_at IS NULL
ON CONFLICT DO NOTHING;
