# Identity Service — API Contract Decisions

## Status của tài liệu này
Đây là tài liệu chốt các quyết định thiết kế API cho identity-service. Mỗi quyết định có trạng thái: **CHỐT**, **DRAFT** hoặc **DEPRECATED**.

---

## 1. Luồng xác thực (Authentication Flow)

### CHỐT — OIDC Authorization Code + PKCE
- Frontend (Web/Mobile) dùng **OIDC Authorization Code + PKCE** thông qua Keycloak.
- identity-service **không** nhận `username/password` rồi tự phát JWT.
- identity-service **không** implement OAuth Resource Owner Password Credentials (ROPC).

### CHỐT — Xác thực tại Gateway
- API Gateway xác minh JWT bằng `issuer`, `audience`, `signature`, `expiry` và JWKS cache cục bộ.
- Gateway **không** gọi identity-service đồng bộ cho mọi request thông thường.
- Chỉ các quyết định rủi ro cao (ví dụ: hoàn tiền lớn, giảm giá B2B) mới gọi `CheckSpecializedPermission` qua gRPC.

---

## 2. REST API (Customer-facing)

### POST /api/v1/public/registration/customer — CHỐT

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
  "principalId": "uuid",
  "status": "ACTIVE",
  "message": "User registered successfully"
}
```

**Response 409 (Conflict — idempotency key tái sử dụng với payload khác):**
```json
{
  "principalId": null,
  "status": "CONFLICT",
  "message": "Idempotency key reused with different payload"
}
```

**Quyết định bảo mật**:
- `rawPassword` KHÔNG được log, lưu cache hoặc truyền ra ngoài Keycloak Admin API.
- `requestFingerprint` phải được tạo phía server (HMAC-SHA256 với secret), KHÔNG tin fingerprint từ client.

---

## 3. REST API (Admin-facing)

### POST /api/v1/admin/iam/assignments — DRAFT (Cần chốt contract ổn định trước khi mở Frontend)

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

**Ràng buộc**:
- `principal` phải có `principal_type = EMPLOYEE`.
- Nếu `scopeType != GLOBAL` thì `scopeId` là bắt buộc.
- Không cho phép tự gán quyền cho chính mình.
- Không cho phép thu hồi `System Administrator` cuối cùng.
- Mọi thay đổi phải được ghi vào `identity_audit_log`.

---

## 4. gRPC API

### CheckSpecializedPermission — CHỐT (dựa trên packages/proto/identity/v1/identity.proto)

**Mô tả**: Kiểm tra quyền đặc biệt cho các thao tác rủi ro cao (ví dụ: hoàn tiền > 5 triệu, giảm giá B2B).

**Quy tắc**:
- Chỉ implement các RPC được định nghĩa trong shared proto. **Không tự viết** `ValidateToken` hay `CheckPermission` nếu chưa có trong proto.
- Phải có deadline/timeout. Nếu timeout → **fail closed** (từ chối quyền).
- `context_attributes` phải được validate schema; không thực thi expression/SpEL từ JSON.
- Mọi decision đều ghi vào `authorization_decisions` và `identity_audit_log`.

### GetJwksPublicKey — DEPRECATED (Kế hoạch xóa bỏ)

> **Lưu ý**: Gateway nên đọc JWKS trực tiếp từ Keycloak OIDC Discovery endpoint thay vì gọi RPC này.
> Nếu quyết định xóa khỏi proto, cần update shared proto, tài liệu API và service boundary trong cùng một PR.

---

## 5. Kafka Events

### identity.events.v1.UserRegistered — DRAFT (Chưa publish production)

**Điều kiện publish**: Chỉ phát sau khi schema versioned tại `packages/events` được phê duyệt.

**Envelope (CloudEvents 1.0)**:
```json
{
  "specversion": "1.0",
  "type": "identity.events.v1.UserRegistered",
  "source": "identity-service",
  "id": "uuid-event-id",
  "time": "2026-10-11T00:00:00Z",
  "datacontenttype": "application/json",
  "data": {
    "principalId": "uuid",
    "email": "user@example.com",
    "firstName": "Nguyen",
    "lastName": "Van A"
  }
}
```

**PII Policy**: `email` là PII. Chỉ đưa vào event nếu consumer đã được phê duyệt nhận PII. Cân nhắc dùng `principalId` và để consumer tự query Profile Service.

### identity.events.v1.UserDeactivated — DRAFT
### identity.events.v1.PasswordChanged — DRAFT

> Không phát các event này cho đến khi consumer contract và PII policy được chốt.

---

## 6. Token Claims

| Claim | Giá trị | Ghi chú |
|---|---|---|
| `iss` | `https://keycloak.omama.vn/realms/omama` | Realm issuer cố định |
| `sub` | Keycloak external subject UUID | Do Keycloak quản lý |
| `user_id` | `principal_id` nội bộ UUID | Đọc từ attribute `app_user_id` qua Protocol Mapper |
| `aud` | `omama-api` | Audience cụ thể, không dùng chung |
| `exp` | `iat + 900` (15 phút) | **CHỐT**: Access Token TTL = 15 phút |
| `roles` | `CUSTOMER` / `EMPLOYEE` | Coarse-grained roles từ Keycloak |

> **Mâu thuẫn đã giải quyết**: Access Token TTL chốt là **15 phút** (không phải 1 giờ). Mọi config, test và tài liệu phải sử dụng giá trị này.

---

## 7. Những gì KHÔNG được triển khai

| Hành động | Lý do |
|---|---|
| API nhận `username/password` và phát JWT | Vi phạm kiến trúc OIDC; Keycloak là nguồn chân lý |
| Bảng `refresh_tokens`, `sessions`, `password_hash` | Keycloak đã sở hữu |
| Guest identity trong identity_db | Cart/Checkout/Order sở hữu Guest session |
| `ValidateToken` gRPC | Không có trong shared proto chuẩn |
| Toàn bộ admin REST trước khi draft OpenAPI được chốt | Contract phải stable trước khi mở Frontend |
