# Checkout MVP — API Contract Candidate Review

> Trạng thái: **PROPOSED / NOT APPROVED**
>
> Phiên bản: `0.1.0-candidate`
>
> Ngày đề xuất: 2026-09-25
>
> OpenAPI: `checkout-mvp.openapi.yaml`

## 1. Điểm dừng workflow

UI Checkout MVP đã được người dùng duyệt, gồm Cart selection → `/checkout`, Guest/Registered Customer, shipping quote, price-change acknowledgment, confirm idempotent, conflict/error/slow state và success boundary tạo Order `UNPAID`.

Artefact này hoàn thành bước đề xuất API Contract Candidate từ requirement và implementation thực tế. Chưa thực hiện:

- Backend/Architecture/Product/Inventory/Shipping/Pricing/Order/Payment/Security review hoặc phê duyệt.
- Backend implementation, API Gateway deployment hoặc contract test.
- Chuyển frontend từ Mockoon sang API Gateway.
- Payment transaction, QR/card integration, webhook hoặc trạng thái `PAID`.

Candidate không phải production contract cho đến khi đáp ứng toàn bộ điều kiện ở mục 10.

## 2. Nguồn hình thành contract

- `docs/01_requirements/01_Product_Backlog.md`: `US-CHK-01~04`; `US-CHK-05` được hoãn.
- `docs/ui_web/CHECKOUT_SCOPE_TRACEABILITY.md`: scope, dependency và quyết định business hiện tại.
- `docs/ui_web/CHECKOUT_DATA_API_MATRIX.md`: payload, validation, error và data boundary.
- UI `/checkout` cùng integration `/cart` đã duyệt.
- `mocks/mockoon/oma-checkout-mvp.json`: executable examples cho Guest/Registered, quote, conflict, error, slow và Order `UNPAID`.
- `src/lib/checkout/*`, `src/services/checkout-service.ts` và BFF `/api/checkout/*`: payload, allowlist, revalidation và calculation thực tế.

## 3. Ranh giới Browser, BFF, Gateway và Mockoon

| Lớp | Path/config | Trách nhiệm |
| --- | --- | --- |
| Browser → Next.js BFF | `POST /api/checkout/prepare`, `/shipping-quotes`, `/confirm` | Same-origin, validate body/query, giữ HttpOnly cart context, không nhận money/parcel do client khai báo. |
| BFF → API Gateway Candidate | `POST /api/v1/checkout/sessions`, `/{id}/shipping-quotes`, `/{id}/orders` | Resource contract được đề xuất trong OpenAPI. |
| BFF → Mockoon local | `CHECKOUT_UPSTREAM_URL`, port `4015`; Cart port `4014` | Executable UI example trước khi Backend sẵn sàng. |
| Local scenario | `X-Mock-Scenario` | Development/test only; không thuộc Gateway contract. |

Browser JavaScript không đọc `oma_cart_context`, không biết upstream URL và không gọi Mockoon trực tiếp. Cơ chế BFF/Gateway identity propagation, ký/mã hóa cookie, binding Registered Customer và session rotation vẫn cần Security/Architecture phê duyệt.

## 4. Capability đề xuất

| ID | Method và Gateway path | Kết quả |
| --- | --- | --- |
| `CHK-C01` | `POST /checkout/sessions` | Validate owned selection, revalidate SKU/price/quantity và tạo server-owned Checkout snapshot. |
| `CHK-C02` | `POST /checkout/sessions/{checkoutSessionId}/shipping-quotes` | Server tính parcel, trả option/fee/ETA có TTL theo address. |
| `CHK-C03` | `POST /checkout/sessions/{checkoutSessionId}/orders` | Revalidate cuối, giữ toàn bộ selection nguyên tử và tạo một Order `UNPAID` theo idempotency key. |

Không đề xuất endpoint `/health`, bootstrap riêng, Cart internal projection, Mockoon scenario, Payment execution, Order history/detail hoặc reservation administration trong contract này.

## 5. Quyết định Candidate

1. Checkout hỗ trợ Guest và Registered Customer, luôn scope theo current cart/customer context.
2. `itemId[]` chỉ xuất hiện khi tạo Checkout session. Session server-side giữ selection; quote/confirm không nhận lại selection để tránh client thay đổi tập dòng ngoài snapshot.
3. Dòng không tồn tại và dòng thuộc context khác không được phân biệt trong response public; không lộ ownership.
4. Prepare revalidate current public price, quantity và purchase eligibility nhưng chưa reserve inventory.
5. Quote chỉ nhận address. Parcel count/weight được server dựng từ session/cart; browser không gửi weight hoặc fee.
6. Empty shipping options là response hợp lệ và không đồng nghĩa với phí `0`. Client không được tạo final total khi chưa có option hợp lệ.
7. Confirm dùng `Idempotency-Key` header, không để key trong business body. Retry kết quả không chắc chắn phải dùng lại key và payload tương đương.
8. Confirm revalidates toàn bộ selection, current price và shipping option ngay trước transaction. Conflict một dòng làm fail toàn bộ Order; không partial checkout.
9. Unit price/subtotal/shipping/discount/total là integer VND do server tính. Client không được gửi bất kỳ money snapshot nào.
10. Price change được hiển thị theo exact public line và cần acknowledgment. `priceRevalidatedAt` hiện là candidate concurrency evidence, chưa được coi là strong version token.
11. Registered Customer chỉ nhận saved-address projection thuộc current customer. Chỉnh form là chỉnh Order snapshot, không mutate Address Book.
12. Confirm thành công trả Order + reservation expiry và `paymentStatus=UNPAID`. Lựa chọn `BANK_TRANSFER`/`COD` chỉ là dữ liệu downstream; Checkout không charge và không trả `PAID`.
13. Promotion/voucher chưa thuộc MVP; `discountVnd=0`. Candidate giữ field để commercial breakdown rõ ràng nhưng không có endpoint/input promotion.
14. Exact stock, warehouse, Batch/Lot, cost/margin, internal reservation allocation và service topology không bao giờ public.

## 6. Phát hiện và sai khác có chủ đích với implementation local

| Implementation/Mockoon local | API Candidate |
| --- | --- |
| Browser gửi `itemIds` ở prepare, quote và confirm | `itemIds` chỉ dùng để tạo session; session server-side bind selection. |
| Browser body chứa `checkoutSessionId` | Session ID chuyển thành path resource. |
| Browser body chứa `idempotencyKey` | Gateway contract dùng required `Idempotency-Key` header. |
| BFF tự tạo `checkoutSessionId` trong memory-less flow | Backend phải persist/bind session hoặc cung cấp cơ chế tương đương. |
| BFF gọi Cart internal buckets và compose snapshot | Gateway/backend phải trả resource projection theo contract; storage/aggregation topology không public. |
| Mockoon `/checkout/bootstrap` | Saved address projection là một phần của create-session response. |
| Mockoon confirm nhận `commercialSnapshot`, shipping object và parcel | Candidate không nhận trusted-looking client snapshot; backend tự resolve/calculates. |
| `mockScenario`, `cartScenario`, `X-Mock-Scenario` | Không tồn tại trong production contract. |
| `/api/v1/health` | Không nằm trong business OpenAPI. |
| Mock order/time fixture cố định | Không phải production semantic. |

UI/BFF local chưa được đổi theo candidate vì contract vẫn **PROPOSED / NOT APPROVED**. Sau phê duyệt, implementation và Mockoon phải được đồng bộ thay vì coi candidate là tương thích ngược mặc định.

## 7. Error và transaction semantics cần giữ

- `409 CART_CHANGED` hoặc `CHECKOUT_ITEMS_UNAVAILABLE`: quay lại Cart/reprepare; item issue chỉ customer-safe, không exact stock.
- `409 PRICE_CHANGED` hoặc `PRICE_ACKNOWLEDGEMENT_REQUIRED`: hiển thị snapshot/tổng mới và lấy explicit acknowledgment.
- `409 SHIPPING_QUOTE_EXPIRED` hoặc `SHIPPING_OPTION_UNAVAILABLE`: quote lại; không tự dùng fee cũ.
- `409 IDEMPOTENCY_CONFLICT`: cùng key nhưng payload khác; không tạo Order thứ hai.
- `422 VALIDATION_ERROR`: field-level error cho recipient/address/payment/session data.
- `500 CHECKOUT_ERROR`: không được suy diễn là chưa tạo Order nếu transaction outcome không chắc chắn; retry cùng idempotency key hoặc query outcome theo quyết định Architecture.
- `503`: dependency unavailable; không hiển thị Order success giả.

Atomic boundary tối thiểu phải bảo đảm hoặc cùng tạo Order + reservation cho toàn selection, hoặc không commit outcome business nào. Cơ chế saga/outbox/transaction cụ thể không thuộc public contract nhưng phải được review.

## 8. Open questions

### Product/Business

1. Canonical administrative dataset/code và migration khi địa giới thay đổi là gì?
2. Required address fields, delivery instruction và phone normalization theo chuẩn nào?
3. COD có dùng `PENDING_PAYMENT/UNPAID` hay chuyển ngay sang `PLACED`; `nextStep` chính xác là gì?
4. Reservation TTL có khác giữa `BANK_TRANSFER` và `COD`; khách được thấy countdown nào?
5. Một dòng conflict luôn fail toàn checkout hay Product muốn hỗ trợ partial split sau MVP?
6. Giá Website đã gồm thuế chưa; shipping tax và rounding do service nào sở hữu?

### Backend/Architecture

7. Checkout session TTL, persistence, cleanup và binding với cart/customer/version được triển khai ra sao?
8. Raw `itemId[]`, Cart version hay signed selection token là handoff production cuối cùng?
9. `priceRevalidatedAt` có đủ hay cần opaque price/cart revision token chống TOCTOU?
10. Order-first hay reservation-first; rollback, outbox và orphan recovery hoạt động thế nào?
11. Idempotency key scope, retention, request hash và replay response/status là gì?
12. Sau Order thành công, selected lines được xóa khỏi Cart khi nào; unselected lines được giữ thế nào?
13. Khi client timeout sau confirm, API nào cho phép resolve outcome an toàn ngoài replay?

### Shipping/Inventory/Order/Payment

14. Shipping quote TTL, carrier timeout/retry, supported-area code và fee-change policy là gì?
15. Inventory service cung cấp atomic multi-line reservation API nào; release/expiry event do ai sở hữu?
16. Reservation gắn Order nào trước khi Payment; duplicate reservation được ngăn bằng key nào?
17. Order status machine và Payment handoff canonical cho BANK_TRANSFER/COD là gì?
18. `estimatedDelivery` nên là text hay structured time window/SLA?

### Security/Privacy

19. PII encryption, log redaction, retention và quyền truy cập recipient/address snapshot là gì?
20. CSRF/Origin, rate limit, anti-automation và fixation protection cho Guest Checkout là gì?
21. Checkout session/Order ID có đủ entropy và anti-enumeration; not-found timing/logging có lộ ownership không?
22. Idempotency key có được browser tạo hay BFF tạo; key có chứa PII/predictable identifier không?

## 9. Checklist review theo team

### Backend/Architecture

- Xác nhận resource path, persistence, session binding, versioning và status code.
- Xác nhận atomic Order/reservation boundary, rollback/recovery và idempotency replay.
- Xác nhận browser/BFF/Gateway mapping mà không public internal topology.

### Product/UX

- Xác nhận Guest/Registered flow, address requirements, price acknowledgment và shipping empty state.
- Xác nhận COD/BANK_TRANSFER status, reservation countdown và partial-conflict behavior.
- Xác nhận voucher/promotion tiếp tục nằm ngoài MVP.

### Inventory/Shipping/Order/Payment

- Xác nhận revalidation và atomic reservation không lộ exact stock/warehouse/Batch.
- Xác nhận quote TTL, fee/ETA format và final re-quote semantics.
- Xác nhận initial Order/payment status và handoff sang EPIC 07/08.

### Pricing/Finance

- Xác nhận active Website price, tax, rounding và `priceRevalidatedAt`/revision.
- Xác nhận mọi total server-side và `discountVnd=0` trong MVP.

### Security/Privacy/Gateway

- Xác nhận cart/customer identity propagation, CSRF, session rotation và anti-enumeration.
- Xác nhận PII encryption/redaction/retention, rate limit và audit.
- Chỉ allow method/path/body/header trong contract; không forward arbitrary URL/query/header.

### Frontend

- Sau Approved, sync generated types, route mapping, idempotency header và session-owned selection.
- Đổi upstream sang API Gateway và chạy regression/E2E/contract test.
- Không gửi browser-supplied price, parcel, fee, discount, total, reservation hoặc status.

## 10. Điều kiện chuyển sang Approved

- Backend/Product xác nhận route, session, selected-line handoff, address và conflict semantics.
- Architecture xác nhận persistence, version token, atomicity, recovery và idempotency.
- Inventory/Shipping/Order/Payment xác nhận reservation, quote, status machine và handoff boundary.
- Pricing/Finance xác nhận price/tax/rounding và server-calculated totals.
- Security/Privacy/Gateway xác nhận session/cookie, CSRF, PII, rate limit, logging và anti-enumeration.
- Mọi open question có quyết định được ghi vào OpenAPI/review log.
- Contract tests bao phủ Guest, Registered address projection, empty quote, quote expiry, cart changed, unavailable quantity, price changed/acknowledged, atomic reservation conflict, idempotent replay, key conflict, dependency failure và success `UNPAID`.
- OpenAPI parse thành công, toàn bộ internal `$ref` resolve và lint không còn lỗi blocking.
- Mockoon, frontend types/BFF và Backend implementation đồng bộ với contract đã review.
- E2E qua API Gateway chứng minh browser không nhận cart context, exact stock, warehouse, cost, Batch/Lot, internal allocation hoặc arbitrary customer address.

Cho đến khi hoàn tất, version giữ `0.1.0-candidate` và trạng thái **PROPOSED / NOT APPROVED**.
