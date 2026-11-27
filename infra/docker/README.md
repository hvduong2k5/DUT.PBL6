# Shared Infrastructure Docker Setup (`infra/docker`)

Thư mục này chứa cấu hình Docker cho các dịch vụ hạ tầng dùng chung của hệ thống O Mạ (DUT.PBL6). Mỗi thành phần được tách thành thư mục riêng để có thể build và deploy độc lập trên Coolify hoặc chạy gom qua `docker-compose.infra.yml` ở local.

---

## 1. Cấu trúc Thư mục

```text
infra/docker/
├── postgres/          # PostgreSQL 16 + init-databases.sql (om_*_db, bao gồm om_media_db)
├── redis/             # Redis 7.2 (Cache, Session, Rate Limit)
├── rabbitmq/          # RabbitMQ 3.13 Management
├── minio/             # MinIO S3 Object Storage (API :9000, Console :9001)
├── adminer/           # Adminer Web UI quản lý PostgreSQL (:8080)
└── docker-compose.infra.yml   # Chạy nhanh toàn bộ hạ tầng ở môi trường local
```

Mỗi thư mục đều có:
- `Dockerfile`: Bản **Local / Dev** (cài sẵn giá trị mặc định để chạy nhanh).
- `Dockerfile.coolify`: Bản **Production / Coolify** (không hardcode mật khẩu, tích hợp `HEALTHCHECK`).
- `.env.example`: Danh sách biến môi trường mẫu cần khai báo.

---

## 2. Chạy ở Môi trường Local

### Cách 1: Chạy toàn bộ hạ tầng bằng Docker Compose
```powershell
docker compose -f infra/docker/docker-compose.infra.yml up -d
```
- PostgreSQL: `localhost:5432` (`postgres` / `postgrespassword`)
- Redis: `localhost:6379` (password: `redispassword`)
- RabbitMQ: `localhost:5672` | UI: `http://localhost:15672` (`rabbitmq` / `rabbitmqpassword`)
- MinIO API: `http://localhost:9000` | Console: `http://localhost:9001` (`minioadmin` / `minioadminpassword`)
- Adminer UI: `http://localhost:8088` (Server: `postgres`)

*(Nếu cần bật thêm MongoDB dự phòng clickstream: thêm `--profile clickstream`)*

### Cách 2: Build & chạy từng container riêng lẻ
```powershell
docker build -t om-minio ./infra/docker/minio
docker run -d --name om-minio -p 9000:9000 -p 9001:9001 -v minio_data:/data om-minio
```

---

## 3. Deploy từng Service Độc lập trên Coolify (Production)

Khi tạo Resource mới (Dockerfile-based) trên Coolify cho từng hạ tầng:

| Service | Base Directory | Dockerfile Location | Volume cần mount | Port nội bộ (Expose) |
| :--- | :--- | :--- | :--- | :--- |
| **PostgreSQL** | `/infra/docker/postgres` | `/Dockerfile.coolify` | `/var/lib/postgresql/data` | `5432` (không public) |
| **Redis** | `/infra/docker/redis` | `/Dockerfile.coolify` | `/data` | `6379` (không public) |
| **RabbitMQ** | `/infra/docker/rabbitmq` | `/Dockerfile.coolify` | `/var/lib/rabbitmq` | `5672`, `15672` |
| **MinIO** | `/infra/docker/minio` | `/Dockerfile.coolify` | `/data` | `9000` (S3 API), `9001` (Console) |
| **Adminer** | `/infra/docker/adminer` | `/Dockerfile.coolify` | *(Không cần)* | `8080` (Web UI) |

### Lưu ý cấu hình trên Coolify:
1. **Environment Variables:** Copy các key từ file `.env.example` trong từng thư mục và đặt mật khẩu mạnh trên UI Coolify.
2. **Network:** Đảm bảo các container hạ tầng và microservices cùng nằm trong mạng `coolify` để giao tiếp nội bộ qua tên container/service (ví dụ `postgres:5432`, `minio:9000`, `redis:6379`).
3. **Domain / Traefik:**
   - **MinIO (tích hợp sẵn UI Console trong cùng container):**
     - Khai báo 2 domain trên Coolify: `https://s3.omama.vn:9000` (S3 API) và `https://minio.omama.vn:9001` (Web Console UI).
     - Nhớ thêm 2 biến môi trường `MINIO_SERVER_URL=https://s3.omama.vn` và `MINIO_BROWSER_REDIRECT_URL=https://minio.omama.vn` để giao diện Web UI preview/login không bị lỗi redirect.
   - **Adminer:** Trỏ domain quản trị vào port `8080` và nên bật **Basic Auth** trên Traefik hoặc giới hạn IP/VPN.
