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
- **Tài liệu thiết kế chi tiết:** [`docs/inventory-service/inventory_architecture_design.md`](file:///d:/PROJECT/PBL6/DUT.PBL6/docs/inventory-service/inventory_architecture_design.md) & [`docs/inventory-service/inventory_database_design.md`](file:///d:/PROJECT/PBL6/DUT.PBL6/docs/inventory-service/inventory_database_design.md).
- **Bộ kịch bản kiểm thử Unit Tests:** [`docs/inventory-service/inventory_unit_tests.md`](file:///d:/PROJECT/PBL6/DUT.PBL6/docs/inventory-service/inventory_unit_tests.md) (Bao phủ Invariants, FEFO Allocator, Use Cases với Mock, Idempotency và Background Workers).
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
    + **DDL Outbox Pattern:** Bổ sung bảng `outbox_events` (`CHECK (retry_count >= 0)`, `error_message TEXT`) và partial index `idx_outbox_pending` (`WHERE status = 'PENDING'`) vào migration `000001_init_inventory_schema.up.sql` / `down.sql`, đồng bộ vào `docs/inventory-service/inventory_database_design.md`.
  - `Slice 4.5 (Tầng Presentation gRPC, Query Use Cases & Entrypoint Server)`:
    + **Sinh mã Go Protobuf (`pkg/proto/`):** Tự động sinh `inventoryv1` (`inventory.pb.go`, `inventory_grpc.pb.go`) và `commonv1` (`types.pb.go`, `errors.pb.go`) qua `buf generate` với module path chuẩn `dut-pbl6/inventory-service/pkg/proto/...`.
    + **2 Query Use Cases mới:** `GetStockLevelUseCase` (tính toán `available_qty`, `stock_status` gồm IN_STOCK, LOW_STOCK, OUT_OF_STOCK, deduplicate SKUs) và `GetBatchFEFODetailsUseCase` (truy vấn danh sách lô theo FEFO `exp_date ASC`, tự động cập nhật trạng thái cận date / expired). Bổ sung phương thức `GetItemsBySKUs` và `GetBatchesBySKU` vào `port.InventoryRepository` và `PostgresInventoryRepository`.
    + **gRPC Presentation Handler (`internal/presentation/grpc/handler.go`):** Cài đặt `InventoryServiceServer` đầy đủ 4 RPCs: `ReserveStock`, `ReleaseReservation`, `GetStockLevel`, `GetBatchFEFODetails`. Xử lý chuyển đổi mã trạng thái chuẩn (`codes.OK`, `codes.InvalidArgument`, `codes.ResourceExhausted`, `codes.NotFound`, `codes.Internal`) và đóng gói `commonv1.ErrorDetail` vào gRPC Status Details.
    + **Hardening & Khắc phục khiếm khuyết Presentation Layer:**
      * Bổ sung `GetReservationByID` vào repository và use case, hỗ trợ giải phóng tồn kho khi caller chỉ cung cấp `reservation_id` (trước đây bị lỗi `ErrInvalidBatchData`).
      * Triển khai `ExecuteWithResult` tính toán chính xác số lượng tồn kho được hoàn trả và gán vào `total_items_restored` trong `ReleaseReservationResponse` (trước đây luôn = 0).
      * Khắc phục triệt để lỗi phân loại chất lượng lô hàng quá hạn trong `GetBatchFEFODetails`: Lô quá hạn (`!now.Before(exp_date)` hoặc `EXPIRED`) bắt buộc chuyển sang `QUARANTINED`, ngăn chặn lỗi bán hàng quá hạn do điều kiện `daysUntil <= 45` bị kích hoạt sai.
      * Tăng cường xác thực chuỗi rỗng/chỉ chứa khoảng trắng (whitespace-only) trên toàn bộ các tham số đầu vào (`order_id`, `sku_code`, `sku_codes`, `reservation_id`).
      * Bổ sung bài kiểm thử tích hợp mạng gRPC thật (In-Memory `bufconn`) xác thực toàn bộ quá trình đóng gói, truyền tải HTTP/2 và giải mã Status Details `ErrorDetail` qua wire.
    + **Entrypoint Server (`cmd/server/main.go`):** Khởi tạo gRPC server lắng nghe cổng `8001` (hỗ trợ ENV `PORT`, `DB_URL`, `REDIS_ADDR`), wire toàn bộ Dependency Injection (PostgreSQL connection pool, Redis client & adapter, repositories, use cases, gRPC server registration, reflection), hỗ trợ Graceful Shutdown an toàn trên `SIGINT`/`SIGTERM` với timeout 10s.
    + **Tổng kết kiểm thử Bước 4:** Đạt **125/125 test cases PASS 100%** (chạy fresh bằng `go test -count=1 ./...`). Bao gồm 26 tests cho Presentation gRPC Handler, 33 tests Usecase, 14 tests Entity, 11 tests Domain Service, 22 tests Postgres Repo, 6 tests Redis Redlock.
  - `Slice 4.6 (Trục sự kiện Kafka, Transactional Outbox Pattern & Background Sweeper Workers)`:
    + **Transactional Outbox Pattern trong Use Cases:**
      * Định nghĩa entity `OutboxEvent` (`internal/domain/entity/outbox.go`) và CloudEvents 1.0 JSON payloads chuẩn (`StockReservedEventData`, `StockReleasedEventData`, `ExpiryWarningEventData`, `OrderPaidEventData`).
      * Trong `ReserveStockUseCase`: ghi bản ghi `outbox_events` với event type `vn.omama.inventory.stock.reserved.v1` trong cùng DB transaction ACID với việc cập nhật items, batches và tạo reservation.
      * Trong `ReleaseReservationUseCase`: ghi bản ghi `outbox_events` với event type `vn.omama.inventory.stock.released.v1` trong cùng DB transaction khi giải phóng hàng.
      * Mở rộng `port.OutboxRepository` và cài đặt `PostgresOutboxRepository` (`internal/infrastructure/postgres/outbox.go`).
    + **Transactional Outbox Publisher Worker (`internal/infrastructure/worker/outbox_publisher.go`):**
      * Quét các sự kiện `PENDING` theo batch (`BatchSize: 50`) từ bảng `outbox_events` sử dụng partial index `idx_outbox_pending`.
      * Tự động trích xuất Partition Key (`sku_code`) từ CloudEvent payload và xuất bản lên topic `inventory.events.v1` qua port `port.EventPublisher`.
      * Cập nhật trạng thái `PUBLISHED` kèm `processed_at = NOW()`, hoặc ghi nhận `retry_count` kèm `error_message` khi gặp lỗi broker.
      * Hỗ trợ Exponential Backoff (`baseBackoff * 2^(retry-1)`) và chuyển sang `FAILED` khi đạt ngưỡng `MaxRetries: 5`.
    + **Background Sweeper Workers:**
      * `TTLReservationCleanupWorker` (`internal/infrastructure/worker/cleanup_worker.go`): Chạy định kỳ (mặc định 60s), quét bảng `stock_reservations` lấy các bản ghi `PENDING` có `expires_at < NOW()`, ủy quyền cho `ReleaseReservationUseCase` để hoàn trả tồn kho an toàn và tự động ghi outbox release event.
      * `ExpiryCheckWorker` (`internal/infrastructure/worker/expiry_worker.go`): Chạy định kỳ (mặc định 24h), quét các lô có `exp_date <= NOW() + 45 days`. Lô cận date chuyển sang `NEAR_EXPIRY` và phát sinh outbox event `vn.omama.inventory.batch.expiry.warning.v1`. Lô đã hết hạn (`exp_date <= NOW()`) chuyển sang `EXPIRED` (hoặc `QUARANTINE` nếu còn hàng đang giữ chỗ).
    + **Tầng Presentation Kafka Consumer (`internal/presentation/kafka/consumer.go`):**
      * Cài đặt `OrderPaidHandler` tiêu thụ sự kiện `vn.omama.order.paid.v1` từ topic `order.events.v1`.
      * Phòng vệ Idempotency 2 lớp: Tầng Consumer qua `IdempotencyRepository` (`idemp:kafka:order.events.v1:order_paid:<order_id>`) và Tầng Use Case qua trạng thái reservation `COMMITTED`.
      * Kích hoạt `CommitStockDeductionUseCase` để trừ kho vật lý vĩnh viễn và đổi reservation sang `COMMITTED`.
      * `KafkaConsumerListener`: Quản lý vòng lặp tiêu thụ và commit offset tin cậy qua `MessageReader`.
    + **Hạ tầng Kafka (`internal/infrastructure/kafka/producer.go`):** Cài đặt `KafkaProducer` dựa trên `segmentio/kafka-go` và `LogEventPublisher` làm fallback khi chạy môi trường local dev chưa có cụm Kafka.
    + **Tích hợp Entrypoint Server (`cmd/server/main.go`):** Wire toàn bộ 3 Background Workers và Kafka Consumer Listener song song cùng gRPC Server; hỗ trợ Graceful Shutdown an toàn dọn dẹp tài nguyên (dừng workers, đóng Kafka connections, đóng gRPC server, đóng DB và Redis pools).
    + **Tổng kết kiểm thử Bước 4:** Đạt **161/161 test cases PASS 100%** (36 test cases mới cho Outbox Entity, Postgres Outbox Repo, Outbox Use Cases, Outbox Publisher Worker, Sweeper Workers, Kafka Consumer & Producer).
  - `Slice 4.7 (Bộ Dữ Liệu Mẫu OCOP Huế, Khởi Tạo Hạ Tầng Docker & Kiểm Thử Postman gRPC)`:
    + **Dữ liệu mẫu (`services/inventory-service/migrations/seed_sample_data.sql`):**
      * Định nghĩa 5 SKU Mè xửng O Mạ Huế (`MX-GION-500G`, `MX-DEO-300G`, `KE-ME-GUONG-250G`, `MX-KHOAI-LANG-400G`, `MX-MAT-ONG-350G`) phủ đủ các trạng thái tồn kho (`IN_STOCK`, `LOW_STOCK`, `OUT_OF_STOCK`).
      * 11 lô sản xuất (`batches`) với các trạng thái (`EXPIRED`, `NEAR_EXPIRY` <= 45 ngày, `ACTIVE`, `QUARANTINE`) tính toán động theo `CURRENT_DATE +/- INTERVAL` duy trì tính hợp lệ vĩnh viễn cho thuật toán FEFO và Zero Expired Sale.
      * Phiếu giữ chỗ mẫu PENDING (`ORD-TEST-HUEDAC-9999`) khóa sẵn 10 gói `MX-DEO-300G` với TTL 2 giờ, phục vụ kiểm thử ngay RPC `ReleaseReservation`.
      * Cơ chế Idempotent Re-run dọn sạch bảng (TRUNCATE bao gồm cả `outbox_events` & `idempotency_keys`).
    + **Tự động hóa 1-click (`infra/scripts/init-inventory-data.ps1`):**
      * Tự động phát hiện và khởi động Windows Service `com.docker.service` & `Docker Desktop.exe` kèm vòng lặp chờ an toàn.
      * Khởi chạy `om-postgres` (PostgreSQL 16) và `om-redis` (Redis 7.2) qua `docker-compose.infra.yml`.
      * Kiểm tra kết nối SQL thật sự (`SELECT 1`), tránh lỗi false-positive do temporary server của `initdb` gây ngắt kết nối (`FATAL: database system is shutting down`).
      * Nạp Schema Migration và Seed Data an toàn qua `docker cp` + `psql -f`, triệt tiêu hoàn toàn lỗi UTF-8 BOM (`\xEF\xBB\xBF`) và lỗi sai lệch encoding trên Windows PowerShell 5.1/7.
    + **Bộ Kiểm Thử & Xác Minh Live (`cmd/testclient/main.go`):**
      * Đã kiểm thử live roundtrip 100% thành công trên gRPC server port 8001 cho toàn bộ 4 RPCs (`GetStockLevel`, `GetBatchFEFODetails`, `ReserveStock`, `ReleaseReservation`), kiểm tra Idempotency và kiểm thử biên (mua vượt tồn kho hợp lệ trả về `ResourceExhausted`).
  - `Slice 4.8 (Senior Engineering & Architectural Hardening - Pragmatic Triage Patch & Review)`:
    + **P0 - Khắc phục Idempotency Tiên Nghiệm trong Kafka Consumer (`internal/presentation/kafka/consumer.go`):**
      * Phát hiện lỗi: Trước đây, `OrderPaidHandler` gọi `idempotencyRepo.CheckOrSet` trước khi gọi `commitStockUC.Execute`. Nếu usecase gặp lỗi tạm thời (ví dụ DB bận, timeout mạng), key idempotency đã nằm trong DB; khi Kafka redeliver message, consumer thấy key đã tồn tại nên bỏ qua luôn, dẫn đến đơn hàng không bao giờ bị trừ tồn vật lý (Lost Stock Deduction) và sau 15 phút sẽ bị worker dọn dẹp xả bán cho người khác.
      * Giải pháp: Đổi thứ tự thực thi — gọi `commitStockUC.Execute` trước (dựa vào tính idempotent sẵn có của usecase/DB: nếu reservation đã `COMMITTED` thì return nil an toàn). Chỉ ghi nhận `idempotencyRepo.CheckOrSet` SAU KHI commit thành công hoặc reservation đã xử lý xong. Khi usecase lỗi, không ghi key, bảo đảm Kafka retry an toàn.
      * Bổ sung Panic Recovery: Tích hợp `defer recover()` bắt mọi runtime panic trong `OrderPaidHandler.Handle` và `KafkaConsumerListener.run`, in stack trace và trả về lỗi, ngăn chặn hoàn toàn việc goroutine consumer làm crash sập toàn bộ microservice.
    + **P1 - Bổ sung gRPC Panic Recovery Interceptor (`cmd/server/main.go` & `cmd/server/main_test.go`):**
      * Đăng ký `recoveryUnaryServerInterceptor` và `recoveryStreamServerInterceptor` chuẩn trong `grpc.NewServer(grpc.ChainUnaryInterceptor(...), grpc.ChainStreamInterceptor(...))`.
      * Tự động bắt mọi runtime panic trong cả Unary lẫn Streaming RPC handlers, ghi log stack trace đầy đủ và trả về mã gRPC `codes.Internal` ("internal server error") cho client, triệt tiêu nguy cơ crash sập toàn bộ tiến trình (bảo vệ sống còn cho 3 background workers và Kafka consumer).
    + **P1 - Bổ sung Outbox Event khi Commit Trừ kho (`internal/application/usecase/commit_stock.go` & `internal/domain/entity/outbox.go`):**
      * Định nghĩa CloudEvents 1.0 schema: `vn.omama.inventory.stock.committed.v1` (`EventTypeStockCommitted`) và tạo file schema JSON chuẩn tại `packages/events/schemas/inventory/v1/stock_committed.event.json`.
      * Inject `outboxRepo` vào `CommitStockDeductionUseCase`.
      * Trong cùng Database Transaction với thao tác trừ kho lô & item, phát sinh sự kiện outbox `vn.omama.inventory.stock.committed.v1` chứa `order_id`, `reservation_id`, `warehouse_id` (đơn định theo `skus[0]`), và danh sách SKU cùng số lượng đã trừ, sẵn sàng phục vụ cho `analytics-service` và `shipping-service`.
    + **Tổng kết kiểm thử:** 100% test cases trên toàn bộ module PASS (**166+ tests**), bao gồm các kịch bản commit retry, outbox transaction failure, panic interceptor recovery (unary & stream), handler panic recovery, CheckOrSet failure, và outbox publisher dispatching với partition key `order_id`.

---

## 8. CÁC ĐIỂM CẦN LƯU Ý & BƯỚC HOÀN THIỆN TIẾP THEO

1. **Kiểm tra tính nhất quán Schema:** Đối chiếu các file `.proto` trong `packages/proto` và event schemas trong `packages/events` với các tương tác dịch vụ đã thiết kế trong tài liệu `02_architecture`.
2. **Thực thi kỷ luật Coder:** Mọi thay đổi logic mã nguồn phải tuân thủ chuẩn 4 Communication Patterns, không tạo phụ thuộc vòng (circular dependency) và không gọi đồng bộ cho các tác vụ phi thời gian thực.
3. **Kiểm thử tải k6:** Thiết lập các kịch bản mô phỏng đặt hàng đồng thời để kiểm chứng độ bền của cơ chế Redis lock chống bán vượt tồn kho.
