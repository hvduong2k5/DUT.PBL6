# DUT.PBL6 — Cấu trúc thư mục dự án

> Dự án hệ thống thương mại điện tử (Omama) — microservices architecture.
> Ngày cập nhật: 2026-11-05

---

## 1. Tổng quan thư mục gốc

```text
DUT.PBL6/
├── .github/                 # CI/CD workflows và templates hiện đại hóa
├── .idea/                   # Cấu hình IntelliJ IDEA
├── .vscode/                 # Cấu hình VS Code
├── .system_generated/       # File generate tự động (venv, artifacts)
├── apps/                    # Ứng dụng frontend (Web + Mobile)
├── docs/                    # Tài liệu: yêu cầu, kiến trúc, API specs, testing, deployment, operations
├── infra/                   # Infrastructure: Docker, K8s, Kafka, Monitoring, Nginx
├── mocks/                   # Mock server (Mockoon) cho phát triển
├── packages/                # Package chia sẻ: proto, contracts, events, common
├── scripts/                 # Script kiểm tra và validate specs
├── services/                # Microservices backend (Java + Go)
├── README.md                # Hướng dẫn tổng quan dự án
├── project_bible.md         # Tham chiếu kỹ thuật chi tiết
└── task.md                  # Danh sách công việc hiện tại
```

---

## 2. Thư mục `apps/` — Frontend applications

```text
apps/
├── web-user/                # Ứng dụng web người dùng (Next.js, TypeScript)
│   ├── public/              # Static assets
│   ├── scripts/             # Script hỗ trợ frontend
│   ├── src/                 # Source code TypeScript
│   │   ├── app/             # Next.js App Router (pages, routes)
│   │   ├── components/      # React components chung
│   │   ├── features/        # Feature-based modules
│   │   │   └── employee/    # Màn hình quản trị nhân viên
│   │   │       ├── access/          # Quản lý quyền truy cập
│   │   │       ├── auth/            # Xác thực nhân viên
│   │   │       ├── b2b/             # B2B admin
│   │   │       ├── catalog/         # Quản lý danh mục sản phẩm (catalog admin)
│   │   │       ├── inventory/       # Quản lý tồn kho
│   │   │       ├── orders/          # Quản lý đơn hàng
│   │   │       ├── packing/         # Quản lý đóng gói
│   │   │       ├── payments/        # Quản lý thanh toán
│   │   │       ├── returns/         # Quản lý đổi trả
│   │   │       ├── shared/          # Component chia sẻ employee
│   │   │       ├── shipping/        # Quản lý vận chuyển
│   │   │       └── support/         # Hỗ trợ khách hàng
│   │   ├── lib/             # Thư viện tiện ích (api-contract, auth, money, etag, ...)
│   │   ├── services/        # Service layer gọi API
│   │   └── styles/          # CSS/Tailwind styles
│   ├── .env.example
│   ├── .env.identity.example
│   ├── .env.oidc.example
│   ├── eslint.config.mjs
│   ├── next.config.mjs
│   ├── package.json
│   ├── tsconfig.json
│   └── vitest.config.mts
│
└── mobile-app/              # Ứng dụng di động (Flutter, Dart)
    ├── android/
    ├── config/
    ├── ios/
    ├── lib/                 # Source code Dart
    ├── linux/
    ├── macos/
    ├── test/
    ├── web/
    ├── windows/
    ├── pubspec.yaml
    ├── analysis_options.yaml
    └── README.md
```

---

## 3. Thư mục `services/` — Microservices backend

Dự án có 10 microservices, phân thành 2 nhóm ngôn ngữ: Java (Spring Boot) và Go.

```text
services/
├── analytics-service/       # Java — Phân tích dữ liệu (chỉ có src/)
├── api-gateway/             # Kong gateway cấu hình (không có source code chính)
│   ├── config/
│   ├── docker-compose.yml
│   ├── docker-compose.prod.yml
│   └── .env.example
├── care-service/            # Java — Chăm sóc khách hàng (chỉ có bin/)
├── catalog-service/         # Java Spring Boot — Danh mục sản phẩm (đã làm việc chi tiết)
│   ├── src/
│   │   ├── main/
│   │   │   ├── java/com/hvduong/catalog/
│   │   │   │   ├── api/                 # REST Controllers (public + admin)
│   │   │   │   │   │   ├── CatalogAdminController.java
│   │   │   │   │   │   ├── CatalogHttpController.java
│   │   │   │   │   │   └── CatalogHttpErrors.java
│   │   │   │   ├── application/
│   │   │   │   │   ├── dto/
│   │   │   │   │   │   ├── request/
│   │   │   │   │   │   │   ├── CheckoutSnapshotRequest.java
│   │   │   │   │   │   ├── CreateProductRequest.java
│   │   │   │   │   │   ├── ProductFilterRequest.java
│   │   │   │   │   │   └── UpdatePriceRequest.java
│   │   │   │   │   │   └── response/
│   │   │   │   │   │       ├── AdminProductDetailResponse.java
│   │   │   │   │   │   │   ├── AdminProductListResponse.java
│   │   │   │   │   │   │   ├── CheckoutSnapshotResponse.java
│   │   │   │   │   │   │   ├── ProductDetailResponse.java
│   │   │   │   │   │   │   └── ProductVariantResponse.java
│   │   │   │   ├── mapper/
│   │   │   │   ├── repository/
│   │   │   │   ├── service/
│   │   │   │   ├── common/
│   │   │   │   ├── grpc/
│   │   │   │   ├── config/
│   │   │   │   └── domain/
│   │   │   └── resources/
│   │   │       ├── application.yml
│   │   │       ├── db/migration/        # Flyway migrations (V1-V11)
│   │   │       └── mapper/sql/         # MyBatis SQL XML
│   │   └── test/
│   │       └── java/com/hvduong/catalog/
│   │           └── ... (unit tests)
│   ├── build.gradle
│   ├── gradlew / gradlew.bat
│   ├── settings.gradle
│   ├── .env / .env.example
│   └── phan-tich.md
├── identity-service/        # Java Spring Boot — Xác thực và phân quyền
├── order-service/           # Go — Quản lý đơn hàng
│   ├── cmd/
│   ├── internal/
│   ├── migrations/
│   ├── scripts/
│   ├── src/
│   ├── tests/
│   ├── tools/
│   ├── go.mod / go.sum
│   ├── Dockerfile
│   └── docker-compose*.yml
├── inventory-service/       # Go — Quản lý tồn kho
├── notification-service/    # Java — Thông báo
├── order-service/           # (đã liệt kê ở trên)
├── profile-service/         # Go — Hồ sơ người dùng
└── shipping-service/        # Go — Vận chuyển
```

### Chi tiết `catalog-service` (đã làm việc)

```
services/catalog-service/
├── src/main/java/com/hvduong/catalog/
│   ├── api/
│   │   ├── CatalogAdminController.java    # Admin REST: GET/POST/PUT products, skus, prices, transition
│   │   ├── CatalogHttpController.java     # Public REST: categories, products, search, checkout snapshot
│   │   └── CatalogHttpErrors.java         # Global exception handler (HTTP)
│   ├── application/
│   │   ├── dto/
│   │   │   ├── request/
│   │   │   │   ├── CheckoutSnapshotRequest.java
│   │   │   │   ├── CreateProductRequest.java
│   │   │   │   ├── CreateVariantRequest.java
│   │   │   │   ├── ProductFilterRequest.java
│   │   │   │   ├── UpdatePriceRequest.java
│   │   │   │   └── ...
│   │   │   └── response/
│   │   │       ├── AdminProductDetailResponse.java
│   │   │       ├── AdminProductListResponse.java
│   │   │       ├── CheckoutSnapshotResponse.java
│   │   │       ├── ProductDetailResponse.java
│   │   │       └── ProductVariantResponse.java
│   │   ├── mapper/
│   │   │   └── CatalogDtoMapper.java      # MapStruct: Entity ↔ DTO
│   │   ├── service/
│   │   │   ├── CatalogService.java        # Interface public catalog
│   │   │   ├── CatalogAdminService.java   # Interface admin catalog
│   │   │   ├── impl/
│   │   │   │   ├── CatalogServiceImpl.java
│   │   │   ├── CatalogAdminServiceImpl.java
│   │   │   └── CheckoutSnapshotService.java
│   │   ├── common/
│   │   │   ├── response/
│   │   │   │   ├── ApiResponse.java       # Envelope: success/code/message/data/errors/requestId/timestamp
│   │   │   │   └── PageResponse.java
│   │   │   ├── enums/
│   │   │   │   ├── ListingStatus.java     # ACTIVE/SUSPENDED/ARCHIVED
│   │   │   │   ├── SalesChannel.java      # D2C_WEB/MOBILE_APP/...
│   │   │   │   └── SortBy.java            # PRICE_ASC/PRICE_DESC/BEST_SELLING/NEWEST
│   │   │   ├── exception/
│   │   │   │   ├── CatalogException.java
│   │   │   │   └── ErrorCode.java
│   │   │   └── handler/
│   │   │       └── GlobalRestExceptionHandler.java
│   │   ├── grpc/
│   │   │   ├── CatalogGrpcService.java    # gRPC internal service
│   │   │   ├── CatalogGrpcMapper.java     # Proto ↔ DTO mapper
│   │   │   └── CatalogAccess.java         # gRPC interceptor: internal token validation
│   │   ├── config/
│   │   │   ├── SecurityConfig.java
│   │   │   ├── KafkaConsumerConfig.java
│   │   │   ├── KafkaProducerConfig.java
│   │   │   ├── MyBatisConfig.java
│   │   │   ├── OpenApiConfig.java
│   │   │   └── UuidTypeHandler.java
│   │   ├── domain/
│   │   │   ├── entity/
│   │   │   │   ├── Product.java
│   │   │   │   ├── ProductVariant.java
│   │   │   │   ├── ProductImage.java
│   │   │   │   ├── Category.java
│   │   │   │   └── ChannelPrice.java
│   │   │   └── ...
│   │   └── repository/
│   │       └── mybatis/
│   │           ├── ProductMapper.java
│   │           ├── ProductVariantMapper.java
│   │           ├── ChannelPriceMapper.java
│   │           ├── CategoryMapper.java
│   │           ├── ProductImageMapper.java
│   │           ├── CheckoutSnapshotMapper.java
│   │           ├── StockAvailabilityMapper.java
│   │           └── projection/
│   │               ├── AdminStatusCountRow.java
│   │               └── AdminCategoryRow.java
│   ├── src/main/resources/
│   │   ├── application.yml
│   │   ├── application-local.yml
│   │   ├── application-test.yml
│   │   ├── db/migration/
│   │   │   ├── V1__create_categories.sql
│   │   │   ├── V2__create_products.sql
│   │   │   ├── V3__create_product_variants.sql
│   │   │   ├── V4__create_channel_prices.sql
│   │   │   ├── V5__create_product_images.sql
│   │   │   ├── V6__add_optimistic_lock_versions.sql
│   │   │   ├── V7__create_sku_stock_projection.sql
│   │   │   ├── V8__create_outbox_events.sql
│   │   │   ├── V9__catalog_constraints_indexes_and_updated_at.sql
│   │   │   ├── V10__checkout_snapshots.sql
│   │   │   └── V11__channel_price_exclusion_constraint.sql
│   │   └── mapper/sql/
│   │       ├── ProductMapper.xml
│   │       ├── ProductVariantMapper.xml
│   │       ├── ChannelPriceMapper.xml
│   │       └── ...
│   └── src/test/java/com/hvduong/catalog/
│       └── ... (33+ unit tests: CatalogServiceImplTest, CatalogAdminControllerTest, etc.)
```

---

## 4. Thư mục `docs/` — Tài liệu dự án

```text
docs/
├── 01_requirements/         # Yêu cầu hệ thống
│   ├── 01_Product_Backlog.md
│   ├── 02_Sprint_Planning.md
│   ├── 03_Non_Functional_Requirements.md
│   ├── 04_Functional_Requirements.md
│   ├── use_case_specification.md
│   ├── README.md
│   ├── diagrams/            # Diagram yêu cầu
│   └── epics/               # Epic specifications
│
├── 02_architecture/         # Kiến trúc hệ thống
│   ├── high_level_design.md
│   ├── system_design.md
│   ├── domain_design.md
│   ├── bounded_context.md
│   ├── service_boundary.md
│   ├── kong_gateway_and_domain_guide.md
│   ├── ai_agent_specification_and_proposal.md
│   ├── high_level_diagram/  # Diagram kiến trúc cao cấp
│   ├── services/            # Thiết kế chi tiết từng service
│   │   ├── identity-service/
│   │   ├── order-service/
│   │   └── profile-service/
│   ├── adr/                 # Architecture Decision Records
│   │   ├── order-service/
│   │   └── profile-service/
│   └── ms15_profile_service_design.md
│
├── 03_api_specs/            # OpenAPI specs (REST contracts) — Nguồn chuẩn API
│   ├── mvp-customer.openapi.yaml          # Public contract (Web/Mobile/BFF)
│   ├── admin-stable.openapi.yaml          # Admin read contract (đã cập nhật ApiResponse envelope)
│   ├── admin-draft-planned.openapi.yaml   # Admin write contract (draft, chưa freeze)
│   ├── catalog-checkout.openapi.yaml      # Internal HTTP contract (checkout/snapshot)
│   ├── openapi_d2c.yaml                   # D2C contract
│   ├── customer-stable.openapi.yaml
│   ├── customer-draft-planned.openapi.yaml
│   ├── identity-service.openapi.yaml
│   ├── order-service.openapi.yaml
│   ├── product-detail-mvp.openapi.yaml
│   ├── product-discovery-mvp.openapi.yaml
│   ├── shopping-cart-mvp.openapi.yaml
│   ├── order-management-mvp.openapi.yaml
│   ├── authentication-mvp.openapi.yaml
│   ├── payment-mvp.openapi.yaml
│   ├── checkout-mvp.openapi.yaml
│   ├── return-refund-mvp.openapi.yaml
│   ├── profile-service.openapi.yaml
│   ├── schema.graphql
│   ├── *.candidate.openapi.yaml           # Draft candidates
│   ├── *._REVIEW.md                       # Contract review docs
│   ├── CATALOG_CONTRACT_REMEDIATION_PLAN.md # Plan tái cấu trúc contract catalog
│   ├── AUTHENTICATION_MVP_CONTRACT_REVIEW.md
│   ├── CHECKOUT_MVP_CONTRACT_REVIEW.md
│   ├── CUSTOMER_PROFILE_MVP_CONTRACT_REVIEW.md
│   ├── ORDER_MANAGEMENT_MVP_CONTRACT_REVIEW.md
│   ├── PAYMENT_MVP_CONTRACT_REVIEW.md
│   ├── PRODUCT_DETAIL_MVP_CONTRACT_REVIEW.md
│   ├── PRODUCT_DISCOVERY_MVP_CONTRACT_REVIEW.md
│   ├── RETURN_REFUND_MVP_CONTRACT_REVIEW.md
│   ├── SHOPPING_CART_MVP_CONTRACT_REVIEW.md
│   ├── README.md
│   └── frontend-contracts/                # Frontend contracts tham khảo
│       ├── web-reference/
│       ├── customer-api-state-matrix.md
│       ├── customer-draft-planned.openapi.yaml
│       ├── customer-stable.openapi.yaml
│       ├── snapshot-hashes.json
│       ├── README.md
│       └── SOURCE.md
│
├── 04_testing/              # Kiểm thử
│   ├── mvp-contracts/       # Verification results (verification.json, order-design-verification.json)
│   ├── order-service/       # Order service test reports (26 files)
│   ├── profile-service/     # Profile service test reports (10 files)
│   ├── inventory-service/   # Inventory service test (1 file)
│   └── 01_tdd_profile_service.md
│
├── 05_deployment/           # Triển khai (chỉ có .gitkeep)
│
├── 06_operations/           # Vận hành
│   ├── mvp_delivery.md
│   ├── mvp_permissions.md
│   ├── order_service_runbook.md
│   └── profile_service_runbook.md
│
├── inventory-service/       # Tài liệu inventory service
│   ├── inventory_architecture_design.md
│   ├── inventory_database_design.md
│   ├── inventory_service_specs.md
│   └── inventory_unit_tests.md
│
├── plans/                   # Kế hoạch triển khai
│   └── active/
│       └── M6_IDENTITY_AUTH.md
│
└── ui_web/                  # Tài liệu giao diện web (UI/UX)
    ├── design/
    ├── MVP_PUBLIC_CONTRACT_MIGRATION_MATRIX.md
    ├── MVP_PUBLIC_FRONTEND_READINESS.md
    ├── MVP_PUBLIC_IDENTITY_AUTHORIZATION.md
    ├── MVP_PUBLIC_RELEASE_VERIFICATION.md
    ├── mvp-public-contract-baseline.json
    ├── ADMIN_*_DATA_API_MATRIX.md (nhiều file matrix API cho từng module admin)
    ├── CUSTOMER_*_DATA_API_MATRIX.md
    ├── ORDER_MANAGEMENT_DATA_API_MATRIX.md
    ├── PAYMENT_DATA_API_MATRIX.md
    ├── RETURN_REFUND_DATA_API_MATRIX.md
    ├── SHOPPING_CART_DATA_API_MATRIX.md
    ├── DATA_API_MATRIX.md
    ├── SCOPE_TRACEABILITY.md
    ├── ORDER_MANAGEMENT_SCOPE_TRACEABILITY.md
    ├── CHECKOUT_SCOPE_TRACEABILITY.md
    ├── PRODUCT_*_DATA_API_MATRIX.md
    ├── WEB_IDENTITY_GATEWAY_INTEGRATION.md
    ├── IDENTITY_OWNER_BFF_MVP.md
    ├── Mockoon_Guide_FE_BE_Design_System.md
    └── UI_SCREEN_SPECIFICATION_FOR_STITCH_AI.md
```

---

## 5. Thư mục `packages/` — Shared packages

```text
packages/
├── common/                  # Code chia sẻ chung (chỉ có src/)
│   └── src/
│
├── contracts/               # API contracts & registry
│   ├── mvp/
│   │   ├── build_public_contract.py   # Sinh public contract (idempotent)
│   │   ├── verify.py                  # Contract verification gate
│   │   ├── registry.yaml              # Registry: owner/consumer/capability/tasks
│   │   ├── sources.json               # Nguồn proto/baseline
│   │   ├── requirements.txt           # Python dependencies (grpc_tools, openapi_spec_validator, jsonschema)
│   │   ├── MIGRATION.md
│   │   ├── README.md
│   │   ├── TEAM-DECISIONS.md
│   │   ├── UI-HANDOFF.md
│   │   ├── baselines/                 # Wire baselines (catalog.json, dev.json, order.json, shipping.json)
│   │   │   ├── catalog.json
│   │   │   ├── dev.json
│   │   │   ├── order.json
│   │   │   └── shipping.json
│   │   └── ...
│   ├── identity-3.3.md
│   ├── order-3.2.md
│   └── verify_order_design.py
│
├── events/                  # Event schema & profiles (Kafka)
│   ├── schemas/             # JSON schemas cho events
│   │   ├── audit/
│   │   ├── care/
│   │   ├── catalog/         # Catalog events (v1/)
│   │   ├── channel/
│   │   ├── finance/
│   │   ├── fulfillment/
│   │   ├── inventory/
│   │   ├── order/
│   │   ├── procurement/
│   │   ├── promotion/
│   │   └── shipping/
│   ├── profiles/            # Event profiles (mvp/)
│   │   └── mvp/
│   │       └── order-source.event.json
│   ├── examples/            # Event examples
│   │   ├── fulfillment/
│   │   ├── order/
│   │   └── shipping/
│   ├── src/
│   └── README.md
│
└── proto/                   # Protobuf definitions (gRPC internal)
    ├── catalog/
    │   └── v1/
    │       └── catalog.proto          # omamx.catalog.v1.CatalogService
    ├── common/
    │   └── v1/
    │       └── types.proto            # PaginationRequest, PaginationResponse, Money, Weight, ...
    ├── fulfillment/
    ├── identity/
    ├── inventory/
    ├── order/
    ├── profile/
    ├── promotion/
    ├── shipping/
    ├── buf.yaml
    ├── buf.gen.yaml
    └── buf.gen.inventory.yaml
```

---

## 6. Thư mục `infra/` — Infrastructure

```text
infra/
├── docker/
│   └── docker-compose.infra.yml     # Docker compose cho infrastructure
├── k8s/                             # Kubernetes manifests (chỉ có .gitkeep)
├── kafka/                           # Kafka configuration
│   ├── docker-compose.yml
│   ├── docker-compose.dev.yml
│   ├── docker-compose.prod.yml
│   ├── .env / .env.example / .env.prod.example
│   └── README.md
├── monitoring/                      # Monitoring stack (Prometheus, Grafana, ...)
├── nginx/                           # Nginx configuration
└── scripts/                         # Infra scripts
```

---

## 7. Thư mục `mocks/` — Mock servers

```text
mocks/
└── mockoon/                         # Mockoon collections cho phát triển
    ├── admin_extensions.json        # Admin catalog mock (port 4030)
    ├── customer_extensions.json
    ├── mobile_pbl.json
    ├── oma-auth-mvp.json
    ├── oma-checkout-mvp.json
    ├── oma-customer-profile-mvp.json
    ├── oma-customer-service-mvp.json
    ├── oma-order-management-mvp.json
    ├── oma-payment-mvp.json
    ├── oma-product-detail-mvp.json
    ├── oma-product-discovery-mvp.json
    ├── oma-return-refund-mvp.json
    ├── oma-shopping-cart-mvp.json
    └── README.md
```

---

## 8. Thư mục `scripts/` — Scripts

```text
scripts/
├── mvp_customer_checks.py
├── mvp_native_checks.py
├── test-order-inventory.py
├── test-web-identity.py
└── validate_specs.py
```

---

## 9. Thư mục `.github/` — CI/CD

```text
.github/
├── workflows/
│   ├── ci.yml                 # CI chính: contract verification + Gradle test
│   ├── contracts.yml          # Contract gate
│   ├── mobile-mvp.yml         # Mobile MVP CI
│   ├── order-inventory.yml    # Order-Inventory integration CI
│   ├── order-service.yml      # Order service CI
│   └── web-identity.yml       # Web identity CI
└── modernize/
    └── java-upgrade/          # Hỗ trợ nâng cấp Java
```

---

## 10. File quan trọng ở root

| File | Mô tả |
|------|-------|
| `README.md` | Hướng dẫn tổng quan dự án |
| `project_bible.md` | Tham chiếu kỹ thuật chi tiết |
| `task.md` | Danh sách công việc hiện tại |
| `.gitattributes` | Cấu hình Git attributes |
| `.gitignore` | Git ignore patterns |

---

## 11. Kiến trúc ngôn ngữ lập trình

| Layer | Ngôn ngữ | Framework |
|-------|----------|-----------|
| Frontend Web | TypeScript | Next.js, React |
| Frontend Mobile | Dart | Flutter |
| Backend (catalog, identity, notification, analytics, care) | Java | Spring Boot 3.3.4, MyBatis, MapStruct, Flyway, Kafka, gRPC |
| Backend (order, inventory, profile, shipping) | Go | Go standard library, gRPC, Kafka |
| API Gateway | Kong | Kong configuration |
| Database | PostgreSQL | Flyway migrations |
| Event Bus | Kafka | Confluent/Apache Kafka |
| Cache | Redis | Spring Data Redis |
| Mock Server | Mockoon | Mockoon collections |

---

## 12. Các công việc đã hoàn thành (từ phiên làm việc)

1. **Contract**: Giữ `ApiResponse` envelope cho Catalog admin read; cập nhật `admin-stable.openapi.yaml` với `CatalogAdminProductListResponse` / `CatalogAdminProductDetailResponse`; thêm `AdminApiSuccessResponse`, `AdminApiErrorResponse`.
2. **Service**: `CatalogService.getAdminProducts()` trả `AdminProductListResponse` trong `data`; batch-load variants + giá; map status `ON_SALE/PAUSED/DISCONTINUED`; detail trả tất cả SKU.
3. **BFF**: `mapList`/`mapDetail` unwrap `payload.data`.
4. **Refactor**: Gộp `CatalogAdminController` từ `controller/` vào `api/`.
5. **P0**: Search server-side (ILIKE SQL), price filter/sort với `listing_status='ACTIVE'`, `effective_from/to`, đúng `channel`; categories phẳng; default sort `BEST_SELLING`; `thumbnail_url` placeholder.
6. **P0-5/P0-6**: Internal auth fail-closed; `validatePriceAndSku` bắt buộc `X-Internal-Token`; gRPC `product_id` ID thật; `complete()` map `CatalogException` theo `ErrorCode`.
7. **P1-6**: Admin write contract-first với `/skus` (thay `/variants`); 5 endpoint bọc `ApiResponse`; thêm transition + price update.
8. **P1-7**: Migration V11 exclusion constraint cho giá tương lai (`tstzrange` + `&&` overlaps).
9. **P2-C-18**: `ListProductVariants` gRPC với phân trang, lọc `category_id`/`ocop_rating`/`active_only`.
10. **P2-C-19**: Kafka retry 3 lần + `DeadLetterPublishingRecoverer` (DLT).
11. **CI**: Tạo `.github/workflows/ci.yml` (contract + test jobs).
12. **DTO Refactor**: Tách record checkout snapshot thành `CheckoutSnapshotRequest` / `CheckoutSnapshotResponse` DTO riêng.
13. **Tests**: Thêm 5 unit test cho search + admin filter (38 tests total pass).
14. **Verification**: `verify.py` PASS (40 OpenAPI specs, 4 protobuf baselines, 13 registry contracts).

---

## 13. Mismatch matrix (tất cả đã xử lý)

Tất cả C-01 đến C-20 trong `CATALOG_CONTRACT_REMEDIATION_PLAN.md` đã được đánh dấu **Done**. Không còn vấn đề mở trong plan.

---
*Tài liệu này được tạo tự động từ cấu trúc thư mục thực tế của dự án DUT.PBL6.*