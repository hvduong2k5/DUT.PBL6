-- F-42 slice 1: real payment permissions in the identity universe plus a
-- FINANCE role that carries them. Payment permissions are deliberately NOT
-- granted to ADMIN: finance access is assigned per person via this role.
-- order-service maps these IAM grants onto its internal payment.* codes
-- (PAYMENT_VIEW is web-UI only and maps to nothing in order-service).

INSERT INTO iam_permissions (permission_code, resource_code, action_code, risk_level, scope_required, description)
VALUES
    ('PAYMENT_VIEW', 'PAYMENT', 'VIEW', 'LOW', FALSE,
     'Xem trang vận hành thanh toán trong khu vực quản trị.'),
    ('PAYMENT_RECONCILIATION_VIEW', 'PAYMENT', 'RECONCILIATION_VIEW', 'MEDIUM', FALSE,
     'Xem danh sách và chi tiết hồ sơ đối soát thanh toán.'),
    ('PAYMENT_REFUND_VIEW', 'PAYMENT', 'REFUND_VIEW', 'MEDIUM', FALSE,
     'Xem danh sách và chi tiết yêu cầu hoàn tiền (cần kèm PAYMENT_SENSITIVE_VIEW).'),
    ('PAYMENT_SENSITIVE_VIEW', 'PAYMENT', 'SENSITIVE_VIEW', 'HIGH', FALSE,
     'Xem thông tin ngân hàng nhạy cảm trong hồ sơ thanh toán và hoàn tiền.'),
    ('PAYMENT_CASE_RESOLVE', 'PAYMENT', 'CASE_RESOLVE', 'HIGH', FALSE,
     'Tiếp nhận và xử lý hồ sơ đối soát thanh toán.')
ON CONFLICT (permission_code) DO NOTHING;

-- Role-code uniqueness is a PARTIAL unique index (V9, then V13:
-- uq_iam_roles_role_code_unarchived on (role_code) WHERE archived_at IS NULL),
-- so ON CONFLICT (role_code) cannot infer it (SQLSTATE 42P10). Seed with a
-- NOT EXISTS guard instead, which is idempotent under any constraint shape.
INSERT INTO iam_roles (role_id, role_code, role_name, description, is_protected, is_enabled)
SELECT
    'a0000000-0000-0000-0000-000000000002',
    'FINANCE',
    'Tài chính — đối soát thanh toán',
    'Nhân viên tài chính xử lý đối soát và hoàn tiền SePay/VietQR',
    FALSE,
    TRUE
WHERE NOT EXISTS (
    SELECT 1 FROM iam_roles WHERE role_code = 'FINANCE' AND archived_at IS NULL
);

INSERT INTO iam_role_permissions (role_id, permission_code)
SELECT r.role_id, p.permission_code
FROM iam_roles r
JOIN iam_permissions p ON p.permission_code IN (
    'PAYMENT_VIEW',
    'PAYMENT_RECONCILIATION_VIEW',
    'PAYMENT_REFUND_VIEW',
    'PAYMENT_SENSITIVE_VIEW',
    'PAYMENT_CASE_RESOLVE'
)
WHERE r.role_code = 'FINANCE'
ON CONFLICT (role_id, permission_code) DO NOTHING;
