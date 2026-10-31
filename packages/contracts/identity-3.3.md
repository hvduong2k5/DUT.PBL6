# Identity 3.3 — RESTful API Integration Contract & Semantic Rules

**Ngày cập nhật:** 31/10/2026  
**Trạng thái:** Integration Contract — Target MVP 3.3  
**Tài liệu quy chuẩn:** Bài giảng Thiết kế Hệ thống & Phân tích Nghiệp vụ — **Mục 3.3 & 3.3.1: Áp dụng thiết kế RESTful API**  
**OpenAPI Spec tương ứng:** [`docs/03_api_specs/identity-service.openapi.yaml`](../../docs/03_api_specs/identity-service.openapi.yaml)  
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

| Phân hệ / Nghiệp vụ | HTTP Method | Endpoint URI (Chuẩn RESTful 3.3.1) | Legacy Alias (Tương thích) | Mô tả hành động & Quy tắc áp dụng |
| :--- | :---: | :--- | :--- | :--- |
| **User Profile** | `GET` | `/api/v1/users/me` | — | Lấy hồ sơ người dùng đang đăng nhập (Quy tắc 1, 2). |
| **User Status** | `PATCH` | `/api/v1/admin/users/{principalId}/status` | `POST .../suspend`<br>`POST .../reactivate` | Cập nhật trạng thái tài khoản (`ACTIVE`, `SUSPENDED`). Dùng `PATCH` thay vì động từ trong URI (Quy tắc 1, 3). |
| **Employee Invitations** | `POST` | `/api/v1/admin/employee-invitations` | `POST .../employees/invitations` | Tạo thư mời nhân viên mới với token có hạn (Quy tắc 2, 4). |
| | `DELETE`| `/api/v1/admin/employee-invitations/{invitationId}` | `POST .../{id}/revoke` | Thu hồi thư mời nhân viên. Dùng `DELETE` method thay cho verb `revoke` (Quy tắc 1, 4). |
| **IAM Role Assignments** | `POST` | `/api/v1/admin/role-assignments` | `POST .../iam/assignments` | Gán vai trò chức danh cho nhân viên có phạm vi nghiệp vụ (Quy tắc 2, 4). |
| | `DELETE`| `/api/v1/admin/role-assignments/{assignmentId}` | — | Thu hồi vai trò đã gán của nhân viên (Quy tắc 1, 4). |
| **IAM Roles** | `GET` | `/api/v1/admin/roles` | — | Lấy danh sách vai trò hệ thống (Quy tắc 1, 2). |
| | `POST` | `/api/v1/admin/roles` | — | Tạo vai trò mới và đồng bộ Realm Role sang Keycloak (Quy tắc 1, 2). |
| | `GET` | `/api/v1/admin/roles/{roleId}` | — | Xem chi tiết vai trò (Quy tắc 1, 2). |
| | `PATCH`| `/api/v1/admin/roles/{roleId}` | — | Cập nhật tên, mô tả hoặc cờ kích hoạt vai trò (Quy tắc 1). |
| | `DELETE`| `/api/v1/admin/roles/{roleId}` | — | Xóa vai trò (chặn xóa vai trò bảo vệ hoặc đang có nhân viên gán) (Quy tắc 1). |
| **Role Permissions** | `GET` | `/api/v1/admin/roles/{roleId}/permissions` | — | Xem danh sách quyền hạn của vai trò (Quy tắc 3). |
| | `PUT` | `/api/v1/admin/roles/{roleId}/permissions` | — | Ghi đè toàn bộ danh sách quyền của vai trò (Batch replace) (Quy tắc 3). |
| | `POST` | `/api/v1/admin/roles/{roleId}/permissions` | — | Gán thêm một quyền vào vai trò (Quy tắc 3). |
| | `DELETE`| `/api/v1/admin/roles/{roleId}/permissions/{permissionCode}` | — | Gỡ một quyền khỏi vai trò (Quy tắc 3). |
| **Customer Registration**| `POST` | `/api/v1/public/customer-registrations` | `POST .../registration/customer` | Đăng ký tài khoản Khách hàng D2C với Idempotency Key (Quy tắc 1, 2, 4). |

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

