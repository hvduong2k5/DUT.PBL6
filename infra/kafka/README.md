# Cụm Kafka Dùng Chung Cho Toàn Hệ Thống (Shared Kafka Cluster)

Cấu hình Kafka dùng chung cho toàn bộ các Microservices trong hệ thống O Mạ (`identity-service`, `order-service`, `inventory-service`, `catalog-service`,...).

---

## 🌟 Điểm nổi bật
1. **KRaft Mode (Native Apache Kafka 3.8):** Hoạt động không cần ZooKeeper, khởi động cực nhanh (< 3 giây), tối ưu tài nguyên RAM/CPU.
2. **Dual-Listeners (Hỗ trợ 2 cổng kết nối song song):**
   - **`kafka:9092` (INTERNAL):** Dành cho các Microservice container chạy chung mạng Docker.
   - **`localhost:9094` (EXTERNAL):** Dành cho ứng dụng chạy từ máy Host (IDE IntelliJ/VS Code, CLI, Postman...).
3. **Tích hợp sẵn Kafka UI:** Quản lý Topics, Events, Consumers qua giao diện trực quan tại: `http://localhost:8085`.
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
Truy cập **Kafka UI**: [http://localhost:8085](http://localhost:8085)

### Cách 2: Triển khai trên Coolify
1. Trên Coolify, chọn **+ Add Resource** -> **Docker Compose**.
2. Dán nội dung file `docker-compose.yml`.
3. Trong tab **Environment Variables**, cấu hình:
   ```env
   DOCKER_NETWORK_NAME=coolify
   DOCKER_NETWORK_EXTERNAL=true
   KAFKA_EXTERNAL_HOST=your-server-ip-or-domain
   ```
4. Bấm **Deploy**.

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

