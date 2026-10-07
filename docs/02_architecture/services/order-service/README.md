# BỘ TÀI LIỆU THIẾT KẾ KIẾN TRÚC CHI TIẾT (LLD): MS-04 ORDER SERVICE
## TRUNG TÂM XỬ LÝ ĐƠN HÀNG ĐA KÊNH, ĐIỀU PHỐI SAGA & ĐỐI SOÁT VIETQR — MÈ XỬNG O MẠ
### PHIÊN BẢN: 2.0 (MODULAR IMPLEMENTATION-READY SPECIFICATION)

---

## 📌 BẢN ĐỒ TÀI LIỆU KIẾN TRÚC MODULAR (DOCUMENT INDEX)

Nhằm tối ưu hóa trải nghiệm đọc, bảo trì độc lập và phân công công việc song song giữa các nhóm kỹ sư (Domain Modeling, Database, Saga/Backend, API/Integration, QA/Testing), toàn bộ bản thiết kế LLD của `order-service` được chia thành **6 Phân Hệ Chuyên Đề (Thematic Modules)** kèm tệp Master Index:

```text
docs/02_architecture/services/order-service/
├── README.md                                  # [BẠN ĐANG Ở ĐÂY] Bản đồ điều hướng & Kiến trúc tổng thể
├── ms04_order_service_design.md               # Tài liệu đặc tả tổng hợp 11 bước (Master Index All-in-One)
│
├── 01_order_domain_and_boundary.md            # [PHÂN HỆ 1] Ranh giới nghiệp vụ & Mô hình miền DDD
├── 02_state_machine_and_lifecycle.md          # [PHÂN HỆ 2] Máy trạng thái FSM & Vòng đời đơn hàng
├── 03_database_and_persistence.md             # [PHÂN HỆ 3] DDL PostgreSQL 16, Redis Cache & Repositories
├── 04_usecases_and_saga_orchestration.md      # [PHÂN HỆ 4] 5 Use Cases, Latency Budget & Điều phối Saga
├── 05_api_contracts_and_transports.md         # [PHÂN HỆ 5] REST OpenAPI, gRPC Protobuf & Kafka CloudEvents
└── 06_testing_and_failure_recovery.md         # [PHÂN HỆ 6] Clean Architecture, 4 Concurrency Tests & Failure Matrix
```

---

## 🗺️ TỔNG HỢP NHANH CÁC PHÂN HỆ THIẾT KẾ (MODULE SUMMARY)

### 1. [Phân Hệ 1: Ranh Giới Nghiệp Vụ & Mô Hình Miền DDD](01_order_domain_and_boundary.md)
- **Nội dung:** Bước 1 (Bounded Context & Scope Isolation) và Bước 2 (Domain Model & Invariants).
- **Điểm then chốt:**
  - Quy định quyền sở hữu dữ liệu độc quyền: Order Service quản lý Order Status và Snapshots; Kho (MS-01) quản lý Inventory; Khuyến mãi (MS-07) quản lý Voucher.
  - Phân loại 3 trạng thái tồn kho: `available` $\rightarrow$ `reserved` $\rightarrow$ `committed`.
  - Mô hình tiền tệ `Money` đơn giản hóa theo đồng VND (`int64`, cấm dùng float/nanos).
  - Snapshot địa chỉ giao hàng: Phân tích đánh đổi riêng tư PII (Lựa chọn Phương án A - Lưu trực tiếp bản chụp địa chỉ giao hàng tại Order Service để đảm bảo tính độc lập vận hành và cách ly ngữ cảnh).
  - 5 Bất biến miền (Domain Invariants), bao gồm luật chiết khấu $\le$ tổng tiền hàng và khớp thanh toán chính xác.
- **Xem chi tiết:** 👉 [Đọc Phân Hệ 1](01_order_domain_and_boundary.md)

### 2. [Phân Hệ 2: Máy Trạng Thái FSM & Vòng Đời Đơn Hàng](02_state_machine_and_lifecycle.md)
- **Nội dung:** Bước 3 (State Machine & Lifecycle Transitions).
- **Điểm then chốt:**
  - Máy trạng thái đơn hàng đầy đủ 14 trạng thái (`DRAFT` $\rightarrow$ `COMPLETED` / `REFUNDED`).
  - Máy trạng thái Saga phân tán: `STARTED` $\rightarrow$ `IN_PROGRESS` $\rightarrow$ `COMPENSATING` $\rightarrow$ `COMPENSATED` / `DEAD_LETTER`.
  - Ma trận chuyển trạng thái kèm điều kiện bảo vệ (Guards) và tác vụ phụ (Side Effects).
  - Cơ chế giải quyết tranh chấp trạng thái cạnh tranh tại giây thứ 900 (Second-899 Race Condition) giữa **Webhook thanh toán** và **Timeout Sweeper** bằng câu lệnh Atomic CAS trên PostgreSQL.
- **Xem chi tiết:** 👉 [Đọc Phân Hệ 2](02_state_machine_and_lifecycle.md)

### 3. [Phân Hệ 3: Mô Hình Dữ Liệu & Tầng Lưu Trữ](03_database_and_persistence.md)
- **Nội dung:** Bước 4 (Database Schema DDL & Cache Storage) và Bước 5 (Repository & Data Access Interfaces).
- **Điểm then chốt:**
  - Sơ đồ quan hệ ERD đầy đủ 7 bảng.
  - Chiến lược khóa chính UUID v7 sinh trực tiếp từ Application Layer (khẳng định chuẩn kỹ thuật PostgreSQL 16).
  - Kịch bản DDL chuẩn production với ràng buộc kiểm tra nghiệp vụ: `CHECK (discount_units <= subtotal_units)` và `CHECK (final_units = subtotal_units - discount_units + shipping_units)`.
  - Bảng `payments` quan hệ 1:N với ràng buộc `UNIQUE (payment_provider, provider_transaction_id)`.
  - Trừu tượng hóa giao dịch CSDL an toàn kiểu `DBTX` và các Repository Interfaces độc lập.
- **Xem chi tiết:** 👉 [Đọc Phân Hệ 3](03_database_and_persistence.md)

### 4. [Phân Hệ 4: Ca Sử Dụng & Điều Phối Saga](04_usecases_and_saga_orchestration.md)
- **Nội dung:** Bước 6 (Application Use Cases & Saga Engine).
- **Điểm then chốt:**
  - Tách bạch vai trò Order Domain Component vs Saga Orchestrator Component.
  - Phân bổ ngân sách độ trễ (Checkout Latency Budget) cam kết **P95 $\sim$ 140ms** (vượt chuẩn NFR-01 < 200ms).
  - 5 ca sử dụng chi tiết: Checkout D2C song song qua `errgroup`, VietQR Webhook đối soát 5 kịch bản (khớp chính xác tuyệt đối `amount == final`), Timeout Compensation Saga tập trung (Model A), Marketplace Inbound Saga, và Bán lẻ tại quầy POS Huế.
- **Xem chi tiết:** 👉 [Đọc Phân Hệ 4](04_usecases_and_saga_orchestration.md)

### 5. [Phân Hệ 5: Giao Thức Mạng & Đặc Tả Hợp Đồng](05_api_contracts_and_transports.md)
- **Nội dung:** Bước 7 (Delivery Layer & Contracts) và Bước 8 (Cross-Cutting Concerns).
- **Điểm then chốt:**
  - Hợp đồng REST API OpenAPI 3.0 cho `POST /api/v1/checkout`.
  - Hợp đồng gRPC Protobuf nội bộ cho `CreatePOSOrder`, `GetOrderDetail`, `CancelOrder` và các gRPC Client trên Critical Path.
  - Danh mục sự kiện Kafka Transactional Outbox chuẩn CNCF CloudEvents 1.0 JSON Schema.
  - Cam kết nhất quán phân tán: **At-least-once Delivery + Idempotent Consumer = Effectively-once Business Outcome**.
  - Bảo vệ đa tầng: Redlock, Circuit Breaker (ngưỡng 2.0s / 50% lỗi), và xác thực Webhook HMAC-SHA256 bằng thuật toán thời gian cố định `ConstantTimeCompare`.
- **Xem chi tiết:** 👉 [Đọc Phân Hệ 5](05_api_contracts_and_transports.md)

### 6. [Phân Hệ 6: Kiểm Thử, Khôi Phục Sự Cố & Truy Vết](06_testing_and_failure_recovery.md)
- **Nội dung:** Bước 9 (Project Directory Blueprint), Bước 10 (Testing & Traceability), và Bước 11 (Distributed Failure & Recovery Matrix).
- **Điểm then chốt:**
  - Bố cục thư mục Go Clean / Hexagonal Architecture chuẩn mực.
  - 4 kịch bản kiểm thử phân tán sống còn (Distributed Concurrency Tests): Race Condition Webhook vs Timeout, Outbox Publisher Crash & Restart, Inventory Ambiguous Timeout, Downstream Idempotent Consumer.
  - Ma trận truy vết yêu cầu chức năng (FR-06 đến FR-28) và phi chức năng (NFR-01 đến NFR-11).
  - Ma trận sự cố phân tán thực chiến gồm 10 kịch bản lỗi mạng, timeout và crash.
- **Xem chi tiết:** 👉 [Đọc Phân Hệ 6](06_testing_and_failure_recovery.md)

---

## 🏛️ SƠ ĐỒ ĐỊNH VỊ TOÀN CẢNH HỆ THỐNG

```text
                                       KONG API GATEWAY
                                              │
                      ┌───────────────────────┴───────────────────────┐
                      │ HTTP REST (North-South)                       │ GraphQL (BFF)
                      ▼                                               ▼
         ┌─────────────────────────────────────────────────────────────────────────┐
         │                    MS-04: ORDER SERVICE (CORE DOMAIN)                   │
         │  ┌───────────────────────────────┐     ┌─────────────────────────────┐  │
         │  │     ORDER DOMAIN ENGINE       │     │      SAGA ORCHESTRATOR      │  │
         │  │  • Quản lý Aggregate Order    │     │  • Điều phối Checkout Saga  │  │
         │  │  • Snapshots bất biến         │     │  • Quản lý saga_instances   │  │
         │  │  • Tính toán số học Money     │     │  • Kích hoạt đền bù tập trung│  │
         │  │  • FSM Order Transitions      │     │    (Centralized Model A)    │  │
         │  └───────────────────────────────┘     └─────────────────────────────┘  │
         └────────┬─────────────────────┬───────────────────────┬──────────────────┘
                  │ gRPC (Critical)     │ gRPC (Critical)       │ Kafka Events (Async)
                  ▼                     ▼                       ▼
         MS-01: INVENTORY       MS-05: CATALOG          APACHE KAFKA BROKER CLUSTER
         (Port: 8001)           (Port: 8005)            (order.events.v1)
```

---

## 🧭 HƯỚNG DẪN ĐỌC THEO VAI TRÒ KỸ SƯ (ROLE-BASED READING GUIDE)

| Vai Trò | Trọng Tâm Nghiên Cứu | Thứ Tự Tài Liệu Khuyên Đọc |
| :--- | :--- | :--- |
| **Backend / Golang Engineer** | Nắm chắc Domain Model, Use Cases, CAS FSM và cấu trúc thư mục code. | [Phân Hệ 1](01_order_domain_and_boundary.md) $\rightarrow$ [Phân Hệ 2](02_state_machine_and_lifecycle.md) $\rightarrow$ [Phân Hệ 4](04_usecases_and_saga_orchestration.md) $\rightarrow$ [Phân Hệ 6](06_testing_and_failure_recovery.md) |
| **Database Administrator (DBA)** | Nghiên cứu DDL PostgreSQL 16, Check Constraints, Indexes, UUID v7 và Cache Schema. | [Phân Hệ 3: Mô hình dữ liệu & Tầng lưu trữ](03_database_and_persistence.md) |
| **Integration / Frontend Engineer** | Xem đặc tả REST APIs, gRPC Services và cơ chế quét mã VietQR. | [Phân Hệ 5: Giao thức mạng & Đặc tả hợp đồng](05_api_contracts_and_transports.md) |
| **QA / Automation Test Engineer** | Xây dựng Test Suite cho 4 bài kiểm thử phân tán và đối chiếu ma trận truy vết FR/NFR. | [Phân Hệ 6: Kiểm thử & Khôi phục sự cố](06_testing_and_failure_recovery.md) |
| **DevOps / SRE / System Architect** | Nghiên cứu Latency Budget, Circuit Breaker, Outbox Worker và Ma trận phục hồi sự cố 10 kịch bản. | [Phân Hệ 4](04_usecases_and_saga_orchestration.md) $\rightarrow$ [Phân Hệ 5](05_api_contracts_and_transports.md) $\rightarrow$ [Phân Hệ 6](06_testing_and_failure_recovery.md) |
