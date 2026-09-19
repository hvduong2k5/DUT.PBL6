# CẨM NANG KIẾN TRÚC & HIỆN TRẠNG HỆ THỐNG (PROJECT BIBLE)

> **Dự án:** Hệ sinh thái thương mại điện tử đa kênh & Hỗ trợ quyết định chiến lược cho Nông đặc sản OCOP Huế (Mè xửng O Mạ) - PBL6 / DUT  
> **Nguồn chân lý duy nhất (Single Source of Truth - SSOT)** cho kiến trúc, nghiệp vụ và hiện trạng kỹ thuật.

---

## 1. TỔNG QUAN & SỨ MỆNH DỰ ÁN

Hệ thống được thiết kế để giải quyết bài toán chuỗi cung ứng và phân phối nông đặc sản OCOP (Mè xửng O Mạ - Thừa Thiên Huế) theo mô hình D2C (Direct to Consumer) kết hợp B2B & O2O (Online to Offline):
- **Tính đặc thù nông sản:** Quản lý hạn sử dụng nghiêm ngặt theo lô/date (FEFO), truy xuất nguồn gốc OCOP.
- **Tính toàn vẹn giao dịch:** Chống bán vượt tồn kho (Anti-overselling) trong các đợt flash-sale hoặc đơn hàng lớn.
- **Hỗ trợ quyết định chiến lược (DSS):** Phân tích dữ liệu hành vi (Clickstream), phân khúc khách hàng RFM và dự báo nhu cầu tái nhập kho bằng Machine Learning/BI.

---

## 2. CẤU TRÚC MONOREPO & BẢN ĐỒ THÀNH PHẦN

```
DUT.PBL6/
├── apps/                          # Tầng Client / Tương tác người dùng
│   ├── web-user/                  # Website D2C Khách hàng (Next.js 14 App Router, SSR, SEO Schema.org)
│   ├── web-admin/                 # Web Admin & BI Dashboard (React Vite, Shadcn UI, ECharts)
│   └── mobile-app/                # Mobile App Khách hàng (Flutter / React Native)
├── services/                      # 18 Bounded Context Microservices (Database-per-Service)
│   ├── api-gateway/               # API Gateway & Mobile BFF (Reverse proxy, Zero-Trust, Rate limit)
│   ├── auth-service/              # Identity & RBAC (JWT, Redis token store, Argon2)
│   ├── catalog-service/           # Quản lý danh mục & SKU OCOP, Media S3 MinIO
│   ├── inventory-service/         # Quản lý tồn kho đa kho, Redis Redlock, xuất kho FEFO
│   ├── order-service/             # Quản lý vòng đời đơn hàng, Saga Orchestrator
│   ├── payment-service/           # Tích hợp VietQR động, Webhook Idempotency
│   ├── shipping-service/          # Tích hợp đơn vị vận chuyển 3PL (GHN/GHTK)
│   ├── notification-service/      # Realtime WebSockets & Firebase Cloud Messaging
│   └── analytics-service/         # BI & DSS, Clickstream MongoDB, RFM Modeling
├── packages/                      # Gói thư viện chia sẻ nội bộ
│   ├── common/                    # Exceptions chuẩn, Logging middleware, Base DTOs
│   ├── events/                    # JSON Schema / CloudEvents cho Kafka Broker
│   └── proto/                     # Protobuf definitions cho gRPC liên dịch vụ
├── infra/                         # Hạ tầng & DevOps
│   ├── docker/                    # docker-compose.infra.yml, docker-compose.monitoring.yml
│   ├── k8s/                       # Kubernetes manifests
│   ├── nginx/                     # Ingress & SSL Termination
│   └── monitoring/                # Prometheus, Grafana, Jaeger, Loki
├── docs/                          # Hồ sơ tài liệu kỹ thuật chuẩn mực
│   ├── 01_requirements/          # 22 Epics, Product Backlog, NFR, FR
│   ├── 02_architecture/          # Bounded Context, Domain Design, High Level Design, SAD
│   ├── 03_api_specs/             # OpenAPI & Event specs
│   ├── 04_testing/               # Kịch bản kiểm thử, Load test k6
│   └── 05_deployment/            # Hướng dẫn triển khai & vận hành
├── task.md                        # Tracker tiến độ công việc (SSOT)
└── project_bible.md               # Cẩm nang kiến trúc hệ thống (SSOT)
```

---

## 3. KIẾN TRÚC MIỀN & 18 BOUNDED CONTEXTS (SHARED-NOTHING)

Áp dụng nguyên tắc **Database-per-Service (Shared-Nothing Architecture)**:
1. **Tuyệt đối cấm truy cập CSDL chéo:** Mỗi microservice độc quyền quản lý CSDL riêng (PostgreSQL schema độc lập hoặc Mongo collection độc lập).
2. **Ngôn ngữ chung (Ubiquitous Language):** Mỗi context định nghĩa thực thể theo góc nhìn của miền đó. Ví dụ:
   - Trong `catalog-service`: Thực thể là `ProductListing` mang tính chất tiếp thị, hình ảnh, mô tả OCOP.
   - Trong `inventory-service`: Thực thể là `StockItem` với `BatchNumber`, `ExpiryDate`, `PhysicalQuantity`, `ReservedQuantity`.
   - Trong `order-service`: Thực thể là `OrderItemSnapshot` bất biến tại thời điểm chốt đơn.
   - Trong `shipping-service`: Thực thể là `Parcel` (trọng lượng, kích thước, địa chỉ lấy/giao).

---

## 4. MÔ HÌNH 3 MẶT PHẲNG KIẾN TRÚC (3 ARCHITECTURAL PLANES)

Hệ thống phân tách rạch ròi 3 luồng tín hiệu để đảm bảo an toàn và hiệu năng:

1. **Business Plane (Mặt phẳng Nghiệp vụ):**
   - Phục vụ luồng mua bán, đặt hàng, xử lý kho.
   - Trục truyền thông: Kafka Topics nghiệp vụ (`order.created`, `inventory.reserved`, `payment.completed`).
2. **Audit Plane (Mặt phẳng Kiểm toán & An ninh):**
   - Lưu vết bằng chứng an ninh (Security Evidence).
   - Cơ chế: **Entity-Level Hash Chain (Tamper-Evident)** vào Append-only Database.
   - Luồng sự kiện chạy qua Kafka Topic riêng biệt (`audit.events`), không bao giờ chặn (block) luồng mua sắm chính.
3. **Observability Plane (Mặt phẳng Giám sát & Viễn trắc):**
   - Metrics, Distributed Traces (TraceID/SpanID), Centralized Logs.
   - **Quy tắc tuyệt đối:** Không gửi dữ liệu telemetry qua Kafka của hệ sinh thái, mà được đẩy trực tiếp qua giao thức **OTLP (OpenTelemetry Protocol)** sang OpenTelemetry Collector -> Prometheus / Jaeger / Loki.

---

## 5. KHUNG 4 QUY TẮC GIAO TIẾP HỆ THỐNG (4 COMMUNICATION PATTERNS)

1. **Ingress Communication (Client -> API Gateway / BFF):**
   - Giao thức: RESTful API (HTTPS/JSON) cho Web/Admin; GraphQL qua Mobile BFF cho thiết bị di động.
   - Zero-Trust Gateway: Thực hiện TLS Termination, Rate limiting, Local JWKS validation, Header sanitization (xóa header client gửi lên) và inject các Header tin cậy (`X-User-Id`, `X-User-Role`).
2. **Synchronous Internal Critical Path (Service <-> Service):**
   - Chỉ dùng cho các thao tác bắt buộc cần kết quả tức thì (ví dụ `order-service` gọi `inventory-service` để Reserve Stock).
   - Giao thức: **gRPC** (Protobuf binary, HTTP/2).
   - Bắt buộc áp dụng: Strict Timeout (tối đa 2.000ms), Retry (exponential backoff) và Circuit Breaker.
3. **Asynchronous Decoupled Processing (Event-Driven):**
   - Dùng cho toàn bộ các bước downstream (Thông báo, Xuất kho đóng gói, Giao vận, Phân tích BI).
   - Giao thức: **Kafka Event Bus** định dạng chuẩn CloudEvents JSON.
4. **Telemetry Ingestion:**
   - Đẩy thẳng qua **OTLP** sang OpenTelemetry Collector.

---

## 6. CÁC CƠ CHẾ KỸ THUẬT QUAN TRỌNG

### 6.1. Hybrid Saga Orchestrator & Centralized Compensation
- **Orchestrator trung tâm:** Đặt tại `order-service`.
- **Bước Critical (gRPC Đồng bộ):** Order gọi `ReserveStock` sang Inventory. Nếu timeout hoặc hết hàng -> Hủy đơn ngay lập tức, không sinh event thừa.
- **Bước Downstream (Kafka Bất đồng bộ):** Khi khách thanh toán qua VietQR, `payment-service` bắn event `payment.succeeded`. Saga Orchestrator nhận event và tuần tự kích hoạt các dịch vụ tiếp theo.
- **Đền bù tập trung (Compensation):** Khi xảy ra sự cố thanh toán quá hạn hoặc hủy đơn, duy nhất Saga Orchestrator được quyền phát lệnh gRPC `ReleaseReservation` để mở lại tồn kho.

### 6.2. Chống Bán Vượt Tồn Kho (Anti-Overselling) & FEFO
- Sử dụng **Redis Redlock** hoặc Redis Lua Script để kiểm tra và giữ chỗ tồn kho khả dụng (`available = physical - reserved`) ở cấp độ vi giây.
- Sau khi giữ chỗ thành công trên Redis, ghi nhận gRPC xuống PostgreSQL của `inventory-service`.
- Phân bổ xuất kho tự động theo quy tắc **FEFO (First Expired, First Out)**: Lô nào cận date nhất sẽ được gán để giao trước, giảm thiểu hao hụt nông sản.

### 6.3. Tích hợp Thanh toán VietQR & Idempotency Key
- Tạo mã VietQR động chứa mã hóa đơn và số tiền chính xác.
- Webhook tiếp nhận thanh toán từ ngân hàng/cổng thanh toán bắt buộc kiểm tra **Idempotency Key** lưu trên Redis/Postgres để ngăn chặn xử lý duplicate webhook khi ngân hàng retry.

### 6.4. Mobile BFF & GraphQL Aggregation
- Thay vì Mobile App phải gửi hàng chục request REST để gom dữ liệu (Sản phẩm + Đánh giá + Tồn kho + Khuyến mãi), Mobile BFF gom các truy vấn nội bộ và trả về kết quả gói gọn trong 1 request GraphQL duy nhất, tối ưu hóa thời lượng pin và chất lượng mạng chập chờn.

---

## 7. HIỆN TRẠNG KỸ THUẬT & QUYẾT ĐỊNH THIẾT KẾ CHO TỪNG SERVICE

### 7.1. MS-01: `inventory-service`
- **Tài liệu thiết kế chi tiết:** [`docs/inventory_architecture_design.md`](file:///d:/PROJECT/PBL6/DUT.PBL6/docs/inventory_architecture_design.md) & [`docs/inventory_database_design.md`](file:///d:/PROJECT/PBL6/DUT.PBL6/docs/inventory_database_design.md).
- **Bộ kịch bản kiểm thử Unit Tests:** [`docs/inventory_unit_tests.md`](file:///d:/PROJECT/PBL6/DUT.PBL6/docs/inventory_unit_tests.md) (Bao phủ Invariants, FEFO Allocator, Use Cases với Mock, Idempotency và Background Workers).
- **Tech Stack:** **Go (Golang)** + **PostgreSQL 16** (`inventory_db`) + **Redis 7.2** (Redlock) + **Apache Kafka** (Saga Events) + **gRPC**.
- **Kiến trúc nội bộ:** Clean Architecture kết hợp DDD (Domain -> Application/Use Cases -> Infrastructure -> Presentation).
- **Cơ sở dữ liệu:** Chuẩn 3NF với các bảng `inventory_items`, `batches`, `stock_reservations`, `stock_reservation_allocations`, `stock_adjustments`, `idempotency_keys`. Tích hợp các ràng buộc CHECK và Partial Index cho FEFO query (`exp_date ASC` với `status IN ('ACTIVE', 'NEAR_EXPIRY')`). Migration DDL: `migrations/000001_init_inventory_schema.up.sql`.
- **Luồng xử lý:** Kết hợp 2 tầng khóa: Redis Redlock (TTL 3s) và Postgres `SELECT ... FOR UPDATE` (Read Committed) để chống race condition và over-selling.
- **Hiện trạng mã nguồn (Hoàn thành Bước 4 - Core Slices & Đã Kiểm Thử Chặt Chẽ):**
  - `Slice 4.1`: Đã khởi tạo module `dut-pbl6/inventory-service` và toàn bộ Domain Entities / Value Objects (`InventoryItem`, `Batch`, `StockReservation`, Domain errors). Đạt 96.5% statement coverage.
  - `Slice 4.2`: Đã triển khai thuật toán phân bổ FEFO (`fefo_allocator.go`), bảo vệ nghiêm ngặt `INV-BI-02` (Zero Expired Sale - tự động hủy điều kiện phân bổ của lô quá hạn), ưu tiên lô cận hạn `NEAR_EXPIRY`, kiểm tra tính nhất quán SKU và vượt qua 100% tests (`UT-INV-DOMAIN-01..03`, `UT-INV-FEFO-01..06` + edge cases). Đạt 93.2% statement coverage.
  - `Slice 4.3`: Đã thiết lập DDL migration PostgreSQL chuẩn hóa cột `status` trong `inventory_items` và chỉ mục FEFO (`status IN ('ACTIVE', 'NEAR_EXPIRY')`), Repository implementation hỗ trợ `SELECT ... FOR UPDATE`, `PostgresTxManager`, và `PostgresIdempotencyRepository`. Kiểm thử toàn diện bằng `sqlmock` đạt 79.6% statement coverage.
  - `Slice 4.4`: Đã triển khai 3 Use Cases cốt lõi (`ReserveStockUseCase`, `ReleaseReservationUseCase`, `CommitStockDeductionUseCase`), Redis Redlock distributed lock, cơ chế Canonical Lock Sorting chống Deadlock đa SKU, gộp SKU trùng lặp (Duplicate SKU consolidation), sửa lỗi double-release trên trạng thái `EXPIRED` và kiểm thử 100% test cases (`UT-INV-APP-01..06` + edge cases). Đạt 79.9% statement coverage.
  - `Slice 4.4b (Hardening Chống Deadlock PostgreSQL & Redis Redlock Toàn diện)`:
    + **Chuẩn hóa Lock Hierarchy một chiều (Items -> Batches):** Khắc phục triệt để lỗi nghịch đảo thứ tự khóa (Inversion) giữa `ReserveStock` (Items -> Batches) và `ReleaseReservation` / `CommitStockDeduction` (trước đây Batches -> Items). Trong Release & Commit, row lock trên `inventory_items` theo `sortedSKUs` luôn được thực hiện trước, sau đó mới gom `batch_id` sắp xếp canonical (UUID string ASC) để khóa và cập nhật `batches`.
    + **Bảo vệ toàn diện bằng Redis Redlock:** Bổ sung `port.LockService` vào `ReleaseReservationUseCase` và `CommitStockDeductionUseCase`. Gom và sắp xếp danh sách SKU theo thứ tự từ điển, acquire lock trước khi mở DB transaction và đảm bảo giải phóng an toàn qua `defer`. Khi xảy ra lock contention giữa chừng, toàn bộ các lock đã acquire trước đó đều được giải phóng triệt để.
    + **Đồng bộ tính Đơn định (Domain Determinism):** Thêm `ReserveAt(qty int, now time.Time) error` vào entity `Batch`, kết hợp `CanAllocateAt(now)` và tái sử dụng trong `AllocateByFEFOAt`. Phương thức `Reserve(qty)` ủy quyền cho `ReserveAt(qty, time.Now())`.
  - `Slice 4.4c (Phòng vệ Context Timeout Leak, Transaction Hygiene, UTC & Outbox DDL)`:
    + **Sửa lỗi Context Timeout Leak trong `defer ReleaseLock`:** Tách context bằng `context.WithoutCancel(ctx)` kết hợp `context.WithTimeout(..., 2*time.Second)` trên cả 3 Use Cases (`ReserveStock`, `ReleaseReservation`, `CommitStockDeduction`). Bổ sung nil-context guard (`if ctx == nil { ctx = context.Background() }`) chống panic và guard `if len(lockedKeys) == 0 { return }` tránh lãng phí timer resource.
    + **Phòng vệ Transaction trong `PostgresTxManager`:** Thêm `defer func() { _ = tx.Rollback() }()` ngay sau khi tạo `sqlTx` đảm bảo rollback an toàn khi xảy ra runtime panic hoặc lỗi trả về từ callback. Bổ sung nil checks cho DB và callback function.
    + **Chuẩn hóa Timezone UTC toàn diện:** Đổi toàn bộ `time.Now()` thành `time.Now().UTC()` trên toàn bộ Domain Entities (`StockReservation`, `InventoryItem`, `Batch`) và tầng PostgreSQL Repository updates (`UpdateItem`, `UpdateReservation`).
    + **Cải tiến độ bền RedlockService:** Giữ token trong sync.Map khi script Lua `Eval` gặp lỗi mạng tạm thời, cho phép caller retry giải phóng lock an toàn thay vì mất token và rò rỉ lock trên Redis; phòng vệ nil context.
    + **DDL Outbox Pattern:** Bổ sung bảng `outbox_events` (`CHECK (retry_count >= 0)`, `error_message TEXT`) và partial index `idx_outbox_pending` (`WHERE status = 'PENDING'`) vào migration `000001_init_inventory_schema.up.sql` / `down.sql`, đồng bộ vào `docs/inventory_database_design.md`.
  - **Tổng kết kiểm thử Bước 4:** Đạt **76/76 test cases PASS 100%** trên toàn bộ service (chạy fresh bằng `go test -count=1 ./...`). Usecase: 27/27, Entity: 14/14, Domain Service: 11/11, Postgres: 18/18, Redis: 6/6.
  - `Slice 4.5 & 4.6`: Tạm hoãn theo kế hoạch chờ UI/Frontend và cụm Kafka.


---

## 8. CÁC ĐIỂM CẦN LƯU Ý & BƯỚC HOÀN THIỆN TIẾP THEO

1. **Kiểm tra tính nhất quán Schema:** Đối chiếu các file `.proto` trong `packages/proto` và event schemas trong `packages/events` với các tương tác dịch vụ đã thiết kế trong tài liệu `02_architecture`.
2. **Thực thi kỷ luật Coder:** Mọi thay đổi logic mã nguồn phải tuân thủ chuẩn 4 Communication Patterns, không tạo phụ thuộc vòng (circular dependency) và không gọi đồng bộ cho các tác vụ phi thời gian thực.
3. **Kiểm thử tải k6:** Thiết lập các kịch bản mô phỏng đặt hàng đồng thời để kiểm chứng độ bền của cơ chế Redis lock chống bán vượt tồn kho.
