# Identity & Access Management (IAM) Service — O Mạ PBL6

Dịch vụ Quản lý Định danh & Phân quyền tập trung (Identity & Access Management) cho hệ sinh thái Thương mại điện tử **Mè Xửng O Mạ**.

---

## 1. Giới Thiệu & Kiến Trúc

`identity-service` áp dụng mô hình phân tách trách nhiệm rõ ràng (**Boundary Pattern**) giữa Identity Provider và Domain Authorization:

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

## 2. Cấu Trúc Thư Mục

```
services/identity-service/
├── deploy/                              # Hạ tầng triển khai Docker
│   ├── docker-compose.yml               # Cụm Postgres, Keycloak, Kafka, Identity Service
│   ├── init-db.sql                      # Tự động tạo keycloak_db và identity_db
│   └── keycloak/
│       └── realm-export.template.json   # Template import sẵn Realm 'omama'
├── docs/                                # Tài liệu kỹ thuật chi tiết
│   ├── architecture.md                  # Kiến trúc hệ thống
│   ├── database-design.md               # Thiết kế CSDL (ERD, Flyway)
│   ├── keycloak-identity-boundary.md    # Ranh giới giữa Keycloak & Service
│   └── runbook.md                       # Vận hành & Troubleshooting
├── src/
│   ├── main/
│   │   ├── java/com/hvduong/identity/   # Mã nguồn nghiệp vụ
│   │   │   ├── api/                     # REST Controllers (Account, Admin IAM, Registration)
│   │   │   ├── application/             # Use cases (Lifecycle, IAM, Audit, Specialized Perms)
│   │   │   ├── grpc/                    # gRPC Services (Port 9016)
│   │   │   ├── infrastructure/          # Keycloak Client, Outbox Kafka, MyBatis Mappers
│   │   │   └── security/                # Spring Security OIDC Resource Server
│   │   └── resources/
│   │       ├── application.yml          # Cấu hình Spring Boot
│   │       └── db/migration/            # Flyway migrations (V1 -> V6)
│   └── test/                            # Bộ kiểm thử Unit Test & Mockito (32 tests)
├── .env.example                         # Mẫu biến môi trường
├── build.gradle                         # Gradle build config
├── Dockerfile                           # Container hóa Identity Service
└── README.md                            # Hướng dẫn này
```

---

## 3. Yêu Cầu Môi Trường (Prerequisites)

- **Java JDK:** Phiên bản **21 LTS** (Khuyên dùng Microsoft HotSpot hoặc Eclipse Temurin).
- **Docker & Docker Compose:** Docker Desktop (v20+) hỗ trợ Compose V2.
- **Gradle:** Đã tích hợp sẵn wrapper (`./gradlew` hoặc `gradlew.bat`).
- **Postman / cURL / Insomnia:** Để kiểm thử các API endpoints.

---

## 4. Hướng Dẫn Cấu Hình Biến Môi Trường

Sao chép file `.env.example` thành `.env`:
```bash
cp .env.example .env
```

Nội dung cấu hình chi tiết:

| Biến môi trường | Giá trị mặc định (Docker) | Ý nghĩa |
| :--- | :--- | :--- |
| `SERVER_PORT` | `8016` | Cổng HTTP REST API |
| `GRPC_SERVER_PORT` | `9016` | Cổng gRPC Server |
| `DB_HOST` | `postgres` (hoặc `localhost`) | Địa chỉ PostgreSQL Host |
| `DB_PORT` | `5432` | Cổng PostgreSQL |
| `DB_NAME` | `identity_db` | Tên CSDL nghiệp vụ của Identity Service |
| `DB_USERNAME` | `postgres` | Tài khoản kết nối DB |
| `DB_PASSWORD` | `password123` | Mật khẩu kết nối DB |
| `KEYCLOAK_SERVER_URL`| `http://keycloak:8080` | URL gốc của Keycloak |
| `KEYCLOAK_REALM` | `omama` | Realm làm việc chính |
| `KEYCLOAK_CLIENT_ID` | `identity-service-admin` | Client ID kết nối Admin REST API |
| `KEYCLOAK_CLIENT_SECRET`| *(Lấy từ Keycloak Admin)* | Secret xác thực của Client Admin |
| `SPRING_KAFKA_BOOTSTRAP_SERVERS` | `kafka:9092` | Cổng Kafka Broker (Outbox) |

---

## 5. Chuẩn Bị Cơ Sở Dữ Liệu PostgreSQL (Đã có sẵn Server)

Vì bạn **đã có sẵn một máy chủ PostgreSQL** (on-premise, cloud hoặc docker host riêng), bạn không cần chạy container Postgres nữa mà chỉ cần tạo 2 cơ sở dữ liệu:

```sql
-- Đăng nhập vào PostgreSQL server hiện có của bạn và chạy lệnh:
CREATE DATABASE keycloak_db;
CREATE DATABASE identity_db;
```

---

## 6. Hướng Dẫn Triển Khai Keycloak Độc Lập (Standalone)

Keycloak được đóng gói trong một file Docker Compose riêng biệt tại thư mục [`deploy/keycloak/`](./deploy/keycloak/):

### Bước 1: Cấu hình biến môi trường cho Keycloak
Di chuyển vào thư mục `deploy/keycloak` và sao chép cấu hình:
```bash
cd deploy/keycloak
cp .env.example .env
```
Mở file `.env` và điền thông tin máy chủ PostgreSQL của bạn:
```properties
POSTGRES_HOST=192.168.1.100    # Địa chỉ IP / Host của PostgreSQL server có sẵn
POSTGRES_PORT=5432
KEYCLOAK_DB_NAME=keycloak_db   # DB dành riêng cho Keycloak
POSTGRES_USER=postgres
POSTGRES_PASSWORD=your_db_password

KEYCLOAK_PORT=8080
KEYCLOAK_ADMIN=admin
KEYCLOAK_ADMIN_PASSWORD=admin
```

### Bước 2: Khởi chạy Keycloak
```bash
docker compose up -d
```
Container `omafood-keycloak-standalone` sẽ kết nối trực tiếp đến PostgreSQL server của bạn và tự động import realm **`omama`** từ file `realm-export.template.json`.

### Bước 3: Lấy `KEYCLOAK_CLIENT_SECRET` cho Identity Service
1. Truy cập Keycloak Admin Console: `http://<KEYCLOAK_HOST>:8080/admin`
2. Đăng nhập bằng `admin` / `admin` (hoặc thông tin trong `.env`).
3. Chuyển sang realm **`omama`** (menu dropdown trên cùng bên trái).
4. Vào menu **Clients** -> Chọn client **`identity-service-admin`**.
5. Chọn tab **Credentials** -> Copy chuỗi **Client Secret**.
6. Vào tab **Service account roles** -> Bấm **Assign role** -> Lọc **Filter by clients** -> Chọn `realm-management` và gán: `manage-users`, `view-users`, `manage-realm`.

---

## 7. Hướng Dẫn Triển Khai Identity Service Độc Lập

File [`deploy/docker-compose.yml`](./deploy/docker-compose.yml) được thiết kế **chỉ triển khai duy nhất `identity-service`**, kết nối tới PostgreSQL server và Keycloak đã chạy:

### Bước 1: Cấu hình biến môi trường tại thư mục `services/identity-service`
Tạo file `.env` (từ `.env.example`):
```properties
SERVER_PORT=8016
GRPC_SERVER_PORT=9016

# Trỏ đến PostgreSQL server có sẵn của bạn
DB_HOST=192.168.1.100
DB_PORT=5432
DB_NAME=identity_db
DB_USERNAME=postgres
DB_PASSWORD=your_db_password

# Trỏ đến Keycloak độc lập vừa chạy ở Mục 6
KEYCLOAK_SERVER_URL=http://192.168.1.100:8080
KEYCLOAK_REALM=omama
KEYCLOAK_CLIENT_ID=identity-service-admin
KEYCLOAK_CLIENT_SECRET=<secret_vừa_lấy_ở_bước_6.3>

# Trỏ đến Kafka (nếu có)
SPRING_KAFKA_BOOTSTRAP_SERVERS=192.168.1.100:9092
```

### Bước 2: Build file JAR
Đứng tại thư mục `services/identity-service`:
```bash
# Windows:
.\gradlew.bat bootJar

# Linux / MacOS:
./gradlew bootJar
```

### Bước 3: Khởi chạy Identity Service
```bash
cd deploy
docker compose up -d
```
Xem log khởi động của service:
```bash
docker compose logs -f identity-service
```
Identity Service sẽ:
1. Kết nối đến `identity_db` trên server PostgreSQL của bạn và tự động chạy Flyway migration (`V1` -> `V6`).
2. Kết nối đến Keycloak để xác thực và đồng bộ.
3. Sẵn sàng nhận request tại cổng `8016` (HTTP) và `9016` (gRPC).

*(Ghi chú: Nếu bạn muốn chạy thử nghiệm toàn bộ cụm gồm cả Postgres, Kafka, Keycloak local all-in-one trên một máy duy nhất để test nhanh, có thể dùng file [`deploy/docker-compose.all-in-one.yml`](./deploy/docker-compose.all-in-one.yml) với lệnh `docker compose -f docker-compose.all-in-one.yml up -d`).*

---

### Lựa Chọn B: Chạy Local Development (Dành cho lập trình viên code và debug)

Phương án này chạy Postgres, Keycloak, Kafka trên Docker, nhưng chạy ứng dụng Spring Boot trực tiếp trên máy để xem log và live-reload nhanh:

#### Bước 1: Khởi động các dịch vụ phụ trợ trên Docker
```bash
cd deploy
docker compose up -d postgres keycloak kafka
```

#### Bước 2: Cập nhật file `.env` tại thư mục `services/identity-service`
Vì chạy ngoài Docker nên địa chỉ host trỏ về `localhost`:
```properties
DB_HOST=localhost
DB_PORT=5432
DB_NAME=identity_db
DB_USERNAME=postgres
DB_PASSWORD=password123

KEYCLOAK_SERVER_URL=http://localhost:8080
KEYCLOAK_REALM=omama
KEYCLOAK_CLIENT_ID=identity-service-admin
KEYCLOAK_CLIENT_SECRET=<secret_lay_o_buoc_5.3>

SPRING_KAFKA_BOOTSTRAP_SERVERS=localhost:9092
```

#### Bước 3: Chạy ứng dụng bằng Gradle Wrapper
```bash
# Windows PowerShell / CMD:
.\gradlew.bat bootRun

# Linux / MacOS:
./gradlew bootRun
```

Dịch vụ sẽ tự động:
1. Kết nối PostgreSQL và kích hoạt Flyway chạy migration từ `V1` đến `V6`.
2. Lắng nghe HTTP REST tại cổng `8016`.
3. Lắng nghe gRPC Server tại cổng `9016`.

---

## 8. Các Bảng Cơ Sở Dữ Liệu Chính (`identity_db`)

Quá trình Flyway migration tự động thiết lập các bảng sau:

| Tên bảng | Chức năng |
| :--- | :--- |
| `identity_principals` | Định danh tài khoản, liên kết IDP (`idp_subject`), phân loại (`CUSTOMER`, `EMPLOYEE`), trạng thái (`ACTIVE`, `SUSPENDED`). |
| `iam_roles` | Danh mục vai trò (`role_code`, `role_name`, `is_protected`). |
| `iam_permissions` | Danh mục quyền hạt nhân (`resource_code`, `action_code`, `risk_level`). |
| `iam_role_permissions`| Bảng quan hệ N-N giữa Role và Permission. |
| `iam_role_assignments`| Gán vai trò cho Principal kèm theo Scope (`GLOBAL`, `WAREHOUSE`, `STORE`), ngày hiệu lực. |
| `outbox_events` | Bảng Outbox lưu sự kiện (`UserRegistered`, `UserSuspended`) trước khi bắn sang Kafka. |
| `identity_audit_logs` | Nhật ký kiểm toán mọi hành động nhạy cảm trong hệ thống. |

---

## 9. Hướng Dẫn Kiểm Thử Các API Chính

### 9.1. Kiểm tra Healthcheck
```bash
curl http://localhost:8016/actuator/health
```
*Kết quả:* `{"status":"UP"}`

### 9.2. Đăng ký tài khoản Khách hàng (`POST /api/v1/public/registration/customer`)
Endpoint hỗ trợ chống trùng lặp bằng `idempotencyKey`:
```bash
curl -X POST http://localhost:8016/api/v1/public/registration/customer \
  -H "Content-Type: application/json" \
  -d '{
    "idempotencyKey": "REG-TEST-0001",
    "requestFingerprint": "fp-1234567890abcdef",
    "email": "customer01@omama.vn",
    "firstName": "Mai",
    "lastName": "Trần",
    "rawPassword": "Password@123"
  }'
```
*Kết quả:* Tạo tài khoản đồng thời trên Keycloak và bảng `identity_principals`, tự động ghi sự kiện Outbox `identity.events.v1.UserRegistered`.

### 9.3. Xem thông tin cá nhân & Danh sách quyền (`GET /api/v1/users/me`)
Gửi kèm Header Authorization mang Access Token từ Keycloak:
```bash
curl -X GET http://localhost:8016/api/v1/users/me \
  -H "Authorization: Bearer <ACCESS_TOKEN>"
```
*Kết quả:* Trả về Profile, `principalType`, danh sách `roles` kèm scope, và toàn bộ `permissions` gộp.

### 9.4. Lấy danh sách Vai trò IAM (`GET /api/v1/admin/access/roles`)
```bash
curl -X GET http://localhost:8016/api/v1/admin/access/roles \
  -H "Authorization: Bearer <ADMIN_ACCESS_TOKEN>"
```

### 9.5. Khóa tài khoản người dùng (`POST /api/v1/admin/users/{id}/suspend`)
```bash
curl -X POST http://localhost:8016/api/v1/admin/users/e9f8d7c6-1111-2222-3333-444455556666/suspend \
  -H "Authorization: Bearer <ADMIN_ACCESS_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"reason": "Tài khoản có dấu hiệu gian lận"}'
```
*Kết quả:* Chuyển trạng thái Principal sang `SUSPENDED`, đồng thời vô hiệu hóa user tương ứng trên Keycloak (`enabled=false`).

---

## 10. Khắc Phục Sự Cố Thường Gặp (Troubleshooting)

### 1. Lỗi `401 Unauthorized` hoặc `Connection Refused` khi gọi Keycloak
- **Nguyên nhân:** Sai `KEYCLOAK_CLIENT_SECRET` hoặc Keycloak chưa khởi động xong.
- **Xử lý:** Vào Keycloak Admin Console kiểm tra lại Client Secret của `identity-service-admin` và đảm bảo biến môi trường trỏ đúng cổng/host.

### 2. Lỗi `403 Forbidden` khi Identity Service tạo User trên Keycloak
- **Nguyên nhân:** Client `identity-service-admin` chưa được cấp quyền quản trị user.
- **Xử lý:** Làm lại **Bước 5.4**, gán role `realm-management -> manage-users` cho Service Account.

### 3. Cảnh báo đỏ trên VS Code: `'Flexible Constructor Bodies' is only available with source level 25`
- **Nguyên nhân:** Bug phân tích cú pháp giữa Eclipse JDT Language Server trên VS Code và Lombok khi dùng `@RequiredArgsConstructor`.
- **Xử lý:** 
  - Thay `@RequiredArgsConstructor` bằng constructor tường minh (Explicit Constructor).
  - Hoặc nâng cấp version Lombok lên `1.18.36` trong `build.gradle` rồi chạy lệnh `Java: Clean Java Language Server Workspace` trên VS Code.

