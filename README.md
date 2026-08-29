# HỆ SINH THÁI THƯƠNG MẠI ĐIỆN TỬ ĐA KÊNH & HỖ TRỢ QUYẾT ĐỊNH CHIẾN LƯỢC CHO NÔNG ĐẶC SẢN OCOP HUẾ (MÈ XỬNG O MẠ)

> **Dự án PBL / Đồ án Chuyên ngành Công nghệ Thông tin**  
> **Kiến trúc Hệ thống**: Phân tán Microservices + Event-Driven Architecture (EDA) + Polyglot Persistence (SQL + NoSQL) + Full Observability Stack.

---

## I. CẤU TRÚC THƯ MỤC CHUẨN MONOREPO (DIRECTORY STRUCTURE)

Dự án được tổ chức theo cấu trúc **Monorepo** chuẩn mực công nghiệp, phân tách độc lập giữa các ứng dụng khách (Apps), các dịch vụ nghiệp vụ (Microservices), các gói thư viện dùng chung (Packages), hạ tầng DevOps (Infra) và tài liệu kỹ thuật (Docs):

```
.
├── .github/                       # Quy trình tự động hóa CI/CD
│   └── workflows/
│       ├── ci.yml                 # Pipeline tự động: Linting, Unit Test, Contract Test
│       └── deploy.yml             # Pipeline tự động: Build Docker image & Auto-deploy
│
├── apps/                          # Tầng Trình diễn (Client Applications)
│   ├── web-user/                  # Website D2C Khách hàng (Next.js 14 App Router, SSR, SEO Schema.org)
│   ├── web-admin/                 # Web Portal Quản trị & BI Dashboard (React.js / Vite / ECharts / Shadcn UI)
│   └── mobile-app/                # Ứng dụng Di động Khách hàng (Flutter / React Native)
│
├── services/                      # Tầng Dịch vụ Nghiệp vụ (Microservices - Domain-Driven Design)
│   ├── api-gateway/               # API Gateway & BFF (Dynamic Routing, Auth Guard, Rate Limiting, Circuit Breaker)
│   ├── auth-service/              # Identity & Phân quyền RBAC (JWT, Refresh Token Redis, Argon2)
│   ├── catalog-service/           # Quản lý Danh mục, Sản phẩm OCOP, Biến thể SKU, Media MinIO S3
│   ├── order-service/             # Quản lý Giỏ hàng, Vòng đời Đơn hàng, SAGA Orchestrator
│   ├── payment-service/           # Cổng thanh toán VietQR động, Webhook listener, Idempotency Handler
│   ├── inventory-service/         # Quản lý Kho phân tán, Khóa chống over-selling (Redis Redlock)
│   ├── shipping-service/          # Tích hợp Giao vận 3PL (GHN/GHTK Mock API), Tracking thời gian thực
│   ├── notification-service/      # Cụm WebSocket Realtime (Socket.io), Firebase FCM Push Notification
│   └── analytics-service/         # BI & Strategic DSS, NoSQL Clickstream Tracking, Phân tích RFM & Dự báo
│
├── packages/                      # Thư viện & Module dùng chung trong toàn bộ Monorepo
│   ├── common/                    # Shared DTOs, Base Exceptions, Middleware, Logger tập trung
│   ├── events/                    # Định nghĩa cấu trúc Sự kiện cho Message Broker (Event Schemas)
│   └── proto/                     # Protocol Buffers / gRPC specs cho giao tiếp liên dịch vụ
│
├── infra/                         # Hạ tầng, DevOps, Containerization & Giám sát
│   ├── docker/
│   │   ├── docker-compose.infra.yml      # Cụm Middleware cục bộ (Postgres, Mongo, Redis, RabbitMQ, MinIO)
│   │   ├── docker-compose.monitoring.yml # Cụm Giám sát (Prometheus, Grafana, Jaeger, Loki)
│   │   └── docker-compose.yml            # Toàn bộ hệ sinh thái chạy trên Docker
│   ├── k8s/                       # Kubernetes Manifests / Helm charts
│   ├── nginx/                     # Cấu hình Ingress Reverse Proxy & SSL Termination
│   ├── monitoring/                # Cấu hình Prometheus, Grafana Dashboards, Loki Logging
│   └── scripts/                   # Scripts khởi tạo CSDL đa dịch vụ & Seed dữ liệu giả lập (1.000 orders)
│
├── docs/                          # Hồ sơ Tài liệu Kỹ thuật Chuẩn mực Học thuật
│   ├── 01_requirements/          # Khảo sát Người dùng, User Personas, User Stories, Use Case Specs
│   ├── 02_architecture/          # Tài liệu Thiết kế Kiến trúc SAD, SAGA Sequences, ERD chuẩn 3NF
│   ├── 03_api_specs/             # OpenAPI / Swagger Specifications & Event Protocols
│   ├── 04_testing/               # Kế hoạch Kiểm thử, Test Suites, Báo cáo Load Test k6
│   └── 05_deployment/            # Hướng dẫn Cài đặt, Triển khai & Vận hành Hệ thống
│
├── backlog_software.md            # Kế hoạch 12 tuần (6 Sprints) & Bảng phân rã Task chi tiết
└── README.md                      # Hướng dẫn tổng quan dự án
```

---

## II. BẢN ĐỒ CÔNG NGHỆ (TECH STACK REFERENCE)

| Phân hệ / Tầng | Công nghệ Sử dụng | Ghi chú & Mục đích |
| :--- | :--- | :--- |
| **Web Khách Hàng (D2C)** | **Next.js 14 (App Router)**, Tailwind CSS, TypeScript | Server-Side Rendering (SSR), Schema.org JSON-LD, SEO Lighthouse > 95. |
| **Web Quản Trị (Admin)** | **React.js (Vite)**, Shadcn UI / Ant Design, **ECharts / Recharts** | Dashboard quản lý đơn hàng realtime và trực quan hóa phân tích BI. |
| **Ứng Dụng Di Động** | **Flutter** (Dart) / React Native | Đồng bộ giỏ hàng, quét QR thanh toán, nhận thông báo đẩy Firebase FCM. |
| **API Gateway & BFF** | Node.js (NestJS/Express) hoặc Spring Cloud Gateway | Định tuyến động, Centralized JWT Auth, Redis Rate Limiting, Circuit Breaker. |
| **Core Microservices** | Node.js (NestJS/TypeScript) / Python FastAPI / Java Spring | Áp dụng mô hình Layered MVC / Clean Architecture (Controller - Service - Repository). |
| **Cơ sở Dữ liệu Quan hệ** | **PostgreSQL 16** (Database-per-Service) | Đảm bảo tính nhất quán ACID tuyệt đối cho đơn hàng, tài khoản, kho, thanh toán. |
| **CSDL NoSQL Hành vi** | **MongoDB 7.0** (Document Store) | Ghi nhận nhật ký Clickstream, từ khóa tìm kiếm, chuỗi điều hướng sản phẩm. |
| **Bộ nhớ đệm & Khóa** | **Redis 7.2** | Quản lý Session, Cache gợi ý tức thì, Khóa phân tán **Redis Redlock**. |
| **Message Broker** | **RabbitMQ 3.13** (AMQP) / Apache Kafka | Trục truyền thông sự kiện bất đồng bộ & điều phối SAGA Pattern. |
| **Object Storage** | **MinIO S3 Compatible** / Cloudinary | Lưu trữ hình ảnh sản phẩm OCOP chất lượng cao, video làng nghề, hóa đơn. |
| **Hệ thống Giám sát** | **Prometheus + Grafana + Jaeger + Loki** | Full 3 trụ cột Observability: Metrics, Tracing (TraceID) và Centralized Logs. |

---

## III. HƯỚNG DẪN KHỞI CHẠY HẠ TẦNG CỤC BỘ (LOCAL SETUP)

### 3.1. Yêu cầu Tiên quyết
- Đã cài đặt **Docker Desktop** (hỗ trợ Docker Compose v2).
- Node.js version `>= 18.x` hoặc runtime tương ứng.
- Git.

### 3.2. Khởi động Tầng Hạ tầng Dữ liệu & Middleware (Postgres, Mongo, Redis, RabbitMQ, MinIO)
Từ thư mục gốc của dự án, chạy lệnh sau:

```bash
docker compose -f infra/docker/docker-compose.infra.yml up -d
```

### 3.3. Các cổng truy cập Web Management của Hạ tầng:
- **PostgreSQL**: `localhost:5432` (User: `postgres`, Pass: `postgrespassword`)
- **MongoDB**: `localhost:27017` (User: `admin`, Pass: `mongopassword`)
- **Redis**: `localhost:6379` (Pass: `redispassword`)
- **RabbitMQ Management UI**: [http://localhost:15672](http://localhost:15672) (User: `rabbitmq`, Pass: `rabbitmqpassword`)
- **MinIO Console UI**: [http://localhost:9001](http://localhost:9001) (User: `minioadmin`, Pass: `minioadminpassword`)

---

## IV. TÀI LIỆU THAM KHẢO CHÍNH
- 📄 [backlog_software.md](file:///e:/DUT.K1N4/PBL/backlog_software.md): Kế hoạch phát triển chi tiết trong 12 tuần (6 Sprints).
- 📄 [docs/01_requirements/khao_sat_nguoi_dung_va_phan_tich_yeu_cau.md](file:///e:/DUT.K1N4/PBL/docs/01_requirements/khao_sat_nguoi_dung_va_phan_tich_yeu_cau.md): Khảo sát người dùng, phân tích đối thủ Thiên Hương, User Stories & Đặc tả Use Cases.
