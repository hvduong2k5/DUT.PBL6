# Product Discovery MVP — API Contract Candidate Review

> Trạng thái: **PROPOSED / NOT APPROVED**
>
> Phiên bản: `0.1.0-candidate`
>
> Ngày đề xuất: 2026-09-25
>
> OpenAPI: `product-discovery-mvp.openapi.yaml`

## 1. Điểm dừng của workflow

Artefact này hoàn thành bước **đề xuất API Contract từ UI đã được duyệt và implementation thực tế**. Nó chưa phải cam kết production của Backend/Architecture/Search/Inventory.

Chưa thực hiện:

- Review và phê duyệt liên team.
- Backend catalog/search/indexing, pricing và inventory integration.
- Contract/load test với API Gateway thật.
- Đồng bộ Mockoon/frontend theo contract đã được duyệt nếu review làm thay đổi candidate.
- Chuyển frontend khỏi Mockoon.

## 2. Nguồn hình thành contract

- `docs/01_requirements/epics/EPIC_03_Product_Discovery.md` trên nhánh `doc`: `US-DISC-01/02/03`.
- `docs/ui_web/PRODUCT_DISCOVERY_SCOPE_TRACEABILITY.md`: phạm vi và quyết định MVP.
- `docs/ui_web/PRODUCT_DISCOVERY_DATA_API_MATRIX.md`: field, query, state và mapping mock.
- D2C-002: visual reference; UI `/products` đã được người dùng duyệt.
- `mocks/mockoon/oma-product-discovery-mvp.json`: executable examples và data bucket.
- `apps/web-user/src/lib/catalog/*`, `src/services/catalog-service.ts` và BFF `/api/catalog/*`: query/payload UI thực sự dùng.

## 3. Ranh giới Browser, BFF, Gateway và Mockoon

| Lớp | Path | Trách nhiệm |
| --- | --- | --- |
| Browser → Next BFF | `/api/catalog/*` | Same-origin, allowlist route, validate query, ẩn upstream. |
| BFF → API Gateway | `/api/v1/catalog/*` | Contract candidate trong OpenAPI. |
| BFF → Mockoon local | Port `4012` | Ánh xạ public query sang Mockoon CRUD query. |
| Mock scenario | `X-Mock-Scenario` | Development/test only, không thuộc production contract. |

Browser và API Contract không biết các field chỉ mục `curatedRank/priceVnd/weightGrams/isAvailable` ở root của fixture Mockoon. BFF loại chúng trước khi trả response public.

## 4. Capability đề xuất

| ID | Method và path | Success |
| --- | --- | --- |
| `DISC-C01` | `GET /catalog/products` | `200 ProductSummary[]` + `X-Total-Count/X-Filtered-Count` |
| `DISC-C02` | `GET /catalog/discovery-config` | `200 DiscoveryConfig` |

Hai endpoint đều public. Gateway vẫn cần rate limit, abuse protection, request ID và observability.

## 5. Quyết định phản ánh trong candidate

- Chỉ product đã approved/published mới xuất hiện.
- Giá, khối lượng, package và availability là thuộc tính SKU/offer; không suy diễn từ Product.
- Product chỉ xuất hiện khi ít nhất một SKU khớp toàn bộ filter; `matchedOffer` là SKU đại diện mà UI hiển thị.
- `inStock=true` loại SKU không có tồn bán được, thuộc lô hết hạn/bị khóa hoặc không còn hiệu lực.
- Giá là số nguyên VND; khối lượng là số nguyên gram.
- Search tiếng Việt không phân biệt hoa thường và dấu; từ khóa sau normalize dài 2–100 ký tự.
- URL/query là nguồn trạng thái của search/filter/sort/page.
- `RELEVANCE` chỉ có ý nghĩa khi có `q`; nếu không có từ khóa thì fallback `CURATED`.
- Offset pagination dùng `page/pageSize`; mọi sort có product ID làm tie-breaker ổn định.
- Tổng sau lọc và trước phân trang trả qua `X-Filtered-Count`; tổng public catalog trước lọc qua `X-Total-Count`.
- Chỉ công bố `isAvailable`, không công bố số lượng tồn kho chính xác.
- `ocopStars` là cấp chứng nhận OCOP, không phải rating của khách hàng.
- Customer rating/filter không có trong candidate cho tới khi EPIC 15 cung cấp dữ liệu thật.
- Product detail, cart, promotions và personalized recommendations không nằm trong contract này.

## 6. Phát hiện từ implementation thực tế

### Query và field UI đang dùng

- UI gửi `q/category/productType/minPrice/maxPrice/weights/inStock/sort/page/pageSize`.
- UI đọc product identity, display content, category, product type, ảnh, chứng nhận/badge và `matchedOffer`.
- UI không cần exact inventory, warehouse, lot, cost price hay unpublished state.
- Config API giúp UI không hard-code danh mục, product type, weight và price presets.
- Combined filters cần semantics **AND giữa các nhóm**, còn nhiều `weights` là **OR trong cùng nhóm**.
- Empty result vẫn là `200` với array rỗng và count bằng `0`; không phải `404`.

### Thu hẹp dữ liệu sau UI review

Mock fixture có `availableQuantity` để mô phỏng inventory nhưng UI không dùng. BFF và candidate đã loại field này khỏi public response để tránh lộ dữ liệu vận hành và tránh tạo coupling không cần thiết.

## 7. Sai khác có chủ đích so với Mockoon

- OpenAPI không chứa `/health`, `X-Mock-Scenario` hay cú pháp CRUD như `priceVnd_gte`.
- Mockoon full-text chỉ chứng minh luồng UI; contract thật yêu cầu normalize/tìm kiếm tiếng Việt không phân biệt dấu.
- Mockoon dùng field phẳng làm index lọc/sort rồi trả chúng ở route trực tiếp; BFF loại các field này. Backend chỉ cần representation canonical trong OpenAPI.
- Một fixture hiện đại diện một `matchedOffer` cho mỗi Product. Backend phải chọn từ toàn bộ SKU hợp lệ theo filter và quy tắc deterministic.
- Mockoon có exact quantity nội bộ; contract chỉ trả boolean availability.
- Mockoon không chứng minh approval, lot expiry, inventory reservation, pricing consistency, search relevance hoặc distributed failure modes.

## 8. Câu hỏi cần Backend/Architecture/Search/Inventory chốt

| ID | Câu hỏi | Candidate hiện tại |
| --- | --- | --- |
| `DISC-CR-01` | Gateway gọi một Catalog Query Service hay aggregate Product + Pricing + Inventory mỗi request? | Ưu tiên read model/search index chuyên cho discovery; không lộ topology qua public API. |
| `DISC-CR-02` | Search engine nào chịu trách nhiệm normalize dấu, typo và relevance? | Contract yêu cầu case/accent-insensitive; typo tolerance chưa bắt buộc. |
| `DISC-CR-03` | Khi nhiều SKU cùng khớp, chọn `matchedOffer` theo quy tắc nào? | Cần deterministic merchandising priority, sau đó price và SKU ID; Backend/Product chốt thứ tự. |
| `DISC-CR-04` | Giá hiển thị là list price, active selling price hay promotional price? | Active selling price không bao gồm model promotion chi tiết; EPIC promotion mở rộng sau. |
| `DISC-CR-05` | Availability được materialize với độ trễ tối đa bao nhiêu? | Discovery là gần thời gian thực; Cart/Checkout vẫn phải revalidate giá và tồn. |
| `DISC-CR-06` | Lot expiry/blocking được đưa vào available stock ở service nào? | Inventory/read model chịu trách nhiệm; Product API không tự suy diễn. |
| `DISC-CR-07` | Offset pagination có đáp ứng quy mô/index hiện tại hay cần cursor? | Candidate giữ page/pageSize vì UI MVP và khả năng bookmark; cần load test. |
| `DISC-CR-08` | Exact `X-Filtered-Count` có quá đắt với search index lớn? | Candidate yêu cầu exact count cho pagination; có thể đổi sang envelope/cursor trước Approved. |
| `DISC-CR-09` | Category/product type/config thuộc master data nào và cache bao lâu? | Discovery Config là nguồn UI; ownership/TTL cần Architecture chốt. |
| `DISC-CR-10` | URL ảnh là CDN public bền vững hay signed URL ngắn hạn? | Candidate yêu cầu URL browser dùng được; ưu tiên CDN URL versioned, không hết hạn trong phiên. |
| `DISC-CR-11` | Nguồn và vòng đời chứng nhận OCOP? | `ocopStars` nullable 3–5, tách biệt customer rating; cần data owner xác nhận. |
| `DISC-CR-12` | Cache policy cho catalog có availability là gì? | Chưa khóa TTL; Cart/Checkout luôn revalidate để chấp nhận eventual consistency. |
| `DISC-CR-13` | Rate limit public search theo IP/device/session là bao nhiêu? | Có response `429 + Retry-After`; ngưỡng cần Gateway/Security chốt. |
| `DISC-CR-14` | Có cần localization cho category/label ngay MVP? | Candidate trả chuỗi tiếng Việt; chưa thiết kế locale/content negotiation. |
| `DISC-CR-15` | Stable sort và consistency khi index đổi giữa hai page? | Product ID tie-breaker; snapshot consistency chưa cam kết. |

## 9. Checklist review đề xuất

### Backend/Product Catalog

- Xác nhận schema Product/SKU, published/approved rules và `matchedOffer` selection.
- Xác nhận active selling price và product/category master data.
- Xác nhận error/status semantics và response count headers.

### Inventory/Pricing

- Xác nhận sellable stock, lot expiry/blocking và staleness SLA.
- Xác nhận không công bố exact quantity và Checkout sẽ revalidate.

### Architecture/Search

- Xác nhận read model/service boundary, Vietnamese normalization và relevance.
- Đánh giá page pagination, exact counts, caching, index refresh và failure strategy.

### Security/Gateway

- Xác nhận public-route rate limit, bot/abuse protection, log redaction và request IDs.
- Chỉ cho phép query/limits trong contract; không forward trực tiếp cú pháp datastore/search engine.

### Frontend

- Sau khi Approved, generate/sync types, cập nhật Mockoon và contract tests.
- Đổi `CATALOG_UPSTREAM_URL` sang API Gateway rồi chạy regression/E2E.

## 10. Tiêu chí đổi sang Approved

- Backend xác nhận route, query, schema, selection logic, price và product state.
- Inventory/Pricing xác nhận availability semantics và revalidation boundary.
- Architecture/Search xác nhận service/read model, normalization, sorting, pagination, count và caching.
- Security/Gateway xác nhận public abuse/rate-limit policy.
- Open questions có quyết định được ghi vào OpenAPI/review log.
- Mockoon, frontend types và generated client đồng bộ theo contract đã review.
- Contract/load tests chứng minh combined filters, expired-stock exclusion, stable pagination, invalid query, empty result, 429 và dependency failures trên Gateway thật.

Cho đến khi hoàn tất, version phải giữ hậu tố `-candidate` và trạng thái **PROPOSED / NOT APPROVED**.
