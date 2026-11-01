# Authentication MVP — API Contract Review & Final Architecture Resolution

> Trạng thái: **APPROVED & RESOLVED BY BFF OIDC (RFC 9700 & RFC 7636 PKCE)**  
> Phiên bản: `2.0.0-final`  
> Ngày cập nhật: 2026-11-01  
> OpenAPI: [`authentication-mvp.openapi.yaml`](./authentication-mvp.openapi.yaml)  
> Tài liệu kỹ thuật triển khai: [`frontend-auth-contract.md`](./frontend-auth-contract.md), [`identity-service.openapi.yaml`](./identity-service.openapi.yaml), [`customer-stable.openapi.yaml`](./customer-stable.openapi.yaml)  

---

## 1. Kết Luận Đánh Giá Của Ban Kiến Trúc, Backend & Security

Hợp đồng Candidate `0.1.0-candidate` đề xuất từ UI (2026-09-24) đã được **Review toàn diện và Nâng cấp lên Chuẩn Kiến Trúc Chính Thức** với các nguyên tắc sau:

1. **Loại bỏ hoàn toàn cơ chế ROPC (Resource Owner Password Credentials) cũ:**
   - Trình duyệt KHÔNG gửi `username/password` trực tiếp qua API `POST /auth/login` để lấy Token về JavaScript.
   - Áp dụng **BFF Pattern (Backend-for-Frontend)** theo khuyến nghị bảo mật cao nhất **RFC 9700** và **RFC 7636 PKCE** nhằm triệt tiêu nguy cơ bị đánh cắp Token khi xảy ra tấn công XSS.
2. **Keycloak 24 OIDC Discovery làm trung tâm Identity Provider (IdP):**
   - Đăng nhập (Web Form, Google, Zalo, Apple) và Quên mật khẩu được xử lý tập trung trên Keycloak mang giao diện chuẩn thương hiệu O Mạ (theme `omama` hoàng gia nâu `#563a26`, hỗ trợ đa ngữ Tiếng Việt / Tiếng Anh).
   - Tắt tính năng `verifyEmail` thủ công cho MVP theo quyết định nghiệp vụ, giảm ma sát chuyển đổi đơn hàng.
3. **Bảo mật phiên bằng Cookie HttpOnly duy nhất:**
   - Phiên đăng nhập giữa Trình duyệt và BFF được duy trì bởi Cookie `OMAMA_SESSION` (`HttpOnly`, `Secure`, `SameSite=Lax`, TTL 30 ngày).
   - Access Token và Refresh Token được lưu trữ bảo mật trong `BffSessionStore` phía backend, không để lộ ra client-side scripts.
4. **Đăng ký tài khoản Khách hàng D2C chuẩn RESTful 3.3.1:**
   - Sử dụng endpoint danh từ số nhiều: `POST /api/v1/public/customer-registrations` với Idempotency Key.
   - Áp dụng Transactional Outbox pattern phát sự kiện CloudEvents 1.0 `identity.events.v1.UserRegistered` sang Apache Kafka để `profile-service` tự động khởi tạo hồ sơ người dùng (`customer_profiles`).

---

## 2. Ranh Giới Kiến Trúc & Cổng Vào (Single Ingress)

| Lớp | Đường dẫn / Host | Mục đích & Ranh giới |
| --- | --- | --- |
| **Browser → BFF** | `https://omama.vn` | Ứng dụng Next.js Frontend duy nhất cho Khách hàng & Nhân sự. |
| **BFF → Kong Gateway** | `https://api.omama.vn/api/v1/public/auth/*` | Các API quản lý vòng đời phiên làm việc (Login, Callback, Session, Refresh, Logout). |
| **Public Registrations** | `https://api.omama.vn/api/v1/public/customer-registrations` | Tiếp nhận đăng ký tài khoản mới D2C. |
| **Identity Core APIs** | `https://api.omama.vn/api/v1/users/me`, `/api/v1/admin/*` | Nhận diện hồ sơ, Roles và Quyền hạn hiệu lực phân cấp. |
| **Keycloak IdP** | `https://api.omama.vn/auth/realms/omama/*` | Giao diện OIDC đăng nhập và phục hồi mật khẩu. |

---

## 3. Bảng Ánh Xạ Khả Năng (Capability Mapping Table)

| ID | Candidate Cũ (`0.1.0-candidate`) | Hợp Đồng Chính Thức Đã Duyệt (`2.0.0`) | Trạng thái | Ghi chú & Quyết định kỹ thuật |
| --- | --- | --- | --- | --- |
| `AUTH-C01` | `POST /auth/login` | `GET /api/v1/public/auth/login`<br/>`GET /api/v1/public/auth/callback` | **Thay thế** | OIDC Authorization Code + PKCE. Trả về `authorizationUrl` sang Keycloak, callback cấp cookie `OMAMA_SESSION`. |
| `AUTH-C02` | `GET /auth/me` | `GET /api/v1/public/auth/session`<br/>`GET /api/v1/users/me` | **Thay thế** | `/session` trả trạng thái xác thực và email; `/users/me` trả chi tiết Profile, Roles và Permissions. |
| `AUTH-C03` | `POST /auth/logout` | `POST /api/v1/public/auth/logout` | **Thay thế** | Idempotent revoke session trên BFF và Keycloak, dọn sạch Cookie `OMAMA_SESSION` (`Max-Age=0`). |
| `AUTH-C04` | `POST /auth/register` | `POST /api/v1/public/customer-registrations` | **Thay thế** | Tuân thủ Quy tắc 2 & 4 (danh từ số nhiều, kebab-case). Đăng ký trực tiếp ACTIVE, bắn event `UserRegistered`. |
| `AUTH-C05` | `POST /auth/registration-verifications/confirm` | N/A (Keycloak Action Token) | **Bỏ khỏi MVP** | Tắt `verifyEmail: false` cho MVP. Khi kích hoạt lại sẽ do Keycloak xử lý tự động. |
| `AUTH-C06` | `POST /auth/registration-verifications/{id}/resend`| N/A (Keycloak Action Token) | **Bỏ khỏi MVP** | Keycloak native resend flow. |
| `AUTH-C07` | `POST /auth/recovery-requests` | Keycloak Reset Password Flow | **Thay thế** | Xử lý trực tiếp qua giao diện Keycloak Ô Mạ (`login-reset-password.ftl`). |
| `AUTH-C08` | `POST /auth/recovery-requests/verify` | Keycloak Action Token | **Thay thế** | Keycloak xác thực token đặt lại mật khẩu qua email link. |
| `AUTH-C09` | `POST /auth/recovery-requests/reset` | Keycloak Form Submit | **Thay thế** | Keycloak cập nhật credential mật khẩu mới và thu hồi mọi session cũ. |
| `AUTH-C10` | N/A | `POST /api/v1/public/auth/tokens/refresh` | **Bổ sung** | Cho phép chủ động làm mới phiên làm việc phía server mà không làm lộ Refresh Token ra JS. |
| `AUTH-C11` | N/A | `PUT /api/v1/users/me/password` | **Bổ sung** | Người dùng đã đăng nhập tự đổi mật khẩu cá nhân (xác thực mật khẩu hiện tại và đặt mật khẩu mới). |
| `AUTH-C12` | N/A | `POST /api/v1/admin/users/{principalId}/password/reset` | **Bổ sung** | SUPER_ADMIN đặt lại mật khẩu mới cho người dùng trên Keycloak IDP. |

---

## 4. Quyết Định Kiến Trúc Đã Chốt (Answers to Review Questions)

| ID | Câu hỏi đánh giá ban đầu | Quyết định kỹ thuật chính thức |
| --- | --- | --- |
| `AUTH-CR-01` | Public contract thuộc BFF hay Gateway; prefix? | Gateway tiếp nhận duy nhất qua prefix chuẩn `/api/v1/public/auth/*` và `/api/v1/users/*`. |
| `AUTH-CR-02` | Cookie được tạo/rotate ở đâu? | BFF (Identity Service MS-16) cấp và quản lý cookie `OMAMA_SESSION` (`HttpOnly`, `SameSite=Lax`, `Secure`). |
| `AUTH-CR-03` | CSRF policy cho cookie? | Trình duyệt dùng `SameSite=Lax`, kết hợp Origin / Referer check tại Kong Gateway. |
| `AUTH-CR-04` | Logout khi thiếu cookie? | Hoàn toàn Idempotent (`200 OK` hoặc `204`), luôn xóa cookie trên response header. |
| `AUTH-CR-05` | Mã lỗi phân loại thế nào? | Chuẩn hóa theo RFC 7807 Envelope: `code`, `message`, `requestId`, `errors[]` (400, 401, 403, 404, 409, 422, 500). |
| `AUTH-CR-06` | Rate limiting? | Kong Gateway plugin Rate Limiting (100 req/min/IP), trả header chuẩn `Retry-After`. |
| `AUTH-CR-07` | Chuẩn hóa họ tên người dùng? | Tách `firstName` và `lastName` tại đăng ký; `profile-service` tự gộp `full_name = strings.TrimSpace(firstName + " " + lastName)`. |
| `AUTH-CR-08` | Xác minh email? | Tắt bắt buộc xác minh email (`verifyEmail=false`) trong MVP để tối ưu tỷ lệ hoàn tất đơn hàng. |
| `AUTH-CR-09` | Khóa công khai JWKS? | API Gateway và microservices nạp trực tiếp JWKS từ Keycloak OIDC Discovery certs (`/certs`); không gọi blocking RPC. |
| `AUTH-CR-10` | Correlation ID? | Gắn `requestId` trong mọi response payload và header `X-Request-Id`. |
| `AUTH-CR-11` | Khóa tài khoản (Account Suspension)? | API `PUT /api/v1/admin/users/{principalId}/status` cập nhật `SUSPENDED` và bắn event `UserSuspended`. |
| `AUTH-CR-12` | Token TTL? | Access Token: 15 phút (900s). Refresh Token / Session Cookie: 30 ngày. |

---

## 5. Kết Luận & Hướng Dẫn Bàn Giao

1. **Đối với Kỹ sư Frontend (Web/Mobile):**
   - Đọc và tuân thủ tài liệu hướng dẫn hợp nhất tại [`frontend-unified-auth-guide.md`](./frontend-unified-auth-guide.md) và hợp đồng BFF [`frontend-auth-contract.md`](./frontend-auth-contract.md).
   - Thiết lập `credentials: 'include'` trên fetch/axios client.
2. **Đối với Kiểm thử Hợp đồng (Contract Testing):**
   - Sử dụng [`customer-stable.openapi.yaml`](./customer-stable.openapi.yaml) và [`identity-service.openapi.yaml`](./identity-service.openapi.yaml) làm Single Source of Truth cho các test suite tự động.
