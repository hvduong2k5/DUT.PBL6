# BẢNG TIẾN ĐỘ & TRACKER DỰ ÁN (TASK TRACKER)

## I. BỐI CẢNH DỰ ÁN & LỜI DẶN CỦA NGƯỜI DÙNG

- **Tên dự án:** Hệ sinh thái thương mại điện tử đa kênh & Hỗ trợ quyết định chiến lược cho Nông đặc sản OCOP Huế (Mè xửng O Mạ) - PBL6 / DUT.
- **Kiến trúc cốt lõi:** Microservices phân tán (18 Bounded Contexts) + Event-Driven Architecture (Kafka / RabbitMQ) + Polyglot Persistence (PostgreSQL 16, MongoDB 7, Redis 7.2) + Full Observability (OTel, Prometheus, Grafana, Jaeger, Loki).
- **Nguyên tắc kỹ thuật sống còn:**
  1. **Source of Truth & Task Tracking:** Duy trì song song `task.md` và `project_bible.md` ở mọi bước.
  2. **Zero-Redundancy Handoff & Model Tiering:** Phân định 4 vai trò (Scribe -> Planner [Pro] -> Coder [Flash] -> Tester [Pro]).
  3. **Micro-Slicing & Anti-Monolith:** Chia nhỏ vi mô (<100 LOC/lần sửa, không quá 2 files).
  4. **Tiết kiệm 70% Token & Chống Rate Limit 429:** Targeted slicing, Silent test mode, Lean docs.
  5. **Quy tắc giao tiếp hệ thống:** Tuân thủ nghiêm ngặt 4 communication patterns (REST Ingress, gRPC Internal Sync, Kafka Business Events, OTLP Telemetry).

---

## II. CHECKLIST TIẾN ĐỘ DỰ ÁN

- [x] **Milestone 0: Khảo sát & Nắm bắt toàn diện Hồ sơ Dự án**
  - [x] Rà soát cấu trúc thư mục Monorepo và tài liệu kỹ thuật (`README.md`, `docs/01_requirements`, `docs/02_architecture`)
  - [x] Phân tích 18 Bounded Contexts, 3 Planes kiến trúc, Hybrid Saga, Zero-Trust Gateway, Mobile BFF
  - [x] Khởi tạo bộ đôi tài liệu SSOT: `task.md` và `project_bible.md`
- [ ] **Milestone 1: Đồng bộ Hợp đồng & Schema Dùng chung (Packages)**
  - [ ] Đối chiếu Protobuf specs (`packages/proto`) với kiến trúc gRPC trong tài liệu thiết kế
  - [ ] Đối chiếu CloudEvents schemas (`packages/events`) với định nghĩa sự kiện trong `02_architecture`
  - [ ] Hoàn thiện `packages/common` (DTOs, Exceptions, Logger, Base middleware)
- [ ] **Milestone 2: Triển khai & Kiểm thử Tầng Dịch vụ Cốt lõi (Core Microservices)**
  - [ ] `auth-service` / `identity-service` (RBAC, JWT, Argon2, Redis Token Store)
  - [ ] `api-gateway` (TLS Termination, Local JWKS validation, Rate limiting, Circuit Breaker)
  - [ ] `catalog-service` & `inventory-service` (Anti-overselling, Redis Redlock, FEFO)
  - [ ] `order-service` & `payment-service` (Saga Orchestration, VietQR dynamic QR, Idempotency)
  - [ ] `shipping-service` & `notification-service` (3PL integration, WebSockets, FCM)
  - [ ] `analytics-service` (Clickstream, RFM Segmentation, BI/DSS)
- [ ] **Milestone 3: Tầng Trình diễn (Apps & Clients)**
  - [ ] `apps/web-user` (Next.js 14 SSR, SEO Schema.org, OCOP Storytelling)
  - [ ] `apps/web-admin` (React Vite, Shadcn UI, BI ECharts Dashboard)
  - [ ] `apps/mobile-app` (Flutter / React Native, Mobile BFF integration)
- [ ] **Lộ trình Triển khai chuyên biệt cho `inventory-service` (MS-01):**
  - [x] **Bước 1:** Thu thập thông tin `inventory-service` từ toàn bộ tài liệu dự án ra file đặc tả (`docs/inventory-service/inventory_service_specs.md`)
  - [x] **Bước 2:** Thiết kế kiến trúc dịch vụ (`docs/inventory-service/inventory_architecture_design.md`), CSDL độc lập (`docs/inventory-service/inventory_database_design.md`), và vẽ sơ đồ luồng Mermaid
  - [x] **Bước 3:** Lập danh sách toàn diện các kịch bản Unit Test cho service (`docs/inventory-service/inventory_unit_tests.md`)
  - [x] **Bước 4: Tiến hành lập trình mã nguồn (Code implementation) & Kiểm thử vi mô:**
    - [x] **Slice 4.1:** Scaffolding Go module (`services/inventory-service`), cấu trúc Clean Architecture & Domain Entities (`InventoryItem`, `Batch`, `StockReservation`)
    - [x] **Slice 4.2:** Thuật toán phân bổ FEFO (`fefo_allocator.go`) & Unit Tests Domain Logic (`go test ./internal/domain/...`) - Đã pass 100% (UT-INV-DOMAIN-01..03, UT-INV-FEFO-01..06, near-expiry & expired tests)
    - [x] **Slice 4.3:** Tầng Infrastructure PostgreSQL (DDL Migration chuẩn hóa cột `status` & index FEFO, Repository với `SELECT ... FOR UPDATE` & Transaction Manager, mock repository & idempotency tests đạt 79.6% coverage)
    - [x] **Slice 4.4:** Tầng Application Use Cases (`ReserveStock`, `ReleaseReservation`, `CommitStockDeduction`), Idempotency & Redis Redlock, deadlock-free canonical sorting, duplicate SKU consolidation - Đã pass 100% (UT-INV-APP-01..06 + edge cases)
    - [x] **Slice 4.4b (Phản biện & Tinh chỉnh Chống Deadlock & Redlock Toàn diện):** Chuẩn hóa Lock Hierarchy một chiều (Items -> Batches) trong `ReleaseReservation` & `CommitStockDeduction`, tích hợp Redis Redlock toàn diện với canonical sorting và defer release an toàn, bổ sung phương thức `ReserveAt(qty, now)` trong `Batch` và đồng bộ vào `AllocateByFEFOAt` đảm bảo tính đơn định Domain (61/61 unit tests PASS 100%).
    - [x] **Slice 4.4c (Phòng vệ Context Leak, Transaction Hygiene, UTC & Outbox DDL):**
      * Sửa lỗi Context Timeout Leak trong `defer ReleaseLock` bằng `context.WithoutCancel(ctx)` kết hợp `context.WithTimeout(..., 2*time.Second)` trên cả 3 use case, bổ sung nil-context guard và guard `len(lockedKeys) == 0`.
      * Phòng vệ Transaction trong `PostgresTxManager.ExecuteInTransaction` với `defer func() { _ = tx.Rollback() }()` ngay sau khi tạo `sqlTx`, bổ sung nil checks cho DB và callback function.
      * Chuẩn hóa triệt để Timezone UTC trên toàn bộ Domain Entities (`StockReservation`, `InventoryItem`, `Batch`) và Postgres Repository updates.
      * Cải tiến `RedlockService`: Giữ lock token trong bộ nhớ khi Lua script `Eval` gặp lỗi tạm thời để hỗ trợ retry giải phóng lock an toàn; phòng vệ nil context.
      * Bổ sung DDL Outbox Pattern (`outbox_events` với `CHECK (retry_count >= 0)`, `error_message TEXT`, và partial index `idx_outbox_pending`) vào `000001_init_inventory_schema.up.sql` / `down.sql` và đồng bộ `inventory_database_design.md`.
    - [x] **Slice 4.5:** Tầng Presentation gRPC, Query Use Cases & Entrypoint Server:
      * Sinh mã Go cho protobuf từ `packages/proto/inventory/v1/inventory.proto` và `packages/proto/common/v1/` vào `pkg/proto/...`.
      * Xây dựng 2 Query Use Cases: `GetStockLevelUseCase` (`internal/application/usecase/get_stock_level.go`) & `GetBatchFEFODetailsUseCase` (`internal/application/usecase/get_batch_fefo.go`). Bổ sung `GetItemsBySKUs` & `GetBatchesBySKU` vào `port.InventoryRepository` và PostgreSQL implementation.
      * Xây dựng gRPC Handler (`internal/presentation/grpc/handler.go`) cài đặt `InventoryServiceServer` với đầy đủ 4 RPC methods, mapping Use Cases và xử lý chuẩn hóa gRPC status (`OK`, `InvalidArgument`, `ResourceExhausted`, `NotFound`, `Internal`) kèm `ErrorDetail` metadata.
      * Hardening Presentation & Use Cases: Bổ sung `GetReservationByID` hỗ trợ giải phóng tồn khi chỉ có `reservation_id`, tính toán trả về `total_items_restored` trong `ReleaseReservationResponse`, sửa lỗi gán nhãn `EXPIRING_SOON` cho lô kẹo quá hạn trong `GetBatchFEFODetails`, và tăng cường validation chuỗi whitespace.
      * Xây dựng Entrypoint Server (`cmd/server/main.go`) cổng 8001 với DI wiring (Postgres, Redis Redlock, Repos, Use Cases, gRPC) và Graceful Shutdown (SIGINT, SIGTERM).
      * Bộ Unit & Wire Integration Tests kiểm thử toàn diện: Đạt 125/125 unit & gRPC wire tests PASS 100% (bao gồm 26 tests cho Presentation gRPC, 33 tests Usecase, 22 tests Postgres Repo).
    - [x] **Slice 4.6:** Trục sự kiện Kafka (Event Producers/Consumers, Transactional Outbox Pattern & Background Workers):
      * **Domain & Outbox Pattern:** Định nghĩa thực thể `OutboxEvent`, schema CloudEvents 1.0 JSON chuẩn (`vn.omama.inventory.stock.reserved.v1`, `vn.omama.inventory.stock.released.v1`, `vn.omama.inventory.batch.expiry.warning.v1`, `vn.omama.order.paid.v1`). Tích hợp ghi sự kiện vào `outbox_events` trong cùng DB transaction tại `ReserveStockUseCase` và `ReleaseReservationUseCase`.
      * **Port & Postgres Implementation:** Bổ sung `port.OutboxRepository`, `port.EventPublisher`, `GetExpiredPendingReservations` và `GetBatchesNearExpiry` vào `port.InventoryRepository`. Cài đặt đầy đủ `PostgresOutboxRepository` và cập nhật `PostgresInventoryRepository`.
      * **Transactional Outbox Publisher Worker:** Xây dựng `OutboxPublisherWorker` tại `internal/infrastructure/worker/outbox_publisher.go` quét các sự kiện `PENDING` theo batch, xuất bản lên Kafka broker (`inventory.events.v1`), cập nhật `PUBLISHED`, hỗ trợ exponential backoff và max retries (chuyển sang `FAILED`).
      * **Background Sweeper Workers:**
        + `TTLReservationCleanupWorker` (`cleanup_worker.go`): Chạy định kỳ (60s), quét các reservations `PENDING` quá hạn (`expires_at < NOW()`), kích hoạt `ReleaseReservationUseCase` để giải phóng tồn kho an toàn và ghi outbox event.
        + `ExpiryCheckWorker` (`expiry_worker.go`): Chạy định kỳ (24h), quét các lô cận date (< 45 ngày) chuyển sang `NEAR_EXPIRY` và ghi outbox event `vn.omama.inventory.batch.expiry.warning.v1`. Lô quá hạn chuyển sang `EXPIRED` (hoặc `QUARANTINE` nếu có phần đang giữ chỗ).
      * **Kafka Consumer (Presentation Layer):** Xây dựng `OrderPaidHandler` và `KafkaConsumerListener` tại `internal/presentation/kafka/consumer.go`, tiêu thụ `OrderPaidEvent` (`order.events.v1`), bảo đảm Idempotency (qua `IdempotencyRepository` và trạng thái reservation), kích hoạt `CommitStockDeductionUseCase` để trừ kho vật lý vĩnh viễn và đổi reservation sang `COMMITTED`.
      * **Entrypoint & DI Integration:** Cập nhật `cmd/server/main.go` khởi chạy song song gRPC Server cùng 3 Background Workers và Kafka Consumer Listener, hỗ trợ fallback `LogEventPublisher` khi chưa có cụm Kafka, thực hiện Graceful Shutdown an toàn (dừng workers, đóng Kafka connections, đóng gRPC server, đóng DB/Redis).
      * **Kiểm thử toàn diện:** Bổ sung 36 test cases mới cho Outbox Entity, Postgres Outbox Repo, Outbox Use Cases, Outbox Publisher Worker, Sweeper Workers, Kafka Consumer & Producer. Nâng tổng số test cases lên **161/161 tests PASS 100%**.
    - [x] **Slice 4.7:** Dữ liệu mẫu (Seed Data) OCOP Huế, Khởi tạo Hạ tầng Docker & Hướng dẫn Postman gRPC:
      * **Seed Data Chuẩn Thực Tế (`seed_sample_data.sql`):** 5 SKU Mè xửng O Mạ Huế, 11 lô hàng FEFO với các trạng thái (`ACTIVE`, `NEAR_EXPIRY`, `EXPIRED`, `QUARANTINE`), 1 đơn giữ chỗ mẫu PENDING, hàm thời gian tương đối `CURRENT_DATE +/- INTERVAL` duy trì tính hợp lệ vĩnh viễn, xử lý triệt để Idempotent Re-run (TRUNCATE bao gồm cả `outbox_events` & `idempotency_keys`).
      * **Tự động hóa Hạ tầng (`init-inventory-data.ps1`):** Script 1-click tự động khởi động Docker Desktop / Windows Service, chạy `om-postgres` và `om-redis`, đợi Postgres phục vụ SQL thật sự (loại trừ false-positive của temporary server trong initdb), nạp schema và seed data qua `docker cp` chống lỗi UTF-8 BOM và lệch encoding trên Windows PowerShell 5.1/7.
      * **Tài liệu Hướng dẫn Kiểm thử Postman gRPC:** Chi tiết từng bước bật Server Reflection, payload JSON mẫu cho toàn bộ 4 RPCs (`GetStockLevel`, `GetBatchFEFODetails`, `ReserveStock`, `ReleaseReservation`), kết quả mong đợi, cơ chế Idempotency và kiểm thử biên (ResourceExhausted khi mua vượt tồn lô hợp lệ).
      * **Bộ Test Client Tự Động (`cmd/testclient/main.go`):** Script Go kiểm thử trực tiếp 100% luồng live roundtrip cho 4 RPCs và edge cases.
    - [x] **Slice 4.8 (Senior Engineering & Architectural Hardening - Pragmatic Triage Patch & Review):**
      * **P0 - Khắc phục Idempotency tiên nghiệm trong Kafka Consumer (`internal/presentation/kafka/consumer.go`):** Đổi thứ tự thực thi: gọi `commitStockUC.Execute(ctx, orderID)` trước (tận dụng tính idempotent sẵn có của domain/DB). Chỉ ghi nhận `idempotencyRepo.CheckOrSet` SAU KHI commit kho thành công (hoặc reservation đã xử lý). Nếu use case gặp lỗi tạm thời, không lưu key idempotency, cho phép Kafka retry message an toàn, triệt tiêu nguy cơ Lost Stock Deduction. Bổ sung `defer recover()` bảo vệ vòng lặp `KafkaConsumerListener.run` và `OrderPaidHandler.Handle` chống crash tiến trình khi gặp payload dị biệt.
      * **P1 - Bổ sung gRPC Panic Recovery Interceptor (`cmd/server/main.go` & `cmd/server/main_test.go`):** Cài đặt `recoveryUnaryServerInterceptor()` và `recoveryStreamServerInterceptor()` chuẩn hóa bọc mọi RPC method qua `grpc.ChainUnaryInterceptor` & `grpc.ChainStreamInterceptor`, recover runtime panics, in stack trace và trả về status `codes.Internal`, bảo vệ toàn diện tiến trình không bị crash sập (bảo vệ 3 background workers và Kafka consumer).
      * **P1 - Bổ sung Outbox Event khi Commit Trừ kho (`internal/application/usecase/commit_stock.go` & `internal/domain/entity/outbox.go`):** Định nghĩa CloudEvent schema `vn.omama.inventory.stock.committed.v1` (`EventTypeStockCommitted`) kèm file schema chuẩn tại `packages/events/schemas/inventory/v1/stock_committed.event.json`. Tích hợp `outboxRepo` vào `CommitStockDeductionUseCase` trong cùng DB transaction, phát sự kiện kèm danh sách SKU và số lượng trừ phục vụ analytics-service và shipping-service. Chuẩn hóa tính đơn định của `warehouse_id` dựa trên `skus[0]` và guard `len(committedItems) > 0`.
      * **Kiểm thử hồi quy & chuyên biệt:** Bổ sung unit tests cho outbox event commit, fail transaction, unary/stream panic interceptor, Kafka consumer retry khi commit lỗi, CheckOrSet thất bại, handler panic recovery, và outbox worker publish `StockCommitted` với partition key `order_id`. Đạt 100% tests PASS across all packages.

- [/] **Tác vụ Môn Thương mại điện tử: Kiểm định & Hoàn thiện Báo cáo Phân tích Phương thức Thanh toán:**
  - [x] Thiết lập bộ khung kiến thức quy chuẩn (API, Sandbox, Callback/IPN, Logging, EMVCo/VietQR, Sandbox Specs, Phương thức chính & phụ).
  - [x] Kiểm định tệp tài liệu `payment_gateway_and_integration_guide.md` đối sánh với 4 yêu cầu bài tập (Đạt 4/4 yêu cầu).
  - [x] Đối chiếu xung đột kiến trúc với toàn bộ tài liệu dự án `DUT.PBL6` (Phát hiện 2 lỗ hổng nghiêm trọng: Thiếu nội dung Chương 7 & Xung đột chiến lược COD giữa ADR-PAY-001 và EPIC_07_Payment).
  - [ ] Biên soạn bổ sung Chương 7 & Điều hòa xung đột COD giữa tài liệu bài tập và tài liệu dự án.
