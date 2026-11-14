INSERT INTO iam_permissions (permission_code, resource_code, action_code, risk_level, scope_required, description)
VALUES ('INVENTORY_RECEIPT_CREATE', 'INVENTORY', 'RECEIPT_CREATE', 'HIGH', TRUE,
        'Ghi nhận nhập kho theo phạm vi được cấp.')
ON CONFLICT (permission_code) DO NOTHING;
