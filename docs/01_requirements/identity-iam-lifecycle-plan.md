# Kế hoạch bổ sung / cập nhật / loại bỏ — Identity Service IAM

**Mục tiêu:** Hoàn thiện mô hình phân quyền, vòng đời tài khoản và role trong `identity-service` để đáp ứng yêu cầu bảo mật, kiểm soát và kiểm tra (audit) cho hệ thống OCOP.

**Phạm vi:** `services/identity-service` (Java Spring Boot + MyBatis + Keycloak), bảng `iam_roles`, `iam_permissions`, `iam_role_permissions`, `iam_role_assignments`, `identity_principals`.

---

## 1. Trạng thái hiện tại (Gap analysis)

| Điểm | Trạng thái hiện tại | Vấn đề |
|------|-------------------|--------|
| 4. Bộ trạng thái Account | `provisioningStatus` (`ACTIVE`, `PENDING_VERIFICATION`) trong DB và `customer_status`. Chưa có endpoint mở khóa/reactivate rõ ràng. | Cần thêm API admin để CRUD tài khoản, khóa tạm/vĩnh viễn, mở khóa/reactivate. |
| 5. Thời gian thu hồi Role | JWT exp do Keycloak cấp; refresh token maxAge 28.800s (8 giờ) trong `oidc.ts`. Không có cấu hình propagate thu hồi role lên session/cache. | Chốt thời gian tối đa (5–15 phút) và xử lý phiên đang thực hiện giao dịch quan trọng. |
| 8. Danh sách Role | Chỉ thấy `ADMIN` (`ROLE_ADMIN` từ Keycloak realm role). `CUSTOMER` và `EMPLOYEE` là `principal_type` (loại tài khoản), không phải role phân quyền. | Chưa có quy trình sửa/clone/archive role và cơ chế bảo vệ Admin hợp lệ cuối cùng. |
| 9. Ma trận thẩm quyền | Không thấy mã nguồn triển khai ma trận chi tiết, self-assignment, break-glass. | Cần bổ sung tài liệu thiết kế (role_permission_matrix.md). |
| 10. Assignment tạm thời | Không thấy mã nguồn. | Cần thiết kế thời điểm hiệu lực, timezone, nhắc hết hạn, người chịu trách nhiệm gia hạn. |
| 11. Access Review | Không thấy mã nguồn/tài liệu. | Cần thiết kế chu kỳ review, owner phê duyệt/thu hồi, export, xử lý quyền thừa/orphaned. |
| Soft delete Role | `iam_roles` có `is_enabled`, `is_protected`. `UNIQUE(role_code)` chặn tạo lại mã trùng sau xóa mềm. | Cần partial unique index, centralized `WHERE is_enabled = TRUE`, không xóa realm role ngay khi soft delete. |

---

## 2. Các mục cần chốt

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

### 11. Chu kỳ Access Review, owner phê duyệt/thu hồi, bằng chứng xác nhận, export, quyền thừa/orphaned Account
- **Chu kỳ Review:** Theo định kỳ (ví dụ quý một lần), owner phê duyệt lại quyền của mỗi nhân viên.
- **Bằng chứng:** Ghi log ai phê duyệt, khi nào, lý do.
- **Export:** Xuất danh sách tài khoản/quyen ra CSV/JSON cho auditor.
- **Quyền thừa/orphaned:** Phát hiện tài khoản có role nhưng không còn cần thiết, hoặc account không có role nào (orphaned), tiến hành thu hồi.

---

## 3. Thay đổi cơ sở dữ liệu (Migration)

### 3.1 Soft delete Role & UNIQUE role_code
```sql
-- Partial unique index: chỉ đảm bảo unique role_code cho role đang kích hoạt
CREATE UNIQUE INDEX uq_iam_roles_role_code_enabled
ON iam_roles (role_code) WHERE is_enabled = TRUE;
```

### 3.2 Centralize truy vấn đọc Role
Tất cả repository/service đọc `iam_roles` phải thêm điều kiện `AND is_enabled = TRUE` (hoặc `deleted_at IS NULL` nếu thêm cột đó).

### 3.3 Bảng hỗ trợ Access Review (nếu cần)
```sql
CREATE TABLE IF NOT EXISTS iam_access_review (
    review_id UUID PRIMARY KEY,
    reviewer_principal_id UUID NOT NULL,
    reviewed_principal_id UUID NOT NULL,
    review_status VARCHAR(32) NOT NULL CHECK (review_status IN ('PENDING', 'APPROVED', 'REVOKED')),
    review_date TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    next_review_date TIMESTAMPTZ,
    evidence TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

---

## 4. API mới cần bổ sung (Identity Service)

| Endpoint | Phương Gesetz | Mô tả | Quyền |
|----------|-------------|-------|-------|
| `/api/v1/admin/accounts` | GET | Liệt kê tài khoản, lọc theo trạng thái | `ROLE_ADMIN` hoặc `USER_VIEW` |
| `/api/v1/admin/accounts/{id}/status` | PATCH | Khóa tạm/vĩnh viễn, mở khóa | `ROLE_ADMIN` hoặc `USER_MANAGE` |
| `/api/v1/admin/accounts/{id}/reactivate` | POST | Khôi phục tài khoản đã khóa | `ROLE_ADMIN` hoặc `USER_MANAGE` |
| `/api/v1/admin/roles` | GET | Liệt kê role (bao gồm `is_enabled`) | `ROLE_ADMIN` hoặc `ROLE_VIEW` |
| `/api/v1/admin/roles/{id}` | DELETE | Soft delete role (chặn nếu là admin cuối cùng) | `ROLE_ADMIN` hoặc `ROLE_MANAGE` |
| `/api/v1/admin/role-assignments` | GET | Xem assignment, lọc theo principal/role/scope | `ROLE_ADMIN` hoặc `ROLE_VIEW` |
| `/api/v1/admin/role-assignments/{id}` | DELETE | Thu hồi assignment (REVOKED) | `ROLE_ADMIN` hoặc `ROLE_MANAGE` |

---

## 5. Thay đổi Keycloak sync

- Khi `iam_role_assignments` thay đổi (gán/thu hồi/hết hạn), gọi `syncPermissionsToKeycloak(principalId)` để cập nhật Keycloak User Attributes.
- **Không xóa realm role ngay khi soft delete trong DB** để tránh làm hỏng JWT/attribute cũ của user đang lưu permissions của role đó.
- Khi thực sự xóa vĩnh viễn role (nếu cần), đồng bộ xóa trên Keycloak sau khi không còn assignment nào tham chiếu.

---

## 6. Audit Log

Tất cả thao tác khóa/mở khóa, gán/thu hồi role, reactivate, access review phải ghi vào `identity_audit_log` (bảng đã có từ migration `V4__create_identity_audit_log.sql`) với trường:
- `actor_principal_id`
- `target_principal_id`
- `action` (ví dụ: `ACCOUNT_SUSPEND`, `ACCOUNT_REACTIVATE`, `ROLE_ASSIGN`, `ROLE_REVOKE`, `ACCESS_REVIEW`)
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
| 2 | Thêm migration soft delete role + partial unique index |
| 3 | Implement API khóa/mở khóa tài khoản + reactivate |
| 4 | Implement API CRUD role & role-assignment (soft delete) |
| 5 | Bổ sung sync Keycloak khi thu hồi role |
| 6 | Thiết kế & implement Access Review (nếu yêu cầu) |
| 7 | Viết unit test & integration test cho tất cả API mới |
| 8 | Review code & merge vào `dev` |

---

**Tài liệu liên quan:**
- `docs/03_api_specs/iam-role-permission-matrix.md`
- `services/identity-service/IDENTITY_SERVICE_AGENT_RULES.md`
- `services/catalog-service/CATALOG_SERVICE_AGENT_RULES.md` (mẫu tham khảo)