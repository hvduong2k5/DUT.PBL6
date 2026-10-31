# Identity Service — API Contract Decisions

## Status của tài liệu này
Đây là tài liệu chốt các quyết định thiết kế API cho identity-service. Mỗi quyết định có trạng thái: **CHỐT**, **DRAFT** hoặc **DEPRECATED**.
Cập nhật tuân thủ **Mục 3.3 & 3.3.1: Áp dụng thiết kế RESTful API** (Bài giảng Thiết kế Hệ thống & Phân tích Nghiệp vụ).

---

## 1. Nguyên Tắc Thiết Kế RESTful API (Mục 3.3.1)

Dịch vụ tuân thủ 4 quy tắc cốt lõi:
1. **Quy tắc 1: Sử dụng Danh từ (Nouns), KHÔNG dùng Động từ (Verbs)**:
   - Bản thân các HTTP Method (`GET`, `POST`, `PUT`, `PATCH`, `DELETE`) đã là động từ. URI chỉ biểu diễn tài nguyên.
   - Ví dụ: `PATCH /api/v1/admin/users/{id}/status` thay cho `POST /suspend` hay `POST /reactivate`.
   - Ví dụ: `DELETE /api/v1/admin/employee-invitations/{id}` thay cho `POST /revoke`.
2. **Quy tắc 2: Luôn sử dụng Danh từ số nhiều (Plural nouns)**:
   - Luôn dùng `users`, `roles`, `permissions`, `employee-invitations`, `role-assignments`, `customer-registrations`.
3. **Quy tắc 3: Thể hiện mối quan hệ phân cấp (Hierarchy)**:
   - Quyền hạn thuộc vai trò: `/api/v1/admin/roles/{roleId}/permissions/{permissionCode}`.
   - Trạng thái thuộc người dùng: `/api/v1/admin/users/{principalId}/status`.
4. **Quy tắc 4: Sử dụng dấu gạch ngang (kebab-case) cho các từ dài**:
   - Dùng `employee-invitations`, `role-assignments`, `customer-registrations`.

---

## 2. Luồng xác thực (Authentication Flow)

### CHỐT — OIDC Authorization Code + PKCE
- Frontend (Web/Mobile) dùng **OIDC Authorization Code + PKCE** thông qua Keycloak.
- identity-service **không** nhận `username/password` rồi tự phát JWT.
- identity-service **không** implement OAuth Resource Owner Password Credentials (ROPC).

### CHỐT — Xác thực tại Gateway
- API Gateway xác minh JWT bằng `issuer`, `audience`, `signature`, `expiry` và JWKS cache cục bộ.
- Gateway **không** gọi identity-service đồng bộ cho mọi request thông thường.
- Chỉ các quyết định rủi ro cao (ví dụ: hoàn tiền lớn, giảm giá B2B) mới gọi `CheckSpecializedPermission` qua gRPC.

---

## 3. REST API (Customer-facing)

### POST /api/v1/public/customer-registrations — CHỐT (RESTful 3.3.1)
*(Hỗ trợ tương thích ngược alias: `POST /api/v1/public/registration/customer`)*

**Mô tả**: Đăng ký tài khoản khách hàng mới. Idempotent theo `idempotencyKey`.

**Request:**
```json
{
  "email": "user@example.com",
  "firstName": "Nguyen",
  "lastName": "Van A",
  "rawPassword": "SecurePass123!",
  "idempotencyKey": "a2f4e5c0-1234-...",
  "requestFingerprint": "sha256-hex-of-body-and-client"
}
```

**Response 200 (OK / idempotent replay):**
```json
{
  "success": true,
  "code": "SUCCESS",
  "message": "Đăng ký tài khoản thành công",
  "data": {
    "principalId": "uuid",
    "status": "ACTIVE",
    "message": "Đăng ký tài khoản thành công"
  },
  "timestamp": "2026-10-31T10:30:00Z"
}
```

**Response 409 (Conflict — idempotency key tái sử dụng với payload khác):**
```json
{
  "success": false,
  "code": "STATE_CONFLICT",
  "message": "Idempotency key reused with different payload",
  "timestamp": "2026-10-31T10:30:00Z"
}
```

**Quyết định bảo mật**:
- `rawPassword` KHÔNG được log, lưu cache hoặc truyền ra ngoài Keycloak Admin API.
- `requestFingerprint` phải được tạo phía server (HMAC-SHA256 với secret), KHÔNG tin fingerprint từ client.

---

## 4. REST API (User Lifecycle & Administration)

### PATCH /api/v1/admin/users/{principalId}/status — CHỐT (RESTful 3.3.1)
*(Hỗ trợ tương thích ngược: `POST .../suspend`, `POST .../reactivate`)*

**Mô tả**: Cập nhật trạng thái tài khoản người dùng (ACTIVE, SUSPENDED). Áp dụng Quy tắc 1 (dùng HTTP PATCH trên noun `status` thay cho verbs `suspend`/`reactivate`).

**Request Body:**
```json
{
  "status": "SUSPENDED",
  "reason": "Tài khoản có dấu hiệu vi phạm chính sách khuyến mãi"
}
```

---

## 5. REST API (Employee Invitations)

### POST /api/v1/admin/employee-invitations — CHỐT (RESTful 3.3.1)
*(Hỗ trợ alias: `POST /api/v1/admin/employees/invitations`)*

**Mô tả**: Tạo thư mời nhân viên mới kèm mã kích hoạt.

### DELETE /api/v1/admin/employee-invitations/{invitationId} — CHỐT (RESTful 3.3.1)
*(Hỗ trợ alias: `POST .../{invitationId}/revoke`)*

**Mô tả**: Thu hồi thư mời nhân viên bằng phương thức HTTP DELETE trên URI danh từ số nhiều (Quy tắc 1, 2, 4).

---

## 6. REST API (IAM Role Assignments)

### POST /api/v1/admin/role-assignments — CHỐT (RESTful 3.3.1)
*(Hỗ trợ alias: `POST /api/v1/admin/iam/assignments`)*

**Mô tả**: Gán role nhân viên có scope và thời hạn.

**Request:**
```json
{
  "principalId": "uuid",
  "roleId": "uuid",
  "scopeType": "GLOBAL | WAREHOUSE | STORE | REGION | DEPARTMENT | BRANCH | BUSINESS_UNIT",
  "scopeId": "warehouse-hcm-01",
  "validFrom": "2026-10-01T00:00:00Z",
  "validUntil": "2026-12-31T23:59:59Z",
  "reason": "Bổ nhiệm trưởng kho Q4 2026"
}
```

### DELETE /api/v1/admin/role-assignments/{assignmentId} — CHỐT (RESTful 3.3.1)
**Mô tả**: Thu hồi phân công vai trò nhân viên bằng HTTP DELETE.

---

## 7. REST API (Role & Permission Management)

### CHỐT — Bộ API Quản Lý Role (Role Lifecycle)

| Method | Endpoint | Chức năng | Mô tả & Ràng buộc |
| :--- | :--- | :--- | :--- |
| `GET` | `/api/v1/admin/roles` | Danh sách Role | Lấy toàn bộ danh sách Role hệ thống. |
| `GET` | `/api/v1/admin/roles/{roleId}` | Chi tiết Role | Xem thông tin chi tiết một Role kèm danh sách Permissions. |
| `POST` | `/api/v1/admin/roles` | Tạo Role mới | Tạo Role với `roleCode`, `roleName`, `description`. Đồng bộ tạo Realm Role trên Keycloak. |
| `PATCH`| `/api/v1/admin/roles/{roleId}` | Cập nhật Role | Cập nhật `roleName`, `description`, `isEnabled`. Áp dụng Optimistic Concurrency Control (`version`). |
| `DELETE`| `/api/v1/admin/roles/{roleId}`| Xóa Role | **Cơ chế an toàn**:<br>1. Từ chối xóa nếu `is_protected = true` (ví dụ `SUPER_ADMIN`).<br>2. Từ chối xóa nếu đang có nhân viên được gán active trong `iam_role_assignments`.<br>3. Tự động xóa liên kết permission con và đồng bộ xóa trên Keycloak. |

---

### CHỐT — Bộ API Quản Lý Permission Của Role (Quan hệ phân cấp - Quy tắc 3)

| Method | Endpoint | Chức năng | Mô tả & Ràng buộc |
| :--- | :--- | :--- | :--- |
| `GET` | `/api/v1/admin/roles/{roleId}/permissions` | Danh sách Permission | Lấy danh sách mã quyền (`permission_code`) thuộc Role. |
| `PUT` | `/api/v1/admin/roles/{roleId}/permissions` | Thay thế Permissions | Thay thế toàn bộ quyền của Role bằng danh sách mới (Transactional replace). |
| `POST`| `/api/v1/admin/roles/{roleId}/permissions` | Gán thêm Permission | Thêm 1 quyền vào Role (Idempotent insert). |
| `DELETE`| `/api/v1/admin/roles/{roleId}/permissions/{permissionCode}` | Gỡ Permission | Gỡ một quyền cụ thể khỏi Role. |

*Lưu ý bảo mật: Mọi thao tác Create, Update, Delete Role và Permission đều được ghi nhật ký kiểm toán không thể sửa xóa vào `identity_audit_log`.*

---

## 8. gRPC API

### CheckSpecializedPermission — CHỐT (dựa trên packages/proto/identity/v1/identity.proto)

**Mô tả**: Kiểm tra quyền đặc biệt cho các thao tác rủi ro cao (ví dụ: hoàn tiền > 5 triệu, giảm giá B2B).

**Quy tắc**:
- Chỉ implement các RPC được định nghĩa trong shared proto. **Không tự viết** `ValidateToken` hay `CheckPermission` nếu chưa có trong proto.
- Phải có deadline/timeout. Nếu timeout → **fail closed** (từ chối quyền).
- `context_attributes` phải được validate schema; không thực thi expression/SpEL từ JSON.
- Mọi decision đều ghi vào `authorization_decisions` và `identity_audit_log`.

### GetJwksPublicKey — DEPRECATED (Kế hoạch xóa bỏ)

> **Lưu ý**: Gateway nên đọc JWKS trực tiếp từ Keycloak OIDC Discovery endpoint thay vì gọi RPC này.

---

## 9. Kafka Events

### identity.events.v1.UserRegistered — DRAFT (Chưa publish production)
Envelope CloudEvents 1.0 với `principalId`, `email`, `firstName`, `lastName`.

### identity.events.v1.UserSuspended — DRAFT
Envelope CloudEvents 1.0 với `principalId`, `reason`, `suspendedAt`.

---

## 10. Token Claims

| Claim | Giá trị | Ghi chú |
|---|---|---|
| `iss` | `https://keycloak.omama.vn/realms/omama` | Realm issuer cố định |
| `sub` | Keycloak external subject UUID | Do Keycloak quản lý |
| `user_id` | `principal_id` nội bộ UUID | Đọc từ attribute `app_user_id` qua Protocol Mapper |
| `aud` | `omama-api` | Audience cụ thể, không dùng chung |
| `exp` | `iat + 900` (15 phút) | **CHỐT**: Access Token TTL = 15 phút |
| `roles` | `CUSTOMER` / `EMPLOYEE` | Coarse-grained roles từ Keycloak |
