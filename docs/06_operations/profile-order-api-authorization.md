# Phân quyền API Profile và Order

## Nguồn quyền và phạm vi

Profile và Order xác minh chữ ký RS256, issuer, audience, hạn token và `user_id`. Với cấu hình Keycloak, đặt `IDENTITY_URL` để service gọi `GET /api/v1/users/me` bằng chính bearer token. Identity phải trả principal ACTIVE có `principalId` và `idpSubject` khớp token. Không nhận quyền nhân viên từ header hoặc danh sách role/permission tự khai trong JWT.

Identity trả thêm `permissionGrants` bên cạnh `permissions` và `roles` cũ. Mỗi phần tử giữ quyền và phạm vi của **cùng một role assignment** đang hiệu lực và role đang enabled.

```json
{"permissionCode":"ORDER_VIEW","scopeType":"WAREHOUSE","scopeId":"WH-01"}
```

Không ghép danh sách `permissions` với mọi `roles[].scopeId`. Ví dụ quyền xem đơn ở WH-01 và quyền xem thông tin nhạy cảm ở WH-02 không cho phép xem thông tin nhạy cảm của WH-01. `GLOBAL` chỉ hợp lệ với `scopeId` rỗng/null. Profile chỉ nhận GLOBAL/DEPARTMENT; Order chỉ nhận GLOBAL/WAREHOUSE. Các loại scope khác bị từ chối. Không có ngoại lệ theo tên role ADMIN trong hai service khi dùng Identity.

## Profile

| API | Quyền | Kiểm tra phạm vi |
|---|---|---|
| `GET /api/v1/admin/employees` | `EMPLOYEE_PROFILE_VIEW` | Lọc department trong SQL trước LIMIT/OFFSET; department_id được yêu cầu ngoài phạm vi trả 403 |
| `GET /api/v1/admin/employees/{id}` | `EMPLOYEE_PROFILE_VIEW` | Hồ sơ ngoài department trả 404 trước giải mã CCCD |
| `POST /api/v1/admin/employees` | `EMPLOYEE_PROFILE_CREATE` | department_id trong payload phải thuộc phạm vi quyền tạo |
| `GET /api/v1/admin/employees/compliance` | `EMPLOYEE_COMPLIANCE_VIEW` | Lọc department trong SQL |
| Số điện thoại, CCCD, số chứng nhận trong các phản hồi trên | `EMPLOYEE_PROFILE_SENSITIVE_VIEW` | Kiểm tra riêng theo department của từng hồ sơ |

Không có quyền nhạy cảm thì các chuỗi tương ứng rỗng. CCCD chỉ được giải mã khi có quyền nhạy cảm cho department đó. Các trường ciphertext/DEK không xuất ra JSON. Quyền nhạy cảm không tự cấp quyền đọc hồ sơ hay compliance. Tạo hồ sơ cần quyền tạo, phản hồi vẫn tuân theo quyền xem nhạy cảm.

`/profile/**` và `/customer/capabilities` chỉ nhận Customer; principal Employee không được chuyển sang customer bằng header. Khách hàng chỉ sửa hồ sơ/địa chỉ thuộc ánh xạ principal của mình, giữ nguyên kiểm tra trạng thái và CAS. Credential nội bộ giữ luồng Order → Profile và gRPC; `X-Internal-Token` cộng `X-User-Role: ADMIN` không mở API HR.

## Order

| API / thao tác | Quyền | Kiểm tra phạm vi |
|---|---|---|
| `GET /api/v1/admin/orders`, `/queue` | `ORDER_VIEW` | Lọc warehouse trong SQL trước phân trang |
| `GET /api/v1/orders/{id}`, `/{id}/tracking`; gRPC GetOrderDetail | `ORDER_VIEW` | Đơn ngoài warehouse trả 404 |
| Người nhận, điện thoại, địa chỉ đường/phố | `ORDER_SENSITIVE_VIEW` | Kiểm tra riêng theo kho của đơn; mặc định che thông tin |
| `POST /api/v1/orders/{id}/cancellation-requests`; gRPC CancelOrder cho staff | `ORDER_CANCEL_REQUEST` | Kiểm tra quyền; vẫn chịu state machine, version và approval protocol hiện có |
| `POST /api/v1/admin/orders/{id}/hold` | `ORDER_HOLD_MANAGE` và quyền đọc đơn | Kho của đơn phải thuộc cả hai quyền |
| `POST /api/v1/admin/checkout-operations/{id}/recover` | `ORDER_RECOVERY_EXECUTE` | Kiểm kho từ accepted checkout đã mã hóa, trong transaction trước thay đổi dữ liệu; hỗ trợ nhiều kho |
| `POST /internal/recovery/{id}` | Credential nội bộ và `ORDER_RECOVERY_EXECUTE` | Job theo kho cần thêm quyền đọc đơn; operation sandbox chưa có kho yêu cầu GLOBAL |

Quyền chỉ là điều kiện cần. Core hiện trả `CARE_APPROVAL_REQUIRED` cho yêu cầu hủy của staff; không tạo quyền phê duyệt hủy trực tiếp hay bỏ qua approval ID của Care. Hold với Fulfillment thật vẫn đóng bằng `FULFILLMENT_STOP_CAPABILITY_UNAVAILABLE` cho tới khi có remote stop handshake. Recovery vẫn giữ merchant, idempotency, trạng thái MANUAL_REVIEW và audit. Không bật thêm một năng lực nghiệp vụ chỉ vì đã seed permission.

Customer giữ quyền truy cập đơn của chính mình; nhân viên không vào cart, quote, checkout hay operation của khách. HTTP và gRPC cùng dùng policy đọc đơn và che PII. gRPC Customer cũng resolve trạng thái Profile. Các API Finance/Payment tiếp tục cần connection/receiver scope và MFA hiện có; principal IAM Employee bị từ chối ở protocol payment staff cũ cho tới khi Finance có hợp đồng cấp quyền tương ứng. `ORDER_*` không cấp quyền thanh toán/hoàn tiền.

## Cấu hình và cấp quyền

1. Triển khai Identity trước để chạy migration `V9__seed_profile_order_permissions.sql` và cung cấp `permissionGrants`. Migration chỉ thêm danh mục quyền và gắn các quyền này cho role ADMIN đã seed. Quyền thực tế vẫn phụ thuộc scope assignment; một ADMIN ở STORE không có scope Profile/Order hợp lệ.
2. Profile đặt `PROFILE_AUTH_MODE=keycloak`, `PROFILE_OIDC_ISSUER`, `PROFILE_OIDC_AUDIENCE`, `IDENTITY_URL`. Order đặt `ORDER_AUTH_MODE=keycloak` và cấu hình OIDC/Profile hiện có, cùng `IDENTITY_URL`. Dùng HTTPS theo validation hiện tại; cờ loopback HTTP chỉ dành cho TEST.
3. Dùng API IAM hiện có để tạo role và gắn permission. Gán role cho employee ACTIVE với `scopeType=DEPARTMENT` và ID phòng ban của Profile, hoặc `scopeType=WAREHOUSE` và ID kho mà Order lưu. Một người phụ trách nhiều kho có nhiều assignments. GLOBAL chỉ cấp cho người có trách nhiệm toàn hệ thống.
4. Triển khai Profile/Order. Identity cũ không có `permissionGrants` khiến API nhân viên bị từ chối; luồng Customer không đổi. Không dùng header phía trình duyệt để giả danh staff.

Mỗi request nhân viên đọc lại Identity, vì vậy thu hồi assignment, hết hạn assignment và disable role có hiệu lực ở request tiếp theo, không chờ JWT hết hạn. Request đã được chấp nhận trước thay đổi quyền không bị hủy ngược. Identity lỗi trả 503 và không dùng quyền cache cũ; token sai trả 401; thiếu quyền trả 403; tài nguyên ngoài phạm vi thường trả 404. Phản hồi admin đặt `Cache-Control: no-store`.

Mode Order `sandbox` giữ role model HS256 phục vụ các fixture cũ; không dùng mode này làm cấu hình dev tích hợp/production. Nội bộ gRPC Profile vẫn là credential service riêng, không phải API HR công khai.

## Kiểm chứng

`go test -race ./...` trong cả hai service kiểm policy, JWT/JWKS, giả mạo header và hồi quy hiện có. Bộ `scripts/test-web-identity.py --fulfillment` chạy Identity/Keycloak, Profile với encryption, Order, PostgreSQL và các provider thật của fixture. Module `scripts/profile_order_authorization_checks.py` kiểm dữ liệu của hai phòng ban/hai kho, quyền nhạy cảm ở scope khác, phân trang, đổi quyền trên cùng token, role disabled, assignment expired/revoked và persona gates. CI Web Identity gọi cùng bộ này.

Fixture dùng dữ liệu tổng hợp dùng một lần; kết quả này không cấp quyền lên môi trường dev đang triển khai. Không thay đổi credential, account hay dữ liệu của dev thật khi chạy test local.
