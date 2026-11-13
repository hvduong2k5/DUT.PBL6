# Identity 3.3 — RESTful API Integration Contract & Semantic Rules

**Ngày cập nhật:** 31/10/2026  
**Trạng thái:** Integration Contract — Target MVP 3.3  
**Tài liệu quy chuẩn:** Bài giảng Thiết kế Hệ thống & Phân tích Nghiệp vụ — **Mục 3.3 & 3.3.1: Áp dụng thiết kế RESTful API**  
**OpenAPI Spec tương ứng:** [`docs/03_api_specs/admin-stable.openapi.yaml`](../../docs/03_api_specs/admin-stable.openapi.yaml), [`docs/03_api_specs/customer-stable.openapi.yaml`](../../docs/03_api_specs/customer-stable.openapi.yaml)  
**Proto gRPC:** [`packages/proto/identity/v1/identity.proto`](../proto/identity/v1/identity.proto)

---

## 1. Bốn Nguyên Tắc Thiết Kế RESTful URI (Mục 3.3.1)

Mọi endpoint REST của `identity-service` phải tuân thủ nghiêm ngặt 4 quy tắc sau:

| Quy tắc | Nội dung quy chuẩn | Sai (RPC / Non-REST) ❌ | Đúng (Chuẩn RESTful 3.3.1) ✅ |
| :--- | :--- | :--- | :--- |
| **Quy tắc 1** | **Sử dụng Danh từ (Nouns), KHÔNG dùng Động từ (Verbs)**.<br>Bản thân các HTTP Method (`GET`, `POST`, `PUT`, `PATCH`, `DELETE`) đã là động từ, URI chỉ chứa danh từ tài nguyên. | `POST /admin/users/{id}/suspend`<br>`POST /admin/users/{id}/reactivate`<br>`POST /invitations/{id}/revoke` | `PATCH /api/v1/admin/users/{id}/status`<br>`DELETE /api/v1/admin/employee-invitations/{id}`<br>`DELETE /api/v1/admin/role-assignments/{id}` |
| **Quy tắc 2** | **Luôn sử dụng Danh từ số nhiều (Plural nouns)**.<br>Giữ tính nhất quán dù truy xuất một danh sách hay một phần tử. | `POST /public/registration/customer`<br>`POST /admin/employee/invitation` | `POST /api/v1/public/customer-registrations`<br>`POST /api/v1/admin/employee-invitations` |
| **Quy tắc 3** | **Thể hiện mối quan hệ phân cấp (Hierarchy)**.<br>Khi một tài nguyên phụ thuộc vào tài nguyên khác, lồng ghép vào URI theo cấu trúc cha/con. | `GET /getRolePermissions?roleId=12`<br>`DELETE /removePermission` | `GET /api/v1/admin/roles/{roleId}/permissions`<br>`DELETE /api/v1/admin/roles/{roleId}/permissions/{code}`<br>`PATCH /api/v1/admin/users/{principalId}/status` |
| **Quy tắc 4** | **Sử dụng dấu gạch ngang (kebab-case) cho từ dài**.<br>Không dùng `snake_case` hoặc `camelCase` trên URL. | `/admin/role_assignments`<br>`/admin/employeeInvitations` | `/api/v1/admin/role-assignments`<br>`/api/v1/admin/employee-invitations`<br>`/api/v1/public/customer-registrations` |

---

## 2. Bảng Ánh Xạ Endpoint Toàn Diện (Canonical URI Matrix)

| Phân hệ / Nghiệp vụ | HTTP Method | Endpoint URI (Chuẩn RESTful 3.3.1) | Mô tả hành động & Quy tắc áp dụng |
| :--- | :---: | :--- | :--- |
| **User Profile** | `GET` | `/api/v1/users/me` | Lấy hồ sơ người dùng đang đăng nhập (Quy tắc 1, 2). |
| | `PUT` | `/api/v1/users/me/password` | Người dùng tự đổi mật khẩu cá nhân (xác thực mật khẩu cũ) (Quy tắc 1, 2). |
| **Admin Users (ADMIN)** | `GET` | `/api/v1/admin/users` | Lấy danh sách người dùng phân trang & bộ lọc (`type`, `status`) (Quy tắc 1, 2). |
| | `GET` | `/api/v1/admin/users/{principalId}` | Xem chi tiết hồ sơ người dùng kèm vai trò và quyền hạn (Quy tắc 1, 2). |
| | `PATCH` | `/api/v1/admin/users/{principalId}/status` | Cập nhật trạng thái tài khoản (`ACTIVE`, `SUSPENDED`). Dùng `PATCH` thay vì động từ trong URI (Quy tắc 1, 3). |
| | `POST` | `/api/v1/admin/users/{principalId}/password/reset` | Super Admin đặt lại mật khẩu cho người dùng (Quy tắc 1, 3). |
| | `POST` | `/api/v1/admin/users/{principalId}/sessions/revoke` | Super Admin thu hồi phiên làm việc, buộc đăng xuất (Quy tắc 1, 3). |
| **Employee Invitations** | `POST` | `/api/v1/admin/employee-invitations` | Tạo thư mời nhân viên mới với token có hạn (Quy tắc 2, 4). |
| | `DELETE`| `/api/v1/admin/employee-invitations/{invitationId}` | Thu hồi thư mời nhân viên. Dùng `DELETE` method thay cho verb `revoke` (Quy tắc 1, 4). |
| **Role Assignments** | `POST` | `/api/v1/admin/role-assignments` | Gán vai trò chức danh cho nhân viên có phạm vi nghiệp vụ (Quy tắc 2, 4). |
| | `DELETE`| `/api/v1/admin/role-assignments/{assignmentId}` | Thu hồi vai trò đã gán của nhân viên (Quy tắc 1, 4). |
| **Roles** | `GET` | `/api/v1/admin/roles` | Lấy danh sách vai trò hệ thống (Quy tắc 1, 2). |
| | `POST` | `/api/v1/admin/roles` | Tạo vai trò mới và đồng bộ Realm Role sang Keycloak (Quy tắc 1, 2). |
| | `GET` | `/api/v1/admin/roles/{roleId}` | Xem chi tiết vai trò (Quy tắc 1, 2). |
| | `PATCH`| `/api/v1/admin/roles/{roleId}` | Cập nhật tên, mô tả hoặc cờ kích hoạt vai trò (Quy tắc 1). |
| | `DELETE`| `/api/v1/admin/roles/{roleId}` | Xóa vai trò (chặn xóa vai trò bảo vệ hoặc đang có nhân viên gán) (Quy tắc 1). |
| **Role Permissions** | `GET` | `/api/v1/admin/roles/{roleId}/permissions` | Xem danh sách quyền hạn của vai trò (Quy tắc 3). |
| | `PUT` | `/api/v1/admin/roles/{roleId}/permissions` | Ghi đè toàn bộ danh sách quyền của vai trò (Batch replace) (Quy tắc 3). |
| | `POST` | `/api/v1/admin/roles/{roleId}/permissions` | Gán thêm một quyền vào vai trò (Quy tắc 3). |
| | `DELETE`| `/api/v1/admin/roles/{roleId}/permissions/{permissionCode}` | Gỡ một quyền khỏi vai trò (Quy tắc 3). |
| **Customer Registration**| `POST` | `/api/v1/public/customer-registrations` | Đăng ký tài khoản Khách hàng D2C với Idempotency Key (Quy tắc 1, 2, 4). |
| **Authentication & BFF Sessions**| `GET` | `/api/v1/public/auth/login` | Khởi tạo đăng nhập OIDC PKCE (Dual-Mode):<br>• **Web Browser:** Trả về `HTTP 302 Found` (Redirect) đưa trình duyệt bay thẳng sang Keycloak.<br>• **Mobile App / API:** Nhận `?format=json` hoặc Header `Accept: application/json` để nhận `HTTP 200 OK` JSON chứa `authorizationUrl` (RFC 9700 & RFC 7636). |
| | `GET` | `/api/v1/public/auth/callback` | Tiếp nhận callback từ Keycloak, đổi code lấy tokens qua backchannel, cấp HttpOnly Cookie OMAMA_SESSION, redirect 302 về Frontend. |
| | `GET` | `/api/v1/public/auth/session` | Kiểm tra phiên làm việc hiện tại từ HttpOnly Cookie OMAMA_SESSION mà không làm lộ token ra trình duyệt. |
| | `POST` | `/api/v1/public/auth/tokens/refresh` | Gia hạn phiên làm việc BFF bằng Refresh Token lưu phía server. |
| | `POST` | `/api/v1/public/auth/logout` | Đăng xuất, thu hồi Token trên Keycloak và xóa Cookie OMAMA_SESSION. |

---

## 3. Cấu Trúc Data Payload (JSON) Chuẩn

### 3.1. Envelope Phản Hồi Thành Công (Success Envelope)
Tất cả response thành công đều bọc trong cấu trúc chuẩn:
```json
{
  "success": true,
  "code": "SUCCESS",
  "message": "Thao tác thành công",
  "data": { ... },
  "timestamp": "2026-10-31T10:30:00Z"
}
```

### 3.2. Envelope Phản Hồi Lỗi (Error Envelope)
```json
{
  "success": false,
  "code": "RESOURCE_NOT_FOUND",
  "message": "Không tìm thấy thông tin tài nguyên",
  "data": null,
  "errors": [
    {
      "field": "principalId",
      "code": "INVALID_UUID",
      "message": "Định dạng UUID không hợp lệ"
    }
  ],
  "requestId": "req-987654",
  "timestamp": "2026-10-31T10:30:00Z"
}
```

### 3.3. Quy Chuẩn Payload Cập Nhật Trạng Thái (Status Update)
Thay vì dùng RPC endpoint như `/suspend`, client gửi payload biểu diễn trạng thái mong muốn:
```json
// PATCH /api/v1/admin/users/{principalId}/status
{
  "status": "SUSPENDED",
  "reason": "Tài khoản có dấu hiệu gian lận khuyến mãi"
}
```
Hoặc để mở khóa:
```json
// PATCH /api/v1/admin/users/{principalId}/status
{
  "status": "ACTIVE",
  "reason": "Đã hoàn thành xác minh danh tính"
}
```

### 3.4. Cơ Chế Dual-Mode Đăng Nhập (Web 302 Redirect vs Mobile JSON)
Endpoint `GET /api/v1/public/auth/login` hỗ trợ cả 2 nền tảng:

1. **Dành cho Trình duyệt Web (SPA, Next.js, React):**
   - Người dùng click thẻ `<a href="/api/v1/public/auth/login?redirect_uri=...">` hoặc gọi `window.location.href = ...`.
   - Trình duyệt gửi header `Accept: text/html...`.
   - Backend phản hồi `HTTP 302 Found` kèm header `Location: <keycloak-auth-url>`. Trình duyệt lập tức chuyển tiếp sang màn hình đăng nhập Keycloak.

2. **Dành cho Mobile App (Flutter, React Native) hoặc API Client:**
   - App gọi qua HTTP Client với `?format=json` hoặc header `Accept: application/json`.
   - Backend phản hồi `HTTP 200 OK` với body JSON:
     ```json
     {
       "success": true,
       "code": "SUCCESS",
       "message": "Khởi tạo đăng nhập thành công",
       "data": {
         "authorizationUrl": "http://localhost:8080/realms/omama/protocol/openid-connect/auth?...",
         "state": "random-state-string"
       }
     }
     ```
   - Mobile App dùng `authorizationUrl` để mở trình duyệt an toàn của hệ điều hành (`ASWebAuthenticationSession` / `CustomTabs`) và tiếp nhận callback qua Deep Link (`omama://oauth/callback`).

---

## 4. Tích Hợp Đa Giao Thức (Cross-Protocol Integration)

```
                            ┌──────────────────────────────────────────────┐
                            │      CLIENTS / GATEWAY / PEER SERVICES       │
                            └───────┬──────────────┬───────────────┬───────┘
                                    │              │               │
                         REST (3.3) │   gRPC (v1)  │    Events (v1)│
                                    ▼              ▼               ▼
                        ┌──────────────────────────────────────────────────┐
                        │                 IDENTITY SERVICE                 │
                        │   - REST: User, Role, Assignment, Registration   │
                        │   - gRPC: CheckSpecializedPermission (9016)      │
                        │   - Outbox: UserRegistered, UserSuspended (Kafka)│
                        └──────────────────────────────────────────────────┘
```

1. **REST API (Port 8016):**
   - Phục vụ ứng dụng Web Admin Portal, Mobile Client và Public Gateway.
   - Định dạng JSON theo cấu trúc `ApiResponse<T>`, OpenAPI 3.0.3.
2. **gRPC Protocol (Port 9016 - `omamx.identity.v1.IdentityService`):**
   - Chỉ dùng cho các quyết định phân quyền động có rủi ro cao:
     `CheckSpecializedPermission(SpecializedPermissionRequest) returns (SpecializedPermissionResponse)`
   - Không lạm dụng gRPC để kiểm tra token thường lệ (Gateway tự xác thực JWT qua JWKS cache).
3. **Outbox Domain Events (Kafka):**
   - Phát sự kiện `identity.events.v1.UserRegistered` và `UserSuspended` theo chuẩn CloudEvents 1.0.

---

## 5. Quy Tắc An Toàn & Phân Quyền Quản Trị (IAM Guardrails)

Hệ thống thiết lập vai trò duy nhất có toàn quyền quản trị tài khoản và phân quyền là **`ADMIN`**, đồng thời kích hoạt 4 quy tắc an toàn cốt lõi:

1. **Rule 1 (Self-Protection):** Chặn `ADMIN` tự khóa tài khoản của chính mình (`actorId == principalId` hoặc so khớp Keycloak subject).
2. **Rule 2 (Super Admin Protection):** Chặn mọi thao tác khóa tài khoản đối với người dùng đang nắm giữ vai trò `ADMIN`.
3. **Rule 3 (Role & Permission Protection):**
   - Chặn xóa hoặc vô hiệu hóa (`isEnabled = false`) các vai trò hệ thống được bảo vệ (`is_protected = true`).
   - Chặn thêm, sửa, hoặc gỡ quyền hạn của vai trò được bảo vệ (`is_protected = true`).
   - Chặn xóa vai trò nếu đang có người dùng được gán còn hiệu lực (`activeAssignments > 0`).
4. **Rule 4 (RBAC Super Admin Enforcement):**
   - Chỉ người dùng có vai trò `ADMIN` mới có thể gọi các API quản trị người dùng (`/api/v1/admin/users/**`). Vi phạm sẽ bị từ chối với mã lỗi `403 Forbidden` / `SecurityException`.


