# Shopping Cart MVP — Data & Mock API Matrix

## Capability inventory

Browser chỉ gọi Next.js same-origin BFF. BFF giữ upstream URL và opaque cart context ở phía server/cookie HttpOnly.

| Capability ID | Method | Browser/BFF route | Mockoon executable route | Mục đích |
| --- | --- | --- | --- | --- |
| `CART-C01` | GET | `/api/cart` | `/api/v1/internal/cart-lines` + `/api/v1/internal/cart-sku-projections` | Lấy cart hiện tại, revalidate và tạo public projection. |
| `CART-C02` | POST | `/api/cart/items` | CRUD internal lines/projections | Thêm exact SKU; merge duplicate. |
| `CART-C03` | PATCH | `/api/cart/items/{itemId}` | CRUD internal lines/projections | Đổi quantity và revalidate. |
| `CART-C04` | DELETE | `/api/cart/items/{itemId}` | CRUD internal lines | Xóa dòng thuộc cart hiện tại. |
| `CART-UI01` | Không gọi API | UI state tại `/cart` | Không có route | Chọn/bỏ chọn các dòng khả dụng và tính subtotal chuẩn bị Checkout; không mutate cart. |
| Local health | GET | Không expose qua BFF | `/api/v1/health` | Kiểm tra Mockoon local; không thuộc API contract. |

Mockoon dùng hai data bucket nội bộ để có mutation stateful. Đây là adapter local, không phải route candidate cho API Gateway. Contract sau UI approval sẽ mô tả API Cart public, không sao chép `/internal/*` hay cú pháp CRUD query của Mockoon.

## Request và validation

### `POST /api/cart/items`

```json
{ "skuId": "SKU-001-250-BOX", "quantity": 1 }
```

- Body phải là JSON object, chỉ có `skuId` và `quantity`.
- `skuId`: `^[A-Z0-9][A-Z0-9-]{1,63}$`.
- `quantity`: integer `1..20` theo guardrail candidate.
- Duplicate `skuId` trong cùng cart context được cộng số lượng và revalidate.

### `PATCH /api/cart/items/{itemId}`

```json
{ "quantity": 3 }
```

- Path chính xác `items/{itemId}`, không cho nested/arbitrary path.
- `itemId`: opaque token `^[A-Za-z0-9-]{1,80}$`.
- Body chỉ có `quantity`, integer `1..20`.
- Quantity `0` không có nghĩa là delete; dùng `DELETE` rõ ràng.

### `GET` và `DELETE`

- `GET /api/cart` không có body.
- `DELETE /api/cart/items/{itemId}` không có body.
- Query production không được chấp nhận. `mockScenario` chỉ dùng development/test và được chuyển thành `X-Mock-Scenario`.
- Method/path ngoài allowlist trả `404`; body/query sai trả `400/422` trước khi gọi upstream.

## Public response

```json
{
  "items": [
    {
      "itemId": "3bb6f4d1-23d2-4cc9-b28e-6da8af9ffab0",
      "productId": "PRD-001",
      "productSlug": "banh-ngu-sac-cung-dinh",
      "productName": "Bánh Ngũ Sắc Cung Đình",
      "imageUrl": "https://...",
      "imageAlt": "Hộp bánh ngũ sắc",
      "skuId": "SKU-001-250-BOX",
      "skuLabel": "Hộp Thượng Hạng 250g",
      "weightGrams": 250,
      "flavor": "Ngũ sắc truyền thống",
      "packageType": "Hộp giấy 16 viên",
      "unitPriceVnd": 185000,
      "previousUnitPriceVnd": null,
      "priceChanged": false,
      "quantity": 2,
      "lineSubtotalVnd": 370000,
      "isAvailable": true,
      "unavailableReason": null
    }
  ],
  "itemCount": 2,
  "subtotalVnd": 370000,
  "hasBlockingIssues": false,
  "notices": [],
  "updatedAt": "2026-09-25T10:00:00.000Z"
}
```

Rules:

- VND và gram là integer.
- `itemCount` là tổng quantity của mọi dòng còn trong cart, kể cả dòng đang cần xử lý.
- `lineSubtotalVnd = unitPriceVnd * quantity` khi available; unavailable trả `null`.
- `subtotalVnd` chỉ cộng các line subtotal khả dụng.
- `previousUnitPriceVnd` chỉ có khi khác current price lúc revalidation.
- `notices` chỉ chứa message public/actionable, không chứa inventory/warehouse/Batch detail.

## HTTP/error matrix

| HTTP | Code | Khi nào | UI state |
| --- | --- | --- | --- |
| `200` | — | GET/PATCH/DELETE thành công | Cart mới nhất hoặc empty. |
| `201` | — | Tạo dòng mới thành công | Product Detail add success. |
| `400` | `INVALID_REQUEST` | JSON/body/query/path shape sai | Field/form error, không gọi upstream. |
| `404` | `CART_ITEM_NOT_FOUND` | Item không tồn tại hoặc không thuộc cart context | Báo dòng đã thay đổi, refresh cart. |
| `404` | `SKU_NOT_FOUND` | SKU không public/không tồn tại | Add error trung tính. |
| `409` | `SKU_UNAVAILABLE` | SKU không còn mua được | PDP/cart line warning; không mutate. |
| `409` | `QUANTITY_UNAVAILABLE` | Quantity vượt khả năng mua hiện tại | Line-level error; không public exact stock. |
| `422` | `INVALID_QUANTITY` | Quantity ngoài guardrail/integer | Field/line validation. |
| `500` | `CART_ERROR` | Mock/API Cart lỗi | Error + retry, giữ UI context. |
| `503` | `CART_UPSTREAM_UNAVAILABLE` | BFF không kết nối upstream | Error + retry, hướng dẫn dev kiểm tra Mockoon. |

## Mock scenarios

Trong development thêm `?mockScenario=<name>` vào page/API. BFF chỉ forward header tới Mockoon port `4014` ngoài production.

| Scenario | HTTP | Dữ liệu/state |
| --- | --- | --- |
| mặc định | `200/201` | Data bucket stateful; cart mới là empty, Add/Update/Delete thay đổi bucket. |
| `cart-prefilled` | `200` | Nhiều dòng khả dụng để kiểm tra layout/subtotal. |
| `cart-empty` | `200` | Empty state. |
| `price-changed` | `200` | Current price khác snapshot lúc thêm; có notice. |
| `cart-unavailable` | `200` | Một dòng unavailable bị khóa checkbox; dòng khả dụng còn lại vẫn có thể được chọn để chuẩn bị Checkout. |
| `sku-unavailable` | `409` khi Add/Update | SKU vừa không còn mua được. |
| `quantity-unavailable` | `409` khi Add/Update | Quantity không còn đáp ứng; không trả exact stock. |
| `cart-error` | `500` | Error/retry state. |
| `cart-slow` | `200` sau khoảng 3 giây | Loading state. |

## Browser → BFF → Mockoon mapping

| Browser | BFF validation/transform | Mockoon local |
| --- | --- | --- |
| `GET /api/cart` | Resolve opaque cookie; allow no business query; project/recompute totals | GET fixed internal buckets with context filter |
| `POST /api/cart/items` | Validate; lookup public SKU projection; merge same SKU; attach context/snapshot | GET projection + POST/PATCH cart line |
| `PATCH /api/cart/items/{itemId}` | Validate ownership and quantity; recheck SKU | GET line/projection + PATCH cart line |
| `DELETE /api/cart/items/{itemId}` | Validate ownership | GET line + DELETE cart line |
| `?mockScenario=cart-slow` (dev only) | Remove query; set allowlisted header | `X-Mock-Scenario: cart-slow` |
| Arbitrary path/method/query/body fields | Reject locally | Không gọi upstream |

## Public, mock-internal và forbidden data

| Phân loại | Field/dữ liệu |
| --- | --- |
| Public | Product/SKU display snapshot, quantity, current unit price, previous price khi changed, line subtotal, cart subtotal, public availability/reason/notices. |
| Mock internal | `contextId`, `unitPriceVndAtAddition`, data bucket row ID, `maxPurchasableQuantity`, Mockoon CRUD filters, fixture/scenario selectors. |
| Không được lộ browser | Opaque cart context, exact stock, warehouse, cost/margin, Batch/Lot/expiry, internal status/reason, auth token, upstream URL, service topology. |

BFF luôn tạo public projection thay vì pass-through data bucket. `contextId` chỉ đi qua server-to-server header/query/body trong local adapter và không xuất hiện trong response.

## UI validation và mutation behavior

- Product Detail chỉ Add sau khi có explicit SKU selection; single available SKU đã auto-select theo feature trước.
- Add button có pending state; response lỗi không hiển thị success giả.
- Quantity controls có accessible label, min/max candidate và disabled khi pending/unavailable.
- Mutation response thay toàn bộ cart snapshot để price/availability/subtotal luôn nhất quán.
- Line-level error được gắn với `itemId`; retry/refetch không xóa context hiển thị.
- Mặc định UI chọn mọi dòng khả dụng. Checkbox từng dòng và “Chọn tất cả” chỉ cập nhật local UI state; dòng unavailable luôn bị loại.
- Selected item count/subtotal được derive từ public cart response và danh sách `itemId` đang chọn; không gửi mutation Cart chỉ để tick checkbox.
- Selection hiện không persist qua reload. Checkout feature phải chốt cách truyền `itemId`/selection token và revalidate giá, availability, ownership trước khi tạo Order.
- UI không tính stock, shipping, promotion hoặc final total.
- Checkout integration không phát request trong feature này.
