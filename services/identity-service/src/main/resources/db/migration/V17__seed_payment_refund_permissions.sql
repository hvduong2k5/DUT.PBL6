-- F-42 slice 2: the refund half of the local payment authorities.
-- Three grants join the FINANCE role next to the read, resolve and
-- decide grants (V15, V16): APPROVE records a manual-refund approval
-- (and accepts approvals made by others into obligations), EXECUTE
-- carries an accepted refund out and attaches its evidence, VERIFY
-- issues the bank-statement certificate and settles the refund against
-- it. order-service separates the duties per person (the approver never
-- executes, the verifier is neither approver nor executor, the evidence
-- issuer is neither), so the grants share one role instead of forming
-- three roles nobody could staff.

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
