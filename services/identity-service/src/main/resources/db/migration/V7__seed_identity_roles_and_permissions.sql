-- 1. Khởi tạo vai trò ADMIN tối cao
INSERT INTO iam_roles (role_id, role_code, role_name, description, is_protected, is_enabled)
VALUES (
    'a0000000-0000-0000-0000-000000000001',
    'ADMIN',
    'Quản trị viên toàn hệ thống',
    'Toàn quyền quản trị tài khoản, phân quyền RBAC và điều hành toàn bộ sàn',
    TRUE,
    TRUE
) ON CONFLICT (role_code) DO NOTHING;

-- 2. Khởi tạo danh mục quyền hạn của identity-service
INSERT INTO iam_permissions (permission_code, resource_code, action_code, risk_level, scope_required, description)
VALUES
    -- Nhóm Quản trị Tài khoản & Người dùng
    ('USER_VIEW', 'USER', 'VIEW', 'LOW', FALSE, 'Xem danh sách và chi tiết thông tin người dùng'),
    ('USER_STATUS_MANAGE', 'USER', 'UPDATE_STATUS', 'HIGH', FALSE, 'Khóa, mở khóa hoặc đình chỉ tài khoản người dùng'),
    ('USER_PASSWORD_RESET', 'USER', 'RESET_PASSWORD', 'HIGH', FALSE, 'Đặt lại mật khẩu cho tài khoản người dùng'),
    ('USER_SESSION_REVOKE', 'USER', 'REVOKE_SESSION', 'MEDIUM', FALSE, 'Cưỡng chế đăng xuất và thu hồi phiên làm việc'),
    ('EMPLOYEE_INVITE', 'INVITATION', 'INVITE', 'HIGH', FALSE, 'Mời nhân viên mới tham gia hệ thống qua email công vụ'),
    ('EMPLOYEE_INVITATION_REVOKE', 'INVITATION', 'REVOKE', 'MEDIUM', FALSE, 'Thu hồi lời mời nhân viên'),

    -- Nhóm Quản trị Vai trò & Quyền hạn (IAM & RBAC)
    ('ROLE_VIEW', 'ROLE', 'VIEW', 'LOW', FALSE, 'Xem danh mục các vai trò và mô tả chi tiết'),
    ('ROLE_MANAGE', 'ROLE', 'MANAGE', 'HIGH', FALSE, 'Tạo mới, chỉnh sửa hoặc xóa vai trò nghiệp vụ'),
    ('ROLE_ASSIGN', 'ROLE_ASSIGNMENT', 'ASSIGN', 'HIGH', FALSE, 'Gán hoặc thu hồi vai trò cho nhân viên'),
    ('PERMISSION_VIEW', 'PERMISSION', 'VIEW', 'LOW', FALSE, 'Xem danh mục các quyền hạn trong hệ thống'),
    ('PERMISSION_MANAGE', 'ROLE_PERMISSION', 'MANAGE', 'CRITICAL', FALSE, 'Gán hoặc gỡ quyền hạn khỏi vai trò'),

    -- Nhóm Nhật ký Kiểm toán (Audit)
    ('AUDIT_LOG_VIEW', 'AUDIT_LOG', 'VIEW', 'MEDIUM', FALSE, 'Tra cứu nhật ký kiểm toán an ninh'),
    ('AUDIT_LOG_EXPORT', 'AUDIT_LOG', 'EXPORT', 'HIGH', FALSE, 'Xuất dữ liệu báo cáo kiểm toán')
ON CONFLICT (permission_code) DO UPDATE 
SET description = EXCLUDED.description,
    risk_level = EXCLUDED.risk_level;

-- 3. Gán toàn bộ quyền trên cho vai trò ADMIN
INSERT INTO iam_role_permissions (role_id, permission_code)
SELECT 'a0000000-0000-0000-0000-000000000001', permission_code
FROM iam_permissions
ON CONFLICT (role_id, permission_code) DO NOTHING;

