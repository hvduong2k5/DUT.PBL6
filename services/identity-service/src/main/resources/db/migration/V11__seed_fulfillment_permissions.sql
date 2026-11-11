-- Khởi tạo danh mục quyền hạn của fulfillment-service
INSERT INTO iam_permissions (permission_code, resource_code, action_code, risk_level, scope_required, description)
VALUES
    ('FULFILLMENT_TASK_READ', 'FULFILLMENT_TASK', 'READ', 'LOW', FALSE, 'Xem danh sách và chi tiết công việc đóng gói'),
    ('FULFILLMENT_TASK_READ_HISTORY', 'FULFILLMENT_TASK', 'READ_HISTORY', 'LOW', FALSE, 'Xem lịch sử thay đổi của công việc đóng gói'),
    ('FULFILLMENT_TASK_ASSIGN_SELF', 'FULFILLMENT_TASK', 'ASSIGN_SELF', 'LOW', FALSE, 'Tự nhận hoặc hủy nhận công việc đóng gói'),
    ('FULFILLMENT_TASK_ASSIGN_OTHER', 'FULFILLMENT_TASK', 'ASSIGN_OTHER', 'HIGH', FALSE, 'Gán công việc đóng gói cho nhân viên khác'),
    ('FULFILLMENT_TASK_WORK', 'FULFILLMENT_TASK', 'WORK', 'MEDIUM', FALSE, 'Thực hiện các bước đóng gói trên công việc được giao'),
    ('FULFILLMENT_TASK_SEAL', 'FULFILLMENT_TASK', 'SEAL', 'MEDIUM', FALSE, 'Niêm phong kiện hàng sau khi đóng gói'),
    ('FULFILLMENT_TASK_BLOCK', 'FULFILLMENT_TASK', 'BLOCK', 'MEDIUM', FALSE, 'Đánh dấu chặn công việc đóng gói'),
    ('FULFILLMENT_TASK_RESOLVE_BLOCK', 'FULFILLMENT_TASK', 'RESOLVE_BLOCK', 'MEDIUM', FALSE, 'Gỡ chặn công việc đóng gói'),
    ('FULFILLMENT_MEDIA_UPLOAD', 'FULFILLMENT_MEDIA', 'UPLOAD', 'MEDIUM', FALSE, 'Tải lên video bằng chứng đóng gói'),
    ('FULFILLMENT_MEDIA_READ_METADATA', 'FULFILLMENT_MEDIA', 'READ_METADATA', 'LOW', FALSE, 'Xem metadata của bằng chứng đóng gói'),
    ('FULFILLMENT_MEDIA_VIEW', 'FULFILLMENT_MEDIA', 'VIEW', 'LOW', FALSE, 'Xem nội dung bằng chứng đóng gói')
ON CONFLICT (permission_code) DO UPDATE
SET description = EXCLUDED.description,
    risk_level = EXCLUDED.risk_level;

-- Tự động gán các quyền Fulfillment cho vai trò ADMIN tối cao, NGOẠI TRỪ
-- FULFILLMENT_TASK_ASSIGN_OTHER: hành vi hiện tại của fulfillment-service
-- cố ý không cấp quyền này cho ADMIN (auth test bảo vệ điều đó); quyền chỉ
-- được seed để sau này gán cho role giám sát kho khi team quyết định.
INSERT INTO iam_role_permissions (role_id, permission_code)
SELECT 'a0000000-0000-0000-0000-000000000001', permission_code
FROM iam_permissions
WHERE resource_code IN ('FULFILLMENT_TASK', 'FULFILLMENT_MEDIA')
  AND permission_code <> 'FULFILLMENT_TASK_ASSIGN_OTHER'
ON CONFLICT (role_id, permission_code) DO NOTHING;
