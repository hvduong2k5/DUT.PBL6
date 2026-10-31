# Admin Access Data/API Matrix

**Cập nhật:** 2026-10-02  
**Backlog:** `EPIC_22_Administration.md`  
**Màn hình:** Employee System Access, Access Review, Role & Permission Catalog  
**Ứng dụng:** `apps/web-admin`  
**Mock Admin:** `mocks/mockoon/admin_extensions.json`, cổng `4030`

**Decision register:** `docs/ui_web/ACCESS_MANAGEMENT_POLICY_DECISIONS.md`. Mutation đang tạm gác cho đến khi các mục `Chưa chốt` liên quan được duyệt.

**Nguồn phạm vi:** `US-ADM-01~07`, `FR-ADM-01~25`. Lát cắt hiện tại chỉ đọc cấu hình và quyền hiệu lực; chưa triển khai mutation khi các chính sách mở ở Mục 8 có thể làm thay đổi hành vi.

## Phạm vi đã triển khai

| Khả năng | UI/BFF | Mock upstream | Trạng thái |
| --- | --- | --- | --- |
| Danh sách Employee System Access | `GET /api/admin/access/employees` | `GET /admin/access/employees` | Đã tích hợp |
| Status tabs và tìm kiếm | Xử lý trên projection đã tải | cùng list route | Đã tích hợp |
| Lọc theo phòng ban/Role | Xử lý trên projection đã tải | cùng list route | Đã tích hợp |
| Lọc theo Permission hiệu lực | Chỉ hiển thị khi có `ACCESS_REVIEW_VIEW` | cùng list route | Đã tích hợp |
| Access Review theo Employee | `GET /api/admin/access/employees/:employeeId` | cùng path không có `/api` | Đã tích hợp trong popup |
| Giải thích Permission trùng nguồn | Một Permission hiển thị một lần, giữ đầy đủ `sources` | detail route | Đã tích hợp |
| Phân biệt cấu hình quyền và khả năng truy cập | `accessState` tách khỏi Role Assignment | list/detail | Đã tích hợp |
| Danh mục và bộ lọc Role | `GET /api/admin/access/roles` | `GET /admin/access/roles` | Đã tích hợp read-only |
| Chi tiết Role, Permission và impact | `GET /api/admin/access/roles/:roleCode` | cùng path không có `/api` | Đã tích hợp trong popup |
| Danh mục và bộ lọc Permission | `GET /api/admin/access/permissions` | `GET /admin/access/permissions` | Đã tích hợp read-only |
| Chi tiết Permission và nơi sử dụng | `GET /api/admin/access/permissions/:permissionCode` | cùng path không có `/api` | Đã tích hợp trong popup |

## Permission và ranh giới dữ liệu

- `EMPLOYEE_ACCOUNT_VIEW` cho phép xem danh sách tài khoản nội bộ trong scope.
- `ACCESS_REVIEW_VIEW` mới cho phép nhận danh mục Permission, lọc theo Permission và mở popup quyền hiệu lực.
- BFF trả `403` cho detail Access Review nếu thiếu quyền; list không trả `effectivePermissionCodes` và permission catalog khi thiếu `ACCESS_REVIEW_VIEW`.
- Response chỉ chứa trường System Access cần thiết và tham chiếu Employee Profile. Họ tên/phòng ban/chức vụ không được chỉnh sửa tại module này.
- Không có password, credential secret, token kích hoạt hoặc dữ liệu xác thực đọc được trong projection.
- Quyền hiệu lực chỉ đi qua Role Assignment; chưa có Direct Permission, Role inheritance, explicit Deny hoặc suy rộng từ quyền xem sang mutation.
- Danh mục Role yêu cầu `ROLE_VIEW`; danh mục Permission yêu cầu `PERMISSION_VIEW`. Có quyền xem một danh mục không đồng nghĩa có quyền quản lý danh mục đó.
- Detail mock dùng projection tĩnh để kiểm tra bố cục và permission gate; backend thật phải tính `impact`, `rolesUsing` và số Employee theo dữ liệu hiện hành.

## Trạng thái thể hiện

- Account: `INVITED`, `PENDING_ACTIVATION`, `ACTIVE`, `LOCKED`, `DISABLED`.
- Assignment: `SCHEDULED`, `ACTIVE`, `EXPIRED`, `REVOKED`.
- Khả năng truy cập: `ENABLED`, `BLOCKED_ACCOUNT`, `NO_ACTIVE_ROLE`.
- Account `LOCKED` vẫn có thể giữ Role Assignment để bảo toàn lịch sử nhưng không được trình bày là đang truy cập được.
- Mỗi Access Review ghi `calculatedAt`; đây là cấu hình hiện tại, không thay thế Audit Log của EPIC 23.

## Chưa triển khai vì cần chốt chính sách

1. Tạo/cập nhật Account và quy trình invite/kích hoạt/MFA/credential.
2. Khóa theo lịch, mở khóa/reactivate, thu hồi session và bảo vệ Admin cuối cùng.
3. Tạo/sửa/archive Role, Permission tùy chỉnh và migration mã Permission; danh mục và detail read-only đã có.
4. Gán/thu hồi Role có thời hạn, approval đặc quyền, self-assignment và phân tách nhiệm vụ.
5. Direct Permission, explicit Deny, Role inheritance và quy tắc hợp nhất scope.
6. Chu kỳ review, xác nhận review, export và Audit explorer.

## Chạy local

```powershell
# apps/web-admin/.env.local
ADMIN_API_UPSTREAM_URL=http://127.0.0.1:4030/api/v1
ADMIN_MOCK_ROLE=SYSTEM_ADMIN

npx @mockoon/cli start --disable-log-to-file --data .\mocks\mockoon\admin_extensions.json
cd apps\web-admin
npm run dev
```

Các màn hình nằm tại `http://localhost:3001/administration/employees`, `/administration/roles` và `/administration/permissions`.
