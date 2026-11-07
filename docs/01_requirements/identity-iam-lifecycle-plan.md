# Kế hoạch bổ sung / cập nhật / loại bỏ — Identity Service IAM

**Mục tiêu:** Hoàn thiện mô hình phân quyền, vòng đời tài khoản và role trong `identity-service` để đáp ứng yêu cầu bảo mật, kiểm soát và kiểm tra (audit) cho hệ thống OCOP.

**Phạm vi:** `services/identity-service` (Java Spring Boot + MyBatis + Keycloak), bảng `iam_roles`, `iam_permissions`, `iam_role_permissions`, `iam_role_assignments`, `identity_principals`.

---

## 1. Trạng thái hiện tại (Gap analysis)

| Điểm | Trạng thái hiện tại | Vấn đề |
|------|-------------------|--------|
| 1. Chuẩn hóa JWT & đồng bộ quyền | JWT, Keycloak attributes và grants đọc chưa thống nhất. Khi đổi permission của role, disable role hoặc thu hồi assignment, không cập nhật dữ liệu phát token kịp thời. | Token cũ vẫn sống lâu hơn assignment, quyền mới không được phản chiếu kịp vào token. |
| 2. An toàn Super Admin & lifecycle role | Chỉ có `ADMIN` (super-admin). Chưa có cơ chế chặn khóa/thu hồi Admin hợp lệ cuối cùng. Chưa có audit chi tiết cho thay đổi role/permission/assignment. | Rủi ro khoá cửa hệ thống nếu xóa nhầm Admin cuối cùng. |
| 4. Bộ trạng thái Account | `provisioningStatus` (`ACTIVE`, `PENDING_VERIFICATION`) trong DB và `customer_status`. Chưa có endpoint mở khóa/reactivate rõ ràng. | Cần thêm API admin để CRUD tài khoản, khóa tạm/vĩnh viễn, mở khóa/reactivate. |
| 5. Thời gian thu hồi Role | JWT exp do Keycloak cấp; refresh token maxAge 28.800s (8 giờ) trong `oidc.ts`. Không có cấu hình propagate thu hồi role lên session/cache. | Chốt thời gian tối đa (5–15 phút) và xử lý phiên đang thực hiện giao dịch quan trọng. |
| 8. Danh sách Role | Chỉ thấy `ADMIN` (`ROLE_ADMIN` từ Keycloak realm role). `CUSTOMER` và `EMPLOYEE` là `principal_type` (loại tài khoản), không phải role phân quyền. | Chưa có quy trình sửa/clone/archive role và cơ chế bảo vệ Admin hợp lệ cuối cùng. |
| 9. Ma trận thẩm quyền | Không thấy mã nguồn triển khai ma trận chi tiết, self-assignment, break-glass. | Cần bổ sung tài liệu thiết kế (role_permission_matrix.md). |
| 10. Assignment tạm thời | Không thấy mã nguồn. | Cần thiết kế thời điểm hiệu lực, timezone, nhắc hết hạn, người chịu trách nhiệm gia hạn. |
| Soft delete Role | `iam_roles` có `is_enabled`, `is_protected`. `UNIQUE(role_code)` chặn tạo lại mã trùng sau xóa mềm. | Cần partial unique index, centralized `WHERE is_enabled = TRUE`, chỉ xóa mềm khi không còn user nào tham chiếu. |

---

## 2. Các mục cần chốt

### 1. Chuẩn hóa JWT và đồng bộ quyền trong Identity (ƯU TIÊN ĐẦU)

Token JWT cần thể hiện được:
- **Principal là CUSTOMER hay EMPLOYEE** (`principal_type`)
- **Principal ID nội bộ** (`principal_id`)
- **Permission gắn đúng scope** (từ `iam_role_permissions` qua `iam_role_assignments`)
- **Thời hạn hiệu lực** (`exp`, `valid_from`, `valid_until`)

Quy trình khi có thay đổi quyền:
- Khi đổi permission của role, disable role, hoặc thu hồi assignment → phải cập nhật dữ liệu dùng để phát token cho tất cả nhân viên bị ảnh hưởng.
- Token mới phải nhận quyền mới.
- Quyền có thời hạn không được sống lâu hơn assignment (khi assignment `REVOKED`/`EXPIRED`, token tương ứng phải bị vô hiệu hóa hoặc không thể refresh được quyền cũ).

### 2. Hoàn thiện an toàn cho Super Admin và lifecycle role

- **Giữ ADMIN là Super Admin hiện tại.**
- **Chặn khóa/thu hồi quyền làm mất Admin hợp lệ cuối cùng:** Kiểm tra trước khi thực hiện `REVOKED` hoặc `TERMINATED` trên Admin cuối cùng, kể cả thao tác đồng thời (concurrent).
- **Ghi Audit cho thay đổi role, permission và assignment.**
- **Ngừng hiệu lực/archive role đã sử dụng:** Không xóa cứng role đã từng có user gán, chỉ xóa mềm (`is_enabled = FALSE`) khi không còn user nào tham chiếu.
- **Chốt và thực thi điều kiện xóa vật lý role chưa sử dụng:** Chỉ xóa vĩnh viễn khi không còn assignment nào tham chiếu (kiểm tra `iam_role_assignments`).
- **Kiểm tra phiên bản khi người dùng lưu form cũ:** Tránh ghi đè thay đổi mới (optimistic locking với `version`).

### 4. Bộ trạng thái Account & quyền mở khóa/reactivate
- **Trạng thái tài khoản:** `ACTIVE`, `PENDING_VERIFICATION`, `SUSPENDED` (khóa tạm), `TERMINATED` (khóa vĩnh viễn).
- **Quy trình khóa:**
  - Khóa tạm: `provisioningStatus = SUSPENDED`, ghi lý do, người khóa, thời gian.
  - Khóa vĩnh viễn: `provisioningStatus = TERMINATED`, yêu cầu phê duyệt theo ma trận thẩm quyền nếu là actor đặc quyền (ADMIN, MANAGER...).
- **Quy trình mở khóa/reactivate:** Yêu cầu permission phù hợp (admin/manager), ghi audit log.
- **API cần bổ sung:** `PATCH /api/v1/admin/accounts/{id}/status`, `POST /api/v1/admin/accounts/{id}/reactivate`, `GET /api/v1/admin/accounts`.

### 5. Thời gian tối đa khóa/thu hồi Role hiệu lực lên session/cache
- **Access Token:** Giữ nguyên exp do Keycloak cấp (khuyên dùng ngắn, ví dụ 5–15 phút cho tài khoản nhạy cảm).
- **Refresh Token:** maxAge 28.800 giây (8 giờ).
- **Thu hồi Role:** Khi `iam_role_assignments` chuyển sang `REVOKED` hoặc `EXPIRED`, identity-service phải sync ngay lên Keycloak (`syncPermissionsToKeycloak`).
- **Phiên đang thực hiện giao dịch quan trọng:** Cho phép hoàn thành giao dịch đang chạy hoặc chặn ngay tuỳ mức độ rủi ro (cần đánh giá theo từng service gọi).

### 8. Danh sách Role hệ thống, Role đặc quyền & bảo vệ Admin cuối cùng
- **Role hệ thống:** Chỉ có `ADMIN` (super-admin, từ Keycloak realm role → `ROLE_ADMIN` trong Spring Security).
- **`principal_type` (không phải role phân quyền):** `CUSTOMER`, `EMPLOYEE` — dùng để phân loại loại tài khoản, không gắn chặt với quyền cụ thể.
- **Role đặc quyền:** Hiện tại chỉ `ADMIN` là role đặc quyền duy nhất. Các role mở rộng (`USER_MANAGER`, `ROLE_MANAGER`, `AUDITOR`) cần được chốt thêm trong thiết kế kỹ thuật nếu hệ thống yêu cầu.
- **Bảo vệ Admin hợp lệ cuối cùng:** Không cho phép xóa hoặc thu hồi toàn quyền của Admin cuối cùng trong hệ thống (tránh khoá cửa). Cần kiểm tra trước khi thực hiện `REVOKED` hoặc `TERMINATED` trên admin cuối cùng.

### 9. Ma trận thẩm quyền gán Role, phân tách nhiệm vụ, break-glass
- **Ma trận:** Xem file `docs/03_api_specs/iam-role-permission-matrix.md`.
- **Phân tách nhiệm vụ (SoD):** Người tạo request không được đồng thời là người phê duyệt cùng request.
- **Self-assignment:** Người dùng không được tự gán role cho mình.
- **Break-glass:** Quy trình khẩn cấp khi cần quyền tạm thời, phải có bản ghi audit và thời gian hiệu lực ngắn.

### 10. Assignment tạm thời, timezone/thời điểm hiệu lực, nhắc hết hạn, người chịu trách nhiệm gia hạn
- `iam_role_assignments` đã có `valid_from`, `valid_until`, `assignment_status`.
- Cần bổ sung: thông báo nhắc hết hạn (ví dụ 7 ngày trước), người chịu trách nhiệm gia hạn (owner), quy trình gia hạn hoặc tự động `EXPIRED`.

---

## 3. Thay đổi cơ sở dữ liệu (Migration)

### 3.1 Soft delete Role & unique role_code

**Quy tắc:**
- Role đã xóa mềm (`is_enabled = FALSE`) **không được phép reactivate** (bật lại) trong bất kỳ trường hợp nào.
- Do đó, có thể **để nguyên `role_code` cũ** trên hàng đã xóa mềm, không cần đổi mã.
- Thêm partial unique index để đảm bảo `role_code` không trùng giữa các role **đang bật** (`is_enabled = TRUE`):
```sql
-- Thay thế UNIQUE(role_code) toàn bảng bằng partial unique index
CREATE UNIQUE INDEX uq_iam_roles_role_code_enabled
ON iam_roles (role_code) WHERE is_enabled = TRUE;
```
- Chỉ được xóa mềm role khi **không còn user nào đang được gán active** (không có `iam_role_assignments` với `assignment_status = 'ACTIVE'` và `valid_until > now` tham chiếu đến role này).
- Các assignment cũ sau khi soft delete phải được chuyển sang `REVOKED`, không được để trỏ nhầm sang role mới (vì `role_id` khác, dù `role_code` có thể giống nhau nếu role mới được tạo sau).

**Lý do:** Vì role đã xóa không bao giờ được reactivate, sẽ không bao giờ có 2 dòng `is_enabled = TRUE` cùng `role_code`. JWT/Keycloak attribute cũ của user không ảnh hưởng vì khi thu hồi role của user, assignment sẽ được xóa/trả về trước, đồng bộ lại Keycloak.

### 3.2 Centralize truy vấn đọc Role
Tất cả repository/service đọc `iam_roles` phải thêm điều kiện `AND is_enabled = TRUE` (hoặc `deleted_at IS NULL` nếu thêm cột đó).

---

## 4. API mới cần bổ sung (Identity Service)

| Endpoint | Phương Gesetz | Mô tả | Quyền |
|----------|-------------|-------|-------|
| `/api/v1/admin/accounts` | GET | Liệt kê tài khoản, lọc theo trạng thái | `ROLE_ADMIN` |
| `/api/v1/admin/accounts/{id}/status` | PATCH | Khóa tạm/vĩnh viễn, mở khóa | `ROLE_ADMIN` |
| `/api/v1/admin/accounts/{id}/reactivate` | POST | Khôi phục tài khoản đã khóa | `ROLE_ADMIN` |
| `/api/v1/admin/roles` | GET | Liệt kê role (bao gồm `is_enabled`) | `ROLE_ADMIN` |
| `/api/v1/admin/roles/{id}` | DELETE | Soft delete role (chỉ khi không còn user tham chiếu; chặn nếu là admin cuối cùng) | `ROLE_ADMIN` |
| `/api/v1/admin/role-assignments` | GET | Xem assignment, lọc theo principal/role/scope | `ROLE_ADMIN` |
| `/api/v1/admin/role-assignments/{id}` | DELETE | Thu hồi assignment (REVOKED) | `ROLE_ADMIN` |

---

## 5. Thay đổi Keycloak sync

- Khi `iam_role_assignments` thay đổi (gán/thu hồi/hết hạn), gọi `syncPermissionsToKeycloak(principalId)` để cập nhật Keycloak User Attributes.
- Khi đổi permission của role, phải sync lại tất cả principal đang được gán role đó.
- **Không xóa realm role ngay khi soft delete trong DB.**
- Khi thực sự xóa vĩnh viễn role (chưa được sử dụng), đồng bộ xóa trên Keycloak sau khi không còn assignment nào tham chiếu.

---

## 6. Audit Log

Tất cả thao tác khóa/mở khóa, gán/thu hồi role, reactivate, thay đổi role/permission/assignment phải ghi vào `identity_audit_log` (bảng đã có từ migration `V4__create_identity_audit_log.sql`) với trường:
- `actor_principal_id`
- `target_principal_id`
- `action` (ví dụ: `ACCOUNT_SUSPEND`, `ACCOUNT_REACTIVATE`, `ROLE_ASSIGN`, `ROLE_REVOKE`, `ROLE_DISABLE`, `PERMISSION_CHANGE`)
- `reason`
- `timestamp`

---

## 7. Quy tắc cho AI (Identity Service)

Xem file `services/identity-service/IDENTITY_SERVICE_AGENT_RULES.md`.

---

## 8. Lộ trình triển khai (gợi ý)

| Tuần | Công việc |
|------|-----------|
| 1 | Thiết kế & chốt ma trận phân quyền (`iam-role-permission-matrix.md`) |
| 2 | Chuẩn hóa JWT & đồng bộ quyền (syncPermissionsToKeycloak khi thay đổi role/permission/assignment) |
| 3 | Thêm migration soft delete role + partial unique index |
| 4 | Implement API khóa/mở khóa tài khoản + reactivate |
| 5 | Implement API CRUD role & role-assignment (soft delete, chỉ xóa khi không còn user tham chiếu) |
| 6 | Bổ sung sync Keycloak khi thu hồi role và thay đổi permission |
| 7 | Viết unit test & integration test cho tất cả API mới |
| 8 | Review code & merge vào `dev` |

---

**Tài liệu liên quan:**
- `docs/03_api_specs/iam-role-permission-matrix.md`
- `services/identity-service/IDENTITY_SERVICE_AGENT_RULES.md`
- `services/catalog-service/CATALOG_SERVICE_AGENT_RULES.md` (mẫu tham khảo)