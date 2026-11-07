# Agent Rules — Identity Service

Quy tắc này áp dụng cho mọi thay đổi code trong `services/identity-service`. AI agent và lập trình viên phải tuân thủ khi implement, review hoặc test dịch vụ này.

## 1. Cơ sở dữ liệu & Migration
- Tất cả migration nằm ở `src/main/resources/db/migration/`.
- Khi tạo bảng mới hoặc thay đổi schema, phải viết file migration theo định dạng `V{so}__mota_chuc_nang.sql`.
- Không được sửa trực tiếp migration đã commit lên nhánh khác.

### 1.1 Bảng IAM cốt lõi
| Bảng | Mục đích |
|------|----------|
| `identity_principals` | Lưu danh tính chính (principal_type: CUSTOMER, EMPLOYEE) |
| `iam_roles` | Danh mục vai trò (role_code, is_enabled, is_protected) |
| `iam_permissions` | Mã quyền (permission_code, resource_code, action_code) |
| `iam_role_permissions` | Ánh xạ role ↔ permission |
| `iam_role_assignments` | Gán role cho principal kèm scope, valid_from/valid_until, status |
| `identity_audit_log` | Ghi nhật ký kiểm toán (actor, target, action, reason, timestamp) |

## 2. Quy tắc Role & Permission
- **Chỉ EMPLOYEE (`principal_type = EMPLOYEE`) được gán role hành chính.** `principal_type = CUSTOMER` không được gán role `ADMIN` hoặc bất kỳ role quản trị nào. Hiện tại chỉ có role `ADMIN` trong hệ thống.
- **Không self-assignment:** Người dùng không được tự gán role cho chính mình.
- **Phân tách nhiệm vụ:** Người tạo request không được đồng thời là người phê duyệt.
- **Role bảo vệ:** `is_protected = TRUE` (ví dụ `ADMIN`) không được xóa cứng hoặc xóa mềm nếu là admin cuối cùng.
- **Chặn mất Admin hợp lệ cuối cùng:** Trước khi thực hiện `REVOKED` hoặc `TERMINATED` trên Admin, phải kiểm tra xem còn ít nhất 1 Admin active khác trong hệ thống không. Áp dụng cả thao tác đồng thời (concurrent).
- **Soft delete role:** Dùng `is_enabled = FALSE`. Khi soft delete, **tự động đổi `role_code` thành giá trị duy nhất** (ví dụ: `ADMIN_deleted_<timestamp>`) để giải phóng `role_code` gốc và cho phép tái sử dụng.
- **Chỉ xóa mềm khi không còn user đang gán active** (không có `iam_role_assignments` với `assignment_status = 'ACTIVE'` và `valid_until > now` tham chiếu đến role này).
- **Xóa vật lý role chưa sử dụng:** Chỉ xóa vĩnh viễn khi không còn assignment nào tham chiếu role đó.
- **UNIQUE role_code:** Sử dụng partial unique index `WHERE is_enabled = TRUE` để cho phép tái sử dụng mã role sau khi xóa mềm.
- **Các assignment cũ sau khi soft delete:** Phải được chuyển sang `REVOKED`, không được để trỏ nhầm sang role mới có cùng `role_code` (vì `role_id` khác).
- **Kiểm tra phiên bản (optimistic locking):** Khi người dùng lưu form cũ, phải kiểm tra `version` để tránh ghi đè thay đổi mới.

## 3. Chuẩn hóa JWT và đồng bộ quyền (ƯU TIÊN ĐẦU)
- Token JWT phải thể hiện được:
  - **Principal là CUSTOMER hay EMPLOYEE** (`principal_type`)
  - **Principal ID nội bộ** (`principal_id`)
  - **Permission gắn đúng scope** (từ `iam_role_permissions` qua `iam_role_assignments`)
  - **Thời hạn hiệu lực** (`exp`, `valid_from`, `valid_until`)
- Khi đổi permission của role, disable role, hoặc thu hồi assignment → phải cập nhật dữ liệu dùng để phát token cho tất cả nhân viên bị ảnh hưởng (gọi `syncPermissionsToKeycloak(principalId)`).
- Token mới phải nhận quyền mới.
- Quyền có thời hạn không được sống lâu hơn assignment (khi assignment `REVOKED`/`EXPIRED`, token tương ứng phải bị vô hiệu hóa hoặc không thể refresh được quyền cũ).

## 4. Truy vấn đọc Role
- Mọi truy vấn đọc `iam_roles` phải thêm điều kiện `AND is_enabled = TRUE` (hoặc `deleted_at IS NULL` nếu có).
- Mọi truy vấn đọc assignment hiệu lực phải kiểm tra:
  ```sql
  assignment_status = 'ACTIVE'
  AND valid_from <= now
  AND (valid_until IS NULL OR valid_until > now)
  ```

## 5. Keycloak Sync
- Khi gán/thu hồi/hết hạn role, phải gọi `syncPermissionsToKeycloak(principalId)` để cập nhật Keycloak User Attributes.
- Khi đổi permission của role, phải sync lại tất cả principal đang được gán role đó.
- **Không xóa realm role ngay khi soft delete trong DB** để tránh làm hỏng JWT/attribute cũ.
- `KeycloakAdminClient.updateUserPermissions(keycloakUserId, permissions)` là điểm kết nối duy nhất.

## 6. API Contract
- Tất cả endpoint admin phải nằm dưới `/api/v1/admin/**`.
- Mọi thay đổi REST API phải cập nhật contract tại `docs/03_api_specs/` (xem quy tắc chung trong `AGENTS.md` → Frontend API Contract Handoff).
- Phân quyền endpoint dùng `@PreAuthorize("hasRole('ADMIN') or hasAuthority('XXX_VIEW/MANAGE')")`.

## 7. Audit Log
- Tất cả thao tác khóa/mở khóa tài khoản, gán/thu hồi role, reactivate, thay đổi role/permission/assignment phải ghi vào `identity_audit_log`.
- Trường bắt buộc: `actor_principal_id`, `target_principal_id`, `action`, `reason`, `timestamp`.
- Ví dụ action: `ACCOUNT_SUSPEND`, `ACCOUNT_REACTIVATE`, `ROLE_ASSIGN`, `ROLE_REVOKE`, `ROLE_DISABLE`, `PERMISSION_CHANGE`.

## 8. Commit Message
- 100% tiếng Anh.
- Ví dụ: `fix(identity): add account reactivate endpoint with permission check`
- Tham khảo `AGENTS.md` phần Commit Message Convention.

## 9. Test
- Viết unit test cho `RoleAssignmentServiceImpl`, `RoleManagementServiceImpl`.
- Viết integration test cho API admin account và role.
- Chạy `./gradlew test` trong `services/identity-service` trước khi commit.

## 10. Tham khảo
- `docs/01_requirements/identity-iam-lifecycle-plan.md`: Kế hoạch chi tiết vòng đời IAM.
- `docs/03_api_specs/iam-role-permission-matrix.md`: Ma trận phân quyền.