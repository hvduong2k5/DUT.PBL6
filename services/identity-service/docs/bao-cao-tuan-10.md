# BÁO CÁO CÔNG VIỆC TUẦN
### Dự án PBL6: Hệ thống Thương mại Điện tử Đặc sản Huế "Mè Xửng O Mạ"

* **Họ và tên:** Hoàng Văn Dương
* **MSSV:** 102230288
* **Lớp:** 22T_DT2
* **Tuần thực hiện:** Tuần 10
* **Phân hệ phụ trách:** Identity Service (MS-16) & Hạ tầng Xác thực / Phân quyền

---

## 1. MỤC TIÊU TRONG TUẦN
1. **Hoàn thiện hệ thống Quản lý Vai trò & Quyền hạn (IAM Role & Permission Management):** Xây dựng đầy đủ bộ REST API CRUD cho IAM Roles và Permissions; thực hiện cơ chế bảo vệ vai trò hệ thống (`is_protected = true`) và đồng bộ 2 chiều với Keycloak Realm Roles.
2. **Nâng cấp Vòng đời Tài khoản & Nhận diện Phiên (User Lifecycle & Session):** Bổ sung Flyway migration `V6` cho trạng thái tài khoản `SUSPENDED`; xây dựng luồng Khóa (`suspend`) và Mở khóa (`reactivate`) tài khoản kèm cơ chế khóa lạc quan (Optimistic Locking), ghi Outbox Event và đồng bộ vô hiệu hóa trên Keycloak.
3. **Tối ưu hóa API Nhận diện Quyền hạn (`/api/v1/users/me`):** Phân giải tường minh mã vai trò (`roleCode`), tên hiển thị (`roleName`), phạm vi phân quyền (`scopeType`, `scopeId`) và tập hợp gộp không trùng lặp các mã quyền (`permissions`); tự động cấp vai trò mặc định `CUSTOMER` cho khách hàng.
4. **Chuẩn hóa Kiến trúc Web Hợp Nhất (Unified Web Portal) & API Contract:** 
   - Thống nhất kiến trúc 1 Web duy nhất phục vụ Khách hàng, Nhân viên và Ban quản trị; phân quyền menu động theo `roles` và `permissions`.
   - Phân tách và chuẩn hóa bộ 4 tài liệu OpenAPI Specs (`customer-stable`, `customer-draft-planned`, `admin-stable`, `admin-draft-planned`) đồng bộ 1:1 với State Matrix.
   - Bổ sung `roles` và `permissions` trực tiếp vào payload phản hồi của `POST /api/v1/auth/login` và `POST /api/v1/auth/register` để giảm tối đa độ trễ điều hướng phía Frontend.
5. **Chuẩn hóa Hạ tầng Triển khai Độc lập & Tích hợp Coolify:** Tách biệt cấu hình Docker Compose cho Keycloak độc lập (kết nối PostgreSQL server có sẵn của trường/hệ thống) và Identity Service; tối ưu hóa Traefik Reverse Proxy, cấu hình SSL/HTTPS và xử lý triệt để các lỗi build-time của Keycloak Quarkus.
6. **Mở rộng Bộ Kiểm thử Tự động:** Tăng gấp đôi độ bao phủ kiểm thử từ 16 tests lên **32 tests (100% PASS)**, kiểm chứng toàn bộ các use cases mới.

---

## 2. CÔNG VIỆC ĐÃ THỰC HIỆN

### 2.1. Hoàn thiện Quản lý Vai trò & Quyền hạn IAM (IAM Role & Permission)
- **Thiết kế & cài đặt `RoleManagementService`:**
  - `getAllRoles()`, `getRoleByCode(roleCode)`: Truy vấn danh mục vai trò kèm danh sách quyền hạn gán kèm.
  - `createRole(request, actorPrincipalId)`: Tạo vai trò mới với mã chuẩn hóa (`ROLE_XXX`), tự động tạo Realm Role tương ứng trên Keycloak qua `KeycloakAdminClient.createOrUpdateRealmRole()`, ghi nhật ký kiểm toán `CREATE_ROLE`.
  - `updateRole(...)`: Cập nhật tên và mô tả vai trò; áp dụng cơ chế khóa lạc quan `version` ngăn chặn ghi đè dữ liệu đồng thời.
  - `deleteRole(roleCode, actorPrincipalId)`: Cơ chế an toàn nghiêm ngặt — **chặn tuyệt đối việc xóa vai trò hệ thống (`is_protected = true`)** hoặc vai trò đang được gán cho nhân viên còn hiệu lực (`activeAssignmentsCount > 0`). Tự động xóa Realm Role tương ứng trên Keycloak khi đủ điều kiện.
  - `assignPermissionsToRole(...)`: Cập nhật danh sách quyền cho vai trò, tự động loại bỏ trùng lặp và xác thực tính hợp lệ của từng `permission_code`.
- **Cài đặt `IamRoleController`:** Expose toàn bộ 7 endpoints RESTful chuẩn OpenAPI dưới tiền tố `/api/v1/admin/access/roles` và `/api/v1/admin/access/permissions`.

### 2.2. Vòng đời Tài khoản & Nhận diện Phiên làm việc (User Lifecycle & Session)
- **Thực thi Migration `V6__allow_suspended_provisioning_status.sql`:** Nới lỏng ràng buộc CHECK trên `identity_principals`, chính thức bổ sung trạng thái `SUSPENDED`.
- **Cài đặt `UserLifecycleService`:**
  - `suspendAccount(principalId, reason, actorId)`: Chuyển trạng thái chủ thể sang `SUSPENDED`, gọi `KeycloakAdminClient.setUserEnabled(idpSubject, false)` để vô hiệu hóa tài khoản trên Keycloak ngay lập tức, ghi sự kiện Outbox `identity.events.v1.UserSuspended` và ghi Audit Log.
  - `reactivateAccount(principalId, actorId)`: Khôi phục trạng thái `ACTIVE` và kích hoạt lại tài khoản trên Keycloak (`setUserEnabled = true`).
  - `getProfileBySubject(subject, email)`: Truy vấn thông tin tài khoản hiện tại từ JWT token, tổng hợp toàn bộ các phân quyền IAM (`iam_role_assignments`) còn hiệu lực, phân giải tên vai trò và gom nhóm toàn bộ mã quyền hiệu lực thành danh sách duy nhất (`LinkedHashSet`).
  - Tự động bổ sung vai trò `CUSTOMER` (Scope `GLOBAL`) nếu tài khoản là khách hàng nhưng chưa có bản ghi IAM riêng biệt trong bảng phân quyền.
- **Cài đặt `UserLifecycleController`:** Cung cấp các endpoint `/api/v1/users/me`, `/api/v1/admin/users/{id}/suspend`, `/api/v1/admin/users/{id}/reactivate`.

### 2.3. Quy hoạch Web Hợp Nhất (Unified Web) & Cải tiến API Contract
- **Phân tích Kiến trúc Hợp nhất Web & Mobile:**
  - Web chạy chung 1 ứng dụng Next.js (port 3000) cho Khách hàng, Nhân viên và Quản trị viên; Mobile App (Flutter/React Native) dành riêng cho Khách hàng.
  - Menu quản trị (`/admin/**`) và các tab chức năng (Kho, Đóng gói, Vận chuyển) được ẩn/hiện động hoàn toàn dựa trên mảng `permissions` và `roles` nhận từ Identity Service.
- **Chuẩn hóa Hợp đồng OpenAPI 3.0:**
  - Tách bạch rõ ràng 4 file OpenAPI: `customer-stable`, `customer-draft-planned`, `admin-stable`, `admin-draft-planned`.
  - Cập nhật endpoint `POST /api/v1/auth/login`, `POST /api/v1/auth/register` và `GET /api/v1/auth/me`: Trả về trực tiếp mảng `roles` và `permissions` ngay trong response. Phía Frontend có thể lưu session và điều hướng trang ngay lập tức (`isStaff ? router.push('/admin') : router.push('/')`), không cần gọi thêm API thứ hai.
  - Biên soạn hoàn chỉnh tài liệu tích hợp bàn giao cho đội ngũ Frontend: `docs/03_api_specs/frontend-unified-auth-guide.md` với code mẫu React, TypeScript types, Route Guard và Axios Interceptors.

### 2.4. Đóng gói & Triển khai Độc lập trên Coolify
- **Tách biệt cấu hình theo hạ tầng thực tế:**
  - Tách Keycloak thành cụm độc lập tại `deploy/keycloak/docker-compose.yml`, kết nối trực tiếp đến **máy chủ PostgreSQL có sẵn** của dự án (vào database `keycloak_db`).
  - Tách `identity-service` thành file compose độc lập tại `deploy/docker-compose.yml`, kết nối linh hoạt tới Postgres server (`identity_db`) và Keycloak.
  - Lưu trữ bản `docker-compose.all-in-one.yml` phục vụ kiểm thử cục bộ ngoại tuyến.
- **Tối ưu hóa triển khai trên Coolify:**
  - Cấu hình Traefik Reverse Proxy và SSL tự động cho domain `https://keycloak-pbl6.hvduong.qzz.io`.
  - Thiết lập chuẩn xác các biến môi trường bắt buộc: `KC_PROXY_HEADERS=xforwarded`, `KC_HOSTNAME`, `KC_HTTP_ENABLED=true`.
  - Xử lý cơ chế mạng Docker của Coolify: Khai báo network `coolify` (`external: true`) giúp container Keycloak kết nối thông suốt với container PostgreSQL nội bộ.
  - Khắc phục lỗi build-time của Quarkus bằng cách chuẩn hóa lệnh `command: start --import-realm` (loại bỏ cờ `--optimized` khi dùng image chuẩn).
  - Biên soạn 2 tài liệu hướng dẫn triển khai chi tiết: `services/identity-service/README.md` và `services/identity-service/deploy/keycloak/README.md`.

### 2.5. Xử lý Tương thích Môi trường Phát triển (IDE & Compiler)
- Phân tích và làm rõ cảnh báo parser Eclipse JDT trên VS Code (`The Java feature 'Flexible Constructor Bodies' is only available with source level 25 and above Java(4195411)`) khi dùng `@RequiredArgsConstructor` của Lombok trên Java 21.
- Hướng dẫn cấu hình constructor tường minh (Explicit Constructor) và nâng cấp Lombok lên phiên bản `1.18.36` giúp mã nguồn sạch bóng lỗi cảnh báo trên IDE.

---

## 3. KẾT QUẢ VÀ SẢN PHẨM BÀN GIAO

### 3.1. Danh mục Endpoints Hoàn Thiện

| Phương thức | Đường dẫn API | Chức năng | Phân quyền / Bảo mật |
|:---:|---|---|:---:|
| `POST` | `/api/v1/auth/login` | Đăng nhập hệ thống (trả về Tokens + Roles + Permissions) | Public |
| `POST` | `/api/v1/auth/register` | Đăng ký khách hàng D2C (Idempotent + Outbox) | Public |
| `POST` | `/api/v1/public/registration/customer` | Endpoint đăng ký qua Orchestrator | Public |
| `GET` | `/api/v1/users/me` | Lấy danh tính, vai trò (kèm scope) và quyền hạn cá nhân | Authenticated (JWT) |
| `GET` | `/api/v1/auth/me` | Lấy profile danh tính chuẩn contract Customer | Authenticated (JWT) |
| `POST` | `/api/v1/admin/users/{id}/suspend` | Khóa tài khoản, vô hiệu hóa Keycloak, ghi Outbox/Audit | `ADMIN` / IAM Scope |
| `POST` | `/api/v1/admin/users/{id}/reactivate` | Mở khóa tài khoản, kích hoạt lại Keycloak | `ADMIN` / IAM Scope |
| `GET` | `/api/v1/admin/access/roles` | Lấy danh mục tất cả vai trò IAM | `ADMIN` / IAM Scope |
| `POST` | `/api/v1/admin/access/roles` | Tạo vai trò IAM mới, đồng bộ Keycloak Realm Role | `ADMIN` / IAM Scope |
| `GET` | `/api/v1/admin/access/roles/{code}` | Xem chi tiết 1 vai trò kèm danh sách quyền | `ADMIN` / IAM Scope |
| `PUT` | `/api/v1/admin/access/roles/{code}` | Cập nhật tên/mô tả vai trò | `ADMIN` / IAM Scope |
| `DELETE`| `/api/v1/admin/access/roles/{code}` | Xóa vai trò (chặn xóa vai trò hệ thống / đang dùng) | `ADMIN` / IAM Scope |
| `PUT` | `/api/v1/admin/access/roles/{code}/permissions` | Gán danh sách quyền hạn cho vai trò | `ADMIN` / IAM Scope |
| `GET` | `/api/v1/admin/access/permissions` | Lấy danh mục tất cả quyền hạn hạt nhân | `ADMIN` / IAM Scope |
| `GET` | `/api/v1/admin/access/permissions/{code}` | Xem chi tiết 1 quyền hạn | `ADMIN` / IAM Scope |
| `POST` | `/api/v1/admin/iam/assignments` | Gán vai trò cho nhân viên theo Scope (Kho/Cửa hàng) | `ADMIN` / IAM Scope |
| `POST` | `/api/v1/admin/invitations` | Tạo lời mời nhân viên nội bộ | `ADMIN` / IAM Scope |
| `gRPC` | `CheckSpecializedPermission` (Port 9016) | Thẩm định quyền động rủi ro cao (ABAC: Tiền, % chiết khấu) | Internal Service |

### 3.2. Kết Quả Kiểm Thử Tự Động (Đạt 32/32 tests — 100% PASS)

Bộ kiểm thử được nâng cấp toàn diện, bổ sung đầy đủ các kịch bản kiểm thử biên, kiểm thử phân quyền và xử lý lỗi:

```text
> Task :compileJava UP-TO-DATE
> Task :compileTestJava UP-TO-DATE
> Task :testClasses UP-TO-DATE
> Task :test UP-TO-DATE

BUILD SUCCESSFUL in 26s
10 actionable tasks: 10 up-to-date
```

| Tên Test Suite | Số lượng Test | Trọng tâm kiểm thử | Kết quả |
|---|:---:|---|:---:|
| `RoleManagementServiceTest` | **10** | Tạo/sửa/xóa role, chặn xóa protected role, gán permissions, đồng bộ Keycloak | ✅ 10/10 PASS |
| `UserLifecycleServiceTest` | **6** | Khóa/Mở khóa tài khoản, đồng bộ Keycloak status, phân giải permissions gộp | ✅ 6/6 PASS |
| `SpecializedPermissionServiceTest` | **5** | Thẩm định hạn mức tiền (10M VND), trần chiết khấu (25%), nguyên tắc Fail-closed | ✅ 5/5 PASS |
| `RoleAssignmentServiceTest` | **3** | Ràng buộc nhân viên, validate scope bắt buộc, kiểm tra thời hạn hiệu lực | ✅ 3/3 PASS |
| `EmployeeInvitationServiceTest` | **3** | Tạo lời mời, chấp thuận và thu hồi lời mời nhân sự | ✅ 3/3 PASS |
| `IdentityGrpcServiceTest` | **3** | Giao tiếp gRPC, mapping request/response protobuf | ✅ 3/3 PASS |
| `IdentityPrincipalMapperIntegrationTest` | **2** | Kiểm thử tương thích kiểu dữ liệu UUID & Timestamp trên PostgreSQL | ✅ 2/2 PASS |
| **TỔNG CỘNG** | **32** | **Bao phủ toàn bộ các tầng Controller, Service, Adapter và DB** | **✅ 32/32 PASS (100%)** |

### 3.3. Các Tài Liệu Bàn Giao Cốt Lõi
1. **Tài liệu Tích hợp Frontend:** [`docs/03_api_specs/frontend-unified-auth-guide.md`](file:///e:/web-deploy/DUT.PBL6/docs/03_api_specs/frontend-unified-auth-guide.md) — Hướng dẫn chi tiết luồng xác thực, phân quyền động, cấu trúc token và code mẫu tích hợp Next.js cho FE.
2. **Bộ Hợp đồng OpenAPI 3.0:** Đã cập nhật và đồng bộ trong [`docs/03_api_specs/`](file:///e:/web-deploy/DUT.PBL6/docs/03_api_specs/):
   - `customer-stable.openapi.yaml`
   - `admin-stable.openapi.yaml`
   - `customer-api-state-matrix.md` & `admin-api-state-matrix.md`
3. **Hướng dẫn Cài đặt & Vận hành Identity Service:** [`services/identity-service/README.md`](file:///e:/web-deploy/DUT.PBL6/services/identity-service/README.md).
4. **Hướng dẫn Triển khai Keycloak Standalone trên Coolify:** [`services/identity-service/deploy/keycloak/README.md`](file:///e:/web-deploy/DUT.PBL6/services/identity-service/deploy/keycloak/README.md).
5. **Cấu hình Hạ tầng Docker:**
   - [`deploy/docker-compose.yml`](file:///e:/web-deploy/DUT.PBL6/services/identity-service/deploy/docker-compose.yml) (Identity Service độc lập)
   - [`deploy/keycloak/docker-compose.yml`](file:///e:/web-deploy/DUT.PBL6/services/identity-service/deploy/keycloak/docker-compose.yml) (Keycloak độc lập)
   - [`deploy/keycloak/realm-export.template.json`](file:///e:/web-deploy/DUT.PBL6/services/identity-service/deploy/keycloak/realm-export.template.json) (Template nạp Realm `omama`)

---

## 4. TỒN ĐỒNG, ĐÁNH GIÁ & KẾ HOẠCH TUẦN 11

### 4.1. Đánh giá Mức độ Hoàn thiện
- Phân hệ **Identity Service (MS-16)** đã **hoàn thành 100% mục tiêu giai đoạn cốt lõi**:
  - Tầng dữ liệu PostgreSQL đạt chuẩn, quản lý qua Flyway migration sạch sẽ.
  - Tầng bảo mật Keycloak OIDC hoạt động ổn định trên nền tảng Coolify.
  - Quản lý phân quyền IAM đầy đủ tính năng RBAC đa phạm vi (`GLOBAL`, `WAREHOUSE`, `STORE`) kết hợp ABAC động qua gRPC.
  - Hợp đồng API với Frontend Web và Mobile App được đặc tả chi tiết, không còn khoảng cách (gap) giữa thiết kế và hiện thực.

### 4.2. Kế hoạch Tuần 11
1. **Hỗ trợ đội ngũ Frontend (FE):** Phối hợp hỗ trợ FE hoàn thành việc merge ứng dụng Web quản trị và Web bán hàng thành **Unified Web**, tích hợp router guard dựa trên mảng `roles` và `permissions` từ Identity Service.
2. **Tích hợp API Gateway:** Cấu hình định tuyến và xác thực JWT tập trung trên API Gateway (route `/api/v1/auth/**` và `/api/v1/admin/access/**` trỏ về Identity Service).
3. **Mở rộng Kafka Consumer cho các Microservices:** Hỗ trợ Catalog Service, Order Service và Customer Profile Service tiêu thụ sự kiện từ Outbox (`identity.events.v1.UserRegistered`, `UserSuspended`) để tự động hóa quy trình nghiệp vụ sau đăng ký.
4. **Triển khai CI/CD Pipeline:** Thiết lập GitHub Actions tự động build JAR, chạy 32 automated tests và kích hoạt webhook deploy lên Coolify khi merge code vào nhánh `main`.

