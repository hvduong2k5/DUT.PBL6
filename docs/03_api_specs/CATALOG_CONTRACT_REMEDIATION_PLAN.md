# Catalog Service — Contract Remediation Plan

> Phạm vi: `services/catalog-service`, contract public/admin/internal, BFF `apps/web-user`, registry/baseline và tài liệu vận hành liên quan.
>
> Trạng thái: contract layer đã được chỉnh trong working tree; code service **chưa** chỉnh. Plan dưới đây là thứ tự triển khai đề xuất.

## 1. Nguồn chuẩn

| Lớp | Artifact | Vai trò |
| --- | --- | --- |
| Public | `docs/03_api_specs/mvp-customer.openapi.yaml` | Hợp đồng Web/Mobile/BFF; sinh từ `build_public_contract.py` |
| Admin read | `docs/03_api_specs/admin-stable.openapi.yaml` | Hợp đồng employee web đang dùng |
| Admin write | `docs/03_api_specs/admin-draft-planned.openapi.yaml` | Đề xuất, chưa freeze |
| Internal HTTP | `docs/03_api_specs/catalog-checkout.openapi.yaml` | Snapshot/revalidate/legacy price check |
| Internal gRPC | `packages/proto/catalog/v1/catalog.proto` | `omamx.catalog.v1.CatalogService` |
| Registry | `packages/contracts/mvp/registry.yaml` | Owner/consumer/capability/tasks |

Quy tắc: public/admin dùng REST; gRPC chỉ dùng nội bộ. Không khôi phục `catalog_public.proto` / `catalog_admin.proto`.

## 2. Contract đã sửa trong working tree

| File | Thay đổi | Lý do |
| --- | --- | --- |
| `packages/contracts/mvp/build_public_contract.py` | `x-provider-contract` của 4 operation Catalog → `catalog-service`; chuẩn hóa `category_id`/`parent_id` thành UUID; sửa mô tả search; bỏ tham chiếu `CatalogPublicService` | Public là REST, không còn proto public |
| `docs/03_api_specs/openapi_d2c.yaml` | `category_id` query + schema có `format: uuid`; mô tả rõ slug không phải alias; `parent_id` UUID | Service bind UUID; example cũ `CAT-ME-XUNG-GION` không dùng được |
| `docs/03_api_specs/mvp-customer.openapi.yaml` | Regenerate từ generator; example/schema category UUID; search summary đúng MVP | Giữ `public == build()` |
| `docs/03_api_specs/admin-stable.openapi.yaml` | 2 GET admin catalog bọc `ApiResponse`: schema 200 → `CatalogAdminProductListResponse`/`CatalogAdminProductDetailResponse`; example có `success/code/message/timestamp/data`; error 400/403/404/500 dùng `AdminApiErrorResponse` | Giữ envelope thực tế của service, BFF chỉ cần unwrap `data` |
| `docs/03_api_specs/catalog-checkout.openapi.yaml` | Port `8006` → `8005`; ghi rõ snapshot chỉ nhận `D2C_WEB`/`MOBILE_APP`; thêm endpoint deprecated `/api/v1/internal/prices/validate` kèm `X-Internal-Token` và schema camelCase | Khớp `application.yml`; không bỏ sót endpoint nội bộ đang tồn tại |
| `packages/contracts/mvp/registry.yaml` | `catalog-public` trỏ `mvp-customer.openapi.yaml`, bỏ `rpc`; task checkout bỏ `CatalogPublicService` | Registry không còn trỏ file proto đã xóa |
| `packages/contracts/mvp/verify.py` | Loại `hvduong.catalog.v1` khỏi compatibility check của baseline Catalog, ghi exception rõ ràng | Baseline Catalog là lịch sử; public/admin gRPC đã bị xóa có chủ đích |
| `packages/contracts/mvp/README.md` | Bảng Catalog và quyết định Catalog dùng REST public/admin | Bỏ mô tả public/admin gRPC |
| `packages/contracts/mvp/MIGRATION.md` | Bỏ mô tả transcoder proto JSON; nêu flat categories, search/pagination server-side | Khớp contract REST |
| `docs/06_operations/mvp_permissions.md` | Catalog admin → REST `/api/v1/admin/catalog/**`; public bỏ "public gRPC đọc" | Khớp ma trận quyền thực tế |

### Đã kiểm

- `python packages/contracts/mvp/build_public_contract.py` chạy được và idempotent.
- `generated == tracked: True`.
- `git diff --check` sạch; `py_compile` cho hai script Python pass.

### Chưa kiểm được

`python packages/contracts/mvp/verify.py` dừng ở `import grpc_tools`; môi trường hiện thiếu `grpc_tools`, `openapi_spec_validator`, `jsonschema`. Cần:

```sh
python -m venv .system_generated/contract-venv
# activate venv
python -m pip install -r packages/contracts/mvp/requirements.txt
python packages/contracts/mvp/verify.py
```

Ghi chú:

- `packages/contracts/mvp/baselines/catalog.json` vẫn là baseline lịch sử (có hash `catalog_public.proto`/`catalog_admin.proto`). Không sửa tay; chỉ regenerate bằng `verify.py --record-baselines` khi có môi trường chuẩn.
- `docs/ui_web/mvp-public-contract-baseline.json` đang pin blob cũ của public contract. Khi commit thay đổi public contract, phải cập nhật đồng thời `publicContractBlob` và `sourceCommit` theo quy định trong `MVP_PUBLIC_CONTRACT_MIGRATION_MATRIX.md`.

## 3. Ma trận mismatch còn lại

| ID | Khu vực | Hiện trạng | Contract / kỳ vọng | Hành động | Phase |
| --- | --- | --- | --- | --- | --- |
| C-01 | Public search | Lấy 1 trang sản phẩm rồi lọc keyword trong bộ nhớ; `total` chỉ đếm trong trang | `q`, `page`, `page_size`; `total` là tổng kết quả | Đẩy filter/pagination xuống SQL | P0 |
| C-02 | Public price filter/sort | Thiếu `pv.listing_status`, `effective_from/to`; sort thiếu `channel` | Chỉ dùng giá ACTIVE, còn hiệu lực, đúng kênh | Sửa SQL filter/sort/count | P0 |
| C-03 | Public default sort | Service mặc định `NEWEST` | Contract mặc định `BEST_SELLING` | Đổi default; chốt semantics | P0 |
| C-04 | Public categories | Service trả cây root + children; controller chỉ trả root, bỏ children | Danh sách phẳng, có `parent_id` | Flatten ở adapter/service | P0 |
| C-05 | Public item price | Lấy giá variant đầu tiên; thiếu giá → 503 cả trang | `base_price` bắt buộc, đúng kênh | Chọn giá thấp nhất còn hiệu lực; loại item không có giá khỏi list | P0 |
| C-06 | Public ảnh | `thumbnail_url` có thể là chuỗi rỗng | `format: uri`, required | Fallback ảnh hợp lệ hoặc chốt nullable | P0 |
| C-07 | Public stock | Có nhánh biến null thành 0 | Null = chưa đồng bộ; `in_stock = stock > 0` | Giữ null xuyên suốt | P0 |
| C-08 | Internal auth | `CatalogAccess` fail-open khi token rỗng; `/internal/prices/validate` không kiểm token | Token ≥ 32 ký tự, fail closed | Sửa interceptor + controller | P0 |
| C-09 | gRPC mapping | `product_id` đang set bằng `variantId` | Phải là product ID thật | Thêm `productId` vào DTO/mapper | P0 |
| C-10 | gRPC error | `complete()` biến mọi lỗi thành `INTERNAL` | Map `CatalogException`/`ErrorCode` đúng status | Sửa interceptor/service | P0 |
| C-11 | Admin response | Bọc `ApiResponse`; BFF đọc trực tiếp `payload.items` | Contract mô tả `ApiResponse<data>`; BFF đọc `payload.data.items` | Giữ envelope, sửa contract + BFF unwrap `data` | P1 |
| C-12 | Admin status | Trả `ACTIVE`/`SUSPENDED`/`ARCHIVED` | `ON_SALE`/`PAUSED`/`DISCONTINUED` | Map ở adapter | P1 |
| C-13 | Admin derived fields | `saleStatusLabel`, `skuCount`, `onSaleSkuCount`, `basePriceFromVnd`, `foodInformationComplete`, `updatedAt`, `categoryName`, ảnh bìa... bị null | Required trong `admin-stable` | Tính/join đầy đủ | P1 |
| C-14 | Admin detail | Chỉ trả SKU ACTIVE; nhiều field mapper ignore | Trả cả PAUSED/DISCONTINUED kèm giá/trạng thái/revision | Sửa query + mapper | P1 |
| C-15 | Admin list filter | Controller/BFF không truyền `q/status/page/pageSize` | Contract hỗ trợ | Thêm filter + BFF passthrough | P1 |
| C-16 | Admin write | Contract draft dùng `/skus`, service dùng `/variants`; response khác | Chưa freeze | Đã chốt contract-first `/skus`; response bọc `ApiResponse` | P1 — Done |
| C-17 | Price future | Unique index `(variant_id, channel) WHERE status='ACTIVE'` chặn giá tương lai | Cần nhiều giá theo thời gian | Exclusion constraint hoặc versioning | P1 |
| C-18 | gRPC list | `ListProductVariants` trả response rỗng | Có phân trang/filter | Implement | P2 |
| C-19 | Kafka stock | Retry vô hạn, không DLT | Poison message không chặn partition | Phân loại lỗi + DLT | P2 |
| C-20 | Public search mô tả | Contract cũ ghi ElasticSearch | MVP dùng PostgreSQL ILIKE | Đã sửa mô tả | Done |

## 4. Phase P0 — Public correctness + internal security

Mục tiêu: public API đúng contract và không rò/khóa sai dữ liệu; internal calls fail closed.

### P0-1. Search server-side

- Thêm `query` vào `ProductMapper.findPublicProducts` và `countPublicProducts`.
- SQL dùng `ILIKE` trên `name`/`description`, giữ `APPROVED` + `ACTIVE`, áp dụng channel mặc định `D2C_WEB`.
- `CatalogServiceImpl.searchProducts` bỏ `stream().filter(...)`; lấy `total` từ count; trả đúng trang.
- `CatalogHttpController.search` giữ `q` bắt buộc, `page_size` mặc định 10.

File: `repository/mybatis/ProductMapper.java`, `resources/mapper/sql/ProductMapper.xml`, `application/service/impl/CatalogServiceImpl.java`, `api/CatalogHttpController.java`.

Test:

- Seed 250 sản phẩm, keyword khớp ở trang 2/3; `page=2&page_size=10` trả đúng 10 item và `total` không đổi.
- Keyword không khớp → `items=[]`, `total=0`.
- Sản phẩm `SUSPENDED`/`PENDING` không xuất hiện.

### P0-2. Price filter/sort đúng hiệu lực

- Trong `findPublicProducts`, `countPublicProducts` và các subquery sort:
  - join `product_variants pv` với `pv.listing_status = 'ACTIVE'`;
  - `cp.status = 'ACTIVE'`;
  - `cp.effective_from <= NOW()`;
  - `(cp.effective_to IS NULL OR cp.effective_to > NOW())`;
  - `cp.channel = #{channel}` ở mọi subquery.
- `enrichListItem` chọn giá thấp nhất trong các variant ACTIVE của đúng channel.
- Sản phẩm không có giá hiệu lực không được trả trong list (tránh 503 toàn trang).

File: `ProductMapper.xml`, `CatalogServiceImpl.java`.

Test:

- Variant SUSPENDED có giá rẻ hơn → không được chọn.
- Giá hết hạn/giá tương lai không tham gia filter/sort.
- `channel=SHOPEE` không lấy giá D2C.
- Product không có giá → không có trong list; detail trả lỗi contract (404/503 theo chốt).

### P0-3. Categories phẳng

- Thêm `getAllCategoriesFlat()` (hoặc flatten trong `CatalogServiceImpl.getAllCategories`) trả toàn bộ category active, giữ `parent_id`.
- Controller map trực tiếp, không trả `children`.
- Giữ cache `catalog:categories`.

File: `CatalogServiceImpl.java`, `api/CatalogHttpController.java`, `application/dto/response/CategoryResponse.java` (chỉ dùng cho nội bộ).

Test: fixture root + 2 con → response có 3 phần tử; con có `parent_id`; không có field `children`.

### P0-4. Public item integrity

- `thumbnail_url`: lấy ảnh approved đầu tiên hoặc placeholder cấu hình; không trả chuỗi rỗng nếu contract yêu cầu URI.
- `summary`, `category_name`: fallback hợp lý, không null.
- `in_stock`/`stock_available`: chỉ null khi projection chưa có; không ép null → 0.
- `base_price`: Money VND, `units` integer, `nanos=0`; không mất precision.

File: `CatalogServiceImpl.java`, `CatalogDtoMapper.java`, `CatalogHttpController.java`.

### P0-5. Internal auth fail-closed

- `CatalogAccess`: nếu `catalog.internal-token` rỗng/ngắn hơn 32 → từ chối mọi call nội bộ (`UNAUTHENTICATED`), log error. Không fail-open.
- `CatalogHttpController.validatePriceAndSku`: thêm `X-Internal-Token` và gọi `authorize(token)`.
- Cân nhắc fail-fast lúc startup ở profile `prod` nếu token rỗng.

File: `grpc/CatalogAccess.java`, `api/CatalogHttpController.java`, `config/SecurityConfig.java` (nếu cần).

Test:

- Token rỗng/sai/ngắn → 401 HTTP, `UNAUTHENTICATED` gRPC.
- Token đúng → 200/OK.
- Cấu hình token rỗng → snapshot/revalidate/validate đều bị chặn.

### P0-6. gRPC mapping + error

- Thêm `productId` vào `ProductVariantResponse`; `CatalogGrpcMapper.toVariantDetailProto` set đúng product ID.
- `complete()` map `CatalogException` theo `ErrorCode`:
  - `*_NOT_FOUND` → `NOT_FOUND`
  - `*_INVALID_STATE`, `PRICE_TAMPERED`, `OUT_OF_SEASON` → `FAILED_PRECONDITION`
  - `INVALID_ARGUMENT` → `INVALID_ARGUMENT`
  - `PERMISSION_DENIED` → `PERMISSION_DENIED`
  - còn lại → `INTERNAL`/`UNAVAILABLE` theo loại.
- `getCheckoutSnapshot` giữ mapping riêng, không nuốt lỗi thành `UNAVAILABLE` nếu là lỗi selection.

File: `application/dto/response/ProductVariantResponse.java`, `application/service/impl/CatalogServiceImpl.java`, `grpc/CatalogGrpcMapper.java`, `grpc/CatalogGrpcService.java`.

### P0-7. Default sort

- `CatalogHttpController.products`: `defaultValue="BEST_SELLING"`.
- `ProductFilterRequest.sortBy` mặc định `BEST_SELLING`.
- Chốt semantics `BEST_SELLING` (xem mục 8).

### P0-8. Tests + gate

- Thêm unit test cho search SQL params, price effective window, flat categories, internal auth, gRPC error mapping.
- Chạy `gradlew test --offline`.
- Sau khi cài đủ dependency: `python packages/contracts/mvp/verify.py`.

## 5. Phase P1 — Admin read model + pricing lifecycle

Mục tiêu: admin BFF đọc được đúng contract, không còn màn hình rỗng/sai trạng thái.

### P1-1. Giữ `ApiResponse` cho GET admin, unwrap ở BFF

Quyết định: **không bỏ** envelope. `admin-stable.openapi.yaml` đã được cập nhật để mô tả đúng `ApiResponse` thay vì object trần.

- `GET /api/v1/admin/catalog/products` trả `ApiResponse<CatalogAdminProductListData>`:
  `success`, `code`, `message`, `timestamp`, `data{items, statusCounts, categories, calculatedAt}`.
- `GET /api/v1/admin/catalog/products/{productId}` trả `ApiResponse<CatalogAdminProductDetailData>`.
- `data` vẫn phải đúng required fields và `additionalProperties: false` của contract; service phải bổ sung `statusCounts`, `categories`, `calculatedAt` (hiện chỉ trả `PageResponse`).
- Error 400/403/404/500 của 2 endpoint dùng `AdminApiErrorResponse` (`success:false`, `code`, `message`, `timestamp`, optional `errors`). 401 vẫn là response chung của Spring Security.
- BFF `apps/web-user/src/features/employee/catalog/server/handler.ts`: đọc `payload.data.items`, `payload.data.statusCounts`, `payload.data.categories`, `payload.data.calculatedAt`; detail đọc `payload.data`.

File: `docs/03_api_specs/admin-stable.openapi.yaml`, `controller/CatalogAdminController.java` (giữ `ApiResponse`), `application/dto/response/AdminProductListResponse.java` (payload trong `data`), `apps/web-user/.../handler.ts`.

### P1-2. Map trạng thái

- `ACTIVE → ON_SALE`, `SUSPENDED → PAUSED`, `ARCHIVED → DISCONTINUED`.
- Label tiếng Việt tương ứng: Đang bán / Tạm ngừng bán / Ngừng kinh doanh.
- Áp dụng cho product và SKU.

File: `application/mapper/CatalogDtoMapper.java` hoặc adapter riêng.

### P1-3. Derived fields

- `skuCount`, `onSaleSkuCount`: đếm theo toàn bộ variant (không chỉ ACTIVE).
- `basePriceFromVnd`: giá thấp nhất còn hiệu lực của channel admin (mặc định D2C_WEB).
- `foodInformationComplete`: đủ 5 trường food information và không rỗng.
- `updatedAt`: lấy `products.updated_at`.
- `categoryName`: join category.
- `coverImageUrl`/`coverImageAlt`: ảnh approved đầu tiên; alt fallback tên sản phẩm.
- Tránh N+1: batch query variants/prices cho cả trang.

File: `CatalogServiceImpl.java`, `ProductMapper.xml`/`ProductVariantMapper.xml`, `CatalogDtoMapper.java`.

### P1-4. Detail trả đủ SKU

- Dùng query lấy tất cả variant (ACTIVE + SUSPENDED + ARCHIVED), kèm giá hiện hành.
- Map `basePriceVnd`, `currency`, `saleStatus`, `saleStatusLabel`, `updatedAt`, `revision`.

File: `ProductVariantMapper.java/.xml`, `CatalogServiceImpl.java`, `CatalogDtoMapper.java`.

### P1-5. Filter + BFF passthrough

- Controller nhận `q`, `status`, `page`, `pageSize`.
- `status` lọc theo vocabulary `ON_SALE/PAUSED/DISCONTINUED`.
- `statusCounts` tính trên tập đã lọc bởi `q` (không tính theo `status` đang chọn) và `categories` trả các category có sản phẩm.
- BFF `apps/web-user/src/features/employee/catalog/server/handler.ts` chuyển `request.nextUrl.searchParams` xuống upstream.

File: `CatalogAdminController.java`, `CatalogServiceImpl.java`, `ProductMapper.xml`, `apps/web-user/.../handler.ts`.

### P1-6. Chốt admin write contract (ĐÃ LÀM)

Quyết định: **contract-first với `/skus`** (theo D-03).

- `POST /api/v1/admin/catalog/products/{productId}/skus` (đã đổi từ `/variants` trong `CatalogAdminController`).
- `PUT /api/v1/admin/catalog/products/{productId}/skus/{skuId}/prices` (đã thêm vào draft contract).
- `POST /api/v1/admin/catalog/products/{productId}/transition` (đã thêm vào draft contract).
- Tất cả response write bọc trong `ApiResponse` envelope (`success/code/message/data/requestId/timestamp`).
- `admin-draft-planned.openapi.yaml`: 5 endpoint catalog write đã được cập nhật envelope và chốt response.

File: `controller/CatalogAdminController.java` (đã đổi `/variants` → `/skus`), `docs/03_api_specs/admin-draft-planned.openapi.yaml`.

### P1-7. Giá tương lai (ĐÃ LÀM)

- Migration `V11__channel_price_exclusion_constraint.sql`: tạo extension `btree_gist`, xóa unique index `uq_channel_prices_active`, thêm exclusion constraint `excl_channel_prices_no_overlap` dùng `tstzrange(effective_from, COALESCE(effective_to, 'infinity'))` với operator `&&` (overlaps) cho cùng `(variant_id, channel)`.
- Giữ index `idx_channel_prices_variant_channel_effective` hỗ trợ query tìm giá hiệu lực.
- Không sửa migration cũ đã chạy.

File: `resources/db/migration/V11__channel_price_exclusion_constraint.sql`.

## 6. Phase P2 — Hardening, docs, CI

- Implement `ListProductVariants` (pagination + `category_id` + `ocop_rating` + `active_only`).
- Kafka: phân loại lỗi; retry có giới hạn cho message độc; DLT cho deserialize/validation; giữ retry dài cho lỗi DB.
- Contract tests:
  - public: categories flat, search pagination, price effective, null stock;
  - admin: envelope `ApiResponse` + `data` required fields, status vocabulary, BFF unwrap;
  - internal: auth fail-closed, port 8005, channel 422.
- CI gate: `build_public_contract.py` idempotent + `verify.py` + `gradlew test`.
- Cập nhật `docs/ui_web/mvp-public-contract-baseline.json` (blob + source commit) khi merge.
- Dọn mô tả còn sót về public/admin gRPC trong docs.
- Cân nhắc bỏ hẳn HTTP `/internal/prices/validate` sau khi Order chuyển sang snapshot.

## 7. Thứ tự triển khai

1. P0-1 search + P0-2 price (cùng sửa `ProductMapper.xml`).
2. P0-3 categories + P0-4 public integrity.
3. P0-5 internal auth + P0-6 gRPC mapping.
4. P0-7 default sort + test P0.
5. P1-1…P1-5 admin read model.
6. P1-6/P1-7 sau khi chốt quyết định.
7. P2 hardening + CI.

Không làm P1 trước P0 vì admin BFF đang phụ thuộc public contract và dễ che lỗi dữ liệu.

## 8. Quyết định cần chốt trước khi code

| ID | Câu hỏi | Đề xuất |
| --- | --- | --- |
| D-01 | `BEST_SELLING` chưa có dữ liệu bán | MVP: map về `created_at DESC` và ghi rõ là fallback; bật analytics ở phase sau. Hoặc đổi contract default sang `NEWEST`. |
| D-02 | `thumbnail_url` không có ảnh | Dùng placeholder CDN cấu hình; giữ contract required + URI. Không trả chuỗi rỗng. |
| D-03 | Admin write dùng `/skus` hay `/variants` | Ưu tiên contract-first: `/skus`; nếu giữ `/variants`, cập nhật draft contract. |
| D-04 | Giá tương lai | Exclusion constraint theo khoảng thời gian; không dùng unique `status='ACTIVE'`. |
| D-05 | `/internal/prices/validate` | Giữ + bắt buộc token trong P0; loại bỏ ở P2 sau khi Order dùng snapshot. |
| D-06 | Product không có giá | Loại khỏi public list; detail trả 404/503 theo contract. Không 503 cả trang. |

## 9. Acceptance criteria

### P0

- `GET /api/v1/categories` trả **flat list**, có category con và `parent_id`.
- `GET /api/v1/products` mặc định `sort_by=BEST_SELLING`; filter/sort giá chỉ dùng variant ACTIVE, giá ACTIVE còn hiệu lực, đúng `channel`.
- `GET /api/v1/products/search?q=...&page=2&page_size=10` trả đúng trang, `total` là tổng thật; không lọc trong bộ nhớ.
- Không còn 503 toàn trang vì một item thiếu giá.
- `stock_available`/`in_stock` giữ null đúng nghĩa; Money không mất precision.
- Snapshot/revalidate/validate từ chối token thiếu/sai/ngắn; cấu hình token rỗng không fail-open.
- gRPC trả `product_id` thật; lỗi nghiệp vụ map đúng status, không thành `INTERNAL`.

### P1

- Admin list/detail giữ `ApiResponse`; schema 200 mô tả envelope và `data` đúng required fields.
- BFF đọc `payload.data`, không đọc trực tiếp `payload.items`/field detail.
- Error 400/403/404/500 của 2 GET admin dùng `AdminApiErrorResponse`.
- `saleStatus` dùng `ON_SALE/PAUSED/DISCONTINUED`; label tiếng Việt đúng.
- `statusCounts`, `categories`, `calculatedAt`, `foodInformation`, `coverImageUrl`, `skus` không null sai contract.
- Detail hiển thị cả SKU PAUSED/DISCONTINUED.
- Admin list nhận `q/status/page/pageSize`; BFF chuyển query xuống.
- Migration giá tương lai chạy được, không phá dữ liệu cũ.

### P2

- `verify.py` pass trong môi trường đủ dependency.
- CI chặn contract stale (`public != build()`), OpenAPI invalid, registry trỏ file thiếu, baseline compatibility fail.
- Kafka có DLT, không chặn partition vì message độc.
- Không còn tài liệu mô tả public/admin gRPC.

## 10. Rủi ro

- `verify.py` exclusion cho baseline Catalog là ngoại lệ có chủ đích; nếu sau này ai đó regenerate baseline phải giữ đúng quyết định xóa public/admin proto.
- Public contract đổi blob; quên cập nhật `docs/ui_web/mvp-public-contract-baseline.json` sẽ làm sai provenance frontend.
- Admin contract `additionalProperties: false`; trả thừa `total/page/...` sẽ làm BFF/validator fail.
- `BEST_SELLING` nếu không chốt sẽ tiếp tục là “NEWEST đội lốt”.
