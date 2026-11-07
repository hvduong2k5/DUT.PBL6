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
- **Soft delete role:** Dùng `is_enabled = FALSE`. Không xóa cứng hàng trong `iam_roles` nếu vẫn có assignment tham chiếu (vi phạm FK `ON DELETE RESTRICT`).
- **UNIQUE role_code:** Sử dụng partial unique index `WHERE is_enabled = TRUE` để cho phép tái sử dụng mã role sau khi xóa mềm.

## 3. Truy vấn đọc Role
- Mọi truy vấn đọc `iam_roles` phải thêm điều kiện `AND is_enabled = TRUE` (hoặc `deleted_at IS NULL` nếu có).
- Mọi truy vấn đọc assignment hiệu lực phải kiểm tra:
  ```sql
  assignment_status = 'ACTIVE'
  AND valid_from <= now
  AND (valid_until IS NULL OR valid_until > now)
  ```

## 4. Keycloak Sync
- Khi gán/thu hồi/hết hạn role, phải gọi `syncPermissionsToKeycloak(principalId)` để cập nhật Keycloak User Attributes.
- **Không xóa realm role ngay khi soft delete trong DB** để tránh làm hỏng JWT/attribute cũ.
- `KeycloakAdminClient.updateUserPermissions(keycloakUserId, permissions)` là điểm kết nối duy nhất.

## 5. API Contract
- Tất cả endpoint admin phải nằm dưới `/api/v1/admin/**`.
- Mọi thay đổi REST API phải cập nhật contract tại `docs/03_api_specs/` (xem quy tắc chung trong `AGENTS.md` → Frontend API Contract Handoff).
- Phân quyền endpoint dùng `@PreAuthorize("hasRole('ADMIN') or hasAuthority('XXX_VIEW/MANAGE')")`.

## 6. Audit Log
- Tất cả thao tác khóa/mở khóa tài khoản, gán/thu hồi role, reactivate, access review phải ghi vào `identity_audit_log`.
- Trường bắt buộc: `actor_principal_id`, `target_principal_id`, `action`, `reason`, `timestamp`.

## 7. Commit Message
- 100% tiếng Anh.
- Ví dụ: `fix(identity): add account reactivate endpoint with permission check`
- Tham khảo `AGENTS.md` phần Commit Message Convention.

## 8. Test
- Viết unit test cho `RoleAssignmentServiceImpl`, `RoleManagementServiceImpl`.
- Viết integration test cho API admin account và role.
- Chạy `./gradlew test` trong `services/identity-service` trước khi commit.

## 9. Tham khảo
- `docs/01_requirements/identity-iam-lifecycle-plan.md`: Kế hoạch chi tiết vòng đời IAM.
- `docs/03_api_specs/iam-role-permission-matrix.md`: Ma trận phân quyền.