# Shopping Cart MVP — Scope & Traceability

## Mục tiêu MVP

Triển khai lát cắt customer-facing của EPIC 05 cho Guest Customer và Registered Customer tại `/cart`: thêm đúng SKU từ Product Detail, xem giỏ, đổi số lượng, xóa dòng, nhận biết thay đổi giá/khả dụng và hiểu subtotal trước Checkout. UI sử dụng cùng `SiteHeader`/`SiteFooter`, gọi same-origin BFF và dừng trước mọi nghiệp vụ tạo Order, giữ tồn, vận chuyển, ưu đãi hoặc thanh toán.

Mockoon là executable example phục vụ UI, không phải contract cuối cùng. API contract chỉ được đề xuất sau checkpoint duyệt UI.

## Actor và user story

| Actor | User story | Phạm vi thực hiện |
| --- | --- | --- |
| Guest Customer, Registered Customer | `US-CART-01` | Thêm đúng `skuId` và số lượng dương; SKU trùng được gộp số lượng; SKU không khả dụng bị từ chối. |
| Guest Customer, Registered Customer | `US-CART-02` | Đổi số lượng; revalidate giá và khả năng mua; lỗi giữ nguyên context của dòng để khách sửa. |
| Guest Customer, Registered Customer | `US-CART-03` | Xóa dòng khỏi giỏ; khi hết dòng hiển thị empty state và lối quay lại catalog. |
| Guest Customer, Registered Customer | `US-CART-04` | Hiển thị đơn giá hiện tại, số lượng, line subtotal và cart subtotal; cho chọn một hoặc nhiều dòng để chuẩn bị Checkout và ghi rõ đây chưa phải tổng Checkout. |

## User story bị hoãn hoặc chỉ là dependency

- Guest cart merge sau đăng nhập, TTL chính thức, đồng bộ đa thiết bị và conflict policy: EPIC 05 để Backend/Architecture chốt; không tự triển khai merge.
- Checkout, địa chỉ nhận hàng, shipping quote, reservation và tạo Order: EPIC 06.
- Payment method, payment status và payment security claims: EPIC 07.
- Exact inventory, warehouse, Batch/Lot, FEFO và actual expiry: EPIC 09.
- Voucher, promotion, combo, loyalty và free-shipping threshold: EPIC 17.
- Gift message/packing option: thuộc flow Checkout/Gifting, không thuộc Cart MVP hiện tại.

## Màn hình và route

- Cart: `/cart`.
- Product Detail `/products/[slug]`: thay integration placeholder bằng thao tác Add to Cart thật sau khi khách chọn SKU; mặc định quantity là `1` và có control rõ ràng.
- Shared header: thêm link `/cart`, không fake cart badge/count khi chưa có bootstrap toàn cục.
- Product Discovery `/products` và các query search/filter/sort/pagination giữ nguyên.

## Happy path và UI state

1. Khách chọn SKU khả dụng tại Product Detail, chọn quantity và bấm Add to Cart.
2. Browser gửi `POST /api/cart/items`; BFF lấy cart context từ cookie HttpOnly, validate body và gọi upstream cố định.
3. Nếu cùng SKU đã có trong cart, upstream/BFF gộp số lượng thay vì tạo dòng trùng.
4. UI báo thành công và cung cấp link đến `/cart`.
5. Cart tải các dòng đã revalidate, hiển thị product/SKU/đơn giá/quantity/line subtotal.
6. Khách dùng checkbox bên trái ảnh để chọn một, nhiều hoặc tất cả dòng khả dụng; khối tóm tắt chỉ cộng số lượng và subtotal của phần đang chọn.
7. Khách tăng, giảm hoặc nhập quantity hợp lệ; response mới thay thế snapshot cart và subtotal nhưng giữ các lựa chọn còn hợp lệ.
8. Khách xóa dòng; empty state xuất hiện khi không còn item.

Các state bắt buộc:

- Loading skeleton có `aria-busy`.
- Empty cart có CTA về `/products`.
- Error/upstream unavailable có retry và không giả mất cart.
- Quantity invalid bị chặn tại client/BFF; quantity vượt khả năng mua trả line-level error, không public exact stock.
- SKU unavailable hoặc paused vẫn giữ trong cart để giải thích, disabled checkbox/quantity, có link mở Product Detail/chọn phương án khác và bị loại khỏi phần chuẩn bị Checkout.
- Price changed hiển thị giá cũ và giá hiện tại; subtotal dùng giá hiện tại.
- Partial invalid cart vẫn hiển thị các dòng hợp lệ, không biến thành lỗi toàn trang.
- Checkout integration point giải thích feature kế tiếp chưa hoạt động và không phát request Checkout.

## Quyết định nghiệp vụ

1. **Cart context:** Guest dùng opaque cookie `oma_cart_context`, `HttpOnly`, `SameSite=Lax`; JavaScript không đọc cart ID. Registered Customer vẫn dùng cùng UI, nhưng mapping session/account chính thức chờ Backend/Auth review.
2. **Ownership:** mọi mutation kiểm tra dòng thuộc cart context hiện tại; item của context khác trả cùng semantics `404` để không lộ dữ liệu.
3. **Duplicate SKU:** thêm cùng `skuId` cộng quantity vào một dòng rồi revalidate; không tạo hai dòng song song.
4. **Quantity:** integer dương. Local MVP đặt guardrail `1..20` mỗi dòng để validation/UI hữu hạn; giới hạn nghiệp vụ chính thức và policy vượt khả năng mua vẫn là open question. Response lỗi không tiết lộ exact inventory.
5. **Availability:** dòng không còn mua được không bị xóa âm thầm. UI giữ dòng, hiển thị public reason, disabled control và đặt `hasBlockingIssues=true`.
6. **Subtotal:** dòng unavailable không đóng góp vào cart subtotal; line subtotal của dòng đó là `null`. Label nói rõ subtotal chỉ gồm hàng đang có thể mua, chưa gồm phí/giảm giá/thuế được quyết định ở Checkout.
7. **Price revalidation:** `unitPriceVnd` luôn là giá bán Website hiện tại. Khi khác giá lúc thêm, trả `previousUnitPriceVnd` và notice; UI không tự tính promotion.
8. **No reservation:** giữ item trong cart không giữ tồn, không bảo đảm giá và không chọn Batch. Cart sẽ tiếp tục được revalidate trước Checkout.
9. **Remove:** thực hiện trực tiếp theo action rõ ràng; không thêm confirm modal hay undo vì đây chỉ là UX recommendation, không phải requirement bắt buộc.
10. **Purchase selection:** mặc định chọn tất cả dòng đang khả dụng. Checkbox của từng dòng nằm bên trái ảnh; khách có thể bỏ/chọn lại độc lập hoặc dùng “Chọn tất cả”. Dòng unavailable luôn disabled và không được chọn. Selection là UI state tạm thời của trang hiện tại, không làm thay đổi Cart API và chưa được persist qua reload/session.
11. **Selected subtotal:** cart subtotal vẫn mô tả toàn bộ hàng khả dụng trong giỏ; khối “Tạm tính hàng đã chọn” chỉ cộng các dòng được tick và là giá trị dùng cho integration point Checkout.
12. **Checkout:** button giữ vai trò integration point và chỉ enabled khi có ít nhất một dòng khả dụng được chọn. Click chỉ thông báo Checkout thuộc feature tiếp theo, không điều hướng tới route chưa triển khai và không tạo Order/reservation/payment.
13. **Stitch filtering:** giữ bố cục cart lines + sticky summary và ngôn ngữ thị giác. Không triển khai free-shipping progress, combo discount, voucher, shipping fee, final total, VAT claim, payment badges, gift note hoặc FEFO/trust claims vì thuộc Epic khác hoặc chưa có nguồn thật.

## Ngoài phạm vi

- Shipping fee, delivery promise, free-shipping eligibility.
- Voucher, coupon, promotion, combo, loyalty points.
- Tax/final payable amount và payment methods.
- Checkout, Order, inventory reservation, Batch allocation.
- Guest/account cart merge, multi-device sync và cross-browser cart recovery.
- Mini-cart toàn cục, live badge count và notification infrastructure.
- Recommendation, related products hoặc replacement SKU suggestion engine.

## Dependency

| Dependency | Dữ liệu/quyết định cần |
| --- | --- |
| EPIC 04 | Stable `skuId`, public product/SKU projection, current Website price và availability summary. |
| EPIC 01/02 | Session boundary cho Registered Customer mà không lộ token cho browser. |
| EPIC 06 | Checkout nhận cart snapshot hợp lệ và revalidate trước reservation/order. |
| EPIC 09 | Nguồn khả dụng tổng hợp; exact stock/warehouse/Batch không public. |
| EPIC 17 | Promotion/voucher/free-shipping breakdown, hoàn toàn không tính ở Cart MVP. |
| Backend/Architecture/Security | Cart ownership, TTL, merge, concurrency/idempotency, cookie/CSRF policy. |

## Mapping requirement → UI state → mock capability

| Requirement | UI state | Mock capability/scenario |
| --- | --- | --- |
| `US-CART-01`, `FR-CART-01` | Add exact SKU từ Product Detail; duplicate merge | `CART-C02` + stateful data bucket |
| `US-CART-01`, `FR-CART-02` | Reject unavailable SKU | `sku-unavailable` |
| `US-CART-02`, `FR-CART-05` | Update quantity và subtotal | `CART-C03` |
| `US-CART-02`, `FR-CART-06` | Quantity vượt khả năng mua, giữ dòng | `quantity-unavailable`/internal purchasability guard |
| `US-CART-02`, `FR-CART-07` | Price/availability changed | `price-changed`, `cart-unavailable` |
| `US-CART-03`, `FR-CART-08` | Remove và empty state | `CART-C04`, `cart-empty` |
| `US-CART-04`, `FR-CART-09..12` | Unit price, quantity, line/cart subtotal | `CART-C01` default/`cart-prefilled` |
| Chuẩn bị Checkout từng phần | Checkbox mỗi dòng, chọn tất cả, selected item count/subtotal; unavailable disabled | UI-derived từ public cart projection; `cart-prefilled`, `cart-unavailable` |
| Ownership | Không đọc/sửa cart context khác | BFF cookie context + `404 CART_ITEM_NOT_FOUND` |
| UI resilience | Loading, server error, retry, upstream unavailable | `cart-slow`, `cart-error`, BFF `503` |

## Câu hỏi cần Backend/Architecture/Security/Business xác nhận

1. Cart của Guest có TTL bao lâu, lưu bằng server-side cart ID hay signed/encrypted cookie; cookie/CSRF policy chính thức là gì?
2. Khi Guest đăng nhập, cart Guest và cart account được replace, merge hay yêu cầu khách review conflict?
3. Giới hạn quantity mỗi dòng/cart là bao nhiêu; error có được public `maxAllowed` hay chỉ trả message trung tính?
4. Khi SKU unavailable, subtotal nên loại dòng như candidate này hay vẫn giữ intended subtotal riêng?
5. Price change có cần khách acknowledge trước Checkout; có public old price và effective time không?
6. Add/update cần idempotency key hoặc cart version/ETag nào để tránh duplicate và lost update?
7. Public unavailable reason code chuẩn hóa gồm những giá trị nào; text thuộc Backend hay Frontend?
8. Cart của Registered Customer có đồng bộ đa thiết bị trong MVP hay chỉ theo session hiện tại?
9. Giá Website trong Cart đã gồm thuế nào; phần tax/final total chỉ bắt đầu ở Checkout ra sao?
10. Checkout API sẽ nhận danh sách `itemId`, một selection token, hay server-side selected state; cần cơ chế cart version nào để revalidate tập dòng đã chọn mà không xử lý nhầm toàn bộ giỏ?
11. Lựa chọn dòng có cần persist giữa reload/thiết bị hay chỉ được truyền tại thời điểm bắt đầu Checkout như candidate UI hiện tại?
