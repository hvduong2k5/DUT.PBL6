# Phân quyền API Profile và Order

## Nguồn quyền và phạm vi

Profile và Order xác minh chữ ký RS256, issuer, audience, hạn token và `user_id`. Staff dùng snapshot permission–scope trong JWT theo [contract tạm v1 và handoff task 1–2](profile-order-jwt-handoff.md). Không đọc quyền staff tươi từ `/users/me`; không nhận quyền nhân viên từ header hay tên role ADMIN. Customer legacy vẫn dùng Identity để kiểm persona.

Claim `permission_grants` trong token giữ quyền và phạm vi của **cùng một nguồn cấp** tại thời điểm phát token. Claim `/users/me.permissionGrants` của Identity hiện tại không thay thế contract JWT. Issuer chưa có contract chính thức thì chưa rollout consumer staff này lên dev chung.

```json
{"permissionCode":"ORDER_VIEW","scopeType":"WAREHOUSE","scopeId":"WH-01"}
```

Không ghép danh sách `permissions` với mọi `roles[].scopeId`. Ví dụ quyền xem đơn ở WH-01 và quyền xem thông tin nhạy cảm ở WH-02 không cho phép xem thông tin nhạy cảm của WH-01. `GLOBAL` chỉ hợp lệ với `scopeId` rỗng/null. HR Profile chỉ nhận GLOBAL theo MVP toàn doanh nghiệp; Order chỉ nhận GLOBAL/WAREHOUSE. Không có ngoại lệ theo tên role ADMIN.

## Profile

| API | Quyền | Kiểm tra phạm vi |
|---|---|---|
| `GET /api/v1/admin/employees` | `EMPLOYEE_PROFILE_VIEW` GLOBAL | Department là bộ lọc, không phải ranh giới quyền |
| `GET /api/v1/admin/employees/{id}` | `EMPLOYEE_PROFILE_VIEW` GLOBAL | Kiểm quyền trước giải mã CCCD |
| `POST /api/v1/admin/employees` | `EMPLOYEE_PROFILE_CREATE` GLOBAL | Giữ validation nghiệp vụ phòng ban |
| `GET /api/v1/admin/employees/compliance` | `EMPLOYEE_COMPLIANCE_VIEW` GLOBAL | HR toàn doanh nghiệp |
| Số điện thoại, CCCD, số chứng nhận trong các phản hồi trên | `EMPLOYEE_PROFILE_SENSITIVE_VIEW` GLOBAL | Kiểm riêng, không mượn scope từ quyền đọc |

Không có quyền nhạy cảm GLOBAL thì các chuỗi tương ứng rỗng. CCCD chỉ được giải mã khi có quyền nhạy cảm. Các trường ciphertext/DEK không xuất ra JSON. Quyền nhạy cảm không tự cấp quyền đọc hồ sơ hay compliance. Tạo hồ sơ cần quyền tạo, phản hồi vẫn tuân theo quyền xem nhạy cảm.

`/profile/**` và `/customer/capabilities` chỉ nhận Customer; principal Employee không được chuyển sang customer bằng header. Khách hàng chỉ sửa hồ sơ/địa chỉ thuộc ánh xạ principal của mình, giữ nguyên kiểm tra trạng thái và CAS. Credential nội bộ giữ luồng Order → Profile và gRPC; `X-Internal-Token` cộng `X-User-Role: ADMIN` không mở API HR.

## Order

| API / thao tác | Quyền | Kiểm tra phạm vi |
|---|---|---|
| `GET /api/v1/admin/orders`, `/queue`, `/{id}` | `ORDER_VIEW` | Lọc warehouse trong SQL trước phân trang; detail projection che recipient riêng |
| `GET /api/v1/orders/{id}`, `/{id}/tracking`; gRPC GetOrderDetail | `ORDER_VIEW` | Đơn ngoài warehouse trả 404 |
| Người nhận, điện thoại, địa chỉ đường/phố | `ORDER_SENSITIVE_VIEW` | Kiểm tra riêng theo kho của đơn; mặc định che thông tin |
| `POST /api/v1/orders/{id}/cancellation-requests`; gRPC CancelOrder cho staff | `ORDER_CANCEL_REQUEST` và quyền đọc đơn | Kho của đơn phải thuộc cả hai quyền trước khi kiểm state machine, version và approval protocol hiện có |
| `POST /api/v1/admin/orders/{id}/hold` | `ORDER_HOLD_MANAGE` và quyền đọc đơn | Kho của đơn phải thuộc cả hai quyền |
| `POST /api/v1/admin/checkout-operations/{id}/recover` | `ORDER_RECOVERY_EXECUTE` | Kiểm kho từ accepted checkout đã mã hóa, trong transaction trước thay đổi dữ liệu; hỗ trợ nhiều kho |
| `POST /internal/recovery/{id}` | Credential nội bộ và `ORDER_RECOVERY_EXECUTE` | Job theo kho cần thêm quyền đọc đơn; operation sandbox chưa có kho yêu cầu GLOBAL |

Quyền chỉ là điều kiện cần. Core hiện trả `CARE_APPROVAL_REQUIRED` cho yêu cầu hủy của staff; không tạo quyền phê duyệt hủy trực tiếp hay bỏ qua approval ID của Care. Hold với Fulfillment thật vẫn đóng bằng `FULFILLMENT_STOP_CAPABILITY_UNAVAILABLE` cho tới khi có remote stop handshake. Recovery vẫn giữ merchant, idempotency, trạng thái MANUAL_REVIEW và audit. Không bật thêm một năng lực nghiệp vụ chỉ vì đã seed permission.

Customer giữ quyền truy cập đơn của chính mình; nhân viên không vào cart, quote, checkout hay operation của khách. HTTP và gRPC cùng dùng policy đọc đơn và che PII. gRPC Customer cũng resolve trạng thái Profile. Các API Finance/Payment tiếp tục cần connection/receiver scope và MFA hiện có; principal IAM Employee bị từ chối ở protocol payment staff cũ cho tới khi Finance có hợp đồng cấp quyền tương ứng. `ORDER_*` không cấp quyền thanh toán/hoàn tiền.

## Cấu hình và cấp quyền

1. Migration `V9__seed_profile_order_permissions.sql` đăng ký quyền và gắn ADMIN seed. Task 1 phải phát JWT staff đúng contract trước khi triển khai consumer. Danh mục quyền trong DB hay projection `/users/me` riêng lẻ chưa đủ.
2. Profile đặt `PROFILE_AUTH_MODE=keycloak`, `PROFILE_OIDC_ISSUER`, `PROFILE_OIDC_AUDIENCE`, `IDENTITY_URL`. Order đặt `ORDER_AUTH_MODE=keycloak` và cấu hình OIDC/Profile hiện có, cùng `IDENTITY_URL`. Dùng HTTPS theo validation hiện tại; cờ loopback HTTP chỉ dành cho TEST.
3. Gán HR scope GLOBAL; Order GLOBAL hoặc WAREHOUSE đúng ID kho thực tế. MVP dùng một kho chung. Quyền nhạy cảm là grant độc lập. Issuer phải giữ permission và scope cùng nguồn cấp.
4. Triển khai Profile/Order sau kiểm chứng token thực. Employee token legacy thiếu contract mới bị từ chối; không fallback đọc quyền tươi. Không dùng header phía trình duyệt để giả danh staff.

Quyền staff giữ snapshot đến khi JWT hết hạn hoặc validUntil sớm hơn. Thu hồi assignment/disable role cần issuer cập nhật token mới; bearer cũ không bị chặn tức thì. TTL chính thức do task 1 chốt. Identity lỗi không làm consumer staff đọc lại quyền; JWKS cache/refresh vẫn cần. Token sai trả 401; principal inactive/thiếu quyền trả 403; tài nguyên ngoài scope thường trả 404. Phản hồi admin đặt `Cache-Control: no-store`. Web BFF hiện còn kiểm account ACTIVE bằng Identity DB; xem giới hạn đường Web trong handoff.

Mode Order `sandbox` giữ role model HS256 phục vụ các fixture cũ; không dùng mode này làm cấu hình dev tích hợp/production. Nội bộ gRPC Profile vẫn là credential service riêng, không phải API HR công khai.

## Kiểm chứng

`go test -race ./...` trong cả hai service kiểm policy, JWT/JWKS, snapshot, grant expiry và giả mạo header. Order PostgreSQL test kiểm scope trước phân trang, cursor và che PII trước giải mã. Bộ `scripts/test-web-identity.py --fulfillment` dùng realm một lần và claims TEST cho consumer, không chứng minh producer IAM sync production đã hoàn tất. Web tests kiểm adapter/projection, upstream lỗi và cookie-only proxy; xem handoff để nghiệm thu task 1–2.

Fixture dùng dữ liệu tổng hợp dùng một lần; kết quả này không cấp quyền lên môi trường dev đang triển khai. Không thay đổi credential, account hay dữ liệu của dev thật khi chạy test local.
