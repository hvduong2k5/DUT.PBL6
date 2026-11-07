# Permission → API → phạm vi: bản gửi Dương tổng hợp

Đối chiếu implementation của bản tích hợp MVP ngày 04/11/2026. Bảng phân biệt **đã kiểm ở backend** với **cần owner thiết kế thêm**. Không thêm claim `principal_type` vào JWT: dùng API Identity đã có `GET /api/v1/users/me`, đối chiếu issuer/sub và `user_id` với principal trong DB. Employee ACTIVE mới được vào nhóm API admin; Customer không được vào dù mang role ADMIN cũ.

Role là nhóm quyền của Employee; permission là hành động cụ thể. Phạm vi là tài nguyên được phép tác động (bản thân, toàn hệ thống, kho/cửa hàng cụ thể). Có permission chưa có nghĩa là đã kiểm đúng phạm vi kho.

## Identity: quyền owner đã có trong controller

Các dòng sau giữ nguyên điều kiện owner `hasRole('ADMIN') OR hasAuthority(permission)`, thêm cổng Employee ACTIVE đã xác minh trước đó. Actor audit lấy `user_id` từ JWT đã kiểm; `X-Actor-Principal-Id` không được dùng làm danh tính người thao tác.

| Permission | API | Phạm vi backend hiện tại |
|---|---|---|
| `USER_VIEW` | `GET /api/v1/admin/users`, `GET /api/v1/admin/users/{principalId}` | Toàn hệ thống |
| `USER_STATUS_MANAGE` | `PATCH /api/v1/admin/users/{principalId}/status` | Tài khoản đích; giữ validation nghiệp vụ của Identity |
| `USER_PASSWORD_RESET` | `POST /api/v1/admin/users/{principalId}/password/reset` | Tài khoản đích |
| `USER_SESSION_REVOKE` | `POST /api/v1/admin/users/{principalId}/sessions/revoke` | Tài khoản đích |
| `EMPLOYEE_INVITE` | `POST /api/v1/admin/employee-invitations` | Lời mời nhân viên |
| `EMPLOYEE_INVITATION_REVOKE` | `DELETE /api/v1/admin/employee-invitations/{invitationId}` | Lời mời đích |
| `ROLE_VIEW` | `GET /api/v1/admin/roles`, `GET /api/v1/admin/roles/{roleId}` | Danh mục vai trò |
| `ROLE_MANAGE` | `POST /api/v1/admin/roles`, `PATCH /api/v1/admin/roles/{roleId}`, `DELETE /api/v1/admin/roles/{roleId}` | Vai trò đích |
| `ROLE_ASSIGN` | `POST /api/v1/admin/role-assignments`, `DELETE /api/v1/admin/role-assignments/{assignmentId}` | Gán/thu hồi role; payload có GLOBAL/WAREHOUSE/STORE, chưa chứng minh enforcement scope đó ở mọi service |
| `PERMISSION_VIEW` | `GET /api/v1/admin/roles/{roleId}/permissions` | Quyền của vai trò đích |
| `PERMISSION_MANAGE` | `PUT/POST /api/v1/admin/roles/{roleId}/permissions`, `DELETE .../permissions/{permissionCode}` | Danh sách quyền của role đích |
| `AUDIT_LOG_VIEW`, `AUDIT_LOG_EXPORT` | Chưa tìm thấy API đọc/export tương ứng trong controller | Không coi mã seed là endpoint đã được bảo vệ và triển khai |

`GET /api/v1/users/me`, `PUT /api/v1/users/me/password` là truy cập chính mình theo principal đã xác minh, không cần quyền admin. Test tích hợp tập trung đăng nhập và ranh giới persona; chưa nghiệm thu toàn bộ thao tác IAM quản trị qua giao diện.

## Các service trong luồng mua

| Service | Quyền/credential thực sự dùng | API hoặc gRPC | Phạm vi hiện được kiểm |
|---|---|---|---|
| Profile | Customer ACTIVE từ Identity + JWT; Profile customer có trạng thái nghiệp vụ phù hợp | `/api/v1/profile`, `/api/v1/profile/addresses/**` | Chỉ hồ sơ/địa chỉ của principal; CAS version khi sửa/xóa; Employee bị chặn |
| Order customer | Customer ACTIVE từ Identity + JWT + ánh xạ Profile thực | `/api/v1/cart/**`, `/api/v1/checkout/**`, `/api/v1/checkout-operations/**`, `/api/v1/orders/**` của customer | Giỏ, báo giá, operation, đơn, payment read của chính Customer; staff có API đọc đơn riêng với ORDER_VIEW và scope kho |
| Identity BFF | Cookie phiên, loại principal từ DB, Origin với browser mutation | `/api/v1/customer-api/**` allowlist | Chỉ chuyển các route Customer đã định nghĩa; không chuyển admin/internal hoặc header danh tính tự khai |
| Catalog public | Anonymous read | REST `/api/v1/categories`, `/api/v1/products`, `/api/v1/products/search`, `/api/v1/products/{id_or_slug}` | Chỉ sản phẩm ACTIVE + APPROVED, giá còn hiệu lực; chưa có quyền nhân viên ở public |
| Catalog admin | Employee ACTIVE và role `ADMIN` do `/users/me` trả từ DB | REST `/api/v1/admin/catalog/**` (JWT + permission `CATALOG_VIEW`/`CATALOG_MANAGE`) | **GLOBAL MVP**; actor lấy từ Identity, không tin `request.actor_id`; chưa kiểm phạm vi kho/cửa hàng |
| Catalog internal | `X-Internal-Token` server, tối thiểu 32 ký tự | Snapshot/revalidate HTTP; `omamx.catalog.v1.CatalogService/GetCheckoutSnapshot`; legacy ValidatePriceAndSku | Caller service có credential; không phải quyền người dùng |
| Inventory integration | Bearer nội bộ qua TLS, CA/server name đã cấu hình | GetStockLevel; terminal reserve/release/commit | Đọc/giữ/nhả/commit theo protocol nội bộ; không suy ra nhân viên nào được chỉnh tồn kho |
| Shipping integration | Credential service riêng; adapter kiểm revision/event | Shipping v2 quote/create/get revision/history | Order gọi provider nội bộ; hãng FAKE trong fixture, không có quyền người dùng từ UI |
| Gateway | Route tới đúng service; không biến header do client gửi thành danh tính | Các route public của Kong | Backend vẫn tự kiểm xác thực và quyền; rate limit không thay authorization |
| Web/Mobile | Guard persona để chọn màn hình và repository | Auth, cart, checkout, order screens | Hỗ trợ UX; quyết định cuối vẫn ở backend |

## Những phần owner cần gửi thêm

| Service/owner | Việc cần chốt trước khi mở nghiệp vụ nội bộ đầy đủ |
|---|---|
| Catalog | Tách quyền đọc/sửa giá/sửa sản phẩm/duyệt; xác định scope; thay guard ADMIN toàn cục bằng map quyền được Dương thống nhất |
| Inventory | Map API đọc tồn, điều chỉnh, nhập/xuất, kiểm kê với quyền và warehouse scope; không dùng internal token làm quyền nhân viên |
| Shipping | Map API cấu hình hãng, điều phối, cập nhật vận đơn với quyền và scope; tách webhook credential khỏi quyền nhân viên |
| Fulfillment | Quyền nhận việc, đóng gói, bàn giao; scope kho và danh tính actor; chưa lấy mã chưa commit của workspace khác để coi là hoàn thành |
| Finance/Payment | Quyền đối soát/hoàn tiền/duyệt và bằng chứng settlement COD; giữ các bước phê duyệt đang có, không tự bỏ kiểm tra |
| Care, các service ngoài luồng đã test | Owner cung cấp endpoint thực, permission và object scope; chưa mở truy cập bằng role chung chỉ để demo |

Tên quyền chi tiết của các service này cần owner xác nhận trước khi seed vào Identity. MVP mua hàng không cần đợi toàn bộ IAM nội bộ hoàn chỉnh, nhưng không được quảng bá là đã phân quyền đủ mọi chức năng nhân viên.

## Profile và Order — quyền nhân viên theo phạm vi

Profile kiểm `EMPLOYEE_PROFILE_VIEW`, `EMPLOYEE_PROFILE_CREATE`, `EMPLOYEE_PROFILE_SENSITIVE_VIEW`, `EMPLOYEE_COMPLIANCE_VIEW` theo GLOBAL/DEPARTMENT. Order kiểm `ORDER_VIEW`, `ORDER_SENSITIVE_VIEW`, `ORDER_CANCEL_REQUEST`, `ORDER_HOLD_MANAGE`, `ORDER_RECOVERY_EXECUTE` theo GLOBAL/WAREHOUSE. Nguồn quyền là `permissionGrants` từ Identity, giữ từng quyền gắn với assignment scope của chính nó.

[Bảng API, phạm vi, giới hạn nghiệp vụ và cách cấp quyền](profile-order-api-authorization.md). Không suy diễn rằng có permission thì hủy/hold/payment đã mở mọi khả năng. Customer và credential service nội bộ giữ ranh giới hiện có.
