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
| `ROLE_VIEW` | `GET /api/v1/admin/roles`, `GET /api/v1/admin/roles/{roleId}`, `GET /api/v1/admin/role-assignments`, `GET /api/v1/admin/iam/principals/{principalId}/effective-permissions` | Danh mục vai trò và quyền đang cấu hình của principal |
| `ROLE_MANAGE` | `POST /api/v1/admin/roles`, `PATCH /api/v1/admin/roles/{roleId}`, `DELETE /api/v1/admin/roles/{roleId}` | Vai trò đích |
| `ROLE_ASSIGN` | `POST /api/v1/admin/role-assignments`, `DELETE /api/v1/admin/role-assignments/{assignmentId}` | Gán/thu hồi role; payload có GLOBAL/WAREHOUSE/STORE, chưa chứng minh enforcement scope đó ở mọi service |
| `PERMISSION_VIEW` | `GET /api/v1/admin/roles/{roleId}/permissions` | Quyền của vai trò đích |
| `PERMISSION_MANAGE` | `PUT/POST /api/v1/admin/roles/{roleId}/permissions`, `DELETE .../permissions/{permissionCode}`, `POST /api/v1/admin/iam/principals/{principalId}/sync-permissions` | Danh sách quyền của role đích và chạy lại đồng bộ sang Keycloak |
| `AUDIT_LOG_VIEW`, `AUDIT_LOG_EXPORT` | Chưa tìm thấy API đọc/export tương ứng trong controller | Không coi mã seed là endpoint đã được bảo vệ và triển khai |

`GET /api/v1/users/me`, `PUT /api/v1/users/me/password` là truy cập chính mình theo principal đã xác minh, không cần quyền admin. Test tích hợp tập trung đăng nhập và ranh giới persona; chưa nghiệm thu toàn bộ thao tác IAM quản trị qua giao diện.

### Quy ước JWT claim (chốt trước PR #71 / #73)

- **Thêm claim `principal_type` vào JWT.** Giá trị: `CUSTOMER` hoặc `EMPLOYEE`. Nguồn: `identity_principals.principal_type`. Được ghi vào Keycloak user attribute `principal_type`, rồi protocol mapper đưa vào claim `principal_type` của access token.
- **Stale data tối đa 5 phút:** Access token chỉ sống 5 phút (`accessTokenLifespan = 300s`). Nếu `principal_type` thay đổi (rất hiếm), token cũ vẫn có giá trị đến khi hết hạn, sau đó phải refresh để nhận claim mới.
- `permission_grants` là nguồn quyền staff có thẩm quyền theo 2-A. Mỗi phần tử trói `permission` với `scope_type` + `scope_id` của chính assignment sinh ra nó; **không ghép chéo** giữa các phần tử.
  - `scope_type = GLOBAL` ⇒ `scope_id` phải `null`.
  - `scope_type = WAREHOUSE/STORE/REGION/DEPARTMENT/BRANCH/BUSINESS_UNIT` ⇒ `scope_id` bắt buộc có giá trị.
- `permissions` (flat list) chỉ giữ cho consumer cũ (Catalog, Identity tự kiểm) trong giai đoạn chuyển tiếp. Consumer mới **không được** dùng danh sách phẳng để kiểm phạm vi; phải dùng `permission_grants`.
- `amr` / `auth_time` phục vụ kiểm tra MFA gần đây cho thao tác rủi cao (6-B) — consumer nào cần thì kiểm, không phải mọi request.

## Các service trong luồng mua

| Service | Quyền/credential thực sự dùng | API hoặc gRPC | Phạm vi hiện được kiểm |
|---|---|---|---|
| Profile | Customer ACTIVE từ Identity + JWT; Profile customer có trạng thái nghiệp vụ phù hợp | `/api/v1/profile`, `/api/v1/profile/addresses/**` | Chỉ hồ sơ/địa chỉ của principal; CAS version khi sửa/xóa; Employee bị chặn |
| Order | Customer ACTIVE từ Identity + JWT + ánh xạ Profile thực | `/api/v1/cart/**`, `/api/v1/checkout/**`, `/api/v1/checkout-operations/**`, `/api/v1/orders/**` của customer | Giỏ, báo giá, operation, đơn, payment read của chính Customer; Employee bị chặn |
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
