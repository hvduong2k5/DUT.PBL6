# Identity & Access Management (IAM) Service — O Mạ PBL6

Dịch vụ Quản lý Định danh & Phân quyền tập trung (Identity & Access Management) cho hệ sinh thái Thương mại điện tử **Mè Xửng O Mạ**.

---

## 1. Giới Thiệu & Kiến Trúc

`identity-service` áp dụng mô hình phân tách trách nhiệm rõ ràng (**Boundary Pattern**) giữa Identity Provider (IdP) và Domain Authorization:

```
                            ┌──────────────────────────────────────────┐
                            │          CLIENTS (Web & Mobile)          │
                            └────────────────────┬─────────────────────┘
                                                 │
                                                 ▼
                            ┌──────────────────────────────────────────┐
                            │    IDENTITY SERVICE (Port 8016/9016)     │
                            │   (Spring Boot 3.3.4 + Java 21 LTS)      │
                            └──────────────┬───────────────────┬───────┘
                                           │                   │
                        Direct Grant / OIDC│                   │ MyBatis / SQL
                                           ▼                   ▼
                     ┌───────────────────────────┐   ┌───────────────────────────┐
                     │    KEYCLOAK IDP (8080)    │   │  POSTGRESQL (identity_db) │
                     │  - Quản lý Credentials    │   │  - IAM Roles & Permissions│
                     │  - Mã hóa mật khẩu        │   │  - Scoped Assignments     │
                     │  - Cấp phát JWT (JWKS)    │   │  - Outbox Pattern & Audit │
                     │  - Phiên làm việc (SSO)   │   │  - User Lifecycle Status  │
                     └───────────────────────────┘   └───────────────────────────┘
```

- **Keycloak (Identity Provider):** Chịu trách nhiệm xác thực danh tính, lưu trữ mật khẩu an toàn, cấp phát Access Token/Refresh Token chuẩn OpenID Connect (OIDC).
- **Identity Service + PostgreSQL:** Quản lý toàn bộ quyền hạn nghiệp vụ RBAC đa cấp (`GLOBAL`, `WAREHOUSE`, `STORE`), chính sách hạn mức chiết khấu/ngưỡng tiền (`Specialized Permissions`), Outbox Events đồng bộ Kafka và Audit Log.

---

## 2. Cấu Trúc Thư Mục Dự Án

```
services/identity-service/
├── deploy/                              # Hạ tầng triển khai Docker Compose
│   ├── docker-compose.all-in-one.yml    # Cụm All-in-One Local Dev (Postgres 5433, Keycloak 8080, Kafka, Identity)
│   ├── docker-compose.yml               # Chỉ chạy riêng Identity Service trên Production / Coolify
│   ├── init-db.sql                      # Script tạo keycloak_db và identity_db cho cụm all-in-one dev
│   └── README.md                        # Hướng dẫn chi tiết triển khai container
├── docs/                                # Tài liệu kỹ thuật chi tiết
├── src/
│   ├── main/
│   │   ├── java/com/hvduong/identity/
│   │   │   ├── api/                     # Interface API (Swagger / OpenAPI 3 annotations)
│   │   │   │   ├── account/             # UserLifecycleApi & impl/UserLifecycleController
│   │   │   │   ├── admin/               # IamRoleApi, EmployeeInvitationApi, IamAssignmentApi & impl/
│   │   │   │   └── registration/        # RegistrationApi & impl/RegistrationController
│   │   │   ├── application/             # Use Cases & Business Logic
│   │   │   │   ├── account/             # Service Interface & impl/ (Lifecycle, Invitation)
│   │   │   │   ├── audit/               # Service Interface & impl/ (Audit Log)
│   │   │   │   ├── authorization/       # Service Interface & impl/ (Specialized Perms)
│   │   │   │   ├── dto/                 # Data Transfer Objects (account, admin, auth, registration, user)
│   │   │   │   ├── iam/                 # Service Interface & impl/ (Role, Assignment)
│   │   │   │   └── registration/        # RegisterPrincipalUseCase & impl/RegistrationOrchestrator
│   │   │   ├── common/                  # ApiResponse, GlobalExceptionHandler, CommonErrorCode, SecurityUtils
│   │   │   ├── config/                  # OpenApiConfig, KeycloakConfig
│   │   │   ├── domain/
│   │   │   │   └── entity/              # Domain / Persistence Entities (11 models)
│   │   │   ├── grpc/                    # gRPC Services (Port 9016)
│   │   │   ├── infrastructure/
│   │   │   │   ├── keycloak/            # KeycloakAdminClient
│   │   │   │   ├── outbox/              # OutboxPublisher (Kafka)
│   │   │   │   └── persistence/mybatis/ # MyBatis Mapper Interfaces & TypeHandlers
│   │   │   └── security/                # Spring Security JWT Resource Server, PKCE Utils
│   │   └── resources/
│   │       ├── application.yml          # Cấu hình Spring Boot
│   │       ├── db/migration/            # Flyway migrations (V1 -> V9)
│   │       └── mapper/sql/              # MyBatis XML Mapper definitions
│   └── test/                            # Bộ kiểm thử Unit & Integration Tests (100% pass)
├── .env.example                         # Mẫu biến môi trường
├── build.gradle                         # Gradle build config
├── Dockerfile                           # Container hóa Identity Service
└── README.md                            # Hướng dẫn này
```

---

## 3. Yêu Cầu Môi Trường (Prerequisites)

- **Docker & Docker Compose:** Docker Desktop (phiên bản 20+ hỗ trợ Compose V2).
- **Java JDK (để build JAR):** Phiên bản **Java 21 LTS** (Microsoft OpenJDK, Eclipse Temurin hoặc Corretto).
- **Gradle:** Đã có sẵn Gradle wrapper (`./gradlew` hoặc `.\gradlew.bat`).
- **RAM khuyến nghị:** Tối thiểu 4GB RAM trống (đặc biệt khi chạy cụm All-in-One gồm Keycloak + PostgreSQL + Kafka).

---

## 4. Cấu Hình Biến Môi Trường (`.env`)

Tại thư mục `services/identity-service`, sao chép file mẫu:

```bash
# Windows PowerShell:
Copy-Item .env.example .env

# Linux / MacOS:
cp .env.example .env
```

### Bảng Giải Thích Các Biến Môi Trường:

| Biến môi trường | Giá trị mặc định (Dev) | Mô tả |
| :--- | :--- | :--- |
| `SERVER_PORT` | `8016` | Cổng HTTP REST API |
| `GRPC_SERVER_PORT` | `9016` | Cổng gRPC Server nội bộ |
| `DB_HOST` | `localhost` | Hostname PostgreSQL (`postgres` nếu chạy trong mạng Docker, `localhost` nếu chạy từ host) |
| `DB_PORT` | `5433` | Cổng kết nối PostgreSQL (`5433` khi dùng container `postgres` từ `docker-compose.all-in-one.yml`, `5432` nếu dùng Postgres mặc định) |
| `DB_NAME` | `identity_db` | Tên Database nghiệp vụ |
| `DB_USERNAME` | `postgres` | Tài khoản kết nối CSDL |
| `DB_PASSWORD` | `password123` | Mật khẩu CSDL (khớp với mật khẩu container `postgres` trong compose) |
| `KEYCLOAK_SERVER_URL` | `http://localhost:8080` | URL backend gọi sang Keycloak (`http://localhost:8080` khi chạy host, `http://keycloak:8080` khi chạy mạng Docker) |
| `KEYCLOAK_PUBLIC_URL` | `http://localhost:8080` | URL công khai mà trình duyệt client truy cập Keycloak |
| `KEYCLOAK_REALM` | `omama` | Tên Realm làm việc trên Keycloak |
| `KEYCLOAK_CLIENT_ID` | `identity-service-admin` | Client ID kết nối Keycloak Admin REST API |
| `KEYCLOAK_CLIENT_SECRET`| `changeme` | Secret lấy từ Keycloak Admin Console (Clients → `identity-service-admin` → Credentials) |
| `KEYCLOAK_PUBLIC_CLIENT_ID`| `frontend-web` | Public Client ID OIDC cho PKCE Flow |
| `IDENTITY_REGISTRATION_FINGERPRINT_KEY` | `synthetic-dev-fingerprint-key-at-least-32-chars` | Khóa bí mật HMAC tối thiểu 32 ký tự dùng để ký và xác thực request fingerprint khi đăng ký khách hàng |
| `AUTH_REDIRECT_URI` | `http://localhost:8016/api/v1/public/auth/callback` | Callback URL Keycloak chuyển hướng về sau khi login |
| `FRONTEND_BASE_URL` | `http://localhost:3000` | URL gốc của Frontend Web (để redirect sau đăng nhập/đăng xuất) |
| `COOKIE_SECURE` | `false` | Cờ bảo mật Cookie (`false` khi chạy local dev HTTP, `true` trên Production HTTPS) |
| `COOKIE_SAME_SITE` | `Lax` | Thuộc tính SameSite của Cookie (`Lax` hoặc `None`) |
| `SPRING_KAFKA_BOOTSTRAP_SERVERS`| `localhost:9092` | Địa chỉ Kafka Broker (`kafka:9092` nếu chạy trong container) |

---

## 5. Hướng Dẫn Khởi Chạy Service (How to Run)

### Kịch Bản 1: Chạy Hybrid Local Dev (Khuyên Dùng Hàng Ngày Để Code & Debug)

Chạy hạ tầng phụ trợ (PostgreSQL, Keycloak, Kafka) trên Docker, chạy `identity-service` trực tiếp bằng Java máy host để dễ dàng debug, hot-reload và xem log chi tiết.

#### Bước 1: Khởi động các hạ tầng phụ trợ
Đứng tại thư mục `services/identity-service`:
```bash
docker compose -f deploy/docker-compose.all-in-one.yml up -d postgres keycloak kafka
```
*Ghi chú:* Bạn cũng có thể khởi động Keycloak độc lập từ `services/keycloak-service/docker-compose.dev.yml` nếu muốn.

#### Bước 2: Lấy Client Secret từ Keycloak
1. Truy cập [http://localhost:8080/admin](http://localhost:8080/admin) (tài khoản: `admin` / `admin`).
2. Chọn Realm **`omama`** -> menu **Clients** -> click **`identity-service-admin`**.
3. Chọn tab **Credentials** -> copy **Client Secret**.
4. Dán vào file `.env` tại `services/identity-service`:
   ```env
   KEYCLOAK_CLIENT_SECRET=<secret_vua_copy>
   ```

#### Bước 3: Khởi chạy Identity Service bằng Gradle Wrapper
```bash
# Trên Windows (PowerShell / CMD):
.\gradlew.bat bootRun

# Trên Linux / macOS:
./gradlew bootRun
```
Ứng dụng sẽ tự động chạy Flyway migration để cập nhật schema CSDL (V1 -> V9) và sẵn sàng tại HTTP `8016` và gRPC `9016`.

---

### Kịch Bản 2: Chạy Toàn Bộ Cụm All-in-One Bằng Docker Compose

Dùng khi bạn muốn kiểm thử toàn diện cả ứng dụng lẫn hạ tầng trong container mà không cần cài đặt JDK/Gradle trên máy host.

#### Bước 1: Build file JAR thực thi
> [!IMPORTANT]
> Dockerfile sử dụng file JAR đã biên dịch sẵn (`COPY build/libs/identity-service-*.jar app.jar`). Bắt buộc build JAR trước:
```bash
# Windows:
.\gradlew.bat bootJar

# Linux / macOS:
./gradlew bootJar
```

#### Bước 2: Khởi chạy toàn bộ cụm Container
```bash
docker compose -f deploy/docker-compose.all-in-one.yml up -d --build
```

#### Bước 3: Kiểm tra trạng thái và log hoạt động
```bash
# Kiểm tra danh sách container đang chạy:
docker compose -f deploy/docker-compose.all-in-one.yml ps

# Xem log realtime của Identity Service:
docker compose -f deploy/docker-compose.all-in-one.yml logs -f identity-service
```

#### Dừng cụm All-in-One:
```bash
# Dừng container, giữ nguyên dữ liệu CSDL:
docker compose -f deploy/docker-compose.all-in-one.yml down

# Dừng container và xóa trắng volume để khởi động lại từ đầu:
docker compose -f deploy/docker-compose.all-in-one.yml down -v
```

---

### Kịch Bản 3: Chạy Riêng Container Identity Service (Kết Nối Hạ Tầng Sẵn Có)

Dùng cho môi trường Production / Coolify:
```bash
# 1. Build file JAR:
.\gradlew.bat bootJar

# 2. Khởi chạy container với compose sản xuất:
cd deploy
docker compose up -d --build

# 3. Theo dõi log:
docker compose logs -f identity-service
```

---

### Kịch Bản 4: Chạy Kiểm Thử Tự Động (Unit & Integration Tests)

Hệ thống tích hợp bộ kiểm thử toàn diện với **55/55 test case đạt tỉ lệ thành công 100%**:

```bash
# Trên Windows:
.\gradlew.bat test

# Trên Linux / macOS:
./gradlew test
```
Báo cáo kiểm thử HTML chi tiết được lưu tại:  
`services/identity-service/build/reports/tests/test/index.html`

---

## 6. Cấu Trúc JWT Access Token Chuẩn Hóa & Kiểm Thử

### 6.1. Cấu Trúc Lean JWT (Chuẩn Hóa Độc Lập)

Keycloak cấp phát JWT Access Token với cấu trúc tối ưu (**Lean JWT**), loại bỏ các trường PII cồng kềnh (`email`, `name`, `given_name`, `family_name`, `preferred_username`) để giảm kích thước gói tin và bảo vệ quyền riêng tư:

```json
{
  "jti": "9c8b7a65-4321-4ef0-9abc-1234567890ab",
  "iss": "https://keycloak.omama.vn/realms/omama",
  "sub": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "aud": [
    "order-api",
    "omama-api"
  ],
  "exp": 1762565100,
  "iat": 1762564800,
  "auth_time": 1762564700,
  "typ": "Bearer",
  "azp": "frontend-web",
  "sid": "7f6e5d4c-3b2a-1908-fedc-ba0987654321",
  "user_id": "a0000000-0000-0000-0000-000000000001",
  "principal_type": "EMPLOYEE",
  "realm_access": {
    "roles": [
      "ADMIN"
    ]
  },
  "permission_grants": [
    "ORDER_VIEW#GLOBAL",
    "ORDER_SENSITIVE_VIEW#WAREHOUSE#WH-HCM-01"
  ]
}
```

### Ý Nghĩa Các Trường:
- `sub`: Keycloak Subject UUID (định danh tài khoản trên Keycloak).
- `user_id`: Application User UUID (khóa chính `users.id` trong PostgreSQL `identity_db`).
- `principal_type`: Loại tài khoản (`EMPLOYEE` hoặc `CUSTOMER`).
- `realm_access.roles`: Danh sách Realm Role của Keycloak (dành cho Admin / System).
- `permission_grants`: Danh sách quyền hạn phân bổ theo phạm vi định dạng `<PERMISSION_CODE>#<SCOPE_TYPE>[#<SCOPE_ID>]`. Spring Security trong Identity Service sẽ tự động phân rã thành các quyền `hasAuthority(...)` để bảo vệ API.
- `aud`: Danh sách Audience dịch vụ được phép tiêu thụ token.
- `azp`: Client ID của ứng dụng gọi OIDC (ví dụ `frontend-web` hoặc `mobile-app`).
- `exp` / `iat`: Thời điểm hết hạn (5 phút) và thời điểm phát hành token.

### Cách Giải Mã & Xem Nội Dung Token:
- **Cách 1 (Giao diện web):** Truy cập [jwt.io](https://jwt.io) và dán chuỗi Access Token vào ô Encoded.
- **Cách 2 (Lệnh PowerShell):**
  ```powershell
  $token = "<DÁN_CHUỖI_JWT_VÀO_ĐÂY>"
  $payload = $token.Split('.')[1]
  $padding = $payload.Length % 4
  if ($padding -eq 2) { $payload += "==" } elseif ($padding -eq 3) { $payload += "=" }
  [System.Text.Encoding]::UTF8.GetString([System.Convert]::FromBase64String($payload.Replace('-', '+').Replace('_', '/'))) | ConvertFrom-Json | ConvertTo-Json -Depth 5
  ```

---

## 7. Swagger Documentation & Hướng Dẫn Gọi API

### 7.1. Swagger UI & OpenAPI 3 Documentation
- **Swagger UI Web:** [http://localhost:8016/swagger-ui.html](http://localhost:8016/swagger-ui.html)
- **OpenAPI JSON Spec:** [http://localhost:8016/v3/api-docs](http://localhost:8016/v3/api-docs)

> [!TIP]
> **Cách Ủy Quyền (Authorize) Trên Swagger UI:**
> 1. Bấm nút **Authorize** (biểu tượng ổ khóa màu xanh lá ở góc phải).
> 2. Dán chuỗi JWT Access Token nhận được từ Keycloak vào ô `Value`.
> 3. Bấm **Authorize** -> **Close**.

---

### 7.2. Kiểm Tra Trạng Thái Sống (Healthcheck)
```bash
curl http://localhost:8016/actuator/health
```
**Kết quả mong đợi:**
```json
{"status":"UP"}
```

---

### 7.3. Đăng Ký Tài Khoản Khách Hàng Công Khai (`POST /api/v1/public/customer-registrations`)
Hỗ trợ cơ chế Idempotent chống đăng ký trùng lặp:

```bash
curl -X POST http://localhost:8016/api/v1/public/customer-registrations \
  -H "Content-Type: application/json" \
  -d '{
    "idempotencyKey": "REG-TEST-001",
    "requestFingerprint": "fp-1234567890abcdef",
    "email": "customer01@omama.vn",
    "firstName": "Văn Dương",
    "lastName": "Hoàng",
    "rawPassword": "Password@123"
  }'
```

---

### 7.4. Xác Thực Theo Kiến Trúc Backend-for-Frontend (BFF Pattern - RFC 9700 & RFC 7636 PKCE)
Bảo vệ Token hoàn toàn phía server bằng HttpOnly Cookie `OMAMA_SESSION`, loại bỏ nguy cơ rò rỉ token qua XSS:

```bash
# 1. Khởi tạo đăng nhập OIDC PKCE (trả về URL xác thực Keycloak kèm state)
curl -X GET "http://localhost:8016/api/v1/public/auth/login?redirect_uri=https://omama.vn&provider=google"

# 2. Callback từ Keycloak (trình duyệt tự động chuyển hướng, cấp HttpOnly Cookie OMAMA_SESSION và redirect 302)
# GET http://localhost:8016/api/v1/public/auth/callback?code=<AUTH_CODE>&state=<STATE>

# 3. Kiểm tra thông tin phiên làm việc hiện tại
curl -X GET http://localhost:8016/api/v1/public/auth/session \
  -H "Cookie: OMAMA_SESSION=<YOUR_SESSION_UUID>"

# 4. Gia hạn phiên làm việc (Refresh Session)
curl -X POST http://localhost:8016/api/v1/public/auth/tokens/refresh \
  -H "Cookie: OMAMA_SESSION=<YOUR_SESSION_UUID>"

# 5. Đăng xuất tài khoản (Thu hồi token trên Keycloak và xóa Cookie)
curl -X POST http://localhost:8016/api/v1/public/auth/logout \
  -H "Cookie: OMAMA_SESSION=<YOUR_SESSION_UUID>"
```

---

### 7.5. Lấy Hồ Sơ & Quyền Của Bản Thân (`GET /api/v1/users/me`)
Gửi kèm JWT Access Token:

```bash
curl -X GET http://localhost:8016/api/v1/users/me \
  -H "Authorization: Bearer <YOUR_ACCESS_TOKEN>"
```

---

### 7.6. Cập Nhật Trạng Thái Người Dùng (`PATCH /api/v1/admin/users/{principalId}/status`)
```bash
# Khóa tài khoản
curl -X PATCH http://localhost:8016/api/v1/admin/users/e9f8d7c6-1111-2222-3333-444455556666/status \
  -H "Authorization: Bearer <ADMIN_ACCESS_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"status": "SUSPENDED", "reason": "Tài khoản có dấu hiệu gian lận"}'

# Mở khóa tài khoản
curl -X PATCH http://localhost:8016/api/v1/admin/users/e9f8d7c6-1111-2222-3333-444455556666/status \
  -H "Authorization: Bearer <ADMIN_ACCESS_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"status": "ACTIVE"}'
```

---

### 7.7. Quản Lý Danh Mục Vai Trò IAM (`GET /api/v1/admin/roles`)
```bash
curl -X GET http://localhost:8016/api/v1/admin/roles \
  -H "Authorization: Bearer <ADMIN_ACCESS_TOKEN>"
```

---

### 7.8. Quản Lý Thư Mời Cho Nhân Viên Mới (`/api/v1/admin/employee-invitations`)
```bash
# Tạo thư mời:
curl -X POST http://localhost:8016/api/v1/admin/employee-invitations \
  -H "Authorization: Bearer <ADMIN_ACCESS_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"validDays": 7}'

# Thu hồi thư mời bằng HTTP DELETE:
curl -X DELETE http://localhost:8016/api/v1/admin/employee-invitations/<INVITATION_UUID> \
  -H "Authorization: Bearer <ADMIN_ACCESS_TOKEN>"
```

---

### 7.9. Gán & Thu Hồi Vai Trò Chức Danh (`/api/v1/admin/role-assignments`)
```bash
# Gán vai trò cho nhân viên:
curl -X POST http://localhost:8016/api/v1/admin/role-assignments \
  -H "Authorization: Bearer <ADMIN_ACCESS_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "principalId": "e9f8d7c6-1111-2222-3333-444455556666",
    "roleId": "d8e7c6b5-0000-1111-2222-333344445555",
    "scopeType": "WAREHOUSE",
    "scopeId": "WH-HUE-01",
    "reason": "Điều chuyển thủ kho chi nhánh Huế"
  }'

# Thu hồi vai trò đã gán bằng HTTP DELETE:
curl -X DELETE http://localhost:8016/api/v1/admin/role-assignments/<ASSIGNMENT_UUID>?reason=KetThucNhiemKy \
  -H "Authorization: Bearer <ADMIN_ACCESS_TOKEN>"
```

---

### 7.10. Kiểm Thử gRPC Service (Port 9016)
Service gRPC cung cấp kiểm tra quyền hạn chuyên sâu và phân phối Public Key:
- `CheckSpecializedPermission`: Đánh giá quyền hạn đặc thù và kiểm soát rủi ro cao.
- `GetJwksPublicKey`: Trả về Public Key để các Microservices khác kiểm tra chữ ký token nội bộ.

Kiểm tra bằng công cụ `grpcurl`:
```bash
grpcurl -plaintext localhost:9016 list
```

---

## 8. Khắc Phục Sự Cố Thường Gặp (Troubleshooting)

### 1. Lỗi `COPY failed: no source files were specified` khi build Docker
- **Nguyên nhân:** Chưa chạy lệnh build file JAR trước khi chạy `docker build` hoặc `docker compose up --build`.
- **Khắc phục:** Chạy `.\gradlew.bat bootJar` (Windows) hoặc `./gradlew bootJar` (Linux/Mac) rồi thử lại.

### 2. Container không kết nối được PostgreSQL / Keycloak trên máy Host
- **Nguyên nhân:** Container dùng `localhost` sẽ trỏ vào chính container thay vì máy tính của bạn.
- **Khắc phục:** 
  - Trên Windows / macOS: Đặt `DB_HOST=host.docker.internal` và `KEYCLOAK_SERVER_URL=http://host.docker.internal:8080`.
  - Trên Linux: Thêm cấu hình `extra_hosts: ["host.docker.internal:host-gateway"]` vào docker-compose service.

### 3. Lỗi `401 Unauthorized` hoặc `Connection Refused` từ Keycloak
- **Nguyên nhân:** Sai `KEYCLOAK_CLIENT_SECRET` hoặc Keycloak chưa sẵn sàng tiếp nhận request.
- **Khắc phục:**
  1. Mở Keycloak Admin Console tại `http://localhost:8080/admin`.
  2. Đăng nhập bằng `admin` / `admin`.
  3. Chọn Realm `omama` -> menu **Clients** -> chọn `identity-service-admin`.
  4. Mở tab **Credentials**, copy Client Secret dán vào file `.env`.
  5. Đảm bảo tab **Service account roles** đã được gán các quyền quản lý `realm-management` (`manage-users`, `view-users`).

### 4. Xung đột cổng (Port already allocated)
- **Nguyên nhân:** Cổng `8016`, `9016`, `5433` hoặc `8080` đang bị chiếm dụng bởi một ứng dụng khác trên máy.
- **Khắc phục:**
  - Kiểm tra tiến trình đang chiếm cổng:
    ```powershell
    # Windows:
    Get-Process -Id (Get-NetTCPConnection -LocalPort 8016).OwningProcess
    # Linux / macOS:
    lsof -i :8016
    ```
  - Hoặc đổi sang cổng khác trong `.env` (ví dụ `SERVER_PORT=8017`).
