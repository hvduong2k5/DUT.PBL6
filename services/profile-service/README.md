# MS-15: PROFILE SERVICE
## HỆ THỐNG THƯƠNG MẠI ĐIỆN TỬ & CHUỖI CUNG ỨNG ĐẶC SẢN OCOP HUẾ (MÈ XỬNG O MẠ)

---

## 1. Giới Thiệu Chung
`profile-service` là microservice quản lý toàn bộ hồ sơ khách hàng, sổ địa chỉ giao hàng, liên kết đơn hàng vãng lai và quản lý hồ sơ nhân sự thợ xưởng kẹo mè xửng đạt chuẩn **OCOP 4 sao tỉnh Thừa Thiên Huế**.

Dịch vụ được thiết kế theo kiến trúc **Clean / Hexagonal Architecture** với ngôn ngữ **Go 1.22+**, bảo đảm hiệu năng siêu việt và độ trễ thấp trên đường găng mua hàng:
- **Port HTTP REST (Chi Router):** `8080` (Kết nối với Kong API Gateway cho Web Next.js & Mobile Flutter).
- **Port gRPC Server:** `50051` (Kết nối nội bộ cho `MS-04 order-service` với cam kết SLA $P99 \le 5\text{ms}$).

---

## 2. Các Giải Thuật & Cơ Chế Kỹ Thuật Lõi
1. **Mô Hình Hành Chính 2 Cấp (Chuẩn Quốc Gia Sau 01/07/2025):** Tỉnh/Thành phố trực thuộc TW $\rightarrow$ Xã/Phường. Bãi bỏ hoàn toàn cấp huyện/thị xã, tinh giản schema và giảm dung lượng payload.
2. **Atomic Default Address Switcher:** Chống Race Condition bằng Transaction kết hợp `SELECT ... FOR UPDATE` và PostgreSQL Partial Unique Index.
3. **Mã Hóa PII Chuẩn Envelope Encryption (DEK/KEK):** Tách biệt DEK mã hóa dữ liệu cục bộ (AES-256-GCM) và KEK quản lý bởi HashiCorp Vault Transit Engine. Xoay khóa định kỳ (Zero-Downtime Re-wrapping) không cần downtime hay giải mã lại ciphertext cũ.
4. **Bộ Đệm Đa Tầng Kết Hợp Pub/Sub Invalidation:** L1 RAM LRU (`golang-lru/v2`) + L2 Redis Cluster (`go-redis/v9`). Xóa sạch cache chéo giữa các Pod khi có thao tác ghi để bảo vệ tính nhất quán của Address Snapshot.
5. **Kiểm Soát Đồng Thời Bằng Khóa Lạc Quan (CAS):** Cột `version INT` phân biệt rành mạch giữa HTTP 404 (Not Found) và HTTP 409 (Conflict).
6. **Transactional Outbox Pattern:** Quét sự kiện nền bằng `SELECT ... FOR UPDATE SKIP LOCKED` và đẩy sang Apache Kafka topic `profile.events.v1`.
7. **Giám Sát Tuân Thủ Chứng Chỉ VSATTP OCOP:** Cron Worker quét định kỳ hạn chứng chỉ của thợ nấu kẹo trước 30 ngày và 7 ngày.

---

## 3. Cấu Trúc Thư Mục
```text
services/profile-service/
├── cmd/server/main.go            # Entrypoint: Boot HTTP, gRPC, Workers, Signal trap
├── internal/
│   ├── config/                   # Đọc biến môi trường
│   ├── domain/                   # Domain models & Error types
│   ├── infrastructure/
│   │   ├── cache/                # DualLayerCache (L1 RAM + L2 Redis + Pub/Sub)
│   │   ├── kafka/                # Pure-Go Kafka producer
│   │   └── security/             # EnvelopeEncryptor & Vault Transit Client
│   ├── repository/               # PostgreSQL repositories (Customer, Address, Employee, Claim)
│   ├── transport/
│   │   ├── grpc/                 # gRPC Server (GetDeliveryAddress SLA P99 <= 5ms)
│   │   └── http/                 # Chi Router, Middlewares (Auth, Metrics, Logger) & Handlers
│   ├── usecase/                  # Business logic (Customer, Address, Employee, 2-Stage Fuzzy)
│   └── worker/                   # Outbox Publisher & Compliance Checker
├── migrations/                   # DDL Migrations (Up & Down)
├── deploy/
│   ├── docker/Dockerfile         # Multi-stage distroless (<25MB)
│   ├── k8s/                      # Kubernetes manifests (Deployment, Service, ConfigMap, HPA)
│   └── monitoring/               # Prometheus alert rules & Grafana dashboard
├── tests/
│   ├── benchmark/                # SLA P99 Benchmark tests
│   ├── contract/                 # Pact Provider verification tests
│   ├── integration/              # Concurrency (100 goroutines), Cache invalidation, Claims tests
│   └── testhelper/               # Testcontainers Go (Postgres 16, Redis 7)
├── docker-compose.yml            # Local development orchestration stack
└── Dockerfile                    # Production Docker build file
```

---

## 4. Hướng Dẫn Khởi Chạy Local Nhanh (1 Command)

Khởi chạy toàn bộ hạ tầng gồm Profile Service, PostgreSQL 16, Redis 7, HashiCorp Vault và Redpanda Kafka:

```bash
docker compose up -d --build
```

Kiểm tra trạng thái dịch vụ:
```bash
# Health probes
curl -i http://localhost:8080/livez
curl -i http://localhost:8080/readyz

# Prometheus metrics
curl -s http://localhost:8080/metrics | grep profile_
```

---

## 5. Hướng Dẫn Chạy Toàn Bộ Test Suite (TDD)

```bash
# 1. Chạy Unit Tests nhanh (Red-Green verification)
go test -v -race -short ./internal/...

# 2. Chạy toàn bộ Integration Tests (Testcontainers PG 16 & Redis 7)
go test -v -race ./tests/integration/... ./internal/repository/...

# 3. Chạy Pact Provider Contract Test
go test -v ./tests/contract/...

# 4. Chạy Benchmark SLA P99 <= 5ms
go test -bench=BenchmarkGetDeliveryAddressCheckout_SLA_P99_Sub5ms -benchmem ./tests/benchmark/...

# 5. Đo Coverage
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

---

## 6. Danh Mục Biến Môi Trường (Environment Variables)

| Biến Môi Trường | Giá Trị Mặc Định | Mô Tả |
| :--- | :--- | :--- |
| `HTTP_PORT` | `8080` | Port lắng nghe HTTP REST API |
| `GRPC_PORT` | `50051` | Port lắng nghe gRPC Server nội bộ |
| `DATABASE_URL` | `postgres://omamx_user:omamx_password@localhost:5432/profile_db?sslmode=disable` | Chuỗi kết nối PostgreSQL 16 |
| `REDIS_ADDR` | `localhost:6379` | Địa chỉ Redis Cluster / Instance |
| `VAULT_ADDR` | `http://localhost:8200` | Địa chỉ máy chủ HashiCorp Vault |
| `VAULT_TOKEN` | `root` | Token xác thực Vault Transit Engine |
| `VAULT_KEY_NAME` | `profile-pii-kek` | Tên khóa KEK Transit dùng mã hóa DEK |
| `KAFKA_BROKERS` | `localhost:9092` | Danh sách địa chỉ Kafka Broker |
| `TOPIC_PROFILE_EVENTS` | `profile.events.v1` | Kafka Topic tiếp nhận sự kiện Profile |
| `OUTBOX_POLL_INTERVAL_MS`| `500` | Chu kỳ quét bảng outbox (mili-giây) |
| `OUTBOX_BATCH_SIZE` | `50` | Số lượng sự kiện outbox xử lý mỗi batch |
