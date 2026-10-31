# Phân tích đầy đủ Identity / Auth Service (MS-16)

> Cập nhật: 10-10-2026 · Phạm vi: tài liệu yêu cầu, kiến trúc, OpenAPI, Protobuf và event package trong workspace.  
> **Trạng thái triển khai trong thư mục này:** `services/auth-service` hiện chưa có source, build file, cấu hình, migration hay test Identity. Các API bên dưới là **contract được công bố**, không phải API đã kiểm chứng chạy được.

## 1. Kết luận điều hành

`auth-service` là tên thư mục hiện có, còn kiến trúc, Protobuf, topic Kafka và service registry đều gọi dịch vụ là **`identity-service` (MS-16, BC-16 Auth & IAM, port 8016)**. Hai tên phải được hợp nhất trước khi tạo artifact runtime, database, topic producer và service discovery.

Identity là nguồn chân lý cho **thông tin xác thực, trạng thái tài khoản, token/JWT, khóa ký JWT và quyền nhân viên**. Nó không phải nguồn chân lý cho hồ sơ khách hàng/nhân viên, địa chỉ, hạng thành viên hoặc điểm loyalty. Những dữ liệu đó thuộc Profile/Promotion. API customer stable hiện trả lẫn dữ liệu hai miền trong `login`, `register` và `me`; đây là API composition bắt buộc phải được thiết kế, không được sao chép các bảng Profile vào Identity.

Thiết kế đích dùng Gateway kiểm tra chữ ký JWT bằng JWKS cache cục bộ. Chỉ các quyết định quyền rủi ro động mới gọi Identity qua gRPC. Một số tài liệu cũ vẫn mô tả `ValidateToken` và `CheckPermission` cho mọi service; hai RPC này **không tồn tại** trong proto hiện tại và đi ngược thiết kế đích.

Các điểm chặn trước khi code nằm trong [contract-gap-va-ke-hoach.md](contract-gap-va-ke-hoach.md): tên dịch vụ, TTL access token, thuật toán hash, mô hình revoke/session, REST response composition, event schema và contract IAM admin.

## 2. Phạm vi sở hữu

| Thuộc Identity / Auth | Không thuộc Identity / Auth | Ranh giới tích hợp |
|---|---|---|
| Account, email/phone định danh, password hash, social identity, trạng thái account và số lần đăng nhập lỗi | Họ tên, avatar, ngày sinh, địa chỉ giao hàng, customer/employee business profile | Profile giữ `user_id` là soft reference, không dùng cross-database FK |
| JWT access/refresh token, signing key/JWKS, session/revocation, guest identity context | Đơn hàng, lịch sử đơn và quyền sở hữu đơn Guest | Order là owner đơn; việc claim đơn Guest là flow liên miền ở Phase 2 |
| Role, permission, role assignment, scope/thời hạn quyền và quyết định quyền đặc biệt | Điều kiện nghiệp vụ riêng như phân công packing, điều kiện refund, stock, workflow approval | RBAC không được bỏ qua điều kiện nghiệp vụ của service đích |
| Login, register, logout, refresh, password recovery, khóa/mở tài khoản | Membership tier và loyalty points | Promotion/Profile là owner; nếu cần trong auth response phải compose/read model |

Nguồn: [BC-16](../../docs/02_architecture/bounded_context.md), [service boundary](../../docs/02_architecture/service_boundary.md), [thiết kế Profile](../../docs/02_architecture/services/ms15_profile_service_design.md), [EPIC 01](../../docs/01_requirements/epics/EPIC_01_Authentication.md) và [EPIC 22](../../docs/01_requirements/epics/EPIC_22_Administration.md).

## 3. Nghiệp vụ phải đáp ứng

### Customer và guest

| Phạm vi | Yêu cầu đã có | Trạng thái contract |
|---|---|---|
| Guest checkout | Khách đặt hàng không cần đăng nhập và không tự tạo account | Requirement có; chưa thấy public auth endpoint định nghĩa guest context/guest grant |
| Đăng ký | Không trùng định danh; trả phiên đăng nhập | `POST /api/v1/auth/register` stable |
| Đăng nhập | Xác thực account; lock sau 5 lần sai liên tiếp | `POST /api/v1/auth/login` stable; policy lock cần hiện thực rõ |
| Session | Refresh/revoke token, logout và cập nhật quyền an toàn | `refresh`, `logout` stable; cơ chế cookie/body và revoke cần chốt |
| Khôi phục mật khẩu | Yêu cầu trung lập, verify token rồi reset; reset thu hồi toàn bộ session | Ba endpoint `recovery-requests` stable |
| Social login | Google/Facebook/Zalo, tránh tạo trùng account | Phase 2, chưa có stable API contract |
| Claim đơn Guest | Chứng minh quyền sở hữu rồi liên kết order cũ | Phase 2, chưa có API contract; Order vẫn owner dữ liệu |

Epic 01 tách quản trị role nhân viên khỏi phạm vi customer-auth; phần này thuộc Epic 22. Vì vậy không nên đưa API role admin vào cùng customer web contract chỉ vì chúng dùng chung Identity database.

### Nhân viên và IAM admin

Identity cung cấp nền tảng kỹ thuật cho các yêu cầu Epic 22: mời/cấp access nhân viên, đổi trạng thái, role, permission, role assignment có scope/thời hạn, review quyền hiệu lực và audit. Quy tắc đáng chú ý:

- Account status, role active, assignment còn hiệu lực và permission/scope đều phải hợp lệ mới có quyền hiệu lực.
- Không được mở rộng quyền `view` thành `update`, `approve` hoặc `export`.
- Role hệ thống/đặc quyền và System Administrator cuối cùng phải được bảo vệ.
- Lock/disable/thu hồi role phải ngăn login mới và xử lý session/quyền còn hoạt động theo policy.
- Mọi thay đổi account, role, permission, assignment phải có audit trail.

Các endpoint `/api/v1/admin/access/**` đang có trong `admin-draft-planned.openapi.yaml` với `x-lifecycle: draft`, chưa phải stable public/admin contract.

## 4. API frontend qua Gateway

### Contract customer stable — nguồn cần ưu tiên cho frontend

Nguồn chính là [customer-stable.openapi.yaml](../../docs/03_api_specs/customer-stable.openapi.yaml). Route được Gateway chuyển tới Identity hoặc một lớp composition đã được chốt; frontend không gọi gRPC trực tiếp.

| Endpoint stable | Xác thực | Input/Output chính | Quy tắc đáng chú ý |
|---|---|---|---|
| `POST /api/v1/auth/login` | Public | `identifier`, `password` → `user` + `tokens` | Có `Idempotency-Key`; response example `expires_in=900`; 401 sai xác thực, 403 account không được phép |
| `POST /api/v1/auth/register` | Public | `phone_number`, `full_name`, `email`, `password` → `user` + `tokens` | Có `Idempotency-Key`, trả 201; `full_name` là Profile-owned input cần orchestration/outbox |
| `POST /api/v1/auth/logout` | `bearerAuth` **hoặc** cookie `customerSession` | → `{success,message}` | Có idempotency; phải revoke refresh/session và xác định access JWT bị revoke kiểm tra thế nào |
| `GET /api/v1/auth/me` | `bearerAuth` **hoặc** cookie `customerSession` | → customer view | Response gồm customer id, phone/email, name/avatar, tier/loyalty — phải compose, không copy ownership |
| `POST /api/v1/auth/refresh` | Contract hiện khai báo bearer **hoặc** cookie, body `refresh_token` | → token pair mới | Cần quyết định một mô hình an toàn, rotation và replay detection |
| `POST /api/v1/auth/recovery-requests` | Public | `{email}` → 202 `{accepted,messageCode}` | Phản hồi trung lập chống account enumeration; rate-limit; có idempotency |
| `POST /api/v1/auth/recovery-requests/verify` | Public | `{recoveryToken}` → `{verified,resetProof,expiresAt}` | Reset proof là capability giới hạn thời gian, không phải user session |
| `POST /api/v1/auth/recovery-requests/reset` | Public | `{resetProof,newPassword}` → 204 | Thu hồi toàn bộ session rồi yêu cầu login lại; 410 khi proof hết hạn |

`customerSession` được khai báo là cookie security scheme; triển khai phải chốt tên cookie, `Secure`, `HttpOnly`, `SameSite`, domain/path, CSRF policy và cách nó liên hệ với refresh token. Không gửi token thật vào log, error, analytics hay Kafka event.

### Định dạng response customer cần composition

`login`, `register`, `me` cùng đòi một object `user` có `customer_id`, `phone_number`, `full_name`, `email`, `avatar_url`, `membership_tier`, `loyalty_points`. Identity chỉ có cơ sở giữ định danh email/phone; Profile có full name/avatar, Promotion/loyalty là owner tier/points. Có ba lựa chọn hợp lệ:

1. Gateway/BFF gọi Identity để xác thực, rồi đọc Profile và Promotion; chỉ trả response sau khi có chiến lược timeout/fallback.
2. Xây dựng read model composition bất đồng bộ; công khai lag và quy tắc khi profile mới chưa tạo xong.
3. Thu hẹp response Auth về identity/tokens, để frontend gọi `/profile` và loyalty endpoint riêng.

Không chọn phương án nào thì service buộc phải sở hữu dữ liệu sai ranh giới hoặc login bị phụ thuộc đồng bộ vào nhiều service.

### Các tài liệu OpenAPI khác

- [openapi_d2c.yaml](../../docs/03_api_specs/openapi_d2c.yaml) có các route auth tương tự nhưng mô tả login bằng **phone** và access token 15 phút. Nó không nhất quán hoàn toàn với customer stable (`identifier`, response giàu dữ liệu, 900 giây) nên không được dùng đồng thời làm nguồn codegen.
- [customer-draft-planned.openapi.yaml](../../docs/03_api_specs/customer-draft-planned.openapi.yaml) thêm verification request/confirm; chúng là **draft**, không phải cam kết frontend stable.
- [admin-stable.openapi.yaml](../../docs/03_api_specs/admin-stable.openapi.yaml) dùng JWT/cookie admin cho các thao tác admin hiện có. Contract tạo/đổi IAM cụ thể hiện chỉ nằm trong [admin-draft-planned.openapi.yaml](../../docs/03_api_specs/admin-draft-planned.openapi.yaml): invitation nhân viên, account status, role assignment, tạo/sửa role, thay permission role.

## 5. gRPC nội bộ — contract thực tế

Nguồn chuẩn code là [identity.proto](../../packages/proto/identity/v1/identity.proto); port kiến trúc là `8016`.

| RPC thực có | Consumer/case dùng | Request quan trọng | Response/quy tắc |
|---|---|---|---|
| `GetJwksPublicKey` | API Gateway nạp/xoay cache public key | `key_id` optional | `key_id`, `algorithm` (`RS256`/`EdDSA`), `public_key_pem`. Gateway tự kiểm JWT cho request bình thường |
| `CheckSpecializedPermission` | Finance refund lớn, Order/B2B discount lớn | `user_id`, `action`, `context_attributes`, `AuditContext` optional | `is_allowed`, `decision_reason`, `required_approval_level` optional, `ErrorDetail` optional |

`AuditContext` dùng chung từ `common/v1/types.proto`: actor id/role, IP client, traceparent và user agent optional. Error nội bộ dùng `common/v1/errors.proto`, gồm lỗi chuẩn như unauthenticated, permission denied, invalid argument, unavailable và deadline exceeded.

HLD quy định chỉ gọi quyền động cho ngưỡng rủi ro, ví dụ refund trên 10 triệu VND hoặc discount B2B trên 25%. Request thông thường phải được Gateway xác thực JWT bằng JWKS cache rồi chuyển các header tin cậy/sanitized (`X-User-*`, role/permission, trace context) vào mạng nội bộ. Service downstream phải chặn headers cùng tên đến trực tiếp từ internet.

> **Không có** `ValidateToken` hoặc `CheckPermission` trong `identity.proto`. Những API này còn xuất hiện ở `service_boundary.md`; đó là documentation drift, không thể code against contract hiện tại.

## 6. Dữ liệu, bảo mật và vận hành

### Mô hình dữ liệu được kiến trúc yêu cầu

| Nhóm | Bảng/khái niệm được nêu | Mục đích |
|---|---|---|
| Account | `user_accounts` | email, phone/username, password hash, status, failed attempts, last login, account type customer/employee |
| Federation | `social_identities` | ánh xạ provider identity với account nội bộ |
| IAM | `roles`, permission catalog, `user_role_assignments` | quyền theo role, scope/thời hạn và trạng thái hiệu lực |
| Session | `refresh_tokens`, token/session revocation | refresh rotation, revoke logout/reset/lock/disable |
| Recovery | `otp_records` hoặc recovery-token record | one-time, expiry, rate-limit, audit; schema chi tiết chưa chốt |
| Redis | `session:*`, `blacklist:*`, rate-limit/counter | dữ liệu nhanh/có TTL; không thay PostgreSQL làm nguồn account/IAM |

Database riêng là PostgreSQL `identity_db`; Profile chỉ lưu `user_id` tham chiếu mềm. Migration do Identity quản lý, không truy cập bảng Profile/Order trực tiếp.

### Security baseline cần kiểm thử

- Không lưu hay trả password, recovery token, reset proof, refresh token plain-text trong database/log/audit/event. Lưu secret/token theo dạng hash phù hợp, one-time use và TTL.
- Lock sau 5 lần đăng nhập sai liên tiếp theo BC-16; chính sách unlock, counter reset, retry/rate limit/IP/device và thông báo an toàn cần được ghi thành contract vận hành.
- Reset password phải revoke toàn bộ session; disable/lock/role revoke cần có policy buộc token cũ mất hiệu lực trong giới hạn thời gian định nghĩa.
- Key signing phải hỗ trợ `kid`, key rotation, JWKS cache TTL, overlap key cũ/mới và behavior khi Identity/JWKS tạm unavailable.
- JWT claim cần tối thiểu gồm subject/user id, roles/permissions theo policy, issue/expiry, issuer/audience, key id và correlation/trace tách khỏi secret. Không đưa PII dư thừa vào token.
- Xác thực mTLS/TLS nội bộ, validate issuer/audience/signature/expiry, sanitized headers tại Gateway, rate limit anti-brute-force và audit cho toàn bộ security action.

## 7. Event Kafka và audit

Kiến trúc và [Kafka API catalog](../../docs/03_api_specs/02_kafka_event_catalog.md) quy định Identity publish topic `identity.events.v1`, partition key `user_id`:

| Event được nêu | Consumer được nêu | Mục đích |
|---|---|---|
| `UserRegistered` | Profile, Marketing | Profile tạo hồ sơ rỗng; Marketing khởi động welcome flow |
| `UserDeactivated` | Chưa đặc tả consumer trong catalog | Đồng bộ vô hiệu hóa/bảo mật downstream nếu được phê duyệt |
| `PasswordChanged` | Chưa đặc tả consumer trong catalog | Theo dõi security side-effect, không mang password/token |

Hiện chưa có thư mục/schema Identity trong [packages/events](../../packages/events); `packages/events/README.md` cũng chưa đăng ký `identity.events.v1`, trong khi API catalog có. Vì vậy không được tự phát JSON tùy ý. Cần tạo schema versioned trước khi producer/consumer code.

Mọi event phải bọc CloudEvents 1.0, backward-compatible, tối thiểu PII, retry 3 lần exponential backoff rồi DLQ theo policy package. Identity cũng là một trong các service phải phát `AuditRecordEvent` vào `audit.events.v1` cho security-sensitive actions; audit không được là dependency đồng bộ chặn login.

## 8. Services và contract bị ảnh hưởng

| Thành phần | Phụ thuộc vào Identity | Việc cần thống nhất |
|---|---|---|
| API Gateway | JWT validation cục bộ bằng JWKS; route `/api/v1/auth/**`; trusted identity headers | cache/key rotation, header sanitation, error mapping, endpoint composition |
| Profile Service | consume `UserRegistered`; soft reference `user_id`; profile data cho response auth | idempotency event, ordering, account/profile creation failure và response mới đăng ký |
| Order/Checkout | guest vs customer context; Phase-2 guest-order claim | guest grant contract, proof of ownership, Order remains owner |
| Finance và Order/B2B | `CheckSpecializedPermission` cho high-risk decision | canonical `action`, context attributes, threshold, audit context, timeout/fail-closed policy |
| Promotion/Loyalty | `membership_tier`, `loyalty_points` hiện có trong auth responses | ownership/read model hoặc BFF composition |
| Marketing | consume registration event | Identity event schema version, PII minimization, dedupe |
| Notification | gửi recovery message theo flow | command/event contract chưa được đặc tả; không publish recovery secret rộng rãi |
| Audit Service | nhận audit event từ security actions | event fields, correlation id, immutable evidence |
| Inventory/Fulfillment/Care và service khác | nhận identity context từ Gateway | không gọi `ValidateToken` giả định; loại bỏ dependency sync cũ hoặc bổ sung proto mới sau ADR |

## 9. Definition of done tối thiểu

Một Identity Service chỉ có thể được coi là hoàn thành MVP khi đồng thời có:

1. Một service name/runtime/database/topic canonical và Flyway migrations độc lập.
2. Stable customer contract được hiện thực hoặc một ADR chỉ rõ các route nào Gateway/BFF compose.
3. JWT signing + JWKS RPC theo đúng proto; Gateway local validation, key rotation và resilience test.
4. Register/login/logout/refresh/recovery có validation, idempotency khi contract yêu cầu, rate limiting, account lock, session revoke và audit.
5. Profile bootstrap và audit/event publishing transactional (outbox), schema versioned, dedupe/DLQ/retry test.
6. IAM admin được tách/lifecycle rõ là draft hay stable, với optimistic concurrency, scope, last-admin protection và immediate/effective revocation policy.
7. Test contract OpenAPI/proto, unit/domain, integration PostgreSQL+Redis+Kafka, security regression và failure/restart/key-rotation scenarios.

## 10. Danh mục nguồn đã đối chiếu

- [EPIC 01 — Authentication](../../docs/01_requirements/epics/EPIC_01_Authentication.md), [EPIC 22 — Administration](../../docs/01_requirements/epics/EPIC_22_Administration.md)
- [Bounded Context](../../docs/02_architecture/bounded_context.md), [Service Boundary](../../docs/02_architecture/service_boundary.md), [High Level Design](../../docs/02_architecture/high_level_design.md), [Profile Service Design](../../docs/02_architecture/services/ms15_profile_service_design.md)
- [Customer stable OpenAPI](../../docs/03_api_specs/customer-stable.openapi.yaml), [Customer planned draft](../../docs/03_api_specs/customer-draft-planned.openapi.yaml), [Admin stable OpenAPI](../../docs/03_api_specs/admin-stable.openapi.yaml), [Admin planned draft](../../docs/03_api_specs/admin-draft-planned.openapi.yaml), [legacy D2C OpenAPI](../../docs/03_api_specs/openapi_d2c.yaml), [gRPC API document](../../docs/03_api_specs/01_grpc_contracts.md), [Kafka API catalog](../../docs/03_api_specs/02_kafka_event_catalog.md)
- [Identity protobuf](../../packages/proto/identity/v1/identity.proto), [shared protobuf types](../../packages/proto/common/v1/types.proto), [shared error model](../../packages/proto/common/v1/errors.proto), [event package policy](../../packages/events/README.md)

Chi tiết các sai khác, mức ưu tiên và trình tự xử lý: [contract-gap-va-ke-hoach.md](contract-gap-va-ke-hoach.md).
