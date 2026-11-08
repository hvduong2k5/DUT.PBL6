# Agent Rules — Identity Service

Quy tắc này áp dụng cho mọi thay đổi code trong `services/identity-service`. AI agent và lập trình viên phải tuân thủ khi implement, review hoặc test dịch vụ này.

## 1. Cơ sở dữ liệu & Migration
- Tất cả migration nằm ở `src/main/resources/db/migration/`.
- Khi tạo bảng mới hoặc thay đổi schema, phải viết file migration theo định dạng `V{so}__mota_chuc_nang.sql`.
- Không được sửa trực tiếp migration đã commit lên nhánh khác.

### 1.1 Bảng IAM cốt lõi
| Bảng | Mục đích |
|------|----------|
| `identity_principals` | Lưu danh tính chính (principal_type: CUSTOMER, EMPLOYEE) |
| `iam_roles` | Danh mục vai trò (role_code, is_enabled, is_protected) |
| `iam_permissions` | Mã quyền (permission_code, resource_code, action_code) |
| `iam_role_permissions` | Ánh xạ role ↔ permission |
| `iam_role_assignments` | Gán role cho principal kèm scope, valid_from/valid_until, status |
| `identity_audit_log` | Ghi nhật ký kiểm toán (actor, target, action, reason, timestamp) |

## 2. Quy tắc Role & Permission
- **Chỉ EMPLOYEE (`principal_type = EMPLOYEE`) được gán role hành chính.** `principal_type = CUSTOMER` không được gán role `ADMIN` hoặc bất kỳ role quản trị nào. Hiện tại chỉ có role `ADMIN` trong hệ thống.
- **Không self-assignment:** Người dùng không được tự gán role cho chính mình.
- **Phân tách nhiệm vụ:** Người tạo request không được đồng thời là người phê duyệt.
- **Role bảo vệ:** `is_protected = TRUE` (ví dụ `ADMIN`) không được xóa cứng hoặc xóa mềm nếu là admin cuối cùng.
- **Chặn mất Admin hợp lệ cuối cùng:** Trước khi thực hiện `REVOKED` hoặc `TERMINATED` trên Admin, phải kiểm tra xem còn ít nhất 1 Admin active khác trong hệ thống không. Áp dụng cả thao tác đồng thời (concurrent).
- **Soft delete role:** Dùng `is_enabled = FALSE`. **Role đã xóa mềm không được phép reactivate** (bật lại) trong bất kỳ trường hợp nào.
- **Chỉ xóa mềm khi không còn user đang gán active** (không có `iam_role_assignments` với `assignment_status = 'ACTIVE'` và `valid_until > now` tham chiếu đến role này).
- **Xóa vật lý role chưa sử dụng:** Chỉ xóa vĩnh viễn khi không có bất kỳ `iam_role_assignments` nào (kể cả lịch sử) tham chiếu role đó; ngược lại phải xóa mềm.
- **UNIQUE role_code:** Dùng partial unique index `uq_iam_roles_role_code_enabled ON iam_roles(role_code) WHERE is_enabled = TRUE` (migration `V9`). **Giữ nguyên `role_code` cũ** trên hàng đã xóa mềm, không đổi tên mã.
- **Các assignment cũ sau khi soft delete:** Phải được chuyển sang `REVOKED`, không được để trỏ nhầm sang role mới (vì `role_id` khác).
- **Kiểm tra phiên bản (optimistic locking):** Khi người dùng lưu form cũ, phải kiểm tra `version` để tránh ghi đè thay đổi mới.
- **Không coi role ADMIN là bypass permission:** Role `ADMIN` chỉ là một role chứa các permission được cấp qua `iam_role_permissions` như mọi role khác. Không được hardcode tên role `ADMIN` hoặc `ROLE_ADMIN` làm điều kiện tự động cho phép truy cập bất kỳ endpoint nào.
- **Phân quyền chỉ dựa trên permission_code:** Tất cả kiểm soát truy cập endpoint phải sử dụng `hasAuthority('<permission_code>')` hoặc `@PreAuthorize("hasAuthority('...')")`, không được dùng `hasRole('ADMIN')` làm điều kiện duy nhất hoặc làm bypass kiểm tra permission chi tiết.
- **Không viết mã kiểm tra theo tên role:** Không được viết code dạng `if (role == "ADMIN")` hoặc `if (authorities.contains("ROLE_ADMIN"))` để bỏ qua kiểm tra permission chi tiết.

## 3. Chuẩn hóa JWT và đồng bộ quyền (ƯU TIÊN ĐẦU)
- **Thêm claim `principal_type` vào JWT.** Giá trị: `CUSTOMER` hoặc `EMPLOYEE`. Nguồn: `identity_principals.principal_type`. Được ghi vào Keycloak user attribute `principal_type`, rồi protocol mapper đưa vào claim `principal_type` của access token.
- **Stale data tối đa 15 phút:** Access token chỉ sống 15 phút (`accessTokenLifespan = 900s`). Nếu `principal_type` thay đổi (rất hiếm), token cũ vẫn có giá trị đến khi hết hạn, sau đó phải refresh để nhận claim mới.
- Token JWT phải thể hiện được:
  - **Loại tài khoản** (`principal_type`: `CUSTOMER` hoặc `EMPLOYEE`)
  - **Principal ID nội bộ** (`user_id`)
  - **Permission gắn đúng scope** (`permission_grants` từ `iam_role_permissions` qua `iam_role_assignments`)
  - **Thời hạn hiệu lực** (`exp`, `iat`)
  - **Realm roles** (`realm_access.roles`) — chỉ dùng để sinh authority `ROLE_*`, không làm bypass kiểm tra permission chi tiết.
- Claim `permissions` (flat list) chỉ giữ cho consumer cũ (Catalog, Identity tự kiểm) trong giai đoạn chuyển tiếp. Consumer mới không được dùng danh sách phẳng để kiểm phạm vi; phải dùng `permission_grants`.
- Khi đổi permission của role, disable role, hoặc thu hồi assignment → phải cập nhật dữ liệu dùng để phát token cho tất cả nhân viên bị ảnh hưởng (gọi `PermissionSyncService.syncPrincipal` / `syncAllPrincipalsForRole`).
- Token mới phải nhận quyền mới.
- Quyền có thời hạn không được sống lâu hơn assignment (khi assignment `REVOKED`/`EXPIRED`, token tương ứng phải bị vô hiệu hóa hoặc không thể refresh được quyền cũ).

### 3.1 Thu hồi phiên tức thời (Force Re-login)
- Access token ngắn hạn: `accessTokenLifespan = 900s` (15 phút). Refresh Token Rotation bật (`revokeRefreshToken = true`, `refreshTokenMaxReuse = 0`) tại `services/keycloak-service/realm-export.template.json`.
- Khi **thu hồi** quyền (revoke assignment, xóa/thay thế bớt permission của role, disable/soft delete role, suspend tài khoản): sync permission, rồi gọi `KeycloakAdminClient.logoutUser(idpSubject)` và `BffSessionStore.removeSessionsByPrincipalId(principalId)` cho mọi principal bị ảnh hưởng.
- Khi chỉ **thêm** permission: chỉ sync Keycloak, không force re-login.
- Khi Keycloak từ chối refresh token (session đã bị revoke): xóa BFF session và trả `401`.

### 3.2 Nguồn sự thật & Keycloak chỉ là bản sao
- Nguồn sự thật của grants là DB IAM: `iam_roles`, `iam_role_permissions`, `iam_role_assignments`.
- Keycloak user attribute `permissions` chỉ là bản sao để protocol mapper đưa vào claim `permissions` của token.
- Không được đọc attribute Keycloak để quyết định quyền hiệu lực. Luôn tính lại từ DB.

### 3.3 Luồng sync bắt buộc
Mọi thay đổi sau phải gọi `PermissionSyncService`:
- Gán role (`ROLE_ASSIGN`) — sync, không force re-login.
- Thu hồi assignment (`ROLE_REVOKE`) — sync + force re-login.
- Sửa permissions của role (`ROLE_PERMISSION_CHANGE`) — sync tất cả holder; nếu quyền giảm thì force re-login.
- Disable role (`ROLE_DISABLE`) — thu hồi assignment + sync + force re-login các principal bị ảnh hưởng.
- Khóa/mở tài khoản (`ACCOUNT_STATUS_CHANGE`) — khi mở khóa phải sync lại trước khi user nhận token mới.
- Chạy tay (`MANUAL_RESYNC`) — endpoint đối soát.

### 3.4 Quy tắc disable role (lỗi P1 của PR #73)
Thứ tự bắt buộc: lấy danh sách principal bị ảnh hưởng **trước khi** revoke → `softDeleteById` → `revokeAllByRoleId` → với từng principal: tính lại quyền, sync Keycloak, xóa BFF session + logout Keycloak. Không được bỏ qua bước session.

### 3.5 Xử lý lỗi sync & đường đối soát
- Không được `catch (...) {}` im lặng. Phải `log.error` và ghi audit `IAM_PERMISSION_SYNC_FAILED` với `outcome=FAILED`, `reasonCode` (`KEYCLOAK_SYNC_FAILED`, `PRINCIPAL_NOT_LINKED`, `BFF_SESSION_REVOKE_FAILED`, `KEYCLOAK_LOGOUT_FAILED`), và `details.trigger`.
- Endpoint đối soát: `POST /api/v1/admin/iam/principals/{principalId}/sync-permissions?forceRelogin=false`. Trả `502` khi Keycloak lỗi.
- Endpoint xem quyền cấu hình: `GET /api/v1/admin/iam/principals/{principalId}/effective-permissions`.

### 3.6 Token đã phát không sửa được
- Không có cách thu hồi quyền trong một access token đã ký. Thay đổi chỉ có hiệu lực từ token phát sau lần refresh kế tiếp.
- Vì vậy mọi đường **giảm quyền** phải force re-login (revoke refresh token + xóa BFF session).
- UI/Access Review phải hiển thị tách biệt "quyền đang cấu hình" (từ DB) và "quyền token đang mang" (claim hiện tại).

## 4. Biến môi trường thời gian sống Token & Session (thêm mới)

Tất cả thời gian sống của Access Token, Refresh Token, BFF Session, Flow Cookie phải đọc từ biến môi trường, không được hardcode trong code Java.

| Biến env | Giá trị mặc định | Ý nghĩa | Đọc trong code |
|----------|-----------------|---------|----------------|
| `ACCESS_TOKEN_LIFESPAN_SECONDS` | `900` (15 phút) | Default Access Token lifespan nếu Keycloak không trả `expires_in` | `AuthenticationServiceImpl.java` |
| `REFRESH_TOKEN_LIFESPAN_SECONDS` | `2592000` (30 ngày) | Default Refresh Token lifespan nếu Keycloak không trả `refresh_expires_in` | `AuthenticationServiceImpl.java` |
| `BFF_SESSION_IDLE_SECONDS` | `2592000` (30 ngày) | Thời gian idle timeout của BFF session trong `BffSessionStore` | `BffSessionStore.java` |
| `BFF_FLOW_COOKIE_SECONDS` | `300` (5 phút) | Flow cookie max age (`oma_oidc_flow`) — đọc ở Next.js frontend | `apps/web-user/.env.example` |

Khai báo đầy đủ trong `services/identity-service/.env.example` (backend) và `apps/web-user/.env.example` (frontend). Khi deploy trên Coolify, gán giá trị phù hợp tại Environment Variables.

## 5. Truy vấn đọc Role
- Mọi truy vấn đọc `iam_roles` phải thêm điều kiện `AND is_enabled = TRUE` (hoặc `deleted_at IS NULL` nếu có).
- Mọi truy vấn đọc assignment hiệu lực phải kiểm tra:
  ```sql
  assignment_status = 'ACTIVE'
  AND valid_from <= now
  AND (valid_until IS NULL OR valid_until > now)
  ```

## 6. Keycloak Sync
- Điểm vào duy nhất là `PermissionSyncService` (`syncPrincipal`, `syncAllPrincipalsForRole`, `resolveEffectivePermissions`).
- `KeycloakAdminClient.updateUserPermissions(keycloakUserId, permissions)` ghi attribute; `logoutUserStrict` dùng cho revocation path, `logoutUser` là best-effort.
- **Không xóa realm role ngay khi soft delete trong DB** để tránh làm hỏng JWT/attribute cũ.
- Không tính quyền bằng cách cộng dồn `findByRoleId` trong vòng lặp; dùng query `findEffectivePermissionCodesByPrincipal`.

## 7. API Contract
- Tất cả endpoint admin phải nằm dưới `/api/v1/admin/**`.
- Mọi thay đổi REST API phải cập nhật contract tại `docs/03_api_specs/` (xem quy tắc chung trong `AGENTS.md` → Frontend API Contract Handoff).
- Phân quyền endpoint **chỉ** dùng `@PreAuthorize("hasAuthority('XXX_VIEW/MANAGE')")`. Không dùng `hasRole('ADMIN')` làm bypass.

## 8. Audit Log
- Tất cả thao tác khóa/mở khóa tài khoản, gán/thu hồi role, reactivate, thay đổi role/permission/assignment phải ghi vào `identity_audit_log`.
- Trường bắt buộc: `actor_principal_id`, `target_principal_id`, `action`, `reason`, `timestamp`.
- Ví dụ action: `ACCOUNT_SUSPEND`, `ACCOUNT_REACTIVATE`, `ROLE_ASSIGN`, `ROLE_REVOKE`, `ROLE_DISABLE`, `PERMISSION_CHANGE`.

## 9. Commit Message
- 100% tiếng Anh.
- Ví dụ: `fix(identity): add account reactivate endpoint with permission check`
- Tham khảo `AGENTS.md` phần Commit Message Convention.

## 10. Test
- Viết unit test cho `RoleAssignmentServiceImpl`, `RoleManagementServiceImpl`.
- Viết integration test cho API admin account và role.
- Chạy `./gradlew test` trong `services/identity-service` trước khi commit.

## 11. Cấu trúc package (`com.hvduong.identity`)
- `api/{account,admin,registration}` (+ `impl/`): interface API và controller.
- `application/{account,audit,authorization,iam,registration}` (+ `impl/`): service nghiệp vụ.
- `application/dto/{request,response}`: DTO. `application/mapper/`: MapStruct mapper.
- `common/{constant,enums,exception,handler,response}`: thành phần dùng chung (`ApiResponse`, `PageResponse`, `IdentityException`, `ErrorCode`, `GlobalExceptionHandler`).
- `domain/entity/`: domain/persistence entity (`*Row`). `infrastructure/persistence/mybatis/mapper/`: MyBatis mapper interfaces.
- MyBatis XML đặt tại `src/main/resources/mapper/sql/` (`mybatis.mapper-locations: classpath:mapper/sql/*.xml`).
- Keycloak (compose, realm, theme) thuộc `services/keycloak-service`; xem `KEYCLOAK_SERVICE_AGENT_RULES.md`.

## 12. Tham khảo
- `docs/01_requirements/identity-iam-lifecycle-plan.md`: Kế hoạch chi tiết vòng đời IAM.
- `services/identity-service/docs/iam-role-permission-matrix.md`: Ma trận phân quyền.