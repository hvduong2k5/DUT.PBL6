# Hướng Dẫn Triển Khai Identity Service (Deployment Guide)

Tài liệu này hướng dẫn chi tiết cách deploy toàn bộ hệ sinh thái **Identity Service** (bao gồm PostgreSQL, Keycloak, Kafka, và Identity Service) bằng Docker Compose hoặc chạy thủ công.

---

## 1. Kiến Trúc Triển Khai

| Container / Service | Port Public | Mục đích |
| :--- | :--- | :--- |
| **postgres** | `5432` | Chứa 2 DB: `keycloak_db` và `identity_db` |
| **keycloak** | `8080` | Quản lý Auth, OIDC JWKS, SSO, Login Portal |
| **kafka** | `9092` | Message Broker (KRaft mode) nhận sự kiện Outbox (`UserRegistered`, `UserSuspended`) |
| **identity-service** | `8016` (HTTP) / `9016` (gRPC) | Microservice nghiệp vụ Spring Boot |

---

## 2. Các Bước Triển Khai Nhanh Bằng Docker Compose

### Bước 1: Build file JAR của Identity Service
Đứng tại thư mục `services/identity-service`:
```bash
./gradlew bootJar
```
*Kết quả:* Sinh ra file `build/libs/identity-service-0.0.1-SNAPSHOT.jar`.

### Bước 2: Khởi động toàn bộ cụm hạ tầng
Di chuyển vào thư mục `deploy/`:
```bash
cd deploy
docker compose up -d
```

### Bước 3: Kiểm tra trạng thái các Container
```bash
docker compose ps
```
Cả 4 container sẽ chuyển sang trạng thái `Up` (hoặc `healthy`).

---

## 3. Kiểm Thử Các Endpoint

### 3.1. Kiểm tra Healthcheck
```bash
curl http://localhost:8016/actuator/health
# Kết quả mong đợi: {"status":"UP"}
```

### 3.2. Đăng ký tài khoản Khách hàng (Idempotent Registration)
```bash
curl -X POST http://localhost:8016/api/v1/public/registration/customer \
  -H "Content-Type: application/json" \
  -d '{
    "idempotencyKey": "REG-TEST-001",
    "requestFingerprint": "fp-1234567890abcdef",
    "email": "khachhang@omama.vn",
    "firstName": "Văn Dương",
    "lastName": "Hoàng",
    "password": "Password123!"
  }'
```

### 3.3. Xem thông tin cá nhân hiện tại (`/api/v1/users/me`)
Gửi kèm JWT token nhận được từ Keycloak:
```bash
curl -X GET http://localhost:8016/api/v1/users/me \
  -H "Authorization: Bearer <KEYCLOAK_ACCESS_TOKEN>"
```

### 3.4. Vô hiệu hóa / Khóa tài khoản (`/api/v1/admin/users/{id}/suspend`)
```bash
curl -X POST http://localhost:8016/api/v1/admin/users/<PRINCIPAL_UUID>/suspend \
  -H "Authorization: Bearer <ADMIN_JWT_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"reason": "Tài khoản gian lận voucher"}'
```

---

## 4. Kiểm Tra gRPC Service

Service gRPC lắng nghe trên port `9016`.
Contract proto dùng chung từ `packages/proto/identity/v1/identity.proto`:
- `CheckSpecializedPermission`: Đánh giá quyền rủi ro cao (ngưỡng tiền `AMOUNT_THRESHOLD`, trần chiết khấu `PERCENT_THRESHOLD`).
- `GetJwksPublicKey`: Cung cấp public key dự phòng cho các client nội bộ.

