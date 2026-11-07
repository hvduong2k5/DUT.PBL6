INSERT INTO iam_permissions (permission_code, resource_code, action_code, risk_level, scope_required, description)
VALUES
    ('ORDER_VIEW', 'ORDER', 'VIEW', 'LOW', TRUE, 'Xem đơn hàng trong phạm vi được cấp'),
    ('ORDER_SENSITIVE_VIEW', 'ORDER', 'VIEW_SENSITIVE', 'HIGH', TRUE, 'Xem thông tin người nhận trong phạm vi được cấp')
ON CONFLICT (permission_code) DO NOTHING;
