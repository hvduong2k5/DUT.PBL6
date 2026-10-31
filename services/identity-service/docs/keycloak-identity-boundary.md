# Ranh Giới Kiến Trúc Giữa Keycloak và Identity Service
**Dự án PBL6 — Mè Xửng O Mạ (MS-16)**

---

## 1. Triết Lý Thiết Kế và Ranh Giới Trách Nhiệm

Kiến trúc định danh của hệ thống không cố gắng "tự viết lại toàn bộ hệ thống xác thực", mà kết hợp sức mạnh của một **Identity Provider (IdP) chuẩn quốc tế (Keycloak)** với một **Microservice nghiệp vụ chuyên biệt (`identity-service`)**:

* **Keycloak (Nền tảng hạ tầng - Authentication & Core IAM):** Sở hữu tính an toàn tuyệt đối của thông tin đăng nhập, phiên làm việc (session), chuẩn OIDC/OAuth 2.0, cấp phát token và bảo mật mật khẩu.
* **`identity-service` (Nghiệp vụ dự án - Scoped Authorization & Orchestration):** Sở hữu định danh nội bộ (`principal_id`), quy trình đăng ký idempotent, quản lý nhân viên theo chi nhánh/kho hàng (`Scope`), chính sách duyệt tiền bạc/chiết khấu động (ABAC qua gRPC), và phát tán sự kiện (`Transactional Outbox`).

```
┌────────────────────────────────────────────────────────────────────────────────────────┐
│                                   CLIENTS / EDGE GATEWAY                               │
└──────────────────┬──────────────────────────────────────────────────┬──────────────────┘
                   │ 1. Đăng nhập OIDC                                │ 2. Gọi API kèm JWT
                   ▼                                                  ▼
┌──────────────────────────────────────┐           ┌─────────────────────────────────────┐
│           KEYCLOAK (IdP)             │           │        API GATEWAY / PROXY          │
│                                      │           └──────────────────┬──────────────────┘
│ • Băm & Quản lý mật khẩu (Bcrypt)    │                              │
│ • OIDC Login, SSO, Session, MFA      │                              │ 3. Forward request
│ • Refresh Token, Password Reset      │                              │    (Bearer JWT)
│ • Cấp phát & Ký số JWT Token         │                              │
│ • Public JWKS Keys (/certs)          │                              ▼
└──────────────────┬───────────────────┘           ┌─────────────────────────────────────┐
                   │                               │      IDENTITY SERVICE (MS-16)       │
                   │ 4. Admin REST API             │                                     │
                   │    (Tạo user, khóa user)      │ • Principal ID nội bộ (UUID)        │
                   └──────────────────────────────>│ • Điều phối Đăng ký Idempotent      │
                                                   │ • Mời nhân viên & Onboarding        │
                                                   │ • Phân quyền Scope (Store/Warehouse)│
                                                   │ • Quyết định rủi ro cao qua gRPC    │
                                                   │ • Transactional Outbox (Kafka)      │
                                                   │ • Audit Log bảo mật độc lập         │
                                                   └──────────────────┬──────────────────┘
                                                                      │
                                                   ┌──────────────────┴──────────────────┐
                                                   ▼                                     ▼
                                      ┌────────────────────────┐            ┌────────────────────────┐
                                      │ PostgreSQL identity_db │            │ Apache Kafka (Events)  │
                                      └────────────────────────┘            └────────────────────────┘
```

---

## 2. Bảng Phân Định Nguồn Chân Lý Dữ Liệu (Source of Truth)

Nguyên tắc bất biến: **Mỗi phần dữ liệu chỉ có DUY NHẤT một nguồn chân lý**. Tuyệt đối không lưu trùng lặp hoặc phân tán trách nhiệm.

| Dữ liệu / Thực thể | Nguồn chân lý (Owner) | Lưu trữ tại | Trách nhiệm và Quyền hạn |
| :--- | :---: | :---: | :--- |
| **Mật khẩu (Password / Hash)** | **Keycloak** | `keycloak_db` | Băm mật khẩu (PBKDF2/Bcrypt), chính sách độ dài, chống dò mật khẩu (Brute-force detection). `identity_db` **không bao giờ lưu mật khẩu**. |
| **Phiên đăng nhập (User Session)** | **Keycloak** | `keycloak_db` / RAM | Quản lý SSO session, phiên nhàn rỗi (idle timeout), cưỡng chế logout (Revoke session). |
| **Token (Access & Refresh Token)** | **Keycloak** | Keycloak RAM / Client | Ký số JWT bằng khóa riêng (RS256 Private Key), cấp lại token mới từ Refresh Token. |
| **Trạng thái Enabled/Disabled** | **Keycloak** | `keycloak_db` | Cho phép hoặc chặn người dùng đăng nhập ngay từ cửa ngõ xác thực. |
| **Cặp khóa ký số (JWKS)** | **Keycloak** | Endpoint `/certs` | Cung cấp Public Key để API Gateway và Backend tự giải mã token offline. |
| **Social Login (Google, Zalo)** | **Keycloak** | `keycloak_db` | Quản lý liên kết tài khoản mạng xã hội với tài khoản hệ thống. |
| **Principal ID nội bộ** | **identity-service** | `identity_principals` | Định danh UUID gốc dùng chung cho toàn bộ hệ sinh thái Microservices (Order, Cart, Profile...). |
| **Mapping ID (`principal_id` $\leftrightarrow$ `idp_subject`)** | **identity-service** | `identity_principals` | Ánh xạ ID nội bộ với Subject ID do Keycloak cấp. |
| **Idempotency Metadata** | **identity-service** | `registration_operations` | Chống gửi trùng request đăng ký (replay attack), lưu request fingerprint. |
| **Lời mời nhân viên (Invitations)** | **identity-service** | `employee_invitations` | Quản lý vòng đời lời mời (Token, Ngày hết hạn, Trạng thái PENDING/ACCEPTED/REVOKED). |
| **Phân quyền theo Chi nhánh/Kho** | **identity-service** | `identity_role_assignments` | Gán vai trò kèm phạm vi: `GLOBAL`, `STORE`, `WAREHOUSE` cùng thời hạn hiệu lực. |
| **Ngưỡng duyệt tiền/Chiết khấu (ABAC)** | **identity-service** | `identity_specialized_policies`| Quy tắc hạn mức duyệt tiền (vd: hoàn tiền tối đa 5 triệu) cho gRPC kiểm tra. |
| **Nhật ký kiểm toán (Audit Trail)** | **identity-service** | `identity_audit_log` | Truy vết bất biến: Ai làm gì, lúc nào, trên tài nguyên nào, từ IP nào. |
| **Sự kiện tích hợp miền (Outbox)** | **identity-service** | `identity_outbox` | Lưu trữ sự kiện (`UserRegistered`, `UserSuspended`) trước khi phát tán lên Kafka. |
| **Hồ sơ cá nhân (Họ tên, SĐT, Avatar)** | **Profile Service** | `profile_db` | `identity-service` chỉ giữ metadata định danh, không quản lý địa chỉ hay ảnh đại diện. |
| **Giỏ hàng & Khách vãng lai (Guest)** | **Cart / Order Service** | `cart_db` / `order_db` | Keycloak và Identity Service **không tạo tài khoản** cho khách vãng lai chỉ xem hàng. |

---

## 3. Giao Thức và Kênh Giao Tiếp Giữa Hai Bên

Keycloak và `identity-service` kết nối với nhau thông qua **3 giao thức chuẩn**:

### 3.1. Admin REST API (Giao tiếp điều khiển)
* **Chiều gọi:** `identity-service` $\longrightarrow$ `Keycloak Server`.
* **Thư viện sử dụng:** `keycloak-admin-client` (Java SDK).
* **Cơ chế xác thực:** OAuth2 Client Credentials Flow (`grant_type=client_credentials`).
* **Service Account:** Client `identity-service-admin` được cấp quyền tối thiểu (`manage-users`, `view-users`) trong Keycloak.
* **Các hành động thực hiện:**
  - `POST /admin/realms/{realm}/users`: Tạo tài khoản Keycloak khi khách đăng ký.
  - `PUT /admin/realms/{realm}/users/{id}`: Bật/tắt trạng thái `enabled` (khóa tài khoản).
  - Gửi email kích hoạt tài khoản (`execute-actions-email`).

### 3.2. OIDC Discovery & JWKS (Giao tiếp xác thực offline)
* **Chiều gọi:** `API Gateway` & `identity-service` $\longrightarrow$ `Keycloak Public JWKS Endpoint`.
* **Endpoint:** `GET /realms/{realm}/protocol/openid-connect/certs`.
* **Cơ chế:**
  - Khi khởi động, Backend tải bộ Public Key của Keycloak về lưu trong RAM Cache.
  - Mỗi khi nhận request có Bearer JWT, Backend tự kiểm tra chữ ký số mà **không cần gửi HTTP request sang Keycloak**, đảm bảo độ trễ $< 0.1$ms.

### 3.3. Token Claim Injection (Custom Protocol Mapper)
* Khi Keycloak phát hành Access Token, thông qua cấu hình **Protocol Mapper** trong file `realm-export.template.json`, Keycloak sẽ tự động nhúng thuộc tính nội bộ vào Payload JWT:
  ```json
  {
    "iss": "https://auth.omama.vn/realms/omama",
    "sub": "b2c1d3e4-keycloak-subject-uuid",
    "user_id": "a1b2c3d4-internal-principal-uuid", // <-- Mapper lấy từ attribute app_user_id
    "email": "duong@omama.vn",
    "roles": ["EMPLOYEE"]
  }
  ```
  Nhờ đó, các service downstream chỉ cần đọc claim `user_id` là có ngay UUID nội bộ của hệ thống.

---

## 4. Ranh Giới Trong Các Luồng Nghiệp Vụ Cốt Lõi

### 4.1. Luồng Đăng ký tài khoản Khách hàng (Customer Registration)
```
[Client] ──> (1) POST /registration/customer ──> [identity-service]
                                                        │
                      ┌─────────────────────────────────┴─────────────────────────────────┐
                      ▼                                                                   ▼
       (2) Tạo User qua Admin API                                          (3) Ghi DB nội bộ
             [Keycloak]                                                  [PostgreSQL identity_db]
       - Lưu credentials an toàn                                         - Lưu principal_id (UUID)
       - Gắn app_user_id = principal_id                                  - Lưu idempotency_key
       - Trả về subject_id                                               - Lưu status = ACTIVE
                                                                         - Ghi Outbox: UserRegistered
                                                                                          │
                                                                                          ▼
                                                                           (4) Kafka phát event
                                                                            ──> [Profile Service]
```
* **Identity Service điều phối**: Kiểm tra tính hợp lệ của request, chống gửi trùng bằng `idempotency_key`, tạo `principal_id`.
* **Keycloak thực hiện**: Tạo user, băm mật khẩu, kích hoạt tài khoản OIDC.
* **Database nội bộ lưu**: Ánh xạ `principal_id` với `subject_id` của Keycloak.

### 4.2. Luồng Đăng nhập (Authentication)
* **100% thuộc trách nhiệm của Keycloak**:
  - Client tương tác trực tiếp với Keycloak qua OIDC Authorization Code + PKCE.
  - `identity-service` **hoàn toàn đứng ngoài luồng đăng nhập**. Không nhận username/password, không sinh JWT token, không lưu session.

### 4.3. Luồng Khóa / Mở khóa tài khoản (Account Suspension)
* **Khi Admin gọi API khóa tài khoản:**
  1. `identity-service` nhận lệnh $\rightarrow$ cập nhật `provisioning_status = 'SUSPENDED'` trong `identity_principals`.
  2. Ghi vết kiểm toán vào `identity_audit_log` (`SUSPEND_ACCOUNT`).
  3. Ghi sự kiện vào bảng Outbox (`UserSuspended`).
  4. Gọi Keycloak Admin API: `setUserEnabled(subjectId, false)`.
  5. **Kết quả**: Keycloak chặn ngay lập tức không cho user này đăng nhập hoặc refresh token.

### 4.4. Luồng Phân quyền Scope & Quyết định rủi ro cao (Authorization & ABAC)
* **100% thuộc trách nhiệm của `identity-service`**:
  - Keycloak **không biết gì** về khái niệm Cửa hàng Huế (`STORE_01`) hay Kho Đà Nẵng (`WH_01`).
  - Keycloak cũng **không biết** quy tắc "Chỉ quản lý mới được duyệt hoàn tiền $> 5$ triệu".
  - Toàn bộ logic Scoped IAM và kiểm tra hạn mức tiền tệ đều nằm trong `identity-service` (được gọi qua gRPC `CheckSpecializedPermission`).

### 4.5. Luồng Quản trị Role & Permission (Role Lifecycle & Keycloak Sync)
* **Identity Service là Single Source of Truth**:
  - Toàn bộ Role (`iam_roles`), Permission (`iam_permissions`) và quan hệ Role - Permission (`iam_role_permissions`) được quản lý toàn vẹn trong PostgreSQL của `identity-service`.
  - Cơ chế an toàn ngăn chặn xóa nhầm:
    1. Chặn xóa các vai trò hệ thống được bảo vệ (`is_protected = true` như `SUPER_ADMIN`).
    2. Chặn xóa vai trò nếu đang có nhân viên được gán active trong `iam_role_assignments`.
* **Đồng bộ tự động sang Keycloak**:
  - Khi Admin tạo Role mới (`POST /api/v1/admin/roles`) hoặc cập nhật thông tin (`PATCH /api/v1/admin/roles/{id}`), `identity-service` tự động gọi `KeycloakAdminClient` để tạo/sửa Realm Role tương ứng trên Keycloak.
  - Khi xóa Role hợp lệ, `identity-service` cũng đồng thời gỡ bỏ Realm Role trên Keycloak để giữ tính nhất quán.

---

## 5. Xử Lý Bất Đồng Bộ & Tính Nhất Quán Phân Tán (Saga Pattern)

Vì **Keycloak** (chạy service riêng với database `keycloak_db`) và **`identity-service`** (dùng `identity_db`) nằm ở 2 hệ thống khác nhau, chúng **KHÔNG THỂ tham gia vào cùng 1 ACID transaction**.

Hệ thống giải quyết bài toán "ghi dữ liệu phân tán" (Dual-write problem) bằng mô hình **Idempotent Saga**:

1. **Giai đoạn 1 (DB Local):** `identity-service` tạo bản ghi `identity_principals` ở trạng thái `provisioning_status = 'PENDING'` kèm bản ghi `registration_operations` trong 1 local transaction.
2. **Giai đoạn 2 (External Call):** Gọi Keycloak Admin API để tạo user.
   - Nếu Keycloak báo lỗi (trùng email, server timeout) $\rightarrow$ Đánh dấu operation `FAILED`.
3. **Giai đoạn 3 (Finalize & Outbox):** Khi Keycloak tạo thành công và trả về `subject_id`:
   - Mở local transaction mới: Cập nhật `provisioning_status = 'ACTIVE'`, gán `idp_subject`, ghi sự kiện `UserRegistered` vào bảng `identity_outbox`.
4. **Reconciliation (Đối soát định kỳ):**
   - Một background job sẽ định kỳ quét các bản ghi có trạng thái `PENDING` quá 5 phút để đối chiếu với Keycloak (nếu Keycloak đã có user thì kích hoạt tiếp, nếu Keycloak chưa có thì dọn dẹp).

---

## 6. Những Điều TUYỆT ĐỐI KHÔNG LÀM (Anti-patterns & Boundary Violations)

Để đảm bảo tính toàn vẹn kiến trúc, lập trình viên cần tuân thủ nghiêm ngặt các điều cấm sau:

| Điều cấm | Lý do vi phạm |
| :--- | :--- |
| ❌ **Tự tạo bảng `users` chứa cột `password_hash` trong `identity_db`** | Vi phạm Single Source of Truth. Keycloak là nơi duy nhất quản lý credentials. |
| ❌ **Tạo API `/login` ở `identity-service` nhận pass rồi tự phát JWT** | Phá vỡ chuẩn OIDC và SSO. Mất toàn bộ tính năng bảo vệ phiên, MFA và Brute-force của Keycloak. |
| ❌ **Tạo tài khoản Keycloak cho khách vãng lai (Guest)** | Gây rác database Keycloak. Khách vãng lai thuộc sở hữu của Cart/Checkout/Order domain. |
| ❌ **Lưu Scope chi nhánh (Store/Warehouse) vào Role của Keycloak** | Keycloak chỉ nên quản lý vai trò thô (`CUSTOMER`, `EMPLOYEE`). Không biến Keycloak thành nơi lưu cấu trúc tổ chức chi nhánh thay đổi liên tục của doanh nghiệp. |
| ❌ **Gọi `identity-service` qua mạng cho mọi request thông thường** | Gây nghẽn cổ chai (Bottleneck). API Gateway và Backend chỉ cần tự verify JWT bằng cached JWKS là đủ. |

