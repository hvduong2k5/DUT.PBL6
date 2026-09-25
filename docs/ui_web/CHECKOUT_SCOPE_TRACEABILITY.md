# Checkout MVP — Scope & Traceability

## Mục tiêu MVP

Triển khai Desktop Web Checkout tại `/checkout` cho Guest Customer và Registered Customer theo `EPIC_06_Checkout`, từ các dòng khả dụng đã được chọn tại `/cart` đến khi:

1. thông tin người nhận và địa chỉ hợp lệ;
2. phí/phương thức vận chuyển được xác định từ địa chỉ và kiện hàng;
3. SKU, giá, số lượng và tổng tiền được server kiểm tra lại;
4. toàn bộ tồn được giữ nguyên tử và Order snapshot được tạo một lần;
5. Order dừng ở trạng thái `UNPAID`, sẵn sàng làm đầu vào cho EPIC 07 Payment.

UI tái sử dụng `SiteHeader`/`SiteFooter`, chỉ tối ưu Desktop (khuyến nghị 1366×768 và 1440×900), gọi same-origin BFF và không public upstream URL hay dữ liệu tồn kho nội bộ.

## Actor và user story được thực hiện

| Actor | User story | Phạm vi thực hiện |
| --- | --- | --- |
| Guest Customer | `US-CHK-01` | Nhập người nhận/địa chỉ và đặt hàng không cần đăng nhập hay tạo tài khoản. |
| Registered Customer | `US-CHK-01` | Scenario `registered-customer` trả đúng địa chỉ thuộc customer hiện tại; khách chọn, kiểm tra và chỉnh snapshot cho đơn mà không mutate Address Book. |
| Guest/Registered Customer | `US-CHK-02` | Tự động yêu cầu lại shipping quote khi địa chỉ thay đổi; hiển thị từng lựa chọn, phí và ETA; không có quote thì không tạo tổng cuối hoặc xác nhận. |
| Guest/Registered Customer | `US-CHK-03` | Confirm gửi server-calculated snapshot; mock mô phỏng giữ tồn nguyên tử, conflict một dòng và reservation expiry; thất bại không chuyển Payment. |
| Guest/Registered Customer | `US-CHK-04` | Hiển thị SKU/quy cách/giá/số lượng/line subtotal, người nhận, shipping và breakdown; revalidate trước confirm; dùng idempotency key; tạo Order `UNPAID`. |

## User story bị hoãn

- `US-CHK-05` — voucher/promotion: Giai đoạn 2, phụ thuộc EPIC 17. UI chỉ hiển thị “Không áp dụng”, không có input voucher và không fake discount.
- Payment transaction, VietQR, card gateway, webhook, trạng thái `PAID`: EPIC 07.
- Order history/detail/tracking và vòng đời sau tạo: EPIC 08.
- Thực thi reservation/release production, Batch/FEFO/exact stock: EPIC 09.
- Gifting, thiệp, ẩn giá, loyalty points và gift packaging trong Stitch: EPIC 17/26 hoặc quyết định nghiệp vụ khác, không thuộc Checkout MVP này.

## Route và màn hình

- `/cart`: integration point được cập nhật để chuyển `itemId` đã chọn sang Checkout; thay đổi này trực tiếp cần cho feature mới.
- `/checkout`: D2C-006, desktop UI gồm progress, receiver/address, shipping, payment integration choice, order review và confirmation boundary.
- `/api/checkout/prepare`: chuẩn bị/revalidate selection.
- `/api/checkout/shipping-quotes`: lấy lại phí theo address + parcel server-side.
- `/api/checkout/confirm`: revalidate, chọn quote hiện hành, tạo Order snapshot + reservation bằng idempotency key.

## Happy path

1. Khách tick một hoặc nhiều dòng khả dụng trong Cart và bấm “Thanh toán”.
2. Cart điều hướng `/checkout?items=...`; `cartScenario` chỉ được mang theo trong local development.
3. BFF lấy `oma_cart_context` HttpOnly, tải cart lines/SKU projection server-to-server và từ chối dòng không thuộc context, unavailable hoặc quantity không còn đáp ứng.
4. Guest nhập người nhận/địa chỉ; Registered Customer có thể sao chép địa chỉ đã lưu thuộc chính account mock.
5. Khi address hợp lệ, UI debounce và lấy shipping quote; đổi address làm quote/tổng được tính lại.
6. UI hiển thị item snapshot, subtotal, shipping, discount `0` và final total. Price changed bắt buộc acknowledgment.
7. Confirm bị khóa khi form/quote/acknowledgment chưa hợp lệ và có pending state chống double-click.
8. BFF revalidate cart lần cuối, tự lấy lại shipping option/fee, tự tính commercial snapshot và gọi upstream cùng idempotency key.
9. Kết quả tạo Order + reservation hiển thị `UNPAID`; không giả Payment thành công hoặc route Order Detail chưa triển khai.

## UI state bắt buộc

- Loading skeleton khi prepare và inline loading khi tính shipping/confirm.
- No selection: CTA về Cart.
- Prepare/upstream error: retry + về Cart.
- Field validation: full name, phone, optional email, province/locality/address line.
- Registered saved-address state; edit chỉ tác động snapshot local.
- Shipping empty/unavailable, server error và slow.
- Price changed + explicit acknowledgment.
- Cart changed, line unavailable/quantity conflict và confirm price conflict: không tạo Order, hướng về Cart/reprepare.
- Confirm server error: giữ form context, cho retry với idempotency key mới khi kết quả là lỗi xác định.
- Success: Order/reservation summary, `UNPAID`, Payment boundary rõ ràng.

## Business decisions

1. **Selection transport:** Checkout nhận danh sách opaque `itemId` đã chọn. BFF đối chiếu ownership bằng HttpOnly cart context và không tin selection là hợp lệ cho đến khi revalidate.
2. **One address per Order:** MVP dùng một người nhận và một địa chỉ cho toàn bộ Order; multi-address/pickup ngoài phạm vi.
3. **Administrative fixture:** local UI chỉ có một tập địa danh minh họa hữu hạn để kiểm thử. Nguồn danh mục, mã chuẩn và xử lý địa giới phải được Backend/Business chốt.
4. **Saved addresses:** bootstrap chỉ trả projection địa chỉ của current customer. Không có endpoint lấy address tùy ý bằng ID; snapshot edit không mutate Customer Profile.
5. **Shipping quote:** BFF tự tạo parcel từ cart lines; browser không gửi weight/fee. Quote được tính lại khi address thay đổi; không quote hoặc option stale thì confirm bị chặn.
6. **Money:** VND integer; subtotal/fee/discount/total do BFF/upstream kiểm tra, không nhận money field từ browser. MVP không áp dụng promotion nên `discountVnd=0`.
7. **Price change:** prepare trả current price và old price public khi changed; UI yêu cầu acknowledgment. Confirm vẫn revalidate và có thể trả `PRICE_CHANGED`.
8. **Reservation:** success mô phỏng toàn bộ selection được giữ nguyên tử và liên kết Order; conflict bất kỳ dòng làm toàn checkout thất bại. Exact inventory/warehouse/Batch không public.
9. **Idempotency:** browser tạo một key ổn định cho một confirm attempt; double-click bị UI khóa và BFF chuyển key server-to-server. Retry giữ nguyên key để an toàn cả khi kết quả trước không chắc chắn; contract chính thức cần chốt retention/replay.
10. **Payment boundary:** lựa chọn `BANK_TRANSFER`/`COD` chỉ là dữ liệu chuyển sang Payment/Order. Checkout không tạo QR, charge, webhook hay `PAID` state.
11. **Order boundary:** mock trả Order identifier và reservation expiry để chứng minh Checkout outcome. Không tạo Order History/Detail route vì thuộc EPIC 08.
12. **Stitch filtering:** giữ cấu trúc nhiều panel + sticky summary và ngôn ngữ thị giác; bỏ voucher, gift, loyalty, VAT/insurance claim và trust claim chưa có nguồn Epic phù hợp.

## Ngoài phạm vi

- Mobile layout, mobile breakpoint hoặc mobile-specific component.
- Voucher/promotion/free-shipping campaign/loyalty.
- Gift recipient, gift message, gift packaging, hidden-price delivery note.
- Payment processing, QR/card SDK, webhook, bank polling hoặc `PAID`.
- Order management, guest order lookup, tracking, cancellation/return.
- Exact inventory, warehouse, Batch/Lot, manufacturing/expiry date.
- Multi-address, store pickup, delivery instruction, address normalization/geocoding.
- Persist Checkout session/reservation thật trong browser hoặc Mockoon database.

## Dependency

| Dependency | Dữ liệu/quyết định sử dụng |
| --- | --- |
| EPIC 01/02 | Guest boundary và projection địa chỉ current customer; không truy cập chéo. |
| EPIC 04/05 | Exact SKU, current Website price, cart ownership, selected quantity và availability summary. |
| EPIC 07 | Tiếp nhận Order/payment method; Checkout không tự xử lý tiền. |
| EPIC 08 | Tạo/lưu Order snapshot và status khởi đầu. |
| EPIC 09 | Atomic reservation, release/expiry và oversell prevention. |
| EPIC 11 | Shipping option, fee, ETA và quote validity. |
| EPIC 17 | Promotion/discount về sau; MVP là `0`. |
| EPIC 23 | Audit cho confirm/order/reservation/idempotency. |

## Mapping requirement → UI state → mock capability

| Requirement | UI state | Mock capability/scenario |
| --- | --- | --- |
| `FR-CHK-01`, `US-CHK-01` | Cart selection → prepare; Guest không login | `CHK-C01`, cart `cart-prefilled` |
| `FR-CHK-02` | Field error, disabled confirm | BFF validation `422` |
| `FR-CHK-03` | Saved-address select + editable snapshot | `registered-customer` |
| `FR-CHK-04` | Current price/availability revalidation, line issues | Cart projection; `price-changed`, `cart-unavailable`, `confirm-price-changed` |
| `FR-CHK-05..06` | Auto quote/requote, loading/error/empty | `CHK-C02`; `shipping-slow`, `shipping-error`, `shipping-unavailable` |
| `FR-CHK-07` | Subtotal + shipping + discount `0` + total | BFF server calculation |
| `FR-CHK-08..10` | Pending confirm, atomic conflict, success snapshot | `CHK-C03`; `confirm-stock-conflict`, default success |
| `FR-CHK-11` | Stable per-attempt idempotency key, button locked | `Idempotency-Key` server-to-server; fixed replay example |
| `FR-CHK-12` | Order `UNPAID`, explicit Payment boundary | success/`confirm-slow`; no Payment call |
| Resilience | Prepare/quote/confirm retry states | `shipping-error`, `confirm-error`, BFF `503` |

## Câu hỏi cần Backend/Architecture/Security/Business xác nhận

1. Canonical administrative dataset/code và lịch cập nhật khi địa giới thay đổi là gì?
2. Required address fields, phone normalization và khả năng hỗ trợ delivery instruction?
3. Shipping quote TTL, supported-area response, carrier timeout/retry và fee-change acknowledgment?
4. Order-first hay reservation-first; cơ chế rollback tránh orphan Order/reservation?
5. Reservation TTL theo `BANK_TRANSFER`/`COD`, thời điểm bắt đầu và release rules?
6. COD khởi tạo Order/status/reservation ra sao; có thật sự dùng `PENDING_PAYMENT/UNPAID` không?
7. Một dòng thiếu tồn làm fail toàn Order hay được phép partial checkout/tách đơn?
8. Idempotency key scope, retention, replay response và binding với customer/cart/session?
9. Cart version/ETag hoặc selection token nào thay cho raw `itemId[]` ở API production?
10. Price/quote acknowledgment cần token/version hay timestamp nào để tránh TOCTOU?
11. Thuế đã nằm trong unit price chưa; rounding và shipping tax được tính ở service nào?
12. PII encryption/log redaction/retention và audit boundary cho Guest checkout?
