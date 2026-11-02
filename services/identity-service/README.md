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
│   ├── docker-compose.all-in-one.yml    # Cụm All-in-One (Postgres, Keycloak, Kafka, Identity)
│   ├── docker-compose.yml               # Chỉ chạy riêng Identity Service trên Docker
│   ├── init-db.sql                      # Script tạo sẵn keycloak_db và identity_db
│   └── keycloak/
│       ├── .env.example                 # Mẫu env cho Keycloak độc lập
│       ├── docker-compose.yml           # Compose chạy riêng Keycloak
│       └── realm-export.template.json   # Template import sẵn Realm 'omama'
├── docs/                                # Tài liệu kỹ thuật chi tiết
├── src/
│   ├── main/
│   │   ├── java/com/hvduong/identity/
│   │   │   ├── api/                     # Interface API (Swagger / OpenAPI 3 annotations)
│   │   │   │   ├── account/             # UserLifecycleApi & impl/UserLifecycleController
│   │   │   │   ├── admin/               # IamRoleApi, EmployeeInvitationApi, IamAssignmentApi & impl/
│   │   │   │   ├── common/              # ApiResponse, GlobalExceptionHandler
│   │   │   │   └── registration/        # RegistrationApi & impl/RegistrationController
│   │   │   ├── application/             # Use Cases & Business Logic
│   │   │   │   ├── account/             # Service Interface & impl/ (Lifecycle, Invitation)
│   │   │   │   ├── audit/               # Service Interface & impl/ (Audit Log)
│   │   │   │   ├── authorization/       # Service Interface & impl/ (Specialized Perms)
│   │   │   │   ├── iam/                 # Service Interface & impl/ (Role, Assignment)
│   │   │   │   └── registration/        # RegisterPrincipalUseCase & impl/RegistrationOrchestrator
│   │   │   ├── config/                  # OpenApiConfig, KeycloakConfig
│   │   │   ├── grpc/                    # gRPC Services (Port 9016)
│   │   │   ├── infrastructure/
│   │   │   │   ├── keycloak/            # KeycloakAdminClient
│   │   │   │   ├── outbox/              # OutboxPublisher (Kafka)
│   │   │   │   └── persistence/
│   │   │   │       ├── entity/          # Row/Entity models (11 Row classes)
│   │   │   │       └── mybatis/mapper/  # MyBatis Mapper Interfaces
│   │   │   └── security/                # Spring Security JWT Resource Server
│   │   └── resources/
│   │       ├── application.yml          # Cấu hình Spring Boot
│   │       ├── db/migration/            # Flyway migrations (V1 -> V6)
│   │       └── mapper/                  # MyBatis XML Mapper definitions
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

| Biến môi trường | Giá trị mặc định | Mô tả |
| :--- | :--- | :--- |
| `SERVER_PORT` | `8016` | Cổng HTTP REST API |
| `GRPC_SERVER_PORT` | `9016` | Cổng gRPC Server nội bộ |
| `DB_HOST` | `localhost` | Hostname PostgreSQL (`postgres` nếu chạy trong mạng Docker, `host.docker.internal` nếu container gọi ra máy host) |
| `DB_PORT` | `5432` | Cổng kết nối PostgreSQL |
| `DB_NAME` | `identity_db` | Tên Database nghiệp vụ |
| `DB_USERNAME` | `postgres` | Tài khoản kết nối CSDL |
| `DB_PASSWORD` | `12345678` | Mật khẩu CSDL |
| `KEYCLOAK_SERVER_URL` | `https://keycloak.omama.vn` | URL của Keycloak (`http://localhost:8080` hoặc `http://keycloak:8080` khi test Docker) |
| `KEYCLOAK_REALM` | `omama` | Tên Realm làm việc trên Keycloak |
| `KEYCLOAK_CLIENT_ID` | `identity-service-admin` | Client ID kết nối Keycloak Admin REST API |
| `KEYCLOAK_CLIENT_SECRET`| `changeme` | Secret lấy từ Keycloak Admin Console |
| `KEYCLOAK_PUBLIC_CLIENT_ID`| `frontend-web` | Public Client ID OIDC cho PKCE Flow |
| `AUTH_REDIRECT_URI` | `https://api.omama.vn/api/v1/public/auth/callback` | Callback URL Keycloak chuyển hướng về sau khi login |
| `COOKIE_SECURE` | `false` | Cờ bảo mật Cookie (`false` khi chạy local dev HTTP, `true` trên Production HTTPS) |
| `COOKIE_SAME_SITE` | `Lax` | Thuộc tính SameSite của Cookie (`Lax` hoặc `None`) |
| `SPRING_KAFKA_BOOTSTRAP_SERVERS`| `localhost:9092` | Địa chỉ Kafka Broker (`kafka:9092` nếu chạy Docker) |

---

## 5. Hướng Dẫn Khởi Chạy Service (How to Run)

Bạn có thể lựa chọn 1 trong các phương thức khởi chạy dưới đây tùy thuộc vào mục đích sử dụng:

---

### Kịch Bản 1: Chạy Trực Tiếp Local Dev Bằng Gradle (Khuyên Dùng Để Lập Trình & Debug)

Phương án này chạy PostgreSQL, Keycloak và Kafka trên Docker, nhưng chạy `identity-service` trực tiếp bằng Java máy host để dễ dàng đặt Breakpoint, xem log màu và hot-reload.

#### Bước 1: Khởi động các hạ tầng phụ trợ trên Docker
Đứng tại thư mục `services/identity-service`:
```bash
docker compose -f deploy/docker-compose.all-in-one.yml up -d postgres keycloak kafka
```

#### Bước 2: Khởi chạy Identity Service bằng Gradle Wrapper
```bash
# Trên Windows (PowerShell / CMD):
.\gradlew.bat bootRun

# Trên Linux / macOS:
./gradlew bootRun
```
Ứng dụng sẽ tự động chạy Flyway migration để khởi tạo các bảng CSDL và sẵn sàng phục vụ tại cổng HTTP `8016` và gRPC `9016`.

---

### Kịch Bản 2: Chạy Toàn Bộ Cụm All-in-One Bằng Docker Compose (Kiểm Thử Môi Trường Đóng Gói)

Khởi tạo tự động **PostgreSQL + Keycloak + Kafka + Identity Service** cùng nhau trong mạng Docker nội bộ mà không cần cài đặt thêm gì trên máy host.

#### Bước 1: Build file JAR thực thi
> [!IMPORTANT]
> Dockerfile sử dụng file JAR đã biên dịch sẵn (`COPY build/libs/identity-service-*.jar app.jar`). Do đó bắt buộc phải build JAR trước:
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
docker compose -f deploy/docker-compose.all-in-one.yml down
# Thêm cờ -v nếu muốn xóa trắng dữ liệu database để khởi động lại từ đầu:
docker compose -f deploy/docker-compose.all-in-one.yml down -v
```

---

### Kịch Bản 3: Chạy Riêng Container Identity Service (Kết Nối Hạ Tầng Đang Có Sẵn)

Dùng khi bạn đã có cụm hạ tầng PostgreSQL, Keycloak chạy sẵn ở ngoài hoặc qua Docker khác.

```bash
# 1. Build file JAR:
.\gradlew.bat bootJar    # (hoặc ./gradlew bootJar trên Linux/Mac)

# 2. Khởi chạy riêng container:
cd deploy
docker compose up -d --build

# 3. Theo dõi log:
docker compose logs -f identity-service

# 4. Dừng container:
docker compose down
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
Báo cáo kiểm thử HTML chi tiết sẽ được tự động xuất tại:  
`services/identity-service/build/reports/tests/test/index.html`

---

## 6. Hướng Dẫn Kiểm Thử & Swagger Documentation

### 6.1. Swagger UI & OpenAPI 3 Documentation
Dịch vụ đã tích hợp sẵn giao diện Swagger UI để trực quan hóa và kiểm thử tất cả các REST API:

- **Swagger UI Web:** [http://localhost:8016/swagger-ui.html](http://localhost:8016/swagger-ui.html) (hoặc `http://localhost:8016/swagger-ui/index.html`)
- **OpenAPI JSON Spec:** [http://localhost:8016/v3/api-docs](http://localhost:8016/v3/api-docs)

> [!TIP]
> **Cách Ủy Quyền (Authorize) Trên Swagger UI:**
> 1. Bấm nút **Authorize** (biểu tượng ổ khóa màu xanh lá ở góc phải).
> 2. Dán chuỗi JWT Access Token nhận được từ Keycloak vào ô `Value`.
> 3. Bấm **Authorize** -> **Close**. Giờ bạn có thể test trực tiếp các API bảo mật.

---

### 6.2. Kiểm Tra Trạng Thái Sống (Healthcheck)
```bash
curl http://localhost:8016/actuator/health
```
**Kết quả mong đợi:**
```json
{"status":"UP"}
```

---

### 6.3. Đăng Ký Tài Khoản Khách Hàng Công Khai (`POST /api/v1/public/customer-registrations`)
Hỗ trợ cơ chế Idempotent chống đăng ký lặp (tuân thủ Chuẩn RESTful 3.3.1):

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

### 6.4. Xác Thực Theo Kiến Trúc Backend-for-Frontend (BFF Pattern - RFC 9700 & RFC 7636 PKCE)
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

### 6.5. Lấy Hồ Sơ & Quyền Của Bản Thân (`GET /api/v1/users/me`)
Gửi kèm JWT Access Token:

```bash
curl -X GET http://localhost:8016/api/v1/users/me \
  -H "Authorization: Bearer <YOUR_ACCESS_TOKEN>"
```

---

### 6.5. Cập Nhật Trạng Thái Người Dùng (`PATCH /api/v1/admin/users/{principalId}/status`)
Tuân thủ Quy tắc 1 (dùng HTTP `PATCH` thay vì động từ `suspend`/`reactivate` trong URI):

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

### 6.6. Quản Lý Danh Mục Vai Trò IAM (`GET /api/v1/admin/roles`)
```bash
curl -X GET http://localhost:8016/api/v1/admin/roles \
  -H "Authorization: Bearer <ADMIN_ACCESS_TOKEN>"
```

---

### 6.7. Quản Lý Thư Mời Cho Nhân Viên Mới (`/api/v1/admin/employee-invitations`)
Tuân thủ Quy tắc 1, 2, 4 (danh từ số nhiều, kebab-case, dùng `DELETE` thay vì động từ `revoke`):

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

### 6.8. Gán & Thu Hồi Vai Trò Chức Danh (`/api/v1/admin/role-assignments`)
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

### 6.9. Kiểm Thử gRPC Service (Port 9016)
Dịch vụ cung cấp gRPC Server tại cổng `9016` (được định nghĩa trong Proto `IdentityService`):
- `CheckSpecializedPermission`: Đánh giá quyền hạn đặc thù và kiểm soát rủi ro cao.
- `GetJwksPublicKey`: Trả về Public Key để các Microservices khác kiểm tra chữ ký token nội bộ.

Kiểm tra bằng công cụ `grpcurl`:
```bash
grpcurl -plaintext localhost:9016 list
```

---

## 7. Khắc Phục Sự Cố Thường Gặp (Troubleshooting)

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
- **Nguyên nhân:** Cổng `8016`, `9016`, `5432` hoặc `8080` đang bị chiếm dụng bởi một ứng dụng khác trên máy.
- **Khắc phục:**
  - Kiểm tra tiến trình đang chiếm cổng:
    ```powershell
    # Windows:
    Get-Process -Id (Get-NetTCPConnection -LocalPort 8016).OwningProcess
    # Linux / macOS:
    lsof -i :8016
    ```
  - Hoặc đổi sang cổng khác trong `.env` (ví dụ `SERVER_PORT=8017`).

