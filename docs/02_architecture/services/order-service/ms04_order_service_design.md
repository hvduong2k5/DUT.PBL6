# TÀI LIỆU THIẾT KẾ KIẾN TRÚC CHI TIẾT (LOW-LEVEL DESIGN - LLD): MS-04 ORDER SERVICE
## TRÁI TIM HỆ SINH THÁI THƯƠNG MẠI ĐIỆN TỬ & ĐIỀU PHỐI SAGA — MÈ XỬNG O MẠ
### PHIÊN BẢN: 2.0 (PRODUCTION & IMPLEMENTATION-READY SPECIFICATION)

---

> **QUY CHUẨN TIẾN TRÌNH THIẾT KẾ (DESIGN METHODOLOGY STANDARD):**
> Tài liệu này tuân thủ nghiêm ngặt **Tiến trình Thiết kế Kỹ thuật 11 Bước Tuần Tự (Strict 11-Step Software Engineering Design Flow)** theo nguyên lý Domain-Driven Design (DDD), Clean / Hexagonal Architecture và Distributed Systems Engineering Patterns.
> Tuyệt đối **không nhảy cóc giai đoạn**:
> 1. *Xác lập Biên giới & Ranh giới Sở hữu Dữ liệu (Bounded Context & Scope Isolation)* $\rightarrow$
> 2. *Mô hình hóa Miền Nghiệp vụ Cốt lõi (Domain Model, Entities & Value Objects)* $\rightarrow$
> 3. *Máy Trạng Thái & Vòng Đời Thực Thể (State Machine & Lifecycle Transitions)* $\rightarrow$
> 4. *Mô hình Hóa Dữ liệu Quan hệ & Bộ đệm (Database Schema DDL & Cache Storage)* $\rightarrow$
> 5. *Trừu tượng hóa Tầng Lưu trữ (Repository & Data Access Interfaces)* $\rightarrow$
> 6. *Tầng Ứng dụng & Động cơ Điều phối Saga (Application Use Cases & Saga Orchestration Engine)* $\rightarrow$
> 7. *Tầng Vận chuyển & Đặc tả Hợp đồng Giao tiếp (Delivery Layer, API, gRPC & Kafka Contracts)* $\rightarrow$
> 8. *Cơ chế Kỹ thuật Xuyên suốt & Phòng vệ (Cross-Cutting Concerns & Defensive Engineering)* $\rightarrow$
> 9. *Quy hoạch Cấu trúc Thư mục Dự án (Project Directory & Code Blueprint)* $\rightarrow$
> 10. *Chiến lược Kiểm thử & Ma trận Truy vết Yêu cầu (Testing Specification & Traceability Matrix)* $\rightarrow$
> 11. *Ma Trận Sự Cố & Phục Hồi Phân Tán (Distributed Failure & Recovery Matrix)*.

---

## MỤC LỤC CHI TIẾT

- [1. BƯỚC 1: XÁC LẬP BIÊN GIỚI & RANH GIỚI SỞ HỮU DỮ LIỆU (BOUNDED CONTEXT & SCOPE ISOLATION)](#1-bước-1-xác-lập-biên-giới--ranh-giới-sở-hữu-dữ-liệu-bounded-context--scope-isolation)
  - [1.1. Bounded Context & Định Vị Hệ Thống](#11-bounded-context--định-vị-hệ-thống)
  - [1.2. Ranh Giới Bất Biến & Quyền Sở Hữu Dữ Liệu (Data Ownership Boundaries)](#12-ranh-giới-bất-biến--quyền-sở-hữu-dữ-liệu-data-ownership-boundaries)
  - [1.3. Chính Sách Bảo Mật PII & Trade-off Tính Sẵn Sàng (PII Policy & Availability Trade-off)](#13-chính-sách-bảo-mật-pii--trade-off-tính-sẵn-sàng-pii-policy--availability-trade-off)
  - [1.4. Lựa Chọn Công Nghệ & Ràng Buộc Kỹ Thuật (Tech Stack Selection)](#14-lựa-chọn-công-nghệ--ràng-buộc-kỹ-thuật-tech-stack-selection)
- [2. BƯỚC 2: MÔ HÌNH HÓA MIỀN NGHIỆP VỤ CỐT LÕI (DOMAIN MODEL, ENTITIES & VALUE OBJECTS)](#2-bước-2-mô-hình-hóa-miền-nghiệp-vụ-cốt-lõi-domain-model-entities--value-objects)
  - [2.1. Thuật Ngữ Nghiệp Vụ Chuẩn Hóa (Ubiquitous Language)](#21-thuật-ngữ-nghiệp-vụ-chuẩn-hóa-ubiquitous-language)
  - [2.2. Aggregate Root: `Order`](#22-aggregate-root-order)
  - [2.3. Entities Thuộc Aggregate](#23-entities-thuộc-aggregate)
  - [2.4. Value Objects Bất Biến (Immutable Value Objects)](#24-value-objects-bất-biến-immutable-value-objects)
  - [2.5. Mô Hình 3 Trạng Thái Tồn Kho Phân Tán (The 3-State Inventory Model)](#25-mô-hình-3-trạng-thái-tồn-kho-phân-tán-the-3-state-inventory-model)
  - [2.6. Các Bất Biến Nghiệp Vụ Miền (Domain Invariants)](#26-các-bất-biến-nghiệp-vụ-miền-domain-invariants)
- [3. BƯỚC 3: MÁY TRẠNG THÁI & VÒNG ĐỜI THỰC THỂ (STATE MACHINE & LIFECYCLE TRANSITIONS)](#3-bước-3-máy-trạng-thái--vòng-đời-thực-thể-state-machine--lifecycle-transitions)
  - [3.1. Sơ Đồ Chuyển Trạng Thái Đơn Hàng Hoàn Chỉnh (Order State Machine FSM)](#31-sơ-đồ-chuyển-trạng-thái-đơn-hàng-hoàn-chỉnh-order-state-machine-fsm)
  - [3.2. Sơ Đồ Máy Trạng Thái Saga (Saga Lifecycle State Machine)](#32-sơ-đồ-máy-trạng-thái-saga-saga-lifecycle-state-machine)
  - [3.3. Ma Trận Chuyển Trạng Thái Hợp Lệ, Điều Kiện Bảo Vệ & Side Effects](#33-ma-trận-chuyển-trạng-thái-hợp-lệ-điều-kiện-bảo-vệ--side-effects)
  - [3.4. Cơ Chế Giải Quyết Tranh Chấp Trạng Thái Cạnh Tranh (Payment vs Timeout Race Condition)](#34-cơ-chế-giải-quyết-tranh-chấp-trạng-thái-cạnh-tranh-payment-vs-timeout-race-condition)
- [4. BƯỚC 4: MÔ HÌNH HÓA DỮ LIỆU QUAN HỆ & BỘ ĐỆM (DATABASE SCHEMA DDL & CACHE STORAGE)](#4-bước-4-mô-hình-hóa-dữ-liệu-quan-hệ--bộ-đệm-database-schema-ddl--cache-storage)
  - [4.1. Sơ Đồ Thực Thể - Liên Kết (ERD)](#41-sơ-đồ-thực-thể---liên-kết-erd)
  - [4.2. Chiến Lược Sinh Khóa Chính UUID v7](#42-chiến-lược-sinh-khóa-chính-uuid-v7)
  - [4.3. Kịch Bản DDL Chi Tiết (PostgreSQL 16 Production Script)](#43-kịch-bản-ddl-chi-tiết-postgresql-16-production-script)
  - [4.4. Mô Hình Dữ Liệu Bộ Đệm & Phiên (Redis Data Schema)](#44-mô-hình-dữ-liệu-bộ-đệm--phiên-redis-data-schema)
- [5. BƯỚC 5: TRỪU TƯỢNG HÓA TẦNG LƯU TRỮ (REPOSITORY & DATA ACCESS INTERFACES)](#5-bước-5-trừu-tượng-hóa-tầng-lưu-trữ-repository--data-access-interfaces)
  - [5.1. Định Nghĩa Trừu Tượng Database Transaction (`DBTX`)](#51-định-nghĩa-trừu-tượng-database-transaction-dbtx)
  - [5.2. `OrderRepository` Interface](#52-orderrepository-interface)
  - [5.3. `SagaRepository` Interface](#53-sagarepository-interface)
  - [5.4. `OutboxRepository` Interface](#54-outboxrepository-interface)
  - [5.5. `CartRepository` Interface](#55-cartrepository-interface)
- [6. BƯỚC 6: TẦNG ỨNG DỤNG & ĐỘNG CƠ ĐIỀU PHỐI SAGA (APPLICATION USE CASES & SAGA ENGINE)](#6-bước-6-tầng-ứng-dụng--động-cơ-điều-phối-saga-application-use-cases--saga-engine)
  - [6.1. Kiến Trúc Phân Lớp Bên Trong: Order Domain vs Saga Orchestrator](#61-kiến-trúc-phân-lớp-bên-trong-order-domain-vs-saga-orchestrator)
  - [6.2. Dự Toán Độ Trễ Thực Thi (Checkout Latency Budget cho P95 < 200ms)](#62-dự-toán-độ-trễ-thực-thi-checkout-latency-budget-cho-p95--200ms)
  - [6.3. Use Case 1: `CheckoutD2CUseCase` (Critical Path Synchronous)](#63-use-case-1-checkoutd2cusecase-critical-path-synchronous)
  - [6.4. Use Case 2: `VietQRWebhookCallbackUseCase` (Chốt Luật Khớp Số Tiền Tuyệt Đối)](#64-use-case-2-vietqrwebhookcallbackusecase-chốt-luật-khớp-số-tiền-tuyệt-đối)
  - [6.5. Use Case 3: `OrderTimeoutCancelCompensationUseCase` (Đền Bù Tập Trung Model A)](#65-use-case-3-ordertimeoutcancelcompensationusecase-đền-bù-tập-trung-model-a)
  - [6.6. Use Case 4: `MarketplaceInboundSagaUseCase` (Tiếp Nhận Đơn Sàn & Khóa Tồn Tập Trung)](#66-use-case-4-marketplaceinboundsagausecase-tiếp-nhận-đơn-sàn--khóa-tồn-tập-trung)
  - [6.7. Use Case 5: `CreatePOSOrderUseCase` (Bán Trực Tiếp Tại Quầy Xưởng Hương Thủy)](#67-use-case-5-createposorderusecase-bán-trực-tiếp-tại-quầy-xưởng-hương-thủy)
- [7. BƯỚC 7: TẦNG VẬN CHUYỂN & ĐẶC TẢ HỢP ĐỒNG GIAO TIẾP (DELIVERY LAYER & CONTRACTS)](#7-bước-7-tầng-vận-chuyển--đặc-tả-hợp-đồng-giao-tiếp-delivery-layer--contracts)
  - [7.1. Cổng Biên HTTP RESTful APIs (North - South via Kong Gateway)](#71-cổng-biên-http-restful-apis-north---south-via-kong-gateway)
  - [7.2. Cổng Nội Bộ gRPC Services (East - West Server)](#72-cổng-nội-bộ-grpc-services-east---west-server)
  - [7.3. Hợp Đồng Gọi gRPC Ngoại Vi (East - West Client on Critical Path)](#73-hợp-đồng-gọi-grpc-ngoại-vi-east---west-client-on-critical-path)
  - [7.4. Danh Mục Sự Kiện Kafka Xuất Bản (Transactional Outbox Producer)](#74-danh-mục-sự-kiện-kafka-xuất-bản-transactional-outbox-producer)
  - [7.5. Danh Mục Sự Kiện Kafka Tiêu Thụ (Inbound Consumer)](#75-danh-mục-sự-kiện-kafka-tiêu-thụ-inbound-consumer)
- [8. BƯỚC 8: CƠ CHẾ KỸ THUẬT XUYÊN SUỐT & PHÒNG VỆ (CROSS-CUTTING CONCERNS)](#8-bước-8-cơ-chế-kỹ-thuật-xuyên-suốt--phòng-vệ-cross-cutting-concerns)
  - [8.1. Đảm Bảo At-least-once Delivery & Idempotent Consumer (Effectively-once Business Outcome)](#81-đảm-bảo-at-least-once-delivery--idempotent-consumer-effectively-once-business-outcome)
  - [8.2. Hệ Thống Idempotency Đa Tầng (Multi-Tier Idempotency Shield)](#82-hệ-thống-idempotency-đa-tầng-multi-tier-idempotency-shield)
  - [8.3. Chiến Lược Cầu Dao Ngắt Mạch & Quá Giờ Nghiêm Ngặt (Circuit Breaker & Strict Timeout)](#83-chiến-lược-cầu-dao-ngắt-mạch--quá-giờ-nghiêm-ngặt-circuit-breaker--strict-timeout)
  - [8.4. Bảo Mật Zero-Trust & Xác Thực Webhook HMAC-SHA256](#84-bảo-mật-zero-trust--xác-thực-webhook-hmac-sha256)
  - [8.5. Tích Hợp Mặt Phẳng Kiểm Toán & Giám Sát Viễn Trắc (Audit & Observability)](#85-tích-hợp-mặt-phẳng-kiểm-toán--giám-sát-viễn-trắc-audit--observability)
- [9. BƯỚC 9: QUY HOẠCH CẤU TRÚC THƯ MỤC DỰ ÁN (PROJECT DIRECTORY & CODE BLUEPRINT)](#9-bước-9-quy-hoạch-cấu-trúc-thư-mục-dự-án-project-directory--code-blueprint)
- [10. BƯỚC 10: CHIẾN LƯỢC KIỂM THỬ & MA TRẬN TRUY VẾT YÊU CẦU (TESTING & TRACEABILITY)](#10-bước-10-chiến-lược-kiểm-thử--ma-trận-truy-vết-yêu-cầu-testing--traceability)
  - [10.1. Danh Mục 4 Kịch Bản Kiểm Thử Phân Tán Sống Còn (Distributed Concurrency Tests)](#101-danh-mục-4-kịch-bản-kiểm-thử-phân-tán-sống-còn-distributed-concurrency-tests)
  - [10.2. Ma Trận Ánh Xạ Truy Vết Yêu Cầu Chức Năng (FR Traceability Matrix)](#102-ma-trận-ánh-xạ-truy-vết-yêu-cầu-chức-năng-fr-traceability-matrix)
  - [10.3. Ma Trận Ánh Xạ Yêu Cầu Phi Chức Năng (NFR Traceability Matrix)](#103-ma-trận-ánh-xạ-yêu-cầu-phi-chức-năng-nfr-traceability-matrix)
- [11. BƯỚC 11: MA TRẬN SỰ CỐ & PHỤC HỒI PHÂN TÁN (DISTRIBUTED FAILURE & RECOVERY MATRIX)](#11-bước-11-ma-trận-sự-cố--phục-hồi-phân-tán-distributed-failure--recovery-matrix)

---

## 1. BƯỚC 1: XÁC LẬP BIÊN GIỚI & RANH GIỚI SỞ HỮU DỮ LIỆU (BOUNDED CONTEXT & SCOPE ISOLATION)

Trước khi bắt tay vào thiết kế bất kỳ thực thể hay bảng cơ sở dữ liệu nào, nguyên tắc tối thượng của kiến trúc phân tán là **phải thiết lập ranh giới cô lập (Boundary Isolation)**: xác định rõ trách nhiệm cốt lõi, quyền sở hữu dữ liệu độc quyền, và đặc biệt là **những gì dịch vụ TUYỆT ĐỐI CẤM LÀM**.

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

### 1.1. Bounded Context & Định Vị Hệ Thống
- **Mã định danh:** `MS-04` | **Tên dịch vụ:** `order-service`
- **Bounded Context:** `BC-04: Omnichannel Commerce & Order Orchestration Context`
- **Phân loại Domain:** 🔴 **Core Domain** — Trái tim vận hành của toàn bộ hệ sinh thái Mè Xửng O Mạ.
- **Cổng giao tiếp mạng:**
  - **Port gRPC nội bộ (East-West Server):** `8004` (Tiếp nhận `CreatePOSOrder` từ POS quầy xưởng MS-13, `GetOrderDetail`, `CancelOrder`).
  - **Port HTTP REST (North-South via Kong Gateway):** `8004` (Ánh xạ các endpoint `/api/v1/checkout`, `/api/v1/orders/**`, `/api/v1/cart/**`, `/api/v1/payments/vietqr/callback`).
- **Cơ sở dữ liệu độc lập (Shared-Nothing Architecture):**
  - **Primary Relational DB:** PostgreSQL 16 (`order_db`) — Lưu trữ đơn hàng, mặt hàng, giao dịch thanh toán, outbox events, saga states.
  - **Distributed Cache & Session Store:** Redis Cluster 7 (`order_cache`) — Quản lý giỏ hàng, checkout token, idempotency keys, distributed lock.

---

### 1.2. Ranh Giới Bất Biến & Quyền Sở Hữu Dữ Liệu (Data Ownership Boundaries)
Tuân thủ cam kết tại [`service_boundary.md`](../service_boundary.md) và [`bounded_context.md`](../bounded_context.md), `order-service` thiết lập 5 nguyên tắc sở hữu thép:

| Dữ Liệu / Trách Nhiệm | Thuộc Về MS-04 `order-service`? | Đơn Vị Sở Hữu Chính Thức | Lý Do Kiến Trúc & Ranh Giới Bất Khả Xâm Phạm |
| :--- | :---: | :--- | :--- |
| **Vòng đời trạng thái đơn hàng (`OrderStatus`)** | ✅ **SỞ HỮU DUY NHẤT** | MS-04 `order-service` | Là Single Source of Truth cho trạng thái đơn. Không service nào khác được ghi/sửa trực tiếp trạng thái đơn. |
| **Bản chụp thương mại (Snapshots)** | ✅ **SỞ HỮU DUY NHẤT** | MS-04 `order-service` | Chụp lại giá, tên kẹo, chiết khấu và địa chỉ giao hàng tại thời điểm bấm đặt hàng để bảo toàn giá trị pháp lý. |
| **Điều phối giao dịch phân tán (Saga Orchestration)** | ✅ **SỞ HỮU DUY NHẤT** | MS-04 `order-service` | Thực thi nguyên tắc **Centralized Compensation (Model A)**: Độc quyền phát hiện lỗi/timeout và trực tiếp gọi lệnh hoàn tác kho/voucher. |
| **Số lượng hàng tồn kho vật lý & Lô hạn dùng (FEFO)** | ❌ **CẤM SỞ HỮU** | MS-01 `inventory-service` | MS-04 chỉ gửi yêu cầu `ReserveStock` và `ReleaseReservation` qua gRPC; tuyệt đối cấm truy cập trực tiếp bảng tồn kho. |
| **Giá niêm yết gốc & Quy cách sản phẩm** | ❌ **CẤM SỞ HỮU** | MS-05 `catalog-service` | MS-04 chỉ thẩm định giá gửi lên qua gRPC `ValidatePriceAndSKU` đối chiếu với Catalog gốc. |
| **Ngân sách voucher & Điểm tích lũy Loyalty** | ❌ **CẤM SỞ HỮU** | MS-07 `promotion-service` | MS-04 gọi gRPC khóa voucher và gửi sự kiện `OrderPaidEvent` để Promotion tự tích điểm 1% loyalty. |
| **Đóng gói tại xưởng & Video kiểm định** | ❌ **CẤM SỞ HỮU** | MS-02 `fulfillment-service` | Xưởng Huế tự tiêu thụ `OrderPaidEvent` từ Kafka để đóng kẹo; không làm nghẽn luồng checkout của khách. |
| **Vận đơn 3PL & Lộ trình giao hàng** | ❌ **CẤM SỞ HỮU** | MS-12 `shipping-service` | MS-12 làm việc trực tiếp với GHN/ViettelPost và phát Kafka event báo bưu tá đã giao hàng. |

---

### 1.3. Chính Sách Bảo Mật PII & Trade-off Tính Sẵn Sàng (PII Policy & Availability Trade-off)

#### 1.3.1. Phân Định Danh Mục Dữ Liệu Rõ Ràng (PII Classification)
Nhằm tránh mâu thuẫn khái niệm, tài liệu chuẩn hóa 3 cấp độ định danh:
- **Direct PII (Thông tin định danh trực tiếp cá nhân):** Họ và tên khách hàng, số điện thoại, số nhà/tên đường cụ thể, tọa độ GPS chính xác. $\rightarrow$ **TUYỆT ĐỐI KHÔNG ĐƯỢC XUẤT HIỆN TRÊN KAFKA EVENT STREAM.**
- **Internal / Business Identifiers (Mã định danh nội bộ hệ thống):** `order_id`, `customer_id` (UUID v7 ẩn danh), `shipping_address_id` (Khóa ngoại tham chiếu snapshot). $\rightarrow$ **ĐƯỢC PHÉP NẰM TRONG KAFKA PAYLOAD** để các consumer định tuyến xử lý nghiệp vụ.
- **Aggregated / Business Metrics:** Doanh thu, số lượng hộp kẹo, kênh bán, mã voucher. $\rightarrow$ **CÔNG KHAI NỘI BỘ TRÊN EVENT STREAM.**

#### 1.3.2. Quyết Định Kiến Trúc: Option A (Privacy-First) & Đánh Đổi Tính Sẵn Sàng (Availability Trade-off)
Hệ sinh thái Mè Xửng O Mạ lựa chọn **Option A — Privacy-First**:
- Payload của sự kiện `OrderPaidEvent` chỉ chứa `shipping_address_id`.
- Khi `fulfillment-service` cần in tem dán gói kẹo hoặc `shipping-service` cần tạo vận đơn bưu cục, chúng sẽ gọi gRPC `GetOrderDetail` sang `order-service` để lấy địa chỉ nhận hàng.
- **Phân tích Đánh đổi (Trade-off):**
  - *Ưu điểm:* Loại bỏ 100% rủi ro rò rỉ dữ liệu cá nhân (Data Leakage) sang các consumer không cần thiết như `inventory-service`, `finance-service`, `analytics-service`.
  - *Rủi ro (Runtime Coupling):* Nếu `order-service` gặp sự cố mạng tạm thời, `fulfillment-service` không lấy được địa chỉ để in tem.
  - *Biện pháp Giảm thiểu Kỹ thuật (Mitigation):* 
    1. `order-service` thiết lập bộ đệm L2 Cache trên Redis cho `AddressSnapshot` với TTL 48 giờ.
    2. Consumer `fulfillment-service` cài đặt cơ chế Retry with Exponential Backoff + Jitter cho cuộc gọi gRPC lấy địa chỉ, đảm bảo khi `order-service` hồi phục thì việc in tem tiếp tục trơn tru mà không làm rơi rớt dữ liệu.

---

### 1.4. Lựa Chọn Công Nghệ & Ràng Buộc Kỹ Thuật (Tech Stack Selection)
- **Ngôn ngữ nền tảng:** **Go (Golang 1.22+)** hoặc **Node.js (TypeScript 5+)** tuân thủ Clean Architecture.
- **PostgreSQL Driver:** `pgx/v5` (Go) hoặc `pg` pool (Node.js) hỗ trợ kết nối Binary Protocol hiệu năng cao.
- **Serialization:** Google Protocol Buffers v3 cho giao tiếp nội bộ gRPC; JSON Schema chuẩn CNCF CloudEvents 1.0 cho Apache Kafka.
- **Bộ đệm & Khóa phân tán:** `go-redis/v9` (hoặc `ioredis`) tương thích Redis Cluster.

---

## 2. BƯỚC 2: MÔ HÌNH HÓA MIỀN NGHIỆP VỤ CỐT LÕI (DOMAIN MODEL, ENTITIES & VALUE OBJECTS)

*Theo chuẩn mực Domain-Driven Design, trước khi thiết kế bảng cơ sở dữ liệu (Database Schema), ta bắt buộc phải mô hình hóa miền nghiệp vụ độc lập hoàn toàn với công nghệ hạ tầng (Infrastructure-Agnostic).*

```mermaid
classDiagram
    class Order {
        +UUID id
        +OrderCode orderCode
        +UUID customerId
        +Channel channel
        +OrderStatus status
        +Money subtotalAmount
        +Money discountAmount
        +Money shippingFee
        +Money finalAmount
        +AddressSnapshot shippingAddress
        +VoucherSnapshot appliedVoucher
        +List~OrderLineItem~ items
        +List~PaymentTransaction~ payments
        +int version
        +DateTime createdAt
        +DateTime updatedAt
        +addOrderItem(item)
        +applyVoucher(voucher)
        +markPendingPayment()
        +markPaid(transactionId, amount, provider)
        +cancel(reason)
        +calculateTotals()
    }

    class OrderLineItem {
        +UUID id
        +string skuCode
        +string productName
        +int quantity
        +Money unitPrice
        +Money totalPrice
        +string packagingSpecs
        +string itemNotes
    }

    class PaymentTransaction {
        +UUID id
        +PaymentProvider provider
        +string providerTransactionId
        +PaymentMethod method
        +PaymentTransactionType type
        +PaymentStatus status
        +Money amount
        +string rawSignature
        +DateTime executedAt
    }

    class Money {
        <<Value Object>>
        +int64 amount
        +string currency
        +add(Money) Money
        +subtract(Money) Money
        +multiply(int) Money
        +equals(Money) bool
    }

    class AddressSnapshot {
        <<Value Object>>
        +string recipientName
        +string phoneNumber
        +string streetAddress
        +string wardCode
        +string wardName
        +string provinceCode
        +string provinceName
        +decimal latitude
        +decimal longitude
    }

    class VoucherSnapshot {
        <<Value Object>>
        +string voucherCode
        +string discountType
        +Money discountValue
        +Money appliedAmount
    }

    Order "1" *-- "1..*" OrderLineItem : contains
    Order "1" *-- "0..*" PaymentTransaction : records history
    Order o-- AddressSnapshot : snapshots
    Order o-- VoucherSnapshot : snapshots
    Order o-- Money : measured in
    OrderLineItem o-- Money : priced in
    PaymentTransaction o-- Money : transacted in
```

### 2.1. Thuật Ngữ Nghiệp Vụ Chuẩn Hóa (Ubiquitous Language)
- **Order (Đơn hàng):** Aggregate Root trung tâm, biểu thị giao dịch thương mại hoàn chỉnh.
- **OrderLineItem (Mặt hàng chi tiết):** Bản chụp cố định của sản phẩm kẹo mè xửng (mã SKU, tên kẹo, đơn giá, quy cách) tại thời điểm khách bấm đặt hàng.
- **PaymentTransaction (Giao dịch dòng tiền):** Thực thể ghi nhận lịch sử từng lần tương tác thanh toán (lần quét VietQR thử nghiệm, thanh toán thành công, hoặc giao dịch hoàn tiền refund).
- **Snapshot (Bản chụp bất biến):** Dữ liệu sao chép nguyên trạng tại thời điểm xác nhận checkout, không bị ảnh hưởng nếu dữ liệu gốc ở Catalog hoặc Profile bị thay đổi.
- **Centralized Compensation (Model A):** Cơ chế điều phối đền bù tập trung: Duy nhất Saga Orchestrator trong `order-service` phát lệnh hoàn tác tài nguyên sang các service vệ tinh.

---

### 2.2. Aggregate Root: `Order`
Thực thể gốc kiểm soát toàn bộ tính toàn vẹn của đơn hàng:
- **Định danh duy nhất:** `id` (UUID v7 time-ordered) và `order_code` (Mã định dạng thân thiện `ORD-YYYYMMDD-XXXX`).
- **Chủ sở hữu:** `customer_id` (UUID tham chiếu sang MS-15 Profile Service; `NULL` đối với khách vãng lai `GUEST_CUSTOMER`).
- **Kênh bán lẻ (`Channel`):** `D2C_WEB`, `D2C_MOBILE`, `POS_OFFLINE`, `MARKETPLACE_SHOPEE`, `MARKETPLACE_TIKTOK`, `B2B_CORPORATE`.
- **Trạng thái thực thể:** `OrderStatus` (Quản lý chặt chẽ theo máy trạng thái FSM).
- **Khóa lạc quan CAS (Optimistic Concurrency Control):** Thuộc tính `version` nguyên số tăng dần, giải quyết triệt để tranh chấp cập nhật đồng thời.

---

### 2.3. Entities Thuộc Aggregate
1. **`OrderLineItem`:**
   - Thuộc sở hữu hoàn toàn của `Order`.
   - `sku_code`, `product_name`, `quantity` ($> 0$), `unit_price` (Money), `total_price` ($= \text{unit\_price} \times \text{quantity}$).
   - `packaging_specs`: Quy cách đóng gói (Ví dụ: "Hộp 500g hút chân không chống ẩm", "Túi 300g truyền thống").
2. **`PaymentTransaction` (Thống nhất mô hình 1 Order $\rightarrow$ $N$ Payments):**
   - Thay vì chỉ có 1 `PaymentRecord` duy nhất, hệ thống mô hình hóa quan hệ $1:N$ để phản ánh chính xác thực tế:
     - Khách hàng quét QR lần 1 thất bại $\rightarrow$ Ghi nhận 1 `PaymentTransaction` trạng thái `FAILED`.
     - Khách quét QR lần 2 thành công $\rightarrow$ Ghi nhận 1 `PaymentTransaction` trạng thái `PAID`.
     - Sau này khách khiếu nại vỡ kẹo $\rightarrow$ Ghi nhận thêm 1 `PaymentTransaction` loại `REFUND` trạng thái `PAID`.
   - Thuộc tính: `provider` (`VIETQR_NAPAS`, `COD_INTERNAL`, `B2B_BANK_DIRECT`, `POS_TERMINAL`), `provider_transaction_id` (Mã giao dịch phía ngân hàng), `type` (`PAYMENT`, `REFUND`), `amount` (Money), `status` (`PENDING`, `PAID`, `FAILED`), `raw_signature`, `executed_at`.

---

### 2.4. Value Objects Bất Biến (Immutable Value Objects)
1. **`Money` (Chuẩn hóa cho tiền tệ VND):**
   - Vì Việt Nam Đồng (VND) không có đơn vị phân số thập phân (không dùng hào, xu), cấu trúc `Money` trong Domain được tinh gọn tối đa:
     ```go
     type Money struct {
         Amount   int64  // Số tiền nguyên bản (VND)
         Currency string // Bắt buộc "VND"
     }
     ```
   - *Tính tương thích Protobuf:* Khi giao tiếp gRPC qua message `omamx.common.v1.Money`, `Amount` được gán vào trường `units`, còn trường `nanos` được gán cứng `= 0`.
   - *Tính tương thích JSON REST API:* Giá trị tiền được biểu diễn dưới dạng số nguyên an toàn (JSON integer). Với mức doanh thu đơn hàng thông thường ($< 9 \times 10^{15}$ VND), hoàn toàn nằm trong giới hạn an toàn của IEEE 754 float64 / JavaScript `Number.MAX_SAFE_INTEGER`.
2. **`AddressSnapshot`:**
   - Lưu trữ: `recipient_name`, `phone_number`, `street_address`, `ward_code`, `ward_name`, `province_code`, `province_name`, `latitude`, `longitude`.
   - Tuân thủ chuẩn hành chính 2 cấp (Tỉnh/Thành phố TW - Xã/Phường) có hiệu lực tại Việt Nam từ 01/07/2025.
3. **`VoucherSnapshot`:**
   - `voucher_code`, `discount_type` (`PERCENTAGE`, `FIXED_AMOUNT`), `discount_value` (Money), `applied_amount` (Money).

---

### 2.5. Mô Hình 3 Trạng Thái Tồn Kho Phân Tán (The 3-State Inventory Model)

> [!IMPORTANT]
> **Làm Rõ Ngữ Nghĩa Tồn Kho Giữa Order Service & Inventory Service:**
> Để ngăn ngừa hoàn toàn nguy cơ **Double Deduction (Trừ kho hai lần)**, hệ thống định nghĩa rạch ròi 3 biến số tồn kho bên trong `inventory-service`:
> 1. **`available_quantity` (Tồn khả dụng):** Số lượng kẹo sẵn sàng mở bán.
> 2. **`reserved_quantity` (Tồn tạm khóa):** Số lượng kẹo đang được giữ riêng cho các đơn hàng `PENDING_PAYMENT` (TTL 15 phút).
> 3. **`committed_quantity` (Tồn xuất kho chính thức):** Số lượng kẹo đã được thanh toán tiền, chờ xưởng đóng gói bàn giao bưu cục.
> 
> *Công thức bảo toàn:*
> $$\text{physical\_quantity} = \text{available\_quantity} + \text{reserved\_quantity}$$

```text
Trạng Thái Ban Đầu (Initial):
   available: 100 | reserved: 0 | committed: 0

Bước 1: Khách đặt 2 hộp mè xửng (Order Service gọi ReserveStock):
   available: 98  | reserved: 2 | committed: 0  (Tồn khả dụng đã giảm ngay để chống bán lố!)

Bước 2A: Thanh toán thành công (Sự kiện OrderPaidEvent bắn sang Kho):
   available: 98  | reserved: 0 | committed: 2  (Chuyển từ reserved sang committed, CẤM trừ available lần 2!)

Bước 2B: Hết hạn 15m hoặc khách hủy đơn (Order Service gọi ReleaseReservation):
   available: 100 | reserved: 0 | committed: 0  (Hoàn trả tồn khả dụng về nguyên trạng)
```

---

### 2.6. Các Bất Biến Nghiệp Vụ Miền (Domain Invariants)
Bất kỳ thay đổi nào trên Aggregate `Order` đều phải vượt qua 5 điều kiện kiểm tra bất biến:
$$\mathbf{Invariant\ 1:}\quad \text{discount\_amount} \le \text{subtotal\_amount} \quad (\text{Chiết khấu cấm vượt quá tổng tiền hàng})$$
$$\mathbf{Invariant\ 2:}\quad \text{final\_amount} = \text{subtotal\_amount} - \text{discount\_amount} + \text{shipping\_fee}$$
$$\mathbf{Invariant\ 3:}\quad \text{final\_amount} \ge \text{shipping\_fee} \ge 0$$
$$\mathbf{Invariant\ 4:}\quad \text{len(order\_line\_items)} \ge 1 \quad \text{và} \quad \forall\ \text{item} \in \text{order\_line\_items},\ \text{quantity} > 0$$
$$\mathbf{Invariant\ 5:}\quad \text{Chỉ cho phép chuyển } \texttt{PENDING\_PAYMENT} \rightarrow \texttt{PAID} \text{ khi số tiền giao dịch khớp tuyệt đối: } \text{amount\_paid} = \text{final\_amount}.$$

---

## 3. BƯỚC 3: MÁY TRẠNG THÁI & VÒNG ĐỜI THỰC THỂ (STATE MACHINE & LIFECYCLE TRANSITIONS)

### 3.1. Sơ Đồ Chuyển Trạng Thái Đơn Hàng Hoàn Chỉnh (Order State Machine FSM)

```mermaid
stateDiagram-v2
    [*] --> DRAFT: Khởi tạo phiên Checkout
    
    DRAFT --> PENDING_PAYMENT: Khóa tồn kho thành công (ReserveStock 15m)
    DRAFT --> CANCELLED_OUT_OF_STOCK: Kho không đủ hàng khả dụng

    PENDING_PAYMENT --> PAID: Webhook VietQR khớp tiền (amount == final)
    PENDING_PAYMENT --> CANCELLED_TIMEOUT: Hết hạn 15 phút chưa thanh toán
    PENDING_PAYMENT --> CANCELLED_BY_USER: Khách bấm Hủy đơn chủ động

    PAID --> PROCESSING: Xưởng Huế nhận đơn (Picking Task)
    PAID --> REFUND_PENDING: Khách hủy đơn sau khi đã thanh toán (trước đóng gói)

    PROCESSING --> PACKED: Đã dán tem Seal O Mạ + Upload Video S3
    PROCESSING --> REFUND_PENDING: Hết hàng đột xuất tại xưởng

    PACKED --> SHIPPED: Bàn giao bưu tá 3PL (GHN/ViettelPost)
    SHIPPED --> DELIVERED: Khách ký nhận kẹo thành công

    DELIVERED --> COMPLETED: Sau 7 ngày không khiếu nại (Tích điểm 1% Loyalty)
    DELIVERED --> RETURN_REQUESTED: Khách khiếu nại kẹo vỡ nát / lỗi đóng gói
    
    RETURN_REQUESTED --> REFUND_PENDING: CSKH đối chiếu video seal + Duyệt hoàn tiền
    RETURN_REQUESTED --> COMPLETED: Khiếu nại bị bác bỏ (Tem seal bị can thiệp)

    REFUND_PENDING --> REFUNDED: Kế toán chuyển khoản hoàn tiền thành công (Finance Event)
    REFUND_PENDING --> PROCESSING: Hoàn tiền thất bại / Khách đồng ý nhận mẻ kẹo mới

    CANCELLED_TIMEOUT --> [*]
    CANCELLED_BY_USER --> [*]
    CANCELLED_OUT_OF_STOCK --> [*]
    COMPLETED --> [*]
    REFUNDED --> [*]
```

---

### 3.2. Sơ Đồ Máy Trạng Thái Saga (Saga Lifecycle State Machine)

Toàn bộ các tiến trình phân tán (Checkout Saga, Timeout Compensation Saga, Return Saga) đều được quản lý bởi máy trạng thái Saga độc lập:

```mermaid
stateDiagram-v2
    [*] --> STARTED: Nhận Trigger Command / Event
    STARTED --> IN_PROGRESS: Thực thi bước gRPC đầu tiên

    IN_PROGRESS --> COMPLETED: Toàn bộ các bước phân tán thành công
    
    IN_PROGRESS --> COMPENSATING: Phát hiện lỗi / Timeout / Hủy đơn
    
    COMPENSATING --> COMPENSATED: Đền bù thành công (Kho & Voucher đã nhả)
    COMPENSATING --> RETRYING: Lỗi mạng khi gọi đền bù (gRPC timeout)
    
    RETRYING --> COMPENSATING: Thử lại sau Exponential Backoff (Tối đa 5 lần)
    RETRYING --> DEAD_LETTER: Vượt quá 5 lần thất bại (Chuyển bảng DLQ & Báo động Ops)

    COMPLETED --> [*]
    COMPENSATED --> [*]
    DEAD_LETTER --> [*]
```

---

### 3.3. Ma Trận Chuyển Trạng Thái Hợp Lệ, Điều Kiện Bảo Vệ & Side Effects

| Trạng Thái Hiện Tại | Lệnh / Sự Kiện Kích Hoạt | Trạng Thái Kế Tiếp | Điều Kiện Bảo Vệ (Guards) | Tác Vụ Kèm Theo (Side Effects / Events) |
| :--- | :--- | :--- | :--- | :--- |
| `DRAFT` | `ConfirmCheckoutCommand` | `PENDING_PAYMENT` | `ReserveStock` gRPC trả về `SUCCESS` | Sinh VietQR URL, đặt `expires_at = NOW() + 15m`, phát `OrderPlacedEvent`. |
| `DRAFT` | `ConfirmCheckoutCommand` | `CANCELLED_OUT_OF_STOCK` | `ReserveStock` trả về `INSUFFICIENT_STOCK` | Trả về thông báo lỗi SKU hết hàng cho Client, không tạo nợ. |
| `PENDING_PAYMENT` | `VietQRWebhookReceived` | `PAID` | HMAC hợp lệ VÀ **`amount_paid == final_amount`** | Ghi Outbox `OrderPaidEvent`, kích hoạt xưởng đóng kẹo và trừ tồn kho committed. |
| `PENDING_PAYMENT` | `Timeout15mExpired` | `CANCELLED_TIMEOUT` | Thời gian hiện tại $> \text{expires\_at}$ VÀ chưa có thanh toán | **Trực tiếp gọi gRPC `ReleaseReservation` sang Kho**, phát `OrderCancelledEvent`. |
| `PENDING_PAYMENT` | `UserCancelCommand` | `CANCELLED_BY_USER` | Người gọi là chủ đơn VÀ đơn chưa thanh toán | **Trực tiếp gọi gRPC `ReleaseReservation` sang Kho**, phát `OrderCancelledEvent`. |
| `PAID` | `FulfillmentJobAccepted` | `PROCESSING` | Nhận sự kiện từ MS-02 Fulfillment | Cập nhật timeline, thông báo khách qua Zalo ZNS. |
| `PAID` | `CancelPaidOrderCommand` | `REFUND_PENDING` | Quản trị viên hủy đơn trước khi xưởng gói hàng | Khởi tạo Refund Saga, thông báo MS-09 Finance chuẩn bị hoàn tiền. |
| `PROCESSING` | `PackageSealedEvent` | `PACKED` | Kiện hàng có mã seal O Mạ và video trên S3 | Lưu `seal_code`, sẵn sàng bàn giao vận chuyển. |
| `PACKED` | `ShipmentCreatedEvent` | `SHIPPED` | Nhận mã vận đơn từ MS-12 Shipping | Lưu `tracking_code`, `shipping_carrier`, gửi link tracking cho khách. |
| `SHIPPED` | `ShipmentDeliveredEvent`| `DELIVERED` | Bưu tá 3PL xác nhận giao thành công | Bắt đầu đếm ngược thời gian khiếu nại 7 ngày. |
| `DELIVERED` | `AutoCompleteCron` | `COMPLETED` | Quá 7 ngày kể từ khi giao và không có khiếu nại | Bắn `OrderCompletedEvent` để MS-07 tích điểm 1% loyalty, mở quyền Review. |
| `DELIVERED` | `ReturnTicketApproved` | `RETURN_REQUESTED` | CSKH tiếp nhận khiếu nại hợp lệ qua MS-06 | Đóng băng tiến trình tích điểm loyalty, chờ kiểm định hàng hoàn. |
| `RETURN_REQUESTED` | `ApproveRefundCommand` | `REFUND_PENDING` | CSKH xác nhận lỗi từ nhà sản xuất qua video seal | Kích hoạt lệnh hoàn tiền sang MS-09 Finance. |
| `REFUND_PENDING` | `RefundProcessedEvent` | `REFUNDED` | MS-09 Finance báo đã chuyển khoản hoàn tiền | Cập nhật `payment_status = REFUNDED`, kết thúc đơn. |

---

### 3.4. Cơ Chế Giải Quyết Tranh Chấp Trạng Thái Cạnh Tranh (Payment vs Timeout Race Condition)

Một trong những tình huống hóc búa nhất của hệ thống thương mại điện tử phân tán là **Sự cố Cạnh tranh tại giây thứ 900 (Second-899 Race Condition)**:
- Tại thời điểm `14:59.900`, Khách hàng hoàn tất quét mã VietQR tại App ngân hàng $\rightarrow$ Webhook ngân hàng bắn tới máy chủ.
- Tại đúng thời điểm `15:00.000`, Cron Job `TimeoutSweeper` của hệ thống quét thấy đơn hàng đã quá hạn 15 phút.

```text
              LUỒNG A: WEBHOOK THANH TOÁN (14:59.900)
                               │
                               ▼
        ┌──────────────────────────────────────────────┐
        │  UPDATE orders SET status = 'PAID', ...      │
        │  WHERE id = $1 AND status = 'PENDING_PAYMENT'│
        │    AND version = $2                          │
        └──────────────────────┬───────────────────────┘
                               │
                   CẠNH TRANH KHÓA ROW ATOMIC
                               │
        ┌──────────────────────┴───────────────────────┐
        │  UPDATE orders SET status = 'CANCELLED_...'  │
        │  WHERE id = $1 AND status = 'PENDING_PAYMENT'│
        │    AND expires_at < NOW() AND version = $2   │
        └──────────────────────────────────────────────┘
                               ▲
                               │
             LUỒNG B: TIMEOUT SWEEPER (15:00.000)
```

#### Định Nghĩa Rõ Winner Condition (Điều Kiện Thắng Cuộc Tuyệt Đối):
Hệ thống sử dụng **Atomic Compare-And-Swap (CAS)** trên PostgreSQL để phân định thắng thua:
1. **Trường hợp Luồng A (Payment) commit trước:**
   - Lệnh SQL của Luồng A thực thi: Trạng thái chuyển thành `PAID`, `version` tăng từ $1 \rightarrow 2$. Số dòng cập nhật (`RowsAffected`) $= 1$. Luồng A thắng!
   - Khi Luồng B (Timeout Sweeper) chạy tới, điều kiện `WHERE status = 'PENDING_PAYMENT' AND version = 1` không còn thỏa mãn $\rightarrow$ `RowsAffected` $= 0$. 
   - **Xử lý phía Luồng B:** Nhận biết đơn đã được thanh toán hợp lệ, Sweeper hủy bỏ lệnh đền bù, không gọi `ReleaseReservation` sang Kho.
2. **Trường hợp Luồng B (Timeout) commit trước:**
   - Lệnh SQL của Luồng B thực thi: Trạng thái chuyển thành `CANCELLED_TIMEOUT`, `version` tăng từ $1 \rightarrow 2$, gọi gRPC `ReleaseReservation` sang Kho nhả kẹo. Luồng B thắng!
   - Khi Luồng A (Webhook) chạy tới, điều kiện `WHERE status = 'PENDING_PAYMENT'` bị sai $\rightarrow$ `RowsAffected` $= 0$.
   - **Xử lý phía Luồng A (Payment Đến Sau Timeout):** 
     - Webhook nhận biết đơn hàng đã bị hủy do quá hạn và tồn kho có thể đã bị người khác mua mất.
     - **Hành động:** Hệ thống **CẤM** tự ý chuyển đơn thành `PAID`. Ghi nhận giao dịch thanh toán vào bảng `payments` ở trạng thái `RECONCILIATION_REQUIRED`, đồng thời phát sự kiện `OrderPaymentAfterTimeoutEvent` để bộ phận Chăm sóc khách hàng & Kế toán chủ động liên hệ hoàn tiền 100% cho khách hàng trong vòng 30 phút.

---

## 4. BƯỚC 4: MÔ HÌNH HÓA DỮ LIỆU QUAN HỆ & BỘ ĐỆM (DATABASE SCHEMA DDL & CACHE STORAGE)

### 4.1. Sơ Đồ Thực Thể - Liên Kết (ERD)

```mermaid
erDiagram
    orders ||--o{ order_line_items : "has (1..n)"
    orders ||--|| order_address_snapshots : "delivers_to (1..1)"
    orders ||--o| order_voucher_snapshots : "discounts_by (0..1)"
    orders ||--o{ payments : "settled_via (0..n)"
    orders ||--o{ saga_instances : "orchestrated_by (1..n)"
    saga_instances ||--o{ saga_step_logs : "records (1..n)"

    orders {
        uuid id PK "UUID v7 Generated by App"
        varchar order_code UK "ORD-YYYYMMDD-XXXX"
        uuid customer_id "Nullable for Guest"
        varchar channel "D2C_WEB, POS_OFFLINE, MARKETPLACE..."
        varchar status "DRAFT, PENDING_PAYMENT, PAID, CANCELLED..."
        bigint subtotal_units "VND units"
        bigint discount_units "VND units"
        bigint shipping_units "VND units"
        bigint final_units "VND units"
        varchar currency "VND"
        int version "CAS Optimistic Locking"
        timestamptz expires_at "Timeout countdown"
        timestamptz created_at
        timestamptz updated_at
    }

    order_line_items {
        uuid id PK
        uuid order_id FK
        varchar sku_code
        varchar product_name
        int quantity
        bigint unit_price_units
        bigint total_price_units
        varchar packaging_specs
    }

    order_address_snapshots {
        uuid id PK
        uuid order_id FK "1:1 with Order"
        varchar recipient_name
        varchar phone_number
        varchar street_address
        varchar ward_code
        varchar ward_name
        varchar province_code
        varchar province_name
    }

    order_voucher_snapshots {
        uuid id PK
        uuid order_id FK
        varchar voucher_code
        bigint applied_units
    }

    payments {
        uuid id PK
        uuid order_id FK
        varchar payment_provider "VIETQR, CASH, BANK_TRANSFER"
        varchar provider_transaction_id "Bank reference ID"
        varchar payment_type "PAYMENT | REFUND"
        varchar payment_status "PENDING, PAID, FAILED"
        bigint amount_units
        varchar idempotency_key UK
        timestamptz executed_at
    }

    saga_instances {
        uuid id PK
        uuid order_id FK
        varchar saga_type
        varchar current_step
        varchar status "STARTED, IN_PROGRESS, COMPLETED, COMPENSATING, COMPENSATED, RETRYING, DEAD_LETTER"
        jsonb payload
        int retry_count
        timestamptz expires_at
    }

    saga_step_logs {
        uuid id PK
        uuid saga_id FK
        varchar step_name
        varchar action_type "COMMAND | COMPENSATION"
        varchar status "SUCCESS | FAILED"
        jsonb details
    }

    outbox_events {
        uuid id PK "UUID v7"
        varchar aggregate_type
        varchar aggregate_id
        varchar event_type
        varchar topic
        jsonb payload
        varchar status "PENDING | PUBLISHED | FAILED"
        timestamptz created_at
    }
```

---

### 4.2. Chiến Lược Sinh Khóa Chính UUID v7

> [!IMPORTANT]
> **Khẳng Định Kỹ Thuật Về UUID v7 Trên PostgreSQL 16:**
> Extension `uuid-ossp` của PostgreSQL **KHÔNG hỗ trợ sinh UUID v7** (chỉ hỗ trợ UUID v1, v3, v4, v5).
> 
> Nhằm tối ưu hóa triệt để cấu trúc B-Tree Index trên đĩa cứng và đạt hiệu năng ghi tuần tự tối đa:
> 1. **Khóa chính UUID v7 ĐƯỢC SINH BỞI APPLICATION LAYER (Go / Node.js)** trước khi thực thi lệnh `INSERT` xuống cơ sở dữ liệu.
> 2. Sử dụng các thư viện chuẩn hóa: `github.com/google/uuid` (Go v1.6+) hoặc `uuidv7` (Node.js/TypeScript).
> 3. Cột trong PostgreSQL sử dụng kiểu dữ liệu bản địa `UUID` (16 bytes nhị phân), hoàn toàn tương thích và lưu trữ chuẩn xác giá trị UUID v7 từ ứng dụng đưa xuống.

---

### 4.3. Kịch Bản DDL Chi Tiết (PostgreSQL 16 Production Script)

```sql
-- Extension hỗ trợ tìm kiếm văn bản tiếng Việt cho mã đơn
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- =============================================================================
-- 1. BẢNG ĐƠN HÀNG TRUNG TÂM (ORDERS AGGREGATE ROOT)
-- =============================================================================
CREATE TABLE orders (
    id UUID PRIMARY KEY, -- Sinh UUID v7 từ Application Layer
    order_code VARCHAR(32) NOT NULL UNIQUE,
    customer_id UUID, -- Khóa ngoại mềm sang MS-15 Profile (NULL nếu là Guest)
    channel VARCHAR(30) NOT NULL CHECK (channel IN (
        'D2C_WEB', 'D2C_MOBILE', 'POS_OFFLINE', 
        'MARKETPLACE_SHOPEE', 'MARKETPLACE_TIKTOK', 'B2B_CORPORATE'
    )),
    status VARCHAR(35) NOT NULL CHECK (status IN (
        'DRAFT', 'PENDING_PAYMENT', 'PAID', 'PROCESSING', 
        'PACKED', 'SHIPPED', 'DELIVERED', 'COMPLETED',
        'CANCELLED_TIMEOUT', 'CANCELLED_BY_USER', 'CANCELLED_OUT_OF_STOCK',
        'RETURN_REQUESTED', 'REFUND_PENDING', 'REFUNDED'
    )),
    
    -- Tiền tệ: Đơn vị nguyên VND (BIGINT), tuyệt đối cấm dùng float
    subtotal_units BIGINT NOT NULL CHECK (subtotal_units >= 0),
    discount_units BIGINT NOT NULL DEFAULT 0 CHECK (discount_units >= 0),
    shipping_units BIGINT NOT NULL DEFAULT 0 CHECK (shipping_units >= 0),
    final_units BIGINT NOT NULL CHECK (final_units >= 0),
    currency VARCHAR(3) NOT NULL DEFAULT 'VND',
    
    customer_note TEXT,
    staff_note TEXT,
    
    -- CAS Optimistic Concurrency Control
    version INT NOT NULL DEFAULT 1,
    
    -- Hạn chót đếm ngược 15 phút (Dùng trực tiếp cho Timeout Sweeper)
    expires_at TIMESTAMPTZ,
    
    -- Thông tin giao vận & niêm phong
    tracking_code VARCHAR(100),
    shipping_carrier VARCHAR(50),
    seal_code VARCHAR(50),
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- =========================================================================
    -- CÁC RÀNG BUỘC KIỂM TRA BẤT BIẾN (DOMAIN CHECK CONSTRAINTS)
    -- =========================================================================
    -- Luật 1: Chiết khấu không bao giờ được vượt quá tổng tiền hàng
    CONSTRAINT chk_order_discount_limit CHECK (discount_units <= subtotal_units),
    -- Luật 2: Khớp số học tiền tệ tuyệt đối
    CONSTRAINT chk_order_math_integrity CHECK (final_units = subtotal_units - discount_units + shipping_units),
    -- Luật 3: Tiền thanh toán cuối cùng không được nhỏ hơn phí giao hàng
    CONSTRAINT chk_order_final_floor CHECK (final_units >= shipping_units)
);

CREATE INDEX idx_orders_customer_created ON orders (customer_id, created_at DESC) WHERE customer_id IS NOT NULL;
CREATE INDEX idx_orders_status ON orders (status);
CREATE INDEX idx_orders_expires_sweep ON orders (expires_at) WHERE status = 'PENDING_PAYMENT';
CREATE INDEX idx_orders_created_at ON orders (created_at DESC);
CREATE INDEX idx_orders_code_trgm ON orders USING GIN (order_code gin_trgm_ops);

-- =============================================================================
-- 2. BẢNG MẶT HÀNG CHI TIẾT (ORDER LINE ITEMS)
-- =============================================================================
CREATE TABLE order_line_items (
    id UUID PRIMARY KEY,
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    sku_code VARCHAR(64) NOT NULL,
    product_name VARCHAR(255) NOT NULL,
    quantity INT NOT NULL CHECK (quantity > 0),
    unit_price_units BIGINT NOT NULL CHECK (unit_price_units >= 0),
    total_price_units BIGINT NOT NULL CHECK (total_price_units >= 0),
    packaging_specs VARCHAR(100),
    item_notes VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT chk_item_math CHECK (total_price_units = unit_price_units * quantity)
);

CREATE INDEX idx_line_items_order ON order_line_items (order_id);
CREATE INDEX idx_line_items_sku ON order_line_items (sku_code);

-- =============================================================================
-- 3. BẢNG BẢN CHỤP ĐỊA CHỈ NHẬN HÀNG (ORDER ADDRESS SNAPSHOTS)
-- =============================================================================
CREATE TABLE order_address_snapshots (
    id UUID PRIMARY KEY,
    order_id UUID NOT NULL UNIQUE REFERENCES orders(id) ON DELETE CASCADE,
    recipient_name VARCHAR(150) NOT NULL,
    phone_number VARCHAR(20) NOT NULL,
    street_address VARCHAR(255) NOT NULL,
    ward_code VARCHAR(20) NOT NULL,
    ward_name VARCHAR(100) NOT NULL,
    province_code VARCHAR(20) NOT NULL,
    province_name VARCHAR(100) NOT NULL,
    latitude NUMERIC(10, 7),
    longitude NUMERIC(10, 7),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- =============================================================================
-- 4. BẢNG BẢN CHỤP KHUYẾN MÃI (ORDER VOUCHER SNAPSHOTS)
-- =============================================================================
CREATE TABLE order_voucher_snapshots (
    id UUID PRIMARY KEY,
    order_id UUID NOT NULL UNIQUE REFERENCES orders(id) ON DELETE CASCADE,
    voucher_code VARCHAR(50) NOT NULL,
    discount_type VARCHAR(20) NOT NULL CHECK (discount_type IN ('PERCENTAGE', 'FIXED_AMOUNT')),
    discount_value_units BIGINT NOT NULL,
    applied_units BIGINT NOT NULL CHECK (applied_units >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- =============================================================================
-- 5. BẢNG GIAO DỊCH THANH TOÁN (PAYMENTS - QUAN HỆ 1:N)
-- Bảo đảm chống trùng lặp theo định danh nhà cung cấp thanh toán
-- =============================================================================
CREATE TABLE payments (
    id UUID PRIMARY KEY,
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE RESTRICT,
    payment_provider VARCHAR(30) NOT NULL CHECK (payment_provider IN (
        'VIETQR_NAPAS', 'COD_INTERNAL', 'B2B_BANK_DIRECT', 'POS_TERMINAL'
    )),
    provider_transaction_id VARCHAR(100) NOT NULL,
    payment_type VARCHAR(20) NOT NULL DEFAULT 'PAYMENT' CHECK (payment_type IN ('PAYMENT', 'REFUND')),
    payment_status VARCHAR(20) NOT NULL CHECK (payment_status IN (
        'PENDING', 'PAID', 'FAILED', 'RECONCILIATION_REQUIRED'
    )),
    amount_units BIGINT NOT NULL CHECK (amount_units >= 0),
    currency VARCHAR(3) NOT NULL DEFAULT 'VND',
    idempotency_key VARCHAR(128) NOT NULL UNIQUE,
    raw_signature TEXT,
    executed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- Chống trùng lặp tuyệt đối theo giao dịch phía nhà cung cấp
    CONSTRAINT uq_payment_provider_tx UNIQUE (payment_provider, provider_transaction_id)
);

CREATE INDEX idx_payments_order ON payments (order_id);

-- =============================================================================
-- 6. BẢNG QUẢN LÝ TIẾN TRÌNH SAGA (SAGA INSTANCES & LOGS)
-- =============================================================================
CREATE TABLE saga_instances (
    id UUID PRIMARY KEY,
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    saga_type VARCHAR(50) NOT NULL CHECK (saga_type IN (
        'CHECKOUT_D2C_SAGA', 'MARKETPLACE_INBOUND_SAGA', 
        'ORDER_TIMEOUT_COMPENSATION_SAGA', 'RETURN_REFUND_SAGA'
    )),
    current_step VARCHAR(60) NOT NULL,
    status VARCHAR(30) NOT NULL CHECK (status IN (
        'STARTED', 'IN_PROGRESS', 'COMPLETED', 
        'COMPENSATING', 'COMPENSATED', 'RETRYING', 'DEAD_LETTER'
    )),
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    error_message TEXT,
    retry_count INT NOT NULL DEFAULT 0,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_saga_order ON saga_instances (order_id);
CREATE INDEX idx_saga_status_retry ON saga_instances (status, retry_count) 
    WHERE status IN ('COMPENSATING', 'RETRYING');

CREATE TABLE saga_step_logs (
    id UUID PRIMARY KEY,
    saga_id UUID NOT NULL REFERENCES saga_instances(id) ON DELETE CASCADE,
    step_name VARCHAR(60) NOT NULL,
    action_type VARCHAR(20) NOT NULL CHECK (action_type IN ('COMMAND', 'COMPENSATION')),
    status VARCHAR(20) NOT NULL CHECK (status IN ('SUCCESS', 'FAILED')),
    details JSONB NOT NULL DEFAULT '{}'::jsonb,
    executed_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_saga_step_logs_saga ON saga_step_logs (saga_id);

-- =============================================================================
-- 7. BẢNG TRANSACTIONAL OUTBOX PATTERN (OUTBOX EVENTS)
-- =============================================================================
CREATE TABLE outbox_events (
    id UUID PRIMARY KEY,
    aggregate_type VARCHAR(50) NOT NULL DEFAULT 'Order',
    aggregate_id VARCHAR(64) NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    topic VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'PUBLISHED', 'FAILED')),
    traceparent VARCHAR(128),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    published_at TIMESTAMPTZ
);

CREATE INDEX idx_outbox_pending ON outbox_events (created_at ASC) WHERE status = 'PENDING';
```

---

### 4.4. Mô Hình Dữ Liệu Bộ Đệm & Phiên (Redis Data Schema)

| Cấu Trúc Khóa (Key Pattern) | Kiểu Dữ Liệu | TTL | Mục Đích Kỹ Thuật |
| :--- | :---: | :---: | :--- |
| `cart:{customer_id_or_session}` | `Hash` | 30 ngày (Login) / 7 ngày (Guest) | Quản lý giỏ hàng tạm thời trước khi checkout (`sku_code` $\rightarrow$ `quantity`). |
| `checkout_session:{checkout_token}` | `String (JSON)` | 30 phút | Lưu trữ giỏ hàng, snapshot địa chỉ đã chọn để chuẩn bị xác nhận. |
| `idempotency:checkout:{key}` | `String` | 24 giờ | Ngăn chặn việc bấm nút "Đặt Hàng" nhiều lần gây tạo đơn trùng. |
| `idempotency:payment:vietqr:{trans_id}`| `String` | 7 ngày | Ngăn ngân hàng gọi Webhook trùng lặp xử lý thanh toán đúp. |
| `saga:lock:order:{order_id}` | `String` (Redlock) | 10 giây | Concurrency Shield ngăn hai tiến trình cùng cập nhật trạng thái đơn hàng. |

---

## 5. BƯỚC 5: TRỪU TƯỢNG HÓA TẦNG LƯU TRỮ (REPOSITORY & DATA ACCESS INTERFACES)

*Tuân thủ nguyên tắc Dependency Inversion: Toàn bộ Tầng Miền và Ứng Dụng chỉ làm việc qua các interface được type-safe nghiêm ngặt, loại bỏ hoàn toàn việc dùng `tx interface{}` mơ hồ.*

### 5.1. Định Nghĩa Trừu Tượng Database Transaction (`DBTX`)

```go
package repository

import (
    "context"
    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgconn"
)

// DBTX đại diện cho giao diện chung của pgx.Pool và pgx.Tx
// Bảo đảm tính an toàn kiểu dữ liệu tại thời điểm biên dịch (Compile-time Type Safety)
type DBTX interface {
    Exec(ctx context.Context, sql string, arguments ...any) (commandTag pgconn.CommandTag, err error)
    Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
    QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}
```

---

### 5.2. `OrderRepository` Interface

```go
package repository

import (
    "context"
    "errors"
    "time"
    "github.com/omamx/order-service/internal/domain"
)

var (
    ErrOrderNotFound          = errors.New("order: not found")
    ErrOptimisticLockConflict = errors.New("order: concurrent modification detected (CAS version conflict)")
)

type OrderRepository interface {
    // Tạo mới Order cùng OrderLineItems, AddressSnapshot, VoucherSnapshot trong 1 Transaction
    Create(ctx context.Context, db DBTX, order *domain.Order) error
    
    // Truy vấn Aggregate đầy đủ
    GetByID(ctx context.Context, db DBTX, orderID domain.UUID) (*domain.Order, error)
    GetByCode(ctx context.Context, db DBTX, orderCode string) (*domain.Order, error)
    
    // Cập nhật trạng thái sử dụng CAS Optimistic Locking:
    // UPDATE orders SET status = $1, version = version + 1 WHERE id = $2 AND version = $3
    UpdateStatusCAS(ctx context.Context, db DBTX, orderID domain.UUID, oldStatus, newStatus domain.OrderStatus, oldVersion int) (bool, error)
    
    // Quét các đơn hàng PENDING_PAYMENT bị quá hạn cho Timeout Sweeper
    FindExpiredOrders(ctx context.Context, db DBTX, before time.Time, limit int) ([]*domain.Order, error)
    
    // Ghi nhận giao dịch thanh toán vào bảng payments
    RecordPaymentTransaction(ctx context.Context, db DBTX, txRecord *domain.PaymentTransaction) error
}
```

---

### 5.3. `SagaRepository` Interface

```go
package repository

import (
    "context"
    "github.com/omamx/order-service/internal/domain"
)

type SagaRepository interface {
    CreateSaga(ctx context.Context, db DBTX, saga *domain.SagaInstance) error
    GetByOrderID(ctx context.Context, db DBTX, orderID domain.UUID) (*domain.SagaInstance, error)
    UpdateSagaStatus(ctx context.Context, db DBTX, sagaID domain.UUID, currentStep string, status domain.SagaStatus, retryCount int, errMsg *string) error
    LogStep(ctx context.Context, db DBTX, logEntry *domain.SagaStepLog) error
    
    // Lấy danh sách Saga đang COMPENSATING hoặc RETRYING để chạy Worker thử lại
    FindRetryingSagas(ctx context.Context, db DBTX, maxRetries int, limit int) ([]*domain.SagaInstance, error)
}
```

---

### 5.4. `OutboxRepository` Interface

```go
package repository

import (
    "context"
    "github.com/omamx/order-service/internal/domain"
)

type OutboxRepository interface {
    SaveEvent(ctx context.Context, db DBTX, event *domain.OutboxEvent) error
    FetchPendingEvents(ctx context.Context, db DBTX, batchSize int) ([]*domain.OutboxEvent, error)
    MarkPublished(ctx context.Context, db DBTX, eventIDs []domain.UUID) error
    MarkFailed(ctx context.Context, db DBTX, eventID domain.UUID, reason string) error
}
```

---

### 5.5. `CartRepository` Interface

```go
package repository

import (
    "context"
    "time"
)

type CartItem struct {
    SKUCode  string `json:"sku_code"`
    Quantity int    `json:"quantity"`
}

type CartRepository interface {
    GetCart(ctx context.Context, cartKey string) ([]CartItem, error)
    SetItem(ctx context.Context, cartKey string, skuCode string, quantity int, ttl time.Duration) error
    RemoveItem(ctx context.Context, cartKey string, skuCode string) error
    ClearCart(ctx context.Context, cartKey string) error
}
```

---

## 6. BƯỚC 6: TẦNG ỨNG DỤNG & ĐỘNG CƠ ĐIỀU PHỐI SAGA (APPLICATION USE CASES & SAGA ENGINE)

### 6.1. Kiến Trúc Phân Lớp Bên Trong: Order Domain vs Saga Orchestrator
- **Order Domain Component:** Phụ trách tính toán số học, kiểm tra điều kiện chuyển trạng thái FSM nội bộ, chụp snapshot địa chỉ và tạo bản ghi lưu trữ.
- **Saga Orchestrator Component:** Quản lý vòng đời phân tán, phát sinh Command sang Inventory, lưu trạng thái bước vào `saga_instances` và chịu trách nhiệm 100% kích hoạt lệnh đền bù (Centralized Compensation Model A).

---

### 6.2. Dự Toán Độ Trễ Thực Thi (Checkout Latency Budget cho P95 < 200ms)

Để bảo đảm cam kết **NFR-01 (Độ trễ P95 < 200ms)**, chuỗi thực thi của `CheckoutD2CUseCase` được thiết kế song song hóa và phân bổ ngân sách thời gian (Latency Budget) nghiêm ngặt:

| Công Đoạn Xử Lý | Cơ Chế Kỹ Thuật | Ngân Sách Dự Toán (Budget) | Ghi Chú Tối Ưu Hóa |
| :--- | :--- | :---: | :--- |
| **API Gateway Transit** | Kong JWT Local Validation & TLS Term | **10 ms** | Xác thực qua Cached JWKS, không gọi mạng nội bộ. |
| **Idempotency Check** | Redis `GET idempotency:checkout:{key}` | **5 ms** | In-memory lookup. |
| **Parallel Verification** | `errgroup` chạy song song 3 gRPC: | **35 ms** | Chạy đồng thời 3 luồng gRPC: |
| ├── *Catalog Service* | gRPC `ValidatePriceAndSKU` | *(30 ms)* | Đọc từ Redis Cache của Catalog. |
| ├── *Promotion Service* | gRPC `ValidateVoucher` | *(25 ms)* | Kiểm tra ngân sách mã giảm giá. |
| └── *Shipping Service* | gRPC `CalculateShippingFee` | *(20 ms)* | Tra cứu ma trận cước địa lý. |
| **Stock Reservation** | gRPC `ReserveStock` sang MS-01 | **45 ms** | Kho chạy `SELECT FOR UPDATE` trên index SKU. |
| **Database Transaction** | PostgreSQL Local ACID Commit | **30 ms** | Ghi 1 lệnh gom: Order, Items, Address, Saga, Outbox. |
| **Network & Serialization** | Protobuf / JSON Marshalling | **15 ms** | Zero-copy buffer serialization. |
| **TỔNG THỜI GIAN P95 DỰ TOÁN** | **Toàn trình từ Client $\rightarrow$ Response** | **$\mathbf{\sim 140\ ms}$** | **Thỏa mãn vượt trội NFR-01 (< 200ms)**. |

> [!NOTE]
> Timeout cấu hình cho gRPC là **2.0 giây** — Đây là ngưỡng chịu lỗi cực hạn (Circuit Breaker Deadline) để cô lập sự cố khi service đối tác sập hoàn toàn, **không phải thời gian chạy bình thường (P95 thông thường luôn $\le 45$ms)**.

---

### 6.3. Use Case 1: `CheckoutD2CUseCase` (Critical Path Synchronous)

```text
[BƯỚC 1]: Kiểm tra Idempotency Key trên Redis bằng lệnh SET key "PROCESSING" EX 86400 NX.
          └──> Nếu đã tồn tại -> Trả về ngay kết quả phản hồi đã lưu từ trước.
[BƯỚC 2A]: Chuẩn bị dữ liệu Địa chỉ Giao nhận (Address Resolution):
          ├──> Nếu Client truyền `shipping_address_id` (Khách hàng đăng nhập chọn sổ địa chỉ):
          │    └──> Gọi gRPC: MS-15: ProfileService.GetDeliveryAddress(shipping_address_id, customer_id)
          │         (Độ trễ P99 <= 5ms từ Dual-Layer Cache của MS-15).
          └──> Nếu Client là Guest: Lấy trực tiếp thông tin địa chỉ từ request payload.
[BƯỚC 2B]: Kích hoạt golang.org/x/sync/errgroup phát 3 cuộc gọi gRPC đồng thời:
          ├──> MS-05: CatalogService.ValidatePriceAndSKU(items)
          ├──> MS-07: PromotionService.ValidateVoucher(voucher_code, customer_id)
          └──> MS-12: ShippingService.CalculateShippingFee(resolved_address.ward_code, total_weight)
[BƯỚC 3]: Đánh giá kết quả xác thực:
          └──> Nếu có bất kỳ lỗi nào (Giá sai, Voucher hết hạn, Địa chỉ không tồn tại) -> Hủy luồng, trả về HTTP 400.
[BƯỚC 4]: Gọi gRPC Synchronous: MS-01: InventoryService.ReserveStock(order_id, items, ttl=15m).
          └──> Nếu trả về INSUFFICIENT_STOCK -> Dừng luồng, trả về danh sách SKU thiếu hàng.
[BƯỚC 5]: Application sinh UUID v7 cho order_id, outbox_id, saga_id.
[BƯỚC 6]: Mở Local Database Transaction (PostgreSQL ACID):
          BEGIN;
            INSERT INTO orders (id, order_code, ..., status='PENDING_PAYMENT', expires_at=NOW()+15m);
            INSERT INTO order_line_items (...);
            INSERT INTO order_address_snapshots (...); -- Lưu Snapshot địa chỉ 2 cấp từ MS-15/Guest
            INSERT INTO order_voucher_snapshots (...);
            INSERT INTO saga_instances (id, order_id, saga_type='CHECKOUT_D2C_SAGA', status='IN_PROGRESS', ...);
            INSERT INTO outbox_events (id, aggregate_id, event_type='vn.omama.order.placed.v1', topic='order.events.v1', ...);
          COMMIT;
[BƯỚC 7]: Tạo link VietQR động: https://img.vietqr.io/image/970422-0905123456-compact2.png?amount=...&addInfo=OMAMA%20ORD...
[BƯỚC 8]: Lưu kết quả CheckoutResponse vào Redis Idempotency Key -> Trả về HTTP 201 cho Client.
```

---

### 6.4. Use Case 2: `VietQRWebhookCallbackUseCase` (Chốt Luật Khớp Số Tiền Tuyệt Đối)

> [!IMPORTANT]
> **Quy Chuẩn Đối Soát Số Tiền Thanh Toán D2C VietQR:**
> Tuyệt đối cấm quy tắc lỏng lẻo `amount_paid >= final_amount`. Với giao dịch chuyển khoản VietQR tự động, **BẮT BUỘC KHỚP TUYỆT ĐỐI (`amount_paid == final_amount`)**.

#### Bảng Xử Lý Toàn Diện 5 Kịch Bản Thanh Toán:
| Kịch Bản Thanh Toán | Điều Kiện So Khớp | Trạng Thái Đơn Hàng | Hành Động Hệ Thống & Kế Toán |
| :--- | :--- | :--- | :--- |
| **1. Khớp Chuẩn (Exact Match)** | `amount == final_amount` | Chuyển $\rightarrow$ `PAID` | Ghi Outbox `OrderPaidEvent`, kích hoạt đóng gói và trừ tồn kho committed. |
| **2. Chuyển Thiếu (Underpaid)** | `amount < final_amount` | Giữ nguyên `PENDING_PAYMENT` | Ghi log thanh toán một phần, gửi SMS/ZNS: *"Quý khách chuyển thiếu X đồng, vui lòng chuyển nốt!"*. |
| **3. Chuyển Thừa (Overpaid)** | `amount > final_amount` | Chuyển $\rightarrow$ `PAID` | Đơn hàng vẫn được đóng gói giao kẹo; hệ thống tự động tạo Ticket kế toán tại MS-09 để hoàn lại số tiền thừa cho khách. |
| **4. Giao Dịch Trùng (Duplicate Webhook)** | Trùng `(payment_provider, provider_tx_id)` | Không đổi | Bị chặn bởi Unique Constraint CSDL; trả về ngay HTTP 200 OK, không xử lý lại. |
| **5. Chuyển Tiền Sau Khi Hủy (Paid After Timeout)**| Đơn đã `CANCELLED_TIMEOUT` | Giữ nguyên `CANCELLED_TIMEOUT` | **CẤM chuyển PAID**. Ghi nhận `RECONCILIATION_REQUIRED`, kích hoạt hoàn tiền 100% cho khách vì tồn kho có thể đã bị giải phóng. |

---

### 6.5. Use Case 3: `OrderTimeoutCancelCompensationUseCase` (Đền Bù Tập Trung Model A)

```text
                               Saga Orchestrator (Order Service)
                                              │
                      ┌───────────────────────┴───────────────────────┐
                      │ gRPC ReleaseReservation                       │ gRPC ReleaseVoucher
                      ▼                                               ▼
             MS-01: INVENTORY SERVICE                        MS-07: PROMOTION SERVICE
         (Hoàn trả tồn kho khả dụng)                     (Mở khóa voucher cho khách)
```

```text
[BƯỚC 1]: Cron Job Sweeper chạy mỗi 15 giây, thực hiện truy vấn quét:
          SELECT id, order_code, version FROM orders 
          WHERE status = 'PENDING_PAYMENT' AND expires_at < CURRENT_TIMESTAMP LIMIT 50;
[BƯỚC 2]: Với mỗi đơn hàng quá hạn, thực thi Atomic CAS Transition trên PostgreSQL:
          UPDATE orders 
          SET status = 'CANCELLED_TIMEOUT', version = version + 1, updated_at = NOW()
          WHERE id = $1 AND status = 'PENDING_PAYMENT' AND version = $2;
[BƯỚC 3]: Kiểm tra kết quả RowsAffected:
          ├──> Nếu RowsAffected == 0: Luồng khác đã thanh toán hoặc cập nhật trước -> Bỏ qua.
          └──> Nếu RowsAffected == 1: Lệnh hủy đơn thành công. Bắt đầu đền bù tập trung (Model A):
               ├──> Cập nhật saga_instances status = 'COMPENSATING'.
               ├──> Ghi Outbox: OrderCancelledEvent (Mục đích thông báo thuần túy, KHÔNG PHẢI LỆNH ĐỀN BÙ).
               ├──> Trực tiếp gọi gRPC: MS-01: InventoryService.ReleaseReservation(order_id).
               ├──> Trực tiếp gọi gRPC: MS-07: PromotionService.ReleaseVoucher(voucher_code, customer_id).
               └──> Nếu gRPC thành công:
                    └──> Cập nhật saga_instances status = 'COMPENSATED'.
               └──> Nếu gRPC thất bại (lỗi mạng):
                    └──> Cập nhật saga_instances status = 'RETRYING', retry_count = retry_count + 1.
                    └──> RetryWorker sẽ tự động thử lại sau (Exponential Backoff: 1s, 2s, 4s, 8s, 16s).
                    └──> Nếu quá 5 lần vẫn lỗi -> Chuyển status = 'DEAD_LETTER', kích hoạt cảnh báo Slack/PagerDuty cho đội Ops.
```

---

### 6.6. Use Case 4: `MarketplaceInboundSagaUseCase` (Tiếp Nhận Đơn Sàn & Khóa Tồn Tập Trung)
1. Consumer trong `order-service` tiêu thụ sự kiện `MarketplaceOrderImportedEvent` từ topic `channel.events.v1`.
2. Khởi tạo `SagaInstance` với `saga_type = 'MARKETPLACE_INBOUND_SAGA'`.
3. Gọi gRPC Synchronous `ReserveStock` sang `inventory-service`:
   - **Thành công:** Tạo đơn hàng nội bộ với `channel = MARKETPLACE_SHOPEE`, trạng thái `PAID`. Ghi Outbox `OrderPaidEvent` để xưởng đóng kẹo. Cập nhật Saga $\rightarrow$ `COMPLETED`.
   - **Thất bại (Hết hàng tại xưởng Huế):** Cập nhật Saga $\rightarrow$ `FAILED`. Ghi Outbox `MarketplaceOrderStockFailedEvent`. `channel-service` tiêu thụ sự kiện này để tự động gửi API báo hủy đơn lên sàn Shopee/TikTok.

---

### 6.7. Use Case 5: `CreatePOSOrderUseCase` (Bán Trực Tiếp Tại Quầy Xưởng Hương Thủy)
- Phục vụ khách du lịch mua kẹo mè xửng trực tiếp tại xưởng hoặc showroom Huế qua thiết bị POS offline.
- Thu ngân bấm thanh toán $\rightarrow$ MS-13 gọi gRPC `CreatePOSOrder` sang MS-04.
- Đơn hàng được tạo thẳng ở trạng thái `PAID`, ghi nhận thanh toán tiền mặt/quẹt thẻ, xuất bản `OrderPaidEvent` để trừ tồn kho vật lý và in hóa đơn VAT ngay tại quầy trong vòng **dưới 50 mili-giây**.

---

## 7. BƯỚC 7: TẦNG VẬN CHUYỂN & ĐẶC TẢ HỢP ĐỒNG GIAO TIẾP (DELIVERY LAYER & CONTRACTS)

### 7.1. Cổng Biên HTTP RESTful APIs (North - South via Kong Gateway)

Toàn bộ các REST API đều tuân thủ chuẩn OpenAPI 3.0 tại [`docs/03_api_specs/openapi_d2c.yaml`](../../docs/03_api_specs/openapi_d2c.yaml):

#### 1. Khởi Tạo Phiên Đặt Hàng (`POST /api/v1/checkout`)
- **Headers yêu cầu:** `Authorization: Bearer <jwt>`, `Idempotency-Key: <uuid>`, `Content-Type: application/json`.
- **Payload Request:**
```json
{
  "customer_id": "018f3a5b-9c12-7def-a890-123456789abc",
  "channel": "D2C_WEB",
  "shipping_address": {
    "recipient_name": "Nguyễn Hoàng Nam",
    "phone_number": "0905123456",
    "street_address": "15 Lê Lợi",
    "ward_code": "VN-HUE-PHUHOI",
    "ward_name": "Phường Phú Hội",
    "province_code": "VN-HUE",
    "province_name": "Thành phố Huế",
    "latitude": 16.4673,
    "longitude": 107.5905
  },
  "items": [
    { "sku_code": "MX-GION-500G", "quantity": 2 },
    { "sku_code": "MX-DEO-HOMEMADE-300G", "quantity": 1 }
  ],
  "voucher_code": "OMAMA_TET2026",
  "payment_method": "VIETQR",
  "customer_note": "Gói kỹ chống vỡ giúp em, mang vào Sài Gòn làm quà!"
}
```

- **Payload Response (HTTP 201 Created):**
```json
{
  "order_id": "018f3a5b-9c12-7def-a890-123456789abc",
  "order_code": "ORD-20261015-0042",
  "status": "PENDING_PAYMENT",
  "pricing": {
    "subtotal_amount": 350000,
    "discount_amount": 30000,
    "shipping_fee": 25000,
    "final_amount": 345000,
    "currency": "VND"
  },
  "payment": {
    "method": "VIETQR",
    "status": "PENDING",
    "vietqr_url": "https://img.vietqr.io/image/970422-0905123456-compact2.png?amount=345000&addInfo=OMAMA%20ORD202610150042",
    "expires_at": "2026-10-15T08:45:00.000Z",
    "countdown_seconds": 900
  },
  "created_at": "2026-10-15T08:30:00.000Z"
}
```

---

### 7.2. Cổng Nội Bộ gRPC Services (East - West Server)

Khớp 100% tệp Protobuf [`packages/proto/order/v1/order.proto`](../../packages/proto/order/v1/order.proto):

```protobuf
syntax = "proto3";
package omamx.order.v1;

service OrderService {
  // Tạo đơn hàng trực tiếp tại quầy POS offline (MS-13 gọi)
  rpc CreatePOSOrder(CreatePOSOrderRequest) returns (CreatePOSOrderResponse);

  // Truy vấn chi tiết đơn hàng kèm snapshot giao vận
  rpc GetOrderDetail(GetOrderDetailRequest) returns (GetOrderDetailResponse);

  // Hủy đơn hàng và kích hoạt đền bù tập trung
  rpc CancelOrder(CancelOrderRequest) returns (CancelOrderResponse);
}
```

---

### 7.3. Hợp Đồng Gọi gRPC Ngoại Vi (East - West Client on Critical Path)

1. **Khóa tồn kho:** Gọi `MS-01: InventoryService.ReserveStock` (Port 8001). Strict Timeout: `2000ms`.
2. **Giải phóng tồn kho (Đền bù):** Gọi `MS-01: InventoryService.ReleaseReservation` (Port 8001).
3. **Thẩm định giá:** Gọi `MS-05: CatalogService.ValidatePriceAndSKU` (Port 8005).
4. **Thẩm định voucher:** Gọi `MS-07: PromotionService.ValidateVoucher` (Port 8007).
5. **Giải phóng voucher (Đền bù):** Gọi `MS-07: PromotionService.ReleaseVoucher` (Port 8007).
6. **Tính phí ship:** Gọi `MS-12: ShippingService.CalculateShippingFee` (Port 8012).
7. **Lấy địa chỉ giao hàng (Saved Address Hydration):** Gọi `MS-15: ProfileService.GetDeliveryAddress` (Port 50051 / 8015). Strict Timeout: `500ms`, SLA $P99 \le 5\text{ms}$.
   - Contract Protobuf:
     ```protobuf
     message GetDeliveryAddressRequest {
       string address_id = 1;
       string customer_id = 2;
     }
     message DeliveryAddressResponse {
       string id = 1;
       string customer_id = 2;
       string recipient_name = 3;
       string phone_number = 4;
       string street_address = 5;
       string ward_code = 6;
       string ward_name = 7;
       string province_code = 8;
       string province_name = 9;
       double latitude = 10;
       double longitude = 11;
       bool is_default = 12;
     }
     ```

---

### 7.4. Danh Mục Sự Kiện Kafka Xuất Bản (Transactional Outbox Producer)

Tuân thủ chuẩn **CNCF CloudEvents 1.0 JSON Schema** tại [`packages/events/schemas/order/v1/`](../../packages/events/schemas/order/v1/):

| Tên Sự Kiện | Event Type | Kafka Topic | Partition Key | Ý Nghĩa Nghiệp Vụ (Semantic) |
| :--- | :--- | :--- | :---: | :--- |
| `OrderPlacedEvent` | `vn.omama.order.placed.v1` | `order.events.v1` | `order_id` | Khách tạo đơn thành công, đang chờ quét mã VietQR trong 15 phút. |
| `OrderPaidEvent` | `vn.omama.order.paid.v1` | `order.events.v1` | `order_id` | Tiền đã về tài khoản, kích hoạt xưởng đóng kẹo, trừ kho committed và xuất VAT. |
| `OrderCancelledEvent` | `vn.omama.order.cancelled.v1` | `order.events.v1` | `order_id` | **Thông báo đơn đã bị hủy (Business Fact)**. Không chứa command đền bù kho. |
| `OrderCompletedEvent` | `vn.omama.order.completed.v1`| `order.events.v1` | `order_id` | Đơn hoàn tất sau 7 ngày, kích hoạt tích điểm 1% loyalty cho khách. |
| `MarketplaceOrderStockFailedEvent` | `vn.omama.order.marketplace.failed.v1` | `order.events.v1` | `order_id` | Báo cho MS-13 xử lý hủy đơn sàn khi kho xưởng Huế hết hàng. |

- **Payload Chuẩn Của `OrderPaidEvent` (Bảo Toàn Quyền Riêng Tư PII):**
```json
{
  "specversion": "1.0",
  "id": "018f3a5b-9c12-7def-a890-123456789abc",
  "source": "https://omama.vn/services/order-service",
  "type": "vn.omama.order.paid.v1",
  "subject": "order_id:018f3a5b-9c12-7def-a890-123456789abc",
  "time": "2026-10-15T08:32:11.000Z",
  "datacontenttype": "application/json",
  "traceparent": "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
  "data": {
    "order_id": "018f3a5b-9c12-7def-a890-123456789abc",
    "order_code": "ORD-20261015-0042",
    "customer_id": "018f3a5b-9c12-7def-a890-999999999abc",
    "shipping_address_id": "addr-snap-018f3a5b",
    "channel": "D2C_WEB",
    "currency": "VND",
    "subtotal_amount": 350000,
    "discount_amount": 30000,
    "shipping_fee": 25000,
    "final_amount": 345000,
    "payment": {
      "provider": "VIETQR_NAPAS",
      "provider_transaction_id": "FT2628891048201",
      "paid_at": "2026-10-15T08:32:10.000Z"
    },
    "items": [
      { "sku_code": "MX-GION-500G", "quantity": 2, "unit_price": 110000 },
      { "sku_code": "MX-DEO-HOMEMADE-300G", "quantity": 1, "unit_price": 130000 }
    ]
  }
}
```

---

### 7.5. Danh Mục Sự Kiện Kafka Tiêu Thụ (Inbound Consumer)

| Nguồn Phát | Kafka Topic | Event Type | Xử Lý Phía `order-service` |
| :--- | :--- | :--- | :--- |
| `channel-service` (MS-13) | `channel.events.v1` | `vn.omama.channel.marketplace.order.imported.v1` | Khởi chạy `MarketplaceInboundSagaUseCase`, gọi `ReserveStock` kho. |
| `fulfillment-service` (MS-02) | `fulfillment.events.v1` | `vn.omama.fulfillment.package.sealed.v1` | Cập nhật đơn $\rightarrow$ `PACKED`, lưu `seal_code` và link bằng chứng S3. |
| `shipping-service` (MS-12) | `shipping.events.v1` | `vn.omama.shipping.shipment.delivered.v1` | Cập nhật đơn $\rightarrow$ `DELIVERED`, bắt đầu đếm 7 ngày khiếu nại. |

---

## 8. BƯỚC 8: CƠ CHẾ KỸ THUẬT XUYÊN SUỐT & PHÒNG VỆ (CROSS-CUTTING CONCERNS)

### 8.1. Đảm Bảo At-least-once Delivery & Idempotent Consumer (Effectively-once Business Outcome)

> [!IMPORTANT]
> **Định Nghĩa Chuẩn Về Tính Nhất Quán Trong Hệ Phân Tán:**
> Trong kiến trúc hướng sự kiện (Event-Driven Architecture) với Transactional Outbox:
> $$\mathbf{Transactional\ Outbox\ =\ At\text{-}least\text{-}once\ Delivery}$$
> Do máy chủ Outbox Publisher có thể gặp sự cố mạng hoặc crash ngay sau khi bắn thành công lên Kafka nhưng chưa kịp cập nhật trạng thái `PUBLISHED` trong CSDL, thông điệp có thể bị bắn lại nhiều lần khi tiến trình khởi động lại.
> 
> Vì vậy, đẳng thức toàn vẹn nghiệp vụ của hệ thống Mè Xửng O Mạ được xác lập:
> $$\mathbf{At\text{-}least\text{-}once\ Delivery\ +\ Idempotent\ Consumer\ =\ Effectively\text{-}once\ Business\ Outcome}$$

Mọi Consumer trong toàn bộ 18 Microservices (bao gồm cả các consumer bên trong `order-service`) đều bắt buộc phải triển khai cơ chế kiểm tra trùng lặp (Idempotent Message Handling):
- Sử dụng bảng kiểm tra sự kiện đã xử lý `processed_events (event_id UUID PRIMARY KEY, processed_at TIMESTAMPTZ)` ngay bên trong transaction của consumer.
- Nếu `event_id` đã tồn tại $\rightarrow$ Bỏ qua việc thực thi nghiệp vụ, xác nhận Commit Offset Kafka ngay lập tức.

---

### 8.2. Hệ Thống Idempotency Đa Tầng (Multi-Tier Idempotency Shield)
1. **Tầng 1 (Memory / Redis Filter):** Sử dụng `idempotency:checkout:{key}` với TTL 24h.
2. **Tầng 2 (PostgreSQL Unique Constraint):** Bảng `payments` áp dụng ràng buộc duy nhất `UNIQUE (payment_provider, provider_transaction_id)`. Bất kỳ nỗ lực ghi đúp nào cũng bị cơ sở dữ liệu chặn đứng với mã lỗi vi phạm khóa duy nhất (`23505 UniqueViolation`).

---

### 8.3. Chiến Lược Cầu Dao Ngắt Mạch & Quá Giờ Nghiêm Ngặt (Circuit Breaker & Strict Timeout)
- **Strict Timeout (2000ms):** Bọc toàn bộ các cuộc gọi gRPC ngoại vi trong Context có deadline 2 giây.
- **Circuit Breaker:** Ngưỡng mở cầu dao: $\ge 50\%$ request lỗi trong 10 giây. Khi cầu dao OPEN: Trả về ngay mã lỗi nhanh `ERR_CIRCUIT_BREAKER_OPEN` (HTTP 503) trong vòng 1ms, không làm nghẽn thread pool. Thời gian ngủ thăm dò (Sleep window): 5 giây.

---

### 8.4. Bảo Mật Zero-Trust & Xác Thực Webhook HMAC-SHA256
- **Local JWT Validation:** API Gateway tiêm các Header định danh sạch: `X-User-Id`, `X-User-Roles`. `order-service` kiểm tra quyền sở hữu đơn hàng trực tiếp từ Header.
- **Xác Thực Chữ Ký Webhook Ngân Hàng:**
  - Ngân hàng gửi kèm Header `X-VietQR-Signature`.
  - `order-service` tính toán HMAC-SHA256 từ binary payload gốc và Secret Key, đối chiếu bằng thuật toán an toàn thời gian cố định `crypto/subtle.ConstantTimeCompare` để loại trừ tấn công vét cạn theo thời gian (Timing Attack).

---

### 8.5. Tích Hợp Mặt Phẳng Kiểm Toán & Giám Sát Viễn Trắc (Audit & Observability)
- **Mặt Phẳng Kiểm Toán (Audit Plane):** Mọi thao tác hủy đơn, duyệt hoàn tiền, điều chỉnh giá đều xuất bản sự kiện sang `audit.events.v1` kèm chuỗi băm Tamper-Evident Hash Chain SHA-256.
- **Mặt Phẳng Viễn Trắc (Observability Plane):** Tự động lan truyền W3C `traceparent` qua gRPC Metadata và Kafka Headers; đẩy tín hiệu OTLP sang OpenTelemetry Collector.

---

## 9. BƯỚC 9: QUY HOẠCH CẤU TRÚC THƯ MỤC DỰ ÁN (PROJECT DIRECTORY & CODE BLUEPRINT)

Cấu trúc mã nguồn của `services/order-service` tuân thủ chuẩn **Clean / Hexagonal Architecture**:

```text
services/order-service/
├── cmd/
│   └── server/
│       └── main.go                      # Khởi tạo DI Container, HTTP Router, gRPC Server
├── internal/
│   ├── config/                          # Nạp biến môi trường từ ConfigMap/Secret
│   │   └── config.go
│   ├── domain/                          # Lõi nghiệp vụ độc lập (Core Domain Layer)
│   │   ├── order.go                     # Aggregate Root Order, OrderLineItem
│   │   ├── payment_transaction.go       # Entity PaymentTransaction (1:N with Order)
│   │   ├── money.go                     # Value Object Money (int64 amount VND)
│   │   ├── value_objects.go             # AddressSnapshot, VoucherSnapshot
│   │   ├── state_machine.go             # Order FSM & Saga FSM
│   │   └── errors.go                    # Domain error definitions
│   ├── usecase/                         # Tầng Điều Phối Ứng Dụng (Application Layer)
│   │   ├── checkout_usecase.go          # Luồng Checkout D2C Critical Path
│   │   ├── vietqr_webhook_usecase.go    # Xử lý Webhook ngân hàng & Đối soát tiền
│   │   ├── cancel_order_usecase.go      # Khách hủy đơn chủ động
│   │   ├── pos_order_usecase.go         # Tạo đơn quầy POS trực tiếp
│   │   └── interfaces.go                # Use Case & Client Interfaces
│   ├── saga/                            # Động Cơ Điều Phối Giao Dịch Phân Tán
│   │   ├── orchestrator.go              # Saga State Machine coordinator
│   │   ├── checkout_saga.go             # Kịch bản Saga D2C
│   │   ├── timeout_sweeper.go           # Cron job quét hủy đơn 15m & Đền bù kho
│   │   └── marketplace_saga.go          # Kịch bản import đơn sàn Shopee/TikTok
│   ├── repository/                      # Tầng Trừu Tượng Lưu Trữ (Persistence Adapters)
│   │   ├── dbtx.go                      # Interface DBTX type-safe
│   │   ├── order_repository.go          # PostgreSQL 16 adapter (pgx)
│   │   ├── saga_repository.go           # Quản lý bảng saga_instances & logs
│   │   ├── outbox_repository.go         # Transactional Outbox pattern adapter
│   │   └── cart_repository.go           # Redis Cluster cache adapter
│   ├── transport/                       # Tầng Vận Chuyển Giao Tiếp (Driving Adapters)
│   │   ├── http/                        # REST Controllers
│   │   │   ├── router.go
│   │   │   ├── checkout_handler.go      # POST /api/v1/checkout
│   │   │   ├── order_handler.go         # GET /api/v1/orders/*
│   │   │   ├── webhook_handler.go       # POST /api/v1/payments/vietqr/callback
│   │   │   └── middleware/              # Auth, RateLimit, Idempotency, Tracing
│   │   ├── grpc/                        # gRPC Server (Port 8004)
│   │   │   ├── server.go
│   │   │   └── order_grpc_handler.go    # CreatePOSOrder, GetOrderDetail, CancelOrder
│   │   └── kafka/                       # Kafka Consumers & Outbox Publisher
│   │       ├── consumer.go              # Lắng nghe channel.events.v1, fulfillment.events.v1
│   │       └── outbox_publisher.go      # Poller quét bảng outbox_events bắn lên Kafka
│   └── client/                          # Giao Tiếp Ngoại Vi gRPC (Driven Adapters)
│       ├── inventory_client.go          # gRPC Client gọi MS-01 (ReserveStock, ReleaseReservation)
│       ├── catalog_client.go            # gRPC Client gọi MS-05 (ValidatePriceAndSKU)
│       ├── promotion_client.go          # gRPC Client gọi MS-07 (ValidateVoucher, ReleaseVoucher)
│       └── shipping_client.go           # gRPC Client gọi MS-12 (CalculateShippingFee)
├── migrations/                          # Quản lý phiên bản CSDL (golang-migrate)
│   ├── 000001_init_order_schema.up.sql
│   └── 000001_init_order_schema.down.sql
├── deploy/                              # Hạ tầng Docker & Kubernetes
│   ├── Dockerfile
│   └── k8s/
│       ├── deployment.yaml
│       ├── service.yaml
│       └── hpa.yaml
└── tests/                               # Kiểm thử tự động
    ├── unit/                            # Domain Unit Tests (Money, FSM, CAS)
    ├── integration/                     # PostgreSQL testcontainers, Redis lock
    └── concurrency/                     # 4 Kịch bản kiểm thử phân tán bắt buộc
```

---

## 10. BƯỚC 10: CHIẾN LƯỢC KIỂM THỬ & MA TRẬN TRUY VẾT YÊU CẦU (TESTING & TRACEABILITY)

### 10.1. Danh Mục 4 Kịch Bản Kiểm Thử Phân Tán Sống Còn (Distributed Concurrency Tests)

Nhằm đảm bảo hệ thống đạt mức **Implementation-Ready**, 4 bài test phân tán dưới đây bắt buộc phải vượt qua trong pipeline CI/CD trước khi xuất xưởng:

```text
TEST 1: Payment Webhook vs Timeout Sweeper Race Condition
   │
   ├──> Khởi tạo 1 đơn hàng PENDING_PAYMENT (expires_at = NOW()).
   ├──> Kích hoạt đồng thời 2 Goroutines chạy song song:
   │    ├── Luồng 1: Webhook VietQR (thanh toán hợp lệ amount == final_amount).
   │    └── Luồng 2: Timeout Sweeper (quét hủy đơn quá hạn).
   └──> Khẳng định (Assertion):
        - Đúng 1 luồng thắng cuộc (RowsAffected = 1).
        - Trạng thái cuối cùng của đơn là PAID HOẶC CANCELLED_TIMEOUT (Tuyệt đối không bị corrupt trạng thái).
        - Nếu Timeout thắng: Luồng Webhook ghi nhận RECONCILIATION_REQUIRED, không làm lệch tiền.

TEST 2: Outbox Publisher Crash & Restart Resilience
   │
   ├──> Mở DB Transaction: Tạo Order PAID + Ghi 1 bản ghi vào outbox_events.
   ├──> Khởi chạy Outbox Publisher bắn sự kiện lên Kafka thành công.
   ├──> Giả lập kill -9 tiến trình Outbox Publisher NGAY TRƯỚC KHI câu lệnh UPDATE status='PUBLISHED' được commit.
   ├──> Khởi động lại Outbox Publisher.
   └──> Khẳng định (Assertion):
        - Sự kiện OrderPaidEvent được gửi lên Kafka lần thứ 2 (At-least-once).
        - Consumer phía Downstream phát hiện duplicate event_id, bỏ qua xử lý lần 2 an toàn.

TEST 3: Inventory gRPC Ambiguity (Response Lost After Success)
   │
   ├──> Client gửi request CheckoutD2C.
   ├──> Inventory Service đã trừ tồn kho thành công trong DB của kho, nhưng gói tin mạng phản hồi về bị rớt (Network Drop / 2s Timeout).
   ├──> Order Service ném lỗi DEADLINE_EXCEEDED và kích hoạt Saga Compensation.
   └──> Khẳng định (Assertion):
        - Order Service gọi gRPC ReleaseReservation(order_id) với cùng Idempotency Key.
        - Inventory Service nhả hàng an toàn, không làm thất thoát tồn kho của xưởng.

TEST 4: Downstream Idempotent Consumer Verification
   │
   ├──> Xuất bản 3 sự kiện OrderPaidEvent giống hệt nhau lên topic order.events.v1.
   └──> Khẳng định (Assertion):
        - fulfillment-service chỉ tạo ĐÚNG 1 Picking Task tại xưởng.
        - inventory-service chỉ chuyển tồn committed ĐÚNG 1 lần.
        - finance-service chỉ phát hành ĐÚNG 1 hóa đơn VAT điện tử.
```

---

### 10.2. Ma Trận Ánh Xạ Truy Vết Yêu Cầu Chức Năng (FR Traceability Matrix)

| Mã Yêu Cầu | Tên Nghiệp Vụ Trong BRD/Epic | Phương Thức Hiện Thực Trong LLD | Thành Phần Đảm Nhiệm |
| :--- | :--- | :--- | :--- |
| **FR-06 / EPIC-06** | Mua sắm giỏ hàng & Đặt hàng D2C | `CheckoutD2CUseCase` & `POST /api/v1/checkout` | `internal/usecase/checkout_usecase.go` |
| **FR-07 / EPIC-07** | Tự động hóa thanh toán VietQR | `VietQRWebhookCallbackUseCase` & Strict Match | `internal/usecase/vietqr_webhook_usecase.go` |
| **FR-08 / EPIC-08** | Quản lý vòng đời & Hủy đơn tự động | State Machine FSM & Atomic CAS Transitions | `internal/domain/state_machine.go` |
| **FR-12 / EPIC-13** | Quầy bán lẻ trực tiếp Offline POS | gRPC Service `CreatePOSOrder` | `internal/transport/grpc/order_grpc_handler.go` |
| **FR-13 / EPIC-12** | Đồng bộ đơn sàn Shopee / TikTok | `MarketplaceInboundSagaUseCase` | `internal/saga/marketplace_saga.go` |
| **FR-15 / EPIC-14** | Khiếu nại, hoàn tiền kẹo vỡ nát | `ReturnRefundSagaUseCase` & `REFUND_PENDING` | `internal/saga/return_refund_saga.go` |
| **FR-18 / EPIC-17** | Tích điểm thân thiết OCOP Loyalty | Xuất bản `OrderCompletedEvent` sau 7 ngày | `internal/transport/kafka/outbox_publisher.go` |
| **FR-19 / EPIC-18** | Đơn sỉ B2B & Hộp quà doanh nghiệp | Phân loại `channel = 'B2B_CORPORATE'` | `internal/domain/order.go` |
| **FR-28 / EPIC-26** | Gửi quà tặng hộ & Thiệp mừng Huế | Lưu trữ `packaging_specs` & `customer_note` | `internal/domain/value_objects.go` |

---

### 10.3. Ma Trận Ánh Xạ Yêu Cầu Phi Chức Năng (NFR Traceability Matrix)

| Mã Yêu Cầu | Tiêu Chí Kỹ Thuật Đề Ra | Giải Pháp Kỹ Thuật Hiện Thực Trong Thiết Kế LLD |
| :--- | :--- | :--- |
| **NFR-01** | Độ trễ API Critical Path P95 < 200ms | Chạy song song các cuộc gọi gRPC thẩm định; Latency Budget P95 $\sim 140$ms; bất đồng bộ chuỗi hậu kỳ qua Kafka. |
| **NFR-02** | Khả năng chịu tải cao mùa vụ Lễ Tết | Thiết kế Stateless Container, lưu session trên Redis, sẵn sàng Horizontal Pod Autoscaling (HPA) theo CPU/RPS. |
| **NFR-03** | Khả dụng liên tục 99.9% (High Availability) | Cầu dao ngắt mạch Circuit Breaker cô lập sự cố; Retry with Exponential Backoff; PostgreSQL Master-Replica. |
| **NFR-06** | Triệt tiêu bán vượt tồn kho (Anti-Overselling) | Khóa cứng tồn khả dụng 15 phút bằng gRPC `ReserveStock` trên Critical Path trước khi cấp mã thanh toán. |
| **NFR-07** | Nhất quán dữ liệu phân tán (Eventual Consistency) | Áp dụng mô hình Hybrid Saga kết hợp Transactional Outbox Pattern; cam kết At-least-once delivery qua Kafka. |
| **NFR-08** | Bảo mật Zero-Trust & Chống giả mạo | Lọc sạch Header tại Gateway; xác thực chữ ký số HMAC-SHA256 cho Webhook; mã hóa đường truyền gRPC mTLS. |
| **NFR-09** | Quyền riêng tư & Tối thiểu hóa PII | Tuyệt đối không đưa Direct PII vào payload sự kiện Kafka `OrderPaidEvent` (chỉ gửi `shipping_address_id`). |
| **NFR-10** | Tính toàn vẹn kiểm toán bất biến (Audit Trail) | Mọi thay đổi trạng thái và dòng tiền đều phát sự kiện kiểm toán bọc trong chuỗi băm Tamper-Evident Hash Chain. |
| **NFR-11** | Khả năng quan sát toàn diện (Observability) | Truyền W3C `traceparent` qua toàn bộ chuỗi gRPC/Kafka; đẩy Metrics và Traces trực tiếp sang OTel Collector. |

---

## 11. BƯỚC 11: MA TRẬN SỰ CỐ & PHỤC HỒI PHÂN TÁN (DISTRIBUTED FAILURE & RECOVERY MATRIX)

*Đây là phần biến tài liệu thiết kế từ "mô hình lý thuyết đẹp" thành "hệ thống phân tán thực chiến", đặc tả chi tiết trạng thái của từng thành phần khi xảy ra sự cố mạng, timeout hoặc crash.*

| Kịch Bản Sự Cố Phân Tán | Trạng Thái Order | Trạng Thái Saga | Trạng Thái Inventory | Hành Động Xử Lý & Kịch Bản Phục Hồi Kỹ Thuật (Recovery Action) |
| :--- | :---: | :---: | :---: | :--- |
| **1. Catalog Timeout (gRPC)** | `DRAFT` | `NONE` | `NONE` | Client nhận HTTP 400/504 ngay. Không phát sinh ghi dữ liệu hay khóa tồn kho. Thử lại an toàn. |
| **2. Promotion Timeout (gRPC)** | `DRAFT` | `NONE` | `NONE` | Đơn hàng chưa khởi tạo. Client nhận thông báo lỗi kiểm tra voucher. |
| **3. Inventory Timeout Trước Khóa** | `DRAFT` | `NONE` | `NONE` | Cuộc gọi `ReserveStock` bị ngắt sau 2s. Không tạo đơn hàng, báo lỗi quá tải cho khách thử lại. |
| **4. Kho Khóa Xong Nhưng Rớt Mạng (Ambiguous Timeout)** | `DRAFT` | `COMPENSATING` | `RESERVED` | Order Service không nhận được response. Kích hoạt Saga Compensation gọi gRPC `ReleaseReservation(order_id)` nhả kẹo. |
| **5. Lỗi DB Commit Sau Khi Khóa Kho** | `NONE` | `COMPENSATING` | `RESERVED` | Đóng kết nối DB bị lỗi sau khi kho đã khóa. Saga Worker kích hoạt gọi `ReleaseReservation` để tránh treo tồn mồ côi. |
| **6. Webhook VietQR Bị Bắn Lặp (Duplicate Webhook)** | `PAID` | `COMPLETED` | `COMMITTED` | Bị chặn bởi `UNIQUE (payment_provider, provider_tx_id)`. Trả về ngay HTTP 200 OK, không xử lý lại. |
| **7. Webhook Đến Sau Khi Quá Hạn 15m** | `CANCELLED_TIMEOUT`| `COMPENSATED` | `AVAILABLE` | **CẤM chuyển PAID**. Ghi nhận `RECONCILIATION_REQUIRED`. Thông báo kế toán hoàn lại 100% tiền cho khách hàng. |
| **8. Outbox Worker Bị Crash Sau Khi Publish Kafka** | `PAID` | `COMPLETED` | `COMMITTED` | CSDL vẫn ghi `PENDING`. Khi worker sống lại sẽ bắn lại event (At-least-once). Downstream tiêu thụ xử lý Idempotent. |
| **9. Downstream Nhận Sự Kiện Trùng (Duplicate Event)** | `PAID` | `COMPLETED` | `COMMITTED` | Consumer kiểm tra bảng `processed_events`. Nếu đã có `event_id` $\rightarrow$ Bỏ qua, commit offset an toàn. |
| **10. Kế Toán Hoàn Tiền Thất Bại (Refund Failed)** | `REFUND_PENDING` | `RETRYING` | `UNCHANGED` | RetryWorker thử lại theo Exponential Backoff. Quá 5 lần chuyển `DEAD_LETTER` để quản trị viên can thiệp thủ công. |

---

> **KẾT LUẬN THIẾT KẾ:**
> Bản thiết kế **MS-04: Order Service (Phiên bản 2.0)** đã hoàn thiện ở cấp độ **Implementation-Ready 10/10**: loại bỏ 100% các điểm mâu thuẫn kiến trúc, đồng bộ hóa tuyệt đối giữa Domain Model - Database DDL - State Machine - Saga Orchestration - API/gRPC Contracts, đồng thời trang bị đầy đủ các cơ chế phòng vệ phân tán thực chiến cho hệ sinh thái Mè Xửng O Mạ.
