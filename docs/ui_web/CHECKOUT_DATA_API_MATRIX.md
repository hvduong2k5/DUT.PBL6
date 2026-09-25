# Checkout MVP — Data & Mock API Matrix

## Capability inventory

Browser chỉ gọi Next.js same-origin BFF. Cart/Checkout upstream URL, cart context, internal availability, parcel calculation và service topology chỉ tồn tại server-side.

| Capability ID | Method | Browser/BFF route | Mockoon executable route | Mục đích |
| --- | --- | --- | --- | --- |
| `CHK-C01` | POST | `/api/checkout/prepare` | GET `/checkout/bootstrap` + Cart internal projection | Validate selection/ownership, revalidate SKU/price/quantity, lấy current-customer address projection. |
| `CHK-C02` | POST | `/api/checkout/shipping-quotes` | POST `/checkout/shipping-quotes` | BFF tự tạo parcel từ cart và lấy options/fee/ETA theo address. |
| `CHK-C03` | POST | `/api/checkout/confirm` | POST `/checkout/confirm` | Revalidate lần cuối, refresh selected shipping fee, tạo server-side snapshot, reservation và Order idempotent. |
| Local health | GET | Không expose qua BFF | `/health` | Kiểm tra Mockoon local, không thuộc contract. |

Cart upstream `4014` chỉ được BFF dùng để đọc internal cart lines và current SKU projections. Browser không gọi route Mockoon hoặc CRUD syntax.

## Request, query và validation

### `POST /api/checkout/prepare`

```json
{ "itemIds": ["line-prefilled-cake", "line-prefilled-tea"] }
```

- Body chỉ có `itemIds`, 1..20 opaque identifiers, unique, `^[A-Za-z0-9-]{1,80}$`.
- BFF lấy `oma_cart_context` HttpOnly và từ chối item không thuộc context bằng semantics không tiết lộ owner khác.
- Query production không được hỗ trợ. `mockScenario` và `cartScenario` chỉ khả dụng ngoài production để kiểm thử local.

### `POST /api/checkout/shipping-quotes`

```json
{
  "checkoutSessionId": "checkout-123e4567-e89b-12d3-a456-426614174000",
  "itemIds": ["line-prefilled-cake"],
  "address": {
    "provinceCode": "HUE",
    "provinceName": "Thành phố Huế",
    "districtCode": "PHU-NHUAN",
    "districtName": "Phường Phú Nhuận",
    "addressLine": "Tầng 3, 84 Nguyễn Huệ"
  }
}
```

- BFF revalidate item và tự tính `itemCount`/`totalWeightGrams`; browser không gửi parcel, price, stock hoặc fee.
- `addressLine` 8..250; code/name bounded và exact object keys.
- Đổi address làm UI bỏ quote cũ và request lại sau debounce.

### `POST /api/checkout/confirm`

```json
{
  "checkoutSessionId": "checkout-123e4567-e89b-12d3-a456-426614174000",
  "itemIds": ["line-prefilled-cake"],
  "recipient": { "fullName": "Nguyễn Văn An", "phone": "0914288668", "email": "an@example.test" },
  "address": {
    "provinceCode": "HUE",
    "provinceName": "Thành phố Huế",
    "districtCode": "PHU-NHUAN",
    "districtName": "Phường Phú Nhuận",
    "addressLine": "Tầng 3, 84 Nguyễn Huệ"
  },
  "shippingOptionId": "STANDARD",
  "paymentMethod": "BANK_TRANSFER",
  "idempotencyKey": "checkout-123e4567-e89b-12d3-a456-426614174000",
  "priceRevalidatedAt": "2026-09-25T10:00:00.000Z",
  "priceChangesAcknowledged": false
}
```

- Full name 2..100; Việt Nam phone candidate; email optional và <=254.
- Payment method allowlist: `BANK_TRANSFER`, `COD`; đây chỉ là lựa chọn cho integration, không xử lý Payment.
- Nếu selection có giá thay đổi, `priceChangesAcknowledged` phải là `true`; BFF vẫn revalidate current price và không dùng cờ này để bỏ qua kiểm tra.
- Không chấp nhận browser-supplied unit price/subtotal/shipping fee/discount/total/status/reservation.
- BFF revalidate cart, gọi lại quote để resolve `shippingOptionId`, tạo snapshot và chuyển idempotency key server-to-server.

## Public response

### Prepare

```json
{
  "checkoutSessionId": "checkout-123e4567-e89b-12d3-a456-426614174000",
  "items": [{
    "itemId": "line-prefilled-cake",
    "productSlug": "banh-ngu-sac-cung-dinh",
    "productName": "Bánh Ngũ Sắc Cung Đình",
    "imageUrl": "https://...",
    "imageAlt": "Hộp Bánh Ngũ Sắc Cung Đình",
    "skuId": "SKU-001-250-BOX",
    "skuLabel": "Hộp Thượng Hạng 250g",
    "weightGrams": 250,
    "flavor": "Ngũ sắc truyền thống",
    "packageType": "Hộp giấy 16 viên",
    "unitPriceVnd": 185000,
    "quantity": 2,
    "lineSubtotalVnd": 370000,
    "priceChanged": false,
    "previousUnitPriceVnd": null
  }],
  "itemCount": 2,
  "subtotalVnd": 370000,
  "priceRevalidatedAt": "2026-09-25T10:00:00.000Z",
  "requiresPriceAcknowledgement": false,
  "notices": [],
  "customerMode": "GUEST",
  "savedAddresses": []
}
```

Registered scenario chỉ public địa chỉ của current customer. Không public `customerId`, auth token hoặc address của account khác.

### Shipping quote

```json
{
  "options": [{
    "shippingOptionId": "STANDARD",
    "name": "Giao hàng tiêu chuẩn toàn quốc",
    "description": "Bảo quản và đóng gói chống va đập",
    "feeVnd": 0,
    "estimatedDelivery": "2–3 ngày"
  }],
  "quotedAt": "2030-09-25T10:00:00.000Z",
  "expiresAt": "2030-09-25T10:15:00.000Z"
}
```

Empty `options` là supported response và UI không hiển thị total như final.

### Confirmation

```json
{
  "orderId": "order-mock-20260925-001",
  "orderNumber": "OMA-260925-001",
  "status": "PENDING_PAYMENT",
  "paymentStatus": "UNPAID",
  "paymentMethod": "BANK_TRANSFER",
  "reservationExpiresAt": "2030-09-25T12:30:00.000Z",
  "subtotalVnd": 665000,
  "shippingFeeVnd": 0,
  "discountVnd": 0,
  "totalVnd": 665000,
  "nextStep": "PAYMENT_REQUIRED",
  "message": "Đơn đã được tạo và hàng đã được tạm giữ."
}
```

- VND và gram là integer.
- Confirmation không bao giờ trả `PAID` trong Checkout.
- Totals/payment method được BFF project từ revalidated data, không pass-through client hoặc arbitrary upstream fields.

## HTTP/error matrix

| HTTP | Code | Khi nào | UI state |
| --- | --- | --- | --- |
| `200` | — | Prepare/quote thành công | Form/summary hoặc shipping options. |
| `201` | — | Confirm idempotent thành công | Order + reservation + `UNPAID` boundary. |
| `400` | `INVALID_REQUEST` | Query/path/body shape sai | Reject locally. |
| `404` | `NOT_FOUND` | Path ngoài allowlist | Không proxy upstream. |
| `409` | `CART_CHANGED` | Item mất/không thuộc cart context | Quay lại Cart. |
| `409` | `CHECKOUT_ITEMS_UNAVAILABLE` | SKU/quantity không đáp ứng | Item issue public, không exact stock. |
| `409` | `PRICE_CHANGED` | Giá đổi tại final check | Reprepare và xác nhận lại tổng. |
| `409` | `PRICE_ACKNOWLEDGEMENT_REQUIRED` | Giá đã đổi nhưng khách chưa đồng ý | Yêu cầu acknowledgment. |
| `409` | `SHIPPING_OPTION_UNAVAILABLE` | Option stale/mất | Tính lại phí. |
| `409` | `SHIPPING_QUOTE_EXPIRED` | Quote hết TTL | Tính lại phí trước confirm. |
| `422` | `VALIDATION_ERROR` | Recipient/address/session/payment invalid | Map field error, không confirm. |
| `500` | `CHECKOUT_ERROR` | Tạo Order/reservation thất bại | Error + retry, không success giả. |
| `503` | `SHIPPING_QUOTE_UNAVAILABLE` | Không xác định được phí | Quote error + retry; không final total. |
| `503` | `CHECKOUT_UPSTREAM_UNAVAILABLE` | Port 4014/4015 không sẵn sàng | Page/form error + hướng dẫn local. |

## Mock scenarios

`X-Mock-Scenario` chỉ được BFF forward trong local/test. UI dùng `?mockScenario=`; Cart fixture dùng riêng `cartScenario=cart-prefilled`.

| Scenario | Capability | HTTP/state |
| --- | --- | --- |
| mặc định | bootstrap/quote/confirm | Guest, 2 shipping options, Order `UNPAID`. |
| `registered-customer` | bootstrap | Một địa chỉ current customer, editable copy. |
| `shipping-unavailable` | quote | `200`, `options=[]`. |
| `shipping-error` | quote | `503`, retry state. |
| `shipping-slow` | quote | `200` sau khoảng 3 giây. |
| `confirm-stock-conflict` | confirm | `409`, line issue, không Order success. |
| `confirm-price-changed` | confirm | `409`, yêu cầu prepare/review lại. |
| `confirm-error` | confirm | `500`, generic retry state. |
| `confirm-slow` | confirm | `201` sau khoảng 3 giây, duplicate-submit lock. |

Cart scenario phối hợp: `cart-prefilled`, `price-changed`, `cart-unavailable`, `cart-empty` từ port 4014.

## Browser → BFF → upstream mapping

| Browser | BFF validation/transform | Mockoon/local dependency |
| --- | --- | --- |
| Prepare `{itemIds}` | Resolve HttpOnly context; ownership + availability + current price; create opaque session; strip internal caps/context | Cart 4014 internal projection + Checkout 4015 bootstrap |
| Quote `{session,itemIds,address}` | Revalidate; compute parcel from server line weight × quantity | Checkout 4015 shipping quote |
| Confirm identifiers + recipient/address | Revalidate; re-quote option; calculate all money; send snapshot + idempotency key | Checkout 4015 confirm |
| Arbitrary path/query/body money | Reject locally | Không gọi upstream |
| `mockScenario`/`cartScenario` in production | Reject locally | Không forward |

## Public, mock/internal và forbidden data

| Phân loại | Field/dữ liệu |
| --- | --- |
| Public | SKU display snapshot, current/previous price when changed, quantity, line/subtotal, current-customer saved address projection, shipping option/fee/ETA, final total, Order number/status `UNPAID`, reservation expiry. |
| Mock/internal | `contextId`, `maxPurchasableQuantity`, Mockoon scenario/header, parcel internals, fixed order fixture, service orchestration. |
| Không được lộ browser | Exact stock, warehouse, cost/margin, Batch/Lot/actual expiry, internal availability/status reason, other-customer address, auth token, upstream URL, service topology, distributed lock/audit internals. |

## UI validation và behavior

- Confirm không chạy nếu receiver/address/shipping/price acknowledgment thiếu.
- Quote cũ bị bỏ ngay khi address không còn hợp lệ; total hiển thị `—` cho đến khi có quote.
- Shipping error/empty không được biến thành fee `0` hoặc final total.
- Price changed hiển thị exact affected public line và cần checkbox acknowledgment.
- Pending states khóa confirm; cùng attempt giữ idempotency key.
- Mọi confirm response lỗi không hiển thị Order success.
- Success ghi rõ Payment chưa triển khai và Order chưa Paid.
