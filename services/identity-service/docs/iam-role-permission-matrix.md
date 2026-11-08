# Ma trận phân quyền IAM — Identity Service

**Mục đích:** Định nghĩa rõ ràng danh sách role, permission, scope và quy tắc gán thu hồi trong hệ thống.

**File liên quan:** `services/identity-service/src/main/resources/db/migration/V2__create_iam_role_permission_model.sql`

---

## 1. Danh sách Role hệ thống

| Role Code | Tên hiển thị | Loại | Mô tả | Có được gán cho EMPLOYEE | Có được gán cho CUSTOMER |
|-----------|-------------|------|-------|--------------------------|--------------------------|
| `ADMIN` | Quản trị viên hệ thống | Hệ thống | Super-admin, toàn quyền mọi endpoint admin | Có (chỉ Admin khác) | Không |

> **Quy tắc:** Hiện tại hệ thống chỉ có 1 role phân quyền là `ADMIN` (từ Keycloak realm role → `ROLE_ADMIN` trong Spring Security).
> `CUSTOMER` và `EMPLOYEE` là `principal_type` (loại tài khoản), không phải role phân quyền. Chúng không được liệt kê trong bảng role này.
>
> Các role mở rộng (`USER_MANAGER`, `ROLE_MANAGER`, `AUDITOR`, …) cần được chốt thêm trong thiết kế kỹ thuật nếu hệ thống yêu cầu.

---

## 2. Danh sách Permission

| Permission Code | Resource | Action | Risk Level | Scope Required | Mô tả |
|-----------------|----------|--------|------------|----------------|-------|
| `CATALOG_VIEW` | CATALOG | VIEW | LOW | FALSE | Xem danh mục và chi tiết sản phẩm trong hệ thống quản trị |
| `CATALOG_MANAGE` | CATALOG | MANAGE | HIGH | TRUE | Tạo/cập nhật/xóa sản phẩm, biến thể, bảng giá, ảnh |
| `CATALOG_APPROVE` | CATALOG | APPROVE | HIGH | TRUE | Phê duyệt/từ chối sản phẩm (chuyển trạng thái vòng đời) |

> **Quy tắc:** Danh sách permission trên được seed trong migration `V7__seed_identity_roles_and_permissions.sql` và `V8__seed_catalog_permissions.sql` của `identity-service`.
>
> Các permission mở rộng (`USER_VIEW`, `USER_MANAGE`, `ROLE_VIEW`, `ROLE_MANAGE`, …) cần được chốt thêm trong thiết kế kỹ thuật nếu hệ thống yêu cầu quản lý tài khoản/vai trò qua API admin của identity-service.

---

## 3. Phạm vi gán quyền (Scope)

| Scope Type | Ý nghĩa | Ví dụ scope_id |
|------------|---------|----------------|
| `GLOBAL` | Toàn hệ thống | `NULL` |
| `WAREHOUSE` | Kho hàng | Mã kho (ví dụ `WH-001`) |
| `STORE` | Cửa hàng | Mã cửa hàng |
| `REGION` | Vùng miền | Mã vùng |
| `DEPARTMENT` | Phòng ban | Mã phòng ban |
| `BRANCH` | Chi nhánh | Mã chi nhánh |
| `BUSINESS_UNIT` | Đơn vị kinh tế | Mã đơn vị |

> **Quy tắc:** Nếu `scope_type <> 'GLOBAL'`, `scope_id` bắt buộc không rỗng. Nếu `scope_type = 'GLOBAL'`, `scope_id` phải `NULL`.

---

## 4. Quy tắc gán / thu hồi Role

1. **Chỉ EMPLOYEE (`principal_type = EMPLOYEE`) được gán role hành chính.** CUSTOMER không được gán role `ADMIN` hoặc bất kỳ role quản trị nào.
2. **Không self-assignment:** Người dùng không được tự gán role cho chính mình.
3. **Phân tách nhiệm vụ (SoD):** Người tạo yêu cầu gán role không được đồng thời là người phê duyệt cùng yêu cầu đó.
4. **Role được gán kèm thời hạn:** `valid_from`, `valid_until`. Khi hết hạn, tự động chuyển thành `EXPIRED` khi truy vấn.
5. **Thu hồi trước hạn:** Chuyển `assignment_status = 'REVOKED'`, ghi `revoked_by_principal_id`, `reason`.
6. **Bảo vệ Admin cuối cùng:** Không cho phép xóa mềm hoặc thu hồi toàn quyền của Admin cuối cùng còn active trong hệ thống.

---

## 5. Break-glass (Khẩn cấp)

- Khi cần cấp quyền tạm thời ngay lập tức cho xử lý sự cố:
  - Tạo assignment với `valid_until` ngắn (ví dụ 24 giờ).
  - Ghi lý do khẩn cấp trong `reason`.
  - Phải có audit log đầy đủ.
  - Sau khi hết thời gian, quyền tự động hết hiệu lực.

---

## 6. Quy tắc chung

- **Role name là assignment bundle:** Không dùng tên role làm điểm kiểm duy nhất cho phân quyền. Phải kiểm tra `permission_code` thông qua `iam_role_permissions`.
- **Admin authorization:** Dựa trên `permission-and-scope`, không chỉ dựa trên `role_name`.
- **Idempotency-Key:** Bắt buộc cho các thao tác create/action có thể thử lại.
- **Empty collections:** Trả về 200 với mảng rỗng và tổng bằng 0.
- **401:** Xác thực bị bỏ lỗi hoặc không hợp lệ.
- **403:** Danh tính đã biết nhưng quyền hạn/khả năng/sở hữu/phạm vi không đủ.
- **`principal_type` không phải phân quyền:** `CUSTOMER` và `EMPLOYEE` chỉ phân loại loại tài khoản, không dùng để kiểm tra quyền truy cập endpoint.

---

## 7. Quy tắc JWT claims (chốt trước PR #71 / #73)

- **Không thêm claim `principal_type` vào JWT.** Loại tài khoản (CUSTOMER/EMPLOYEE) lấy từ DB qua `GET /api/v1/users/me`, đối chiếu `user_id` (Identity Principal UUID) từ JWT. JWT là immutable, không baking metadata persona có thể thay đổi.
- `permission_grants` là nguồn quyền staff có thẩm quyền theo 2-A. Mỗi phần tử trói `permission` với `scope_type` + `scope_id` của chính assignment sinh ra nó; **không ghép chéo** giữa các phần tử.
  - `scope_type = GLOBAL` ⇒ `scope_id` phải `null`.
  - `scope_type = WAREHOUSE/STORE/REGION/DEPARTMENT/BRANCH/BUSINESS_UNIT` ⇒ `scope_id` bắt buộc có giá trị.
- `permissions` (flat list) chỉ giữ cho consumer cũ (Catalog, Identity tự kiểm) trong giai đoạn chuyển tiếp. Consumer mới **không được** dùng danh sách phẳng để kiểm phạm vi; phải dùng `permission_grants`.
- `realm_access.roles` chỉ dùng để sinh authority `ROLE_*` (ví dụ `ROLE_ADMIN`), không làm bypass kiểm tra permission chi tiết.
- `amr` / `auth_time` phục vụ kiểm tra MFA gần đây cho thao tác rủi cao (6-B) — consumer nào cần thì kiểm, không phải mọi request.