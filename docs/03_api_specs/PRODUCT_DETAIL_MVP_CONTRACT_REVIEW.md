# Product Detail & SKU Selection MVP — API Contract Candidate Review

> Trạng thái: **PROPOSED / NOT APPROVED**
>
> Phiên bản: `0.1.0-candidate`
>
> Ngày đề xuất: 2026-09-25
>
> OpenAPI: `product-detail-mvp.openapi.yaml`

## 1. Điểm dừng workflow

UI Product Detail & SKU Selection đã được người dùng duyệt. Artefact này hoàn thành bước đề xuất API Contract dựa trên requirement, quyết định phạm vi, Mockoon và implementation thực tế.

Chưa thực hiện:

- Backend/Architecture/Inventory/Pricing/Security review hoặc phê duyệt.
- Backend implementation, API Gateway deployment và contract test.
- Chuyển frontend từ Mockoon sang API Gateway.
- Cart/Checkout integration, inventory reservation hoặc Batch allocation.

Candidate không được xem là production-ready cho đến khi đáp ứng toàn bộ điều kiện ở mục 10.

## 2. Nguồn hình thành contract

- `docs/01_requirements/epics/EPIC_04_Product_SKU.md` trên branch `doc`: `US-PROD-03/04`, phần đọc của `US-PROD-05/06`, `FR-PROD-05~09`.
- `docs/ui_web/PRODUCT_DETAIL_SCOPE_TRACEABILITY.md`: phạm vi customer-facing và business decisions đã dùng để triển khai.
- `docs/ui_web/PRODUCT_DETAIL_DATA_API_MATRIX.md`: capability, field, validation, state và security boundary.
- D2C-004 `screen.png`/`code.html`: visual reference; rating, combo, shipping promise và traceability lô đã được loại khỏi MVP.
- UI `/products/[slug]` đã được duyệt; Product Discovery card đã nối bằng slug.
- `mocks/mockoon/oma-product-detail-mvp.json`: executable examples cho multi-SKU, single-SKU, unavailable, optional missing, error, slow và not-found.
- `src/lib/product-detail/*`, `src/services/product-detail-service.ts` và BFF `/api/catalog/*`: payload và validation mà UI thực tế dùng.

## 3. Ranh giới Browser, BFF, Gateway và Mockoon

| Lớp | Path/config | Trách nhiệm |
| --- | --- | --- |
| Browser → Next.js BFF | `GET /api/catalog/products/{slug}` | Same-origin, validate slug/query, fixed allowlist, ẩn upstream. |
| BFF → API Gateway | `GET /api/v1/catalog/products/{slug}` | Contract candidate trong OpenAPI. |
| BFF → Mockoon local | `PRODUCT_DETAIL_UPSTREAM_URL`, port `4013` | Executable UI example trước khi Backend sẵn sàng. |
| Local scenario | `X-Mock-Scenario` | Development/test only; không thuộc Gateway contract. |

BFF loại `internalStatus`, `publicationStatus`, `curatedRank`, search/audit data và các field SKU như exact quantity, warehouse, cost price, internal reason, Batch/Lot trước khi trả browser.

## 4. Capability đề xuất

| ID | Method và Gateway path | Security | Success |
| --- | --- | --- | --- |
| `DETAIL-C01` | `GET /catalog/products/{slug}` | Public, không yêu cầu đăng nhập | `200 ProductDetail` |

Không đề xuất endpoint Cart, inventory, Batch, review/rating, promotion, related product hoặc traceability trong contract này.

## 5. Quyết định candidate

1. Product detail chỉ trả Product/SKU đã approved và được phép public.
2. Product không tồn tại, unpublished hoặc không visible đều dùng `404 PRODUCT_NOT_FOUND`; response không tiết lộ trạng thái nội bộ.
3. Product public vẫn có thể trả `200` khi toàn bộ SKU unavailable để khách đọc thông tin; UI disabled mọi lựa chọn mua.
4. Response trả toàn bộ public SKU, gồm SKU unavailable. Backend không cung cấp một implicit selected/default SKU trong candidate.
5. UI multi-SKU bắt đầu chưa có confirmed purchase selection. Single-SKU chỉ được auto-select phía UI khi response có đúng một SKU và SKU đó available.
6. Giá, khối lượng, flavor, package và availability nằm ở SKU. Giá là integer VND và được mô tả là active Website selling price; promotion nằm ngoài phạm vi.
7. `isAvailable` là projection public; Cart/Checkout tương lai vẫn phải revalidate giá và khả năng bán.
8. `unavailableReason` chỉ là customer-safe text. Exact quantity, warehouse, internal reason và service topology không public.
9. `manufacturingDatePolicy` và `shelfLifeDescription` là policy catalog. Candidate không có actual manufacturing/expiry date, Batch ID hoặc Lot ID.
10. Gallery hỗ trợ product images và optional SKU image override. URL phải dùng được từ browser trong vòng đời hiển thị.
11. `ocopCertification` chỉ là public summary đã được phê duyệt, không phải customer rating và không thay thế EPIC 19.
12. Food information bắt buộc để public gồm ingredients, storage instructions, manufacturing-date policy và shelf-life description; allergen statement nullable theo chính sách được duyệt.

## 6. Phát hiện từ implementation thực tế

### Selection behavior

- UI không dùng `recommendedSkuId`, cheapest-SKU hoặc thứ tự array để tự chọn khi có nhiều SKU.
- `selectedSkuId` chỉ được đặt từ thao tác khách, ngoại trừ một SKU available duy nhất.
- Khi chọn SKU, UI cập nhật giá, khối lượng, flavor, package, availability và SKU image nếu có.
- SKU unavailable render trong cùng danh sách nhưng radio disabled; không thể trở thành selected value.
- CTA hiện là integration point rõ ràng và không gửi Cart request.

### Resilience và error semantics

- Invalid slug hoặc unsupported query bị BFF chặn trước upstream.
- `404` có state riêng và link về catalog.
- `500/503` dùng retry state; slow response giữ loading skeleton.
- Optional certification/allergen có thể không xuất hiện mà UI không tạo dữ liệu thay thế.

### Data minimization

Mock fixture cố ý chứa `availableQuantity`, `warehouseCode`, `costPriceVnd`, `internalUnavailableReason`, publication/internal status và search index để chứng minh BFF projection. Runtime check xác nhận browser payload không nhận các field này.

## 7. Sai khác có chủ đích với Mockoon

- OpenAPI không chứa `/health`, `X-Mock-Scenario`, port hoặc tên environment.
- Mockoon có header scenario để ép UI state; Gateway chọn response từ dữ liệu nghiệp vụ thật.
- Mock fixture chứa internal fields để kiểm thử stripping; OpenAPI `additionalProperties: false` chỉ mô tả public projection.
- Mockoon trả customer-safe reason dưới dạng tiếng Việt cố định. Backend cần chốt code/localization ownership trước Approved; candidate hiện giữ string vì UI implementation đang dùng string.
- Mockoon không chứng minh approval workflow, price effectiveness, inventory staleness, Batch eligibility, CDN lifecycle, rate limiting hoặc distributed dependency failures.
- Mockoon `product-not-found` mô phỏng cả unknown và unpublished semantics; Backend phải áp dụng authorization/publication rule thật nhưng không phân biệt nguyên nhân trong public response.

## 8. Open questions

| ID | Team | Câu hỏi | Candidate hiện tại |
| --- | --- | --- | --- |
| `DETAIL-CR-01` | Backend/Architecture | Product Detail aggregate Catalog, Pricing và Availability trực tiếp hay qua read model? | Không lộ topology; ưu tiên public projection có SLA rõ. |
| `DETAIL-CR-02` | Product/Backend | Product discontinued nhưng còn giá trị đọc/SEO có public read-only hay trả 404? | Public product có thể `200` all-unavailable; unpublished/not-visible là `404`. |
| `DETAIL-CR-03` | Inventory | `isAvailable` tổng hợp SKU status, sellable stock, Batch expiry/blocking với độ trễ tối đa bao nhiêu? | Boolean projection; Cart/Checkout revalidate bắt buộc. |
| `DETAIL-CR-04` | Inventory/Security | Bộ unavailable reason public được chuẩn hóa theo code/text nào? | Candidate dùng nullable customer-safe string, không lộ internal reason. |
| `DETAIL-CR-05` | Pricing/Finance | `priceVnd` đã gồm thuế/phí nào và hiệu lực giá được bảo đảm thế nào? | Active Website selling price, integer VND; promotion ngoài phạm vi. |
| `DETAIL-CR-06` | Pricing/Cart | Khi giá đổi sau khi mở trang, Cart nhận SKU rồi xử lý price mismatch ra sao? | Cart phải revalidate và thông báo thay đổi; chưa thuộc endpoint này. |
| `DETAIL-CR-07` | Food Compliance | Manufacturing date public nên là policy text hay structured code + localized content? | Candidate dùng approved text, không có actual Batch date. |
| `DETAIL-CR-08` | Food Compliance | Allergen statement có thật sự optional với mọi product type không? | Nullable trong candidate; publish rule phải enforce theo loại sản phẩm/pháp lý. |
| `DETAIL-CR-09` | EPIC 19/Data Owner | Field OCOP summary nào Product Catalog được phép sao chép/public? | Chỉ stars + label optional; traceability chi tiết bị loại. |
| `DETAIL-CR-10` | Media/Architecture | Product/SKU image dùng CDN versioned URL hay signed URL; TTL bao lâu? | URI browser-usable; không được hết hạn giữa phiên thông thường. |
| `DETAIL-CR-11` | Backend | Có cần explicit SKU sort/display order? | Candidate giữ array order nhưng chưa contract hóa merchandising rule. |
| `DETAIL-CR-12` | Backend | Flavor/package có cần structured option model để tạo selector theo dimension? | MVP trả flat SKU list vì UI chọn SKU trực tiếp, không tổng hợp tổ hợp giả. |
| `DETAIL-CR-13` | Gateway/Security | Rate limit public detail theo IP/device/cache key là bao nhiêu? | Có `429 + Retry-After`; ngưỡng cần chốt. |
| `DETAIL-CR-14` | Architecture | Cache/ETag policy cho detail có availability là gì? | Chưa khóa TTL; availability có eventual consistency và phải revalidate ở Cart. |
| `DETAIL-CR-15` | Localization | Public text và reason có cần locale/content negotiation ở MVP? | Candidate trả text tiếng Việt; localization chưa thiết kế. |

## 9. Checklist review từng team

### Backend/Product Catalog

- Xác nhận publish/visibility và `404` non-disclosure semantics.
- Xác nhận schema Product, SKU, food information, image ordering và SKU ordering.
- Xác nhận all-unavailable product có được public read-only.
- Xác nhận validation, error code, request ID và `additionalProperties` behavior.

### Inventory/Batch

- Xác nhận nguồn, staleness SLA và failure behavior của `isAvailable`.
- Xác nhận Batch hết hạn/bị khóa không thể tạo `isAvailable=true`.
- Xác nhận exact stock, warehouse, Batch/Lot và actual dates không public.

### Pricing/Finance

- Xác nhận active Website selling price, thuế/phí, currency và thời điểm hiệu lực.
- Xác nhận Cart/Checkout revalidation khi giá thay đổi.

### Food Compliance/EPIC 19

- Xác nhận publish rule cho ingredient, allergen, storage và shelf-life policy.
- Xác nhận manufacturing/expiry catalog text không gây nhầm với Batch actual date.
- Xác nhận quyền sở hữu và phạm vi public của OCOP summary.

### Architecture/Media

- Xác nhận aggregation/read-model boundary, caching, ETag và dependency failure semantics.
- Xác nhận CDN URL lifecycle, media approval và SKU image override.

### Security/Gateway

- Xác nhận endpoint public, rate limit, bot protection, request ID và log redaction.
- Bảo đảm unpublished/internal product không bị enumeration qua khác biệt status/timing.
- Chỉ allow path/query trong contract; không forward arbitrary URL/header.

### Frontend

- Sau Approved, sync generated types, error handling và Mockoon theo contract đã review.
- Đổi `PRODUCT_DETAIL_UPSTREAM_URL` sang API Gateway và chạy regression/E2E.
- Tích hợp Cart chỉ khi EPIC 05 có contract riêng và luôn truyền explicit `skuId`.

## 10. Điều kiện chuyển sang Approved

- Backend/Product xác nhận route, schema, publication, SKU order và all-unavailable semantics.
- Inventory/Batch xác nhận availability projection, expiry/blocking và revalidation boundary.
- Pricing/Finance xác nhận active price, tax/fee semantics và price-change behavior.
- Food Compliance/EPIC 19 xác nhận food policy, allergen và OCOP ownership.
- Architecture/Media xác nhận aggregation, caching, CDN và dependency failure strategy.
- Security/Gateway xác nhận public exposure, anti-enumeration, rate limit và observability.
- Mọi open question có quyết định được ghi vào OpenAPI/review log.
- Contract tests bao phủ multi/single SKU, unavailable, all-unavailable, invalid slug, unpublished/not-found, error và dependency failure.
- Mockoon, frontend types/BFF và Backend implementation đồng bộ với contract đã review.
- Regression/E2E qua API Gateway chứng minh browser không nhận exact stock, warehouse, cost, Batch/Lot hoặc internal status.

Cho đến khi hoàn tất, version giữ `0.1.0-candidate` và trạng thái **PROPOSED / NOT APPROVED**.
