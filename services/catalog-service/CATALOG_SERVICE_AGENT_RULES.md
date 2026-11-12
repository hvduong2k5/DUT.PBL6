# Catalog Service — Agent Rules & Structure Guide

> File này được viết để agent (AI coding assistant) hiểu nhanh cấu trúc, quy tắc và convention khi làm việc với `services/catalog-service`.
> Mọi thay đổi lớn trong catalog service phải tuân thủ các quy tắc dưới đây.

---

## 1. Mục đích của Catalog Service

Catalog Service là microservice quản lý danh mục sản phẩm (products, variants/SKU, categories, channel prices) cho hệ thống Oma.
- **Public REST** (`/api/v1/*`): Dành cho Web/Mobile/BFF — chỉ trả sản phẩm `APPROVED` + `ACTIVE`.
- **Admin REST** (`/api/v1/admin/catalog/*`): Dành cho nhân viên quản trị — CRUD sản phẩm, SKU, giá, transition trạng thái.
- **Internal gRPC** (`omamx.catalog.v1.CatalogService`): Nội bộ microservice-to-microservice (order-service, inventory-service...).
- **Internal HTTP** (`/api/v1/internal/*`): Checkout snapshot, revalidate, price validation.

---

## 2. Cấu trúc thư mục chi tiết

```
services/catalog-service/
├── build.gradle                 # Spring Boot 3.3.4, MyBatis, MapStruct, Flyway, Kafka, gRPC, Spotless
├── gradlew / gradlew.bat
├── settings.gradle
├── .env / .env.example
├── phan-tich.md
│
├── src/main/java/com/hvduong/catalog/
│   ├── api/
│   │   ├── CatalogAdminController.java    # Admin REST endpoints
│   │   ├── CatalogHttpController.java     # Public + Internal REST endpoints
│   │   └── CatalogHttpErrors.java         # Global exception handler (HTTP)
│   │
│   ├── application/
│   │   ├── dto/
│   │   │   ├── request/
│   │   │   │   ├── CheckoutSnapshotRequest.java
│   │   │   │   ├── CreateProductRequest.java
│   │   │   │   ├── CreateVariantRequest.java
│   │   │   │   ├── ProductFilterRequest.java
│   │   │   │   ├── UpdatePriceRequest.java
│   │   │   │   └── UpdateProductRequest.java
│   │   │   └── response/
│   │   │       ├── AdminProductDetailResponse.java
│   │   │       ├── AdminProductListResponse.java
│   │   │       ├── CheckoutSnapshotResponse.java
│   │   │       ├── ProductDetailResponse.java
│   │   │       └── ProductVariantResponse.java
│   │   │
│   │   ├── mapper/
│   │   │   ├── AdminCatalogMapper.java    # Centralized admin DTO mapping
│   │   │   └── CatalogDtoMapper.java      # MapStruct: Entity ↔ DTO
│   │   │
│   │   ├── service/
│   │   │   ├── CatalogCacheEvictor.java   # @Component, xóa cache category khi admin mutate
│   │   │   ├── CatalogService.java        # Interface: PUBLIC catalog only
│   │   │   ├── CatalogAdminService.java   # Interface: ADMIN read + write
│   │   │   ├── impl/
│   │   │   │   ├── CatalogServiceImpl.java   # Public impl
│   │   │   │   ├── CatalogAdminServiceImpl.java # Admin impl
│   │   │   │   └── CheckoutSnapshotService.java
│   │   │
│   │   ├── common/
│   │   │   ├── constant/
│   │   │   │   ├── CatalogConstants.java    # Magic strings: channels, contract version
│   │   │   │   └── SaleStatus.java          # ListingStatus <-> ON_SALE/PAUSED/DISCONTINUED
│   │   │   ├── response/
│   │   │   │   ├── ApiResponse.java         # Envelope: success/code/message/data/errors/requestId/timestamp
│   │   │   │   └── PageResponse.java
│   │   │   ├── enums/
│   │   │   │   ├── ListingStatus.java       # ACTIVE/SUSPENDED/ARCHIVED (DB)
│   │   │   │   ├── SalesChannel.java        # D2C_WEB/MOBILE_APP/...
│   │   │   │   └── SortBy.java              # PRICE_ASC/PRICE_DESC/BEST_SELLING/NEWEST
│   │   │   ├── exception/
│   │   │   │   ├── CatalogException.java
│   │   │   │   └── ErrorCode.java
│   │   │   └── handler/
│   │   │       └── GlobalRestExceptionHandler.java
│   │   │
│   │   ├── grpc/
│   │   │   ├── CatalogGrpcService.java      # gRPC internal service
│   │   │   ├── CatalogGrpcMapper.java       # Proto ↔ DTO mapper
│   │   │   └── CatalogAccess.java           # gRPC interceptor: internal token validation
│   │   │
│   │   ├── config/
│   │   │   ├── SecurityConfig.java
│   │   │   ├── KafkaConsumerConfig.java     # Retry 3 lần + DLT
│   │   │   ├── KafkaProducerConfig.java
│   │   │   ├── MyBatisConfig.java
│   │   │   ├── OpenApiConfig.java
│   │   │   └── UuidTypeHandler.java
│   │   │
│   │   ├── domain/
│   │   │   ├── entity/
│   │   │   │   ├── Product.java
│   │   │   │   ├── ProductVariant.java
│   │   │   │   ├── ProductImage.java
│   │   │   │   ├── Category.java
│   │   │   │   └── ChannelPrice.java
│   │   │   └── ...
│   │   │
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
│   │
│   └── src/main/resources/
│       ├── application.yml
│       ├── application-local.yml
│       ├── application-test.yml
│       ├── db/migration/
│       │   ├── V1__create_categories.sql
│       │   ├── V2__create_products.sql
│       │   ├── V3__create_product_variants.sql
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
│   │
│   └── src/test/java/com/hvduong/catalog/
│       └── ... (38 unit tests)
```

---

## 3. Quy tắc chung (General Rules)

### 3.1. Package convention
- Tất cả class nằm trong package `com.hvduong.catalog.*`.
- Không được tạo package mới ngoài `com.hvduong.catalog` mà không có sự đồng ý.

### 3.2. Java version & format
- Java 21 (toolchain).
- Sử dụng **Spotless** với `googleJavaFormat('1.17.0')`.
- Import order: `com.hvduong`, `java`, `javax`, `org`, `com`, `''`.
- Chạy `./gradlew spotlessApply` trước khi commit.

### 3.3. Lombok & MapStruct
- Sử dụng Lombok (`@Data`, `@Builder`, `@NoArgsConstructor`, `@AllArgsConstructor`, `@RequiredArgsConstructor`) cho DTO/Entity.
- Sử dụng MapStruct cho mapping Entity ↔ DTO (`CatalogDtoMapper`).
- Không viết getter/setter thủ công cho DTO.

### 3.4. Naming convention
- Class: `PascalCase` (ví dụ: `CatalogServiceImpl`, `AdminCatalogMapper`).
- Method/variable: `camelCase` (ví dụ: `getAdminProducts`, `currentPriceByVariant`).
- Constant: `UPPER_SNAKE_CASE` (ví dụ: `CHANNEL_D2C_WEB`).
- Enum: `PascalCase` với giá trị `UPPER_SNAKE_CASE` (ví dụ: `ListingStatus.ACTIVE`).
- File migration: `V{number}__{description}.sql` (ví dụ: `V11__channel_price_exclusion_constraint.sql`).

---

## 4. Quy tắc Contract (API Contract)

### 4.1. Public Contract (`mvp-customer.openapi.yaml`)
- Nguồn chuẩn: `docs/03_api_specs/mvp-customer.openapi.yaml`.
- Sinh từ `packages/contracts/mvp/build_public_contract.py`.
- Phải chạy `./gradlew spotlessApply` + kiểm tra `generated == tracked` trước khi commit.
- Public response **KHÔNG** dùng `ApiResponse` envelope; trả DTO trực tiếp + `PublicError`.

### 4.2. Admin Contract (`admin-stable.openapi.yaml`)
- Admin read (2 GET): Bọc trong `ApiResponse` envelope.
  - Schema 200 → `CatalogAdminProductListResponse` / `CatalogAdminProductDetailResponse`.
  - Example: `success: true`, `code: SUCCESS`, `message: ...`, `timestamp: ...`, `data: {...}`.
- Error 400/403/404/500 của 2 GET admin: Dùng `AdminApiErrorResponse` (`success: false`).
- 401: Giữ response chung của Spring Security.
- Admin write (POST/PUT): Bọc trong `ApiResponse` envelope.

### 4.3. Internal Contract (`catalog-checkout.openapi.yaml`)
- Port 8005 (không phải 8006).
- Snapshot chỉ nhận `D2C_WEB` / `MOBILE_APP`.
- Endpoint `/api/v1/internal/prices/validate` deprecated + bắt buộc `X-Internal-Token`.

### 4.4. gRPC Contract (`packages/proto/catalog/v1/catalog.proto`)
- Chỉ dùng nội bộ. Không khôi phục `catalog_public.proto` / `catalog_admin.proto`.
- `product_id` phải là product ID thật (không phải variantId).
- Lỗi nghiệp vụ phải map đúng gRPC status (không phải tất cả thành `INTERNAL`).

---

## 5. Quy tắc Service Layer

### 5.1. Phân tách Public / Admin
- `CatalogService` interface: Chỉ chứa public methods (`getPublicProducts`, `searchProducts`, `getAllCategories`, `getVariantBySkuCode`, `validatePrices`).
- `CatalogAdminService` interface: Chứa admin read + write (`getAdminProducts`, `getAdminProductDetail`, `createProduct`, `updateProduct`, `transition`, `addVariant`, `updateVariantPrices`).
- `CatalogServiceImpl`: Chỉ implement `CatalogService`.
- `CatalogAdminServiceImpl`: Chỉ implement `CatalogAdminService`.
- **Không được** để một class implement cả hai interface.

### 5.2. Transaction
- `@Transactional(readOnly = true)` trên class level cho service chỉ đọc.
- `@Transactional` (không readOnly) cho method ghiDB (create/update/delete).
- `@CacheEvict(value = "catalog:categories", allEntries = true)` khi mutation thay đổi danh mục.

### 5.3. DTO Mapping
- Sử dụng `AdminCatalogMapper` cho toàn bộ mapping admin (list item, SKU, status counts, categories).
- Không viết mapping thủ công trong service nếu có thể dùng MapStruct/ AdminCatalogMapper.
- Không để magic strings trong service (dùng `CatalogConstants`, `SaleStatus`).

### 5.4. Cache
- `getAllCategories()` có `@Cacheable(value = "catalog:categories", key = "'all'")`.
- Khi mutation admin (create/update/approve/suspend/archive/addVariant/updateVariantPrices), gọi `cacheEvictor.evictCategoriesCache()`.

### 5.5. Error Handling
- Ném `CatalogException(ErrorCode.XXX, message)` cho lỗi nghiệp vụ.
- `GlobalRestExceptionHandler` chuyển `CatalogException` thành `ApiResponse.error(...)` với `requestId` (UUID).
- Không swallow exception không lý do.

---

## 6. Quy tắc Database (MyBatis + Flyway)

### 6.1. Migration
- Mỗi migration là file SQL mới: `V{so}__ten_mo_ta.sql`.
- **KHÔNG** sửa migration cũ đã chạy (tránh vỡ DB production).
- Dùng Flyway `validate-on-migrate: true`.
- Chỉ thêm index/constraint mới, không đổi cấu trúc cột cũ.

### 6.2. MyBatis Mapper XML
- Sử dụng `<sql>` fragment cho điều kiện lặp lại (ví dụ: `publicSearchCondition`, `publicPriceCondition`).
- Sử dụng `<include refid="..." />` để tái sử dụng fragment.
- Parameter bằng `#{paramName}` (không dùng string concatenation).
- Enum dùng `::enum_type` (ví dụ: `channel = #{channel}::sales_channel`).

### 6.3. MyBatis Mapper Interface
- `@Param("ten")` cho mọi parameter.
- Trả về `List<T>`, `Optional<T>`, `long`, `boolean`.
- Không viết SQL inline trong Java.

---

## 7. Quy tắc Test

### 7.1. Test Framework
- JUnit 5 + Mockito + MockitoExtension.
- Chạy `./gradlew test --offline`.
- Mỗi test class đặt tên `*Test.java`.

### 7.2. Test Structure
- Given / When / Then.
- Sử dụng `lenient()` cho stubbing không chắc chắn sẽ được gọi.
- Tránh `UnnecessaryStubbingException` — chỉ stub method thực sự được gọi.
- Test cả success case và error case.

### 7.3. Coverage Mínimum
- Tất cả public method trong service phải có ít nhất 1 test.
- Error paths (not found, invalid input, conflict) phải có test.

---

## 8. Quy tắc Commit

### 8.1. Commit Message Convention
**All commit messages MUST be written in 100% English.** No Vietnamese text in commit subject or body.
```
<type>(<scope>): <short description in English>

- Detail of change 1
- Detail of change 2
- ./gradlew test --offline: PASS
```

### 8.2. Các loại commit
| Loại | Ý nghĩa |
|------|---------|
| `feat` | Tính năng mới |
| `fix` | Sửa lỗi |
| `refactor` | Refactor không đổi hành vi |
| `docs` | Chỉ thay đổi tài liệu |
| `chore` | Dự phòng, build, config |

### 8.3. Chia commit nhỏ
- Mỗi commit chỉ làm **một việc** (ví dụ: chỉ refactor, chỉ thêm test, chỉ sửa contract).
- Không gộp nhiều việc khác nhau vào 1 commit.
- Commit message phải rõ ràng, không dùng "fix", "update", "done".

---

## 9. Lệnh Thường Dùng

```bash
# Biên dịch
./gradlew compileJava --offline

# Chạy test
./gradlew test --offline

# Format code (Spotless)
./gradlew spotlessApply --offline

# Chạy verify contract (cần venv Python)
python -m venv .system_generated/contract-venv
.source_generated/contract-venv/Scripts/activate
python -m pip install -r packages/contracts/mvp/requirements.txt
python packages/contracts/mvp/verify.py

# Xem trạng thái git
git status --short
git log --oneline -10
```

---

## 10. Các Lỗi Thường Gặp & Cách Sửa

| Lỗi | Nguyên nhân | Cách sửa |
|-----|-------------|----------|
| `UnnecessaryStubbingException` | Stub method không được gọi | Xóa stub hoặc dùng `lenient()` |
| `cannot find symbol` | Thiếu import | Thêm import class bị thiếu |
| `method does not override` | `@Override` trên method không ở interface | Xóa `@Override` |
| `incompatible types: long cannot be converted to Integer` | `int` field nhận `long` | Dùng `.intValue()` hoặc đổi kiểu field |
| `verify.py` lỗi import | Thiếu dependency Python | Cài `requirements.txt` trong venv |

---

## 11. Các File Không Được Sửa Tuyệt Đối (trừ khi có lý do đặc biệt)

- `V1__` đến `V10__` migrations (đã chạy trên production).
- `packages/proto/catalog/v1/catalog.proto` (gRPC contract đã freeze).
- `docs/03_api_specs/mvp-customer.openapi.yaml` (phải sinh từ `build_public_contract.py`, không sửa tay).

---

## 12. Checklist Trước Khi Commit

- [ ] `./gradlew spotlessApply --offline` đã chạy.
- [ ] `./gradlew compileJava --offline` pass.
- [ ] `./gradlew test --offline` pass (38 tests).
- [ ] `python packages/contracts/mvp/verify.py` pass (nếu thay đổi contract).
- [ ] Git status sạch (không có file lẻ).
- [ ] Commit message tuân thủ convention.

---

*Tài liệu này được tạo để agent hiểu nhanh cấu trúc và quy tắc của Catalog Service. Nếu có thay đổi cấu trúc lớn, cần cập nhật lại file này.*
