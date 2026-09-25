# Shopping Cart MVP — API Contract Candidate Review

> Trạng thái: **PROPOSED / NOT APPROVED**
>
> Phiên bản: `0.1.0-candidate`
>
> Ngày đề xuất: 2026-09-25
>
> OpenAPI: `shopping-cart-mvp.openapi.yaml`

## 1. Điểm dừng workflow

UI Shopping Cart MVP đã được người dùng duyệt, gồm Add to Cart từ Product Detail, route `/cart`, cập nhật quantity, xóa dòng, chọn một phần giỏ để chuẩn bị Checkout và các state loading/empty/error/unavailable.

Artefact này hoàn thành bước đề xuất API Contract Candidate từ requirement và implementation thực tế. Chưa thực hiện:

- Backend/Architecture/Product/Inventory/Pricing/Security review hoặc phê duyệt.
- Backend implementation, API Gateway deployment hoặc contract test.
- Chuyển frontend từ Mockoon sang API Gateway.
- Checkout, Order, inventory reservation, shipping, promotion hoặc payment.

Candidate không phải production contract cho đến khi đáp ứng toàn bộ điều kiện ở mục 10.

## 2. Nguồn hình thành contract

- `docs/01_requirements/epics/EPIC_05_Shopping_Cart.md` trên branch `doc`: `US-CART-01~04`, `FR-CART-01~09`.
- `docs/ui_web/SHOPPING_CART_SCOPE_TRACEABILITY.md`: customer-facing scope, business decisions và dependency.
- `docs/ui_web/SHOPPING_CART_DATA_API_MATRIX.md`: capability, validation, response, error và data boundary.
- UI `/cart` đã duyệt và Add to Cart từ `/products/[slug]`.
- `mocks/mockoon/oma-shopping-cart-mvp.json`: executable examples cho stateful Add/Update/Delete, empty, price changed, unavailable, error và slow.
- `src/lib/cart/*`, `src/services/cart-service.ts` và BFF `/api/cart/*`: payload, allowlist và validation UI thực tế đang sử dụng.

## 3. Ranh giới Browser, BFF, Gateway và Mockoon

| Lớp | Path/config | Trách nhiệm |
| --- | --- | --- |
| Browser → Next.js BFF | `GET /api/cart`, `POST /api/cart/items`, `PATCH/DELETE /api/cart/items/{itemId}` | Same-origin, validate path/body/query, quản lý opaque HttpOnly cart context, không lộ upstream. |
| BFF → API Gateway | `/api/v1/cart` và `/api/v1/cart/items/{itemId}` | Contract Candidate trong OpenAPI. |
| BFF → Mockoon local | `CART_UPSTREAM_URL`, port `4014` | Executable UI example trước khi Backend sẵn sàng. |
| Local scenario | `X-Mock-Scenario` | Development/test only; không thuộc Gateway contract. |

Browser JavaScript không đọc cart context, không biết upstream URL và không gọi Mockoon trực tiếp. Candidate mô tả cookie `oma_cart_context` như opaque Cart Session; cơ chế ký/mã hóa, mapping Registered Customer và propagation BFF → Gateway vẫn cần Security/Architecture phê duyệt.

## 4. Capability đề xuất

| ID | Method và Gateway path | Kết quả |
| --- | --- | --- |
| `CART-C01` | `GET /cart` | Cart hiện tại đã revalidate, gồm empty/unavailable/price-change projection. |
| `CART-C02` | `POST /cart/items` | Thêm exact SKU hoặc gộp quantity khi cùng SKU đã tồn tại. |
| `CART-C03` | `PATCH /cart/items/{itemId}` | Thay quantity của đúng owned line và trả full cart snapshot mới. |
| `CART-C04` | `DELETE /cart/items/{itemId}` | Xóa owned line và trả full cart snapshot mới. |

Không đề xuất endpoint `/health`, Mockoon CRUD bucket, cart selection, Checkout, shipping, promotion, reservation, Order hoặc payment trong contract này.

## 5. Quyết định Candidate

1. Cart phục vụ Guest và Registered Customer, không ép Guest đăng nhập. Mọi read/mutation được scope theo cart context hiện tại.
2. `itemId` là opaque line identifier. Item không tồn tại và item thuộc context khác đều trả `404 CART_ITEM_NOT_FOUND` để không lộ ownership.
3. Mỗi line xác định một exact `skuId`; thêm lại cùng SKU sẽ merge quantity, không tạo duplicate line.
4. `quantity` là integer `1..20` theo guardrail Candidate đã dùng trong UI/BFF. Quantity `0` không đồng nghĩa với delete.
5. Add/Update luôn revalidate current public selling state và khả năng mua. `409` chỉ trả customer-safe `SKU_UNAVAILABLE` hoặc `QUANTITY_UNAVAILABLE`, không trả exact stock.
6. Mutation thành công trả full revalidated Cart snapshot để UI không ghép subtotal/availability từ state cũ.
7. Dòng vừa trở nên unavailable được giữ lại để khách hiểu và xử lý. `lineSubtotalVnd` của dòng đó là `null`; cart `subtotalVnd` chỉ cộng dòng available.
8. `itemCount` là tổng quantity của mọi dòng còn giữ trong cart, kể cả unavailable; UI phải dùng label phù hợp và không coi đây là số lượng có thể Checkout.
9. Giá là active Website selling price bằng integer VND. `previousUnitPriceVnd` chỉ có khi revalidation phát hiện giá thay đổi; Cart không áp promotion.
10. Cart không giữ tồn, không khóa giá, không chọn warehouse/Batch/Lot và không tạo Order.
11. Checkbox chọn một phần giỏ là UI state tạm thời, không mutate Cart API. Checkout tương lai phải nhận tập dòng được chọn theo cơ chế riêng và revalidate ownership, price, availability cùng cart version.
12. Candidate trả `200 Cart` sau DELETE thay vì `204` vì UI cần snapshot và subtotal mới mà không phát thêm request.
13. POST trả `201` khi tạo line mới và `200` khi merge line có sẵn. Backend cần xác nhận client có cần phân biệt hai trường hợp qua status hay response metadata.

## 6. Phát hiện từ implementation thực tế

- Product Detail chỉ gọi Add khi đã có explicit SKU và quantity hợp lệ; payload không chứa Product name/price do client tự khai báo.
- BFF dùng allowlist cố định cho đúng bốn method/path; arbitrary nested path và query bị từ chối trước upstream.
- Cookie context là `HttpOnly`, `SameSite=Lax`, `Path=/`; `Secure` được thêm trong production. TTL local hiện là 30 ngày và chưa phải quyết định nghiệp vụ.
- BFF dựng public Cart projection từ line storage và current SKU projection, rồi loại `contextId`, giá snapshot nội bộ, max purchasable quantity và CRUD metadata.
- UI giữ line unavailable, khóa quantity/checkbox và cho phép các line available khác vẫn được chọn cho Checkout tương lai.
- UI giữ selection hợp lệ khi quantity/update/remove thành công, nhưng selection không persist qua reload.
- Local BFF đang dùng `INVALID_QUANTITY` cho một số lỗi body Add rộng hơn quantity; Candidate chuẩn hóa thành `VALIDATION_FAILED`. BFF chỉ được đổi sau khi contract được review.
- Local BFF chưa triển khai optimistic concurrency/cart version, idempotency key, Origin/CSRF enforcement hoặc Registered Customer merge.

## 7. Sai khác có chủ đích với Mockoon

| Mockoon/local adapter | API Candidate |
| --- | --- |
| `/api/v1/internal/cart-lines` và `/api/v1/internal/cart-sku-projections` | Một Cart resource `/cart` với owned mutation. |
| Data bucket CRUD query như `contextId_eq`, `sort`, `order` | Không public query syntax hoặc storage topology. |
| `X-Mock-Scenario` và `mockScenario` | Không tồn tại trong production contract. |
| `/api/v1/health` | Không nằm trong business OpenAPI. |
| `contextId`, `unitPriceVndAtAddition`, `maxPurchasableQuantity`, bucket row data | Không public cho browser/client. |
| Fixture contexts như `mock-cart-prefilled` | Không phải identity/session production. |
| BFF tự compose/recompute Cart | Backend/Gateway phải cung cấp projection đúng contract; vị trí aggregation là quyết định Architecture. |

Mockoon tiếp tục là executable example, không phải nguồn sự thật cuối cùng. Sau khi Candidate được Approved, Mockoon và BFF phải đồng bộ theo contract được duyệt.

## 8. Open questions

### Product/Business

1. Maximum quantity cho mỗi line/toàn cart là bao nhiêu; guardrail `20` có được phê duyệt không?
2. Add cùng SKU phải luôn cộng quantity hay có trường hợp replace/cap?
3. Dòng unavailable nên được giữ bao lâu, và public reason code/text chuẩn hóa gồm những gì?
4. `itemCount` nên gồm unavailable như Candidate hay cần thêm `availableItemCount`?
5. Customer có phải acknowledge price change trước khi Checkout không?

### Backend/Architecture

6. Guest cart TTL, storage owner và cleanup policy là gì?
7. Registered Customer cart có sync đa thiết bị không; Guest cart được merge/replace khi đăng nhập?
8. Có cần `cartVersion`/ETag/`If-Match` để chống lost update và để Checkout revalidate đúng snapshot không?
9. POST Add cần `Idempotency-Key` để tránh double-add khi retry mạng không?
10. API có giữ status `201` mới/`200` merge hay luôn trả một status?
11. Dependency Product/Pricing/Inventory lỗi một phần sẽ fail toàn cart hay trả stale/partial projection có đánh dấu?

### Inventory/Pricing

12. Quy tắc tổng hợp `isAvailable` từ selling status, inventory và Batch eligibility là gì?
13. `QUANTITY_UNAVAILABLE` có được public giới hạn mua hay phải luôn giữ message trung tính?
14. Active Website price đã gồm thuế nào; hiệu lực và rounding được xác định ra sao?
15. `previousUnitPriceVnd` được phép public trong bao lâu và lấy từ snapshot nào?

### Security

16. Cart context được random, signed hay encrypted; Gateway nhận cookie hay trusted principal/header do BFF tạo?
17. Chính sách CSRF/Origin cho mutation dùng cookie là gì ngoài `SameSite=Lax`?
18. Session rotation, fixation protection, rate limit và abuse limit cho Guest cart là gì?
19. Khi item không thuộc cart, timing/logging có bảo đảm không lộ ownership không?

### Checkout dependency

20. Checkout nhận `itemId[]`, selection token hay server-side selection; có cho cùng cart mở nhiều checkout subset không?
21. Dòng không được chọn có tiếp tục nằm trong cart sau khi Order tạo thành công không?
22. Checkout sẽ revalidate/lock price và reserve inventory ở bước nào; response conflict quay lại Cart ra sao?

## 9. Checklist review theo team

### Backend/Cart

- Xác nhận route, status, schema, merge semantics và full-snapshot mutation response.
- Xác nhận ownership, TTL, Registered Customer mapping, merge và concurrency/idempotency.
- Bảo đảm error không public exact inventory hoặc cart context của người khác.

### Product/UX

- Xác nhận quantity limit, unavailable retention/reason và price-change acknowledgement.
- Xác nhận cart subtotal/item count semantics và partial Checkout behavior.
- Xác nhận unavailable line không chặn các dòng available được chọn riêng.

### Inventory/Batch

- Xác nhận purchase-eligibility projection và thời điểm revalidation.
- Xác nhận exact stock, warehouse, Batch/Lot, FEFO và actual expiry không public qua Cart.
- Xác nhận Cart không reserve inventory.

### Pricing/Finance

- Xác nhận active Website price, thuế, rounding và price snapshot policy.
- Xác nhận promotion/shipping/final total không thuộc Cart Candidate.

### Architecture

- Xác nhận storage/aggregation boundary, versioning, idempotency và dependency-failure strategy.
- Xác nhận Browser → BFF → Gateway identity propagation mà không lộ topology/upstream.
- Xác nhận selection handoff từ Cart sang Checkout.

### Security/Gateway

- Xác nhận opaque cart context, cookie flags, CSRF/Origin, fixation protection và rotation.
- Xác nhận rate limit, anti-enumeration, request ID, logging và redaction.
- Chỉ allow method/path/body trong contract; không forward arbitrary URL/header/query.

### Frontend

- Sau Approved, sync generated types, validation, error code và BFF với contract.
- Đổi `CART_UPSTREAM_URL` sang API Gateway và chạy regression/E2E.
- Không gửi UI checkbox selection vào Cart mutation; chỉ handoff theo Checkout contract được duyệt.

## 10. Điều kiện chuyển sang Approved

- Backend/Product xác nhận route, status, quantity, merge, unavailable và subtotal semantics.
- Architecture xác nhận cart/session identity, persistence, concurrency, idempotency và partial Checkout handoff.
- Inventory/Batch xác nhận availability projection và non-public data boundary.
- Pricing/Finance xác nhận active price, tax và price-change policy.
- Security/Gateway xác nhận cookie/session, CSRF, ownership, rate limit và anti-enumeration.
- Checkout team xác nhận selected-line handoff và revalidation/reservation boundary.
- Mọi open question có quyết định được ghi vào OpenAPI/review log.
- Contract tests bao phủ empty, create, duplicate merge, update, delete-last-line, invalid quantity, SKU unavailable, quantity unavailable, price changed, ownership not-found và dependency failure.
- OpenAPI parse thành công, toàn bộ internal `$ref` resolve và lint không còn lỗi blocking.
- Mockoon, frontend types/BFF và Backend implementation đồng bộ với contract đã review.
- Regression/E2E qua API Gateway chứng minh browser không nhận cart context value qua JavaScript, exact stock, warehouse, cost, Batch/Lot hoặc internal storage data.

Cho đến khi hoàn tất, version giữ `0.1.0-candidate` và trạng thái **PROPOSED / NOT APPROVED**.
