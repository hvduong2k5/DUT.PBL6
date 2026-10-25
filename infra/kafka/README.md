# Cụm Kafka Dùng Chung Cho Toàn Hệ Thống (Shared Kafka Cluster)

Cấu hình Kafka dùng chung cho toàn bộ các Microservices trong hệ thống O Mạ (`identity-service`, `order-service`, `inventory-service`, `catalog-service`,...).

---

## 🌟 Điểm nổi bật
1. **KRaft Mode (Native Apache Kafka 3.8):** Hoạt động không cần ZooKeeper, khởi động cực nhanh (< 3 giây), tối ưu tài nguyên RAM/CPU.
2. **Dual-Listeners (Hỗ trợ 2 cổng kết nối song song):**
   - **`kafka:9092` (INTERNAL):** Dành cho các Microservice container chạy chung mạng Docker.
   - **`localhost:9094` (EXTERNAL):** Dành cho ứng dụng chạy từ máy Host (IDE IntelliJ/VS Code, CLI, Postman...).
3. **Bảo mật Kafka UI (Form Login):** Hỗ trợ đặt User/Password cho giao diện Web khi gán Domain công khai trên Coolify.
4. **Auto-create Topics:** Tự động tạo topic khi có service produce message đầu tiên (VD: `identity.events.v1`).
5. **Dữ liệu bền vững:** Lưu tại volume `om_kafka_shared_data`.

---

## 🚀 Hướng dẫn khởi chạy

### Cách 1: Chạy Local (Docker Compose)
```bash
cd infra/kafka
cp .env.example .env
docker compose up -d
```
* Truy cập **Kafka UI**: [http://localhost:8085](http://localhost:8085)
* Tài khoản mặc định: `admin` / `OmamaSecretPass@2026` (hoặc đặt `KAFKA_UI_AUTH_TYPE=DISABLED` trong `.env` nếu muốn tắt đăng nhập).

### Cách 2: Triển khai trên Coolify (Có gán Domain)
1. Trên Coolify, chọn **+ Add Resource** -> **Docker Compose**.
2. Dán nội dung file `docker-compose.yml`.
3. Trong tab **Environment Variables**, cấu hình:
   ```env
   DOCKER_NETWORK_NAME=coolify
   DOCKER_NETWORK_EXTERNAL=true
   
   # Khóa cổng broker chỉ cho localhost/nội bộ truy cập
   KAFKA_EXTERNAL_BIND_IP=127.0.0.1
   
   # Mật khẩu quản trị Kafka UI qua Domain
   KAFKA_UI_AUTH_TYPE=LOGIN_FORM
   KAFKA_UI_USERNAME=admin
   KAFKA_UI_PASSWORD=MatKhauBaoMatCuaBan@123
   ```
4. Gán Domain cho service `kafka-ui` (VD: `https://kafka-ui.omama.vn`).
5. Bấm **Deploy**.

---

## 🔌 Cấu hình kết nối từ các Microservices

### 1. Nếu Microservice chạy bằng Docker Container (trong cùng mạng `om-network` hoặc `coolify`)
Khai báo biến môi trường:
```yaml
SPRING_KAFKA_BOOTSTRAP_SERVERS: kafka:9092
```

### 2. Nếu Microservice chạy trực tiếp từ máy tính/IDE (chạy local bằng `./gradlew bootRun`)
Khai báo trong `application.yml` hoặc file `.env`:
```yaml
spring:
  kafka:
    bootstrap-servers: localhost:9094
```

