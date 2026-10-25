# Cụm Kafka Dùng Chung Cho Toàn Hệ Thống (Shared Kafka Cluster)

Cấu hình Kafka dùng chung cho toàn bộ các Microservices trong hệ thống O Mạ (`identity-service`, `order-service`, `inventory-service`, `catalog-service`,...).

Hệ thống cung cấp **2 file compose chuyên biệt** cho từng môi trường:
1. **`docker-compose.prod.yml`**: Dành cho **Production trên Coolify** (Tối ưu bảo mật, không mở port VPS, Traefik domain routing, bảo vệ form login cho UI).
2. **`docker-compose.dev.yml`** (hoặc `docker-compose.yml`): Dành cho **Local Development** (Mở port 9094 cho IDE và 8085 cho Web UI, không cần đăng nhập).

---

## 📁 So sánh 2 phiên bản Compose

| Đặc điểm | `docker-compose.dev.yml` (Local Dev) | `docker-compose.prod.yml` (Coolify Prod) |
| :--- | :--- | :--- |
| **Mục đích** | Code & test trên máy cá nhân | Triển khai trên máy chủ VPS Coolify |
| **Mạng Docker** | `om-network` (bridge độc lập) | `coolify` (`external: true`) |
| **Port Broker Host** | `9094:9094` (để IDE/bootRun kết nối) | **Không map** (Tránh trùng port & lộ Internet) |
| **Port UI Host** | `8085:8080` (trực tiếp trình duyệt) | **Không map** (Traefik route từ Domain vào 8080) |
| **Xác thực Kafka UI** | `DISABLED` (Vào thẳng UI nhanh) | `LOGIN_FORM` (Bắt buộc nhập User/Password) |
| **Dữ liệu Volume** | `om_kafka_dev_data` | `om_kafka_prod_data` |

---

## 🚀 1. Hướng dẫn chạy Local Dev (Docker Desktop / Docker)

Tại thư mục gốc `infra/kafka`:
```bash
cd infra/kafka

# Khởi động cụm Kafka local (dùng docker-compose.dev.yml)
docker compose -f docker-compose.dev.yml up -d
# hoặc gõ ngắn gọn: docker compose up -d
```
* **Kafka UI:** Mở trình duyệt [http://localhost:8085](http://localhost:8085) (vào thẳng không cần mật khẩu).
* **Ứng dụng Local (Spring Boot trên IDE):** Kết nối qua `localhost:9094`.

---

## 🌐 2. Hướng dẫn triển khai Production trên Coolify

1. Đăng nhập vào Coolify Dashboard.
2. Chọn **Project / Environment** ➔ Chọn **+ Add Resource** ➔ Chọn **Docker Compose**.
3. **Copy toàn bộ nội dung file [`docker-compose.prod.yml`](./docker-compose.prod.yml)** dán vào ô Compose Content.
4. Chuyển sang tab **Environment Variables**, khai báo thông tin đăng nhập UI:
   ```env
   KAFKA_UI_USERNAME=admin
   KAFKA_UI_PASSWORD=MatKhauBaoMatCuaBan@2026
   ```
5. Tại mục cài đặt service **`kafka-ui`**, điền Domain của bạn (ví dụ: `https://kafka-ui.omama.vn`).
6. Bấm **Deploy**.

---

## 🔌 Cấu hình kết nối từ các Microservices

### A. Microservice chạy trong Docker container (Coolify hoặc Docker compose)
Khai báo biến môi trường:
```yaml
SPRING_KAFKA_BOOTSTRAP_SERVERS: kafka:9092
```

### B. Microservice chạy từ máy cá nhân (chạy qua IDE / `./gradlew bootRun`)
Khai báo trong file `.env` hoặc `application.yml`:
```yaml
spring:
  kafka:
    bootstrap-servers: localhost:9094
```
