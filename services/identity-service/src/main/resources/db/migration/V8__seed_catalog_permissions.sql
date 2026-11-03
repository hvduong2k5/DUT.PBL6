-- Khởi tạo danh mục quyền hạn của catalog-service
INSERT INTO iam_permissions (permission_code, resource_code, action_code, risk_level, scope_required, description)
VALUES
    ('CATALOG_VIEW', 'CATALOG', 'VIEW', 'LOW', FALSE, 'Xem danh mục và chi tiết sản phẩm trong hệ thống quản trị'),
    ('CATALOG_MANAGE', 'CATALOG', 'MANAGE', 'HIGH', FALSE, 'Tạo mới, chỉnh sửa thông tin sản phẩm, biến thể và bảng giá kênh bán'),
    ('CATALOG_APPROVE', 'CATALOG', 'APPROVE', 'HIGH', FALSE, 'Phê duyệt, từ chối hoặc đình chỉ kinh doanh sản phẩm OCOP')
ON CONFLICT (permission_code) DO UPDATE 
SET description = EXCLUDED.description,
    risk_level = EXCLUDED.risk_level;

-- Tự động gán toàn bộ quyền Catalog cho vai trò ADMIN tối cao
INSERT INTO iam_role_permissions (role_id, permission_code)
SELECT 'a0000000-0000-0000-0000-000000000001', permission_code
FROM iam_permissions
WHERE resource_code = 'CATALOG'
ON CONFLICT (role_id, permission_code) DO NOTHING;

