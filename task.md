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
      * Bổ sung bộ Unit Tests mới (76/76 unit tests PASS 100%).
    - [ ] **Slice 4.5 (Tạm hoãn - Đợi UI/Frontend sẵn sàng):** Tầng Presentation (Protobuf, gRPC Handlers, REST Gateway)
    - [ ] **Slice 4.6 (Tạm hoãn - Đợi hạ tầng Kafka sẵn sàng):** Trục sự kiện Kafka (Event Producers/Consumers & Background Cron Workers)

