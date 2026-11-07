# Ma trận phân quyền IAM — Identity Service

**Mục đích:** Định nghĩa rõ ràng danh sách role, permission, scope và quy tắc gán thu hồi trong hệ thống.

**File liên quan:** `services/identity-service/src/main/resources/db/migration/V2__create_iam_role_permission_model.sql`

---

## 1. Danh sách Role hệ thống

| Role Code | Tên hiển thị | Loại | Mô tả | Có được gán cho EMPLOYEE | Có được gán cho CUSTOMER |
|-----------|-------------|------|-------|--------------------------|--------------------------|
| `ADMIN` | Quản trị viên hệ thống | Hệ thống | Super-admin, toàn quyền mọi endpoint admin | Có (chỉ Admin khác) | Không |
| `USER_MANAGER` | Quản lý người dùng | Nghiệp vụ | Quản lý tài khoản, khóa/mở khóa, reactivate | Có | Không |
| `ROLE_MANAGER` | Quản lý vai trò | Nghiệp vụ | Quản lý catalog role, gán thu hồi role | Có | Không |
| `AUDITOR` | Kiểm toán viên | Nghiệp vụ | Xem audit log, export báo cáo | Có | Không |
| `EMPLOYEE` | Nhân viên | Cơ sở | Nhân viên cơ bản, không có quyền quản trị | Có (mặc định) | Không |
| `CUSTOMER` | Khách hàng | Cơ sở | Khách hàng mua sắm, xem profile cá nhân | Không | Có (mặc định) |

> **Quy tắc:** `principal_type = EMPLOYEE` mới được gán các role hành chính (`ADMIN`, `USER_MANAGER`, `ROLE_MANAGER`, `AUDITOR`). `CUSTOMER` chỉ giữ role `CUSTOMER`.

---

## 2. Danh sách Permission

| Permission Code | Resource | Action | Risk Level | Scope Required | Mô tả |
|-----------------|----------|--------|------------|----------------|-------|
| `USER_VIEW` | USER | VIEW | LOW | FALSE | Xem danh sách và chi tiết tài khoản |
| `USER_MANAGE` | USER | MANAGE | HIGH | TRUE | Khóa/mở khóa, reactivate tài khoản |
| `ROLE_VIEW` | ROLE | VIEW | LOW | FALSE | Xem danh sách vai trò |
| `ROLE_MANAGE` | ROLE | MANAGE | HIGH | TRUE | Tạo/sửa/xóa mềm role, gán thu hồi role |
| `ACCOUNT_READ` | ACCOUNT | READ | LOW | FALSE | Đọc profile tài khoản của mình |
| `ACCOUNT_WRITE` | ACCOUNT | WRITE | MEDIUM | FALSE | Cập nhật profile tài khoản của mình |

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

1. **Chỉ EMPLOYEE được gán role hành chính.** CUSTOMER không được gán `ADMIN`, `USER_MANAGER`, ...
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