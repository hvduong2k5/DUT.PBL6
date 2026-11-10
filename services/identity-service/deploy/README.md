# Hướng Dẫn Triển Khai Identity Service (Deployment Guide)

Tài liệu này hướng dẫn chi tiết cách triển khai cụm **Identity Service** (bao gồm PostgreSQL, Keycloak, Kafka, và Spring Boot Identity Service) bằng Docker Compose cho môi trường Local Dev và Production.

---

## 1. Kiến Trúc Cụm All-in-One (Local Dev)

| Container / Service | Port Host | Port Container | Mục đích |
| :--- | :--- | :--- | :--- |
| **postgres** | `5433` | `5432` | PostgreSQL 16 (tự khởi tạo 2 DB: `keycloak_db` và `identity_db`) |
| **keycloak** | `8080` | `8080` | Keycloak 24 (chế độ start-dev, auto-import realm `omama`, lean JWT) |
| **kafka** | `9092` | `9092` | Apache Kafka 3.7 (KRaft mode) nhận sự kiện Outbox (`UserRegistered`, `UserSuspended`) |
| **identity-service** | `8016` / `9016` | `8016` / `9016` | Spring Boot 3.3.4 (REST API port 8016, gRPC port 9016) |

---

## 2. Các Bước Triển Khai Nhanh

### Cách 1: Khởi động All-in-One bằng Docker Compose (Khuyên dùng khi kiểm thử container)

#### Bước 1: Build file JAR của Identity Service
Đứng tại thư mục `services/identity-service`:
```bash
# Windows:
.\gradlew.bat bootJar

# Linux / macOS:
./gradlew bootJar
```
*Kết quả:* Sinh ra file `build/libs/identity-service-0.0.1-SNAPSHOT.jar`.

#### Bước 2: Khởi động toàn bộ cụm hạ tầng
```bash
cd deploy
docker compose -f docker-compose.all-in-one.yml up -d --build
```

#### Bước 3: Kiểm tra trạng thái các Container
```bash
docker compose -f docker-compose.all-in-one.yml ps
```
Cả 4 container sẽ chuyển sang trạng thái `Up` (hoặc `healthy`).

---

### Cách 2: Chạy Hạ Tầng Phụ Trợ Trên Docker + Chạy Ứng Dụng Trên Máy Host (Khuyên dùng khi lập trình & debug)

```bash
# 1. Khởi động PostgreSQL, Keycloak và Kafka trên Docker:
cd services/identity-service
docker compose -f deploy/docker-compose.all-in-one.yml up -d postgres keycloak kafka

# 2. Chạy Spring Boot trên máy host:
.\gradlew.bat bootRun
```
*Lưu ý khi chạy trên host:* DB PostgreSQL lắng nghe tại port **`5433`** (`DB_PORT=5433`).

---

## 3. Kiểm Thử Các Endpoint

### 3.1. Kiểm tra Healthcheck
```bash
curl http://localhost:8016/actuator/health
# Kết quả mong đợi: {"status":"UP"}
```

### 3.2. Đăng ký tài khoản Khách hàng (Idempotent Customer Registration)
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

### 3.3. Xem thông tin cá nhân hiện tại (`/api/v1/users/me`)
Gửi kèm JWT token nhận được từ Keycloak:
```bash
curl -X GET http://localhost:8016/api/v1/users/me \
  -H "Authorization: Bearer <KEYCLOAK_ACCESS_TOKEN>"
```

### 3.4. Cập nhật trạng thái người dùng (`/api/v1/admin/users/{principalId}/status`)
```bash
curl -X PATCH http://localhost:8016/api/v1/admin/users/<PRINCIPAL_UUID>/status \
  -H "Authorization: Bearer <ADMIN_JWT_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"status": "SUSPENDED", "reason": "Tài khoản gian lận voucher"}'
```

---

## 4. Kiểm Tra gRPC Service

Service gRPC lắng nghe trên port `9016`.
Contract proto dùng chung từ `packages/proto/identity/v1/identity.proto`:
- `CheckSpecializedPermission`: Đánh giá quyền rủi ro cao (ngưỡng tiền `AMOUNT_THRESHOLD`, trần chiết khấu `PERCENT_THRESHOLD`).
- `GetJwksPublicKey`: Cung cấp public key dự phòng cho các client nội bộ.
