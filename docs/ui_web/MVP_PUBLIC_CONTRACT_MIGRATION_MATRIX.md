# MVP Public Contract — Frontend migration matrix

> Giai đoạn 1–4 hoàn tất: đã chốt baseline, tạo lớp contract dùng chung, chuyển Catalog/Product và Profile/Address sang public MVP contract.

## 1. Baseline được pin

| Mục | Giá trị |
| --- | --- |
| Release | `mvp-2026-10-26` |
| Trạng thái contract | `integration-target` |
| Runtime acceptance | `pending` |
| Contract branch | `integration/mvp-contracts` |
| Contract commit | `9416090bb476406b4928f376c594cc1748671c49` |
| Public contract | `docs/03_api_specs/mvp-customer.openapi.yaml` |
| Public contract Git blob | `af684a9e747d42a8b4c77412a8b9f5324a77a3f7` |
| Frontend baseline | `feature/ui-web@b1ff2918c7297fc93d0cb9a15cc1d0a0ee46b72e` |
| Public operations | 30 |

Mốc máy đọc nằm tại [`mvp-public-contract-baseline.json`](./mvp-public-contract-baseline.json). Khi PR contract đổi HEAD hoặc nội dung public contract, phải review diff rồi cập nhật đồng thời commit và blob; không tự trỏ theo tên branch động.

## 2. Thứ tự ưu tiên tài liệu

1. `docs/03_api_specs/mvp-customer.openapi.yaml`: wire contract public cho Web, Mobile và BFF.
2. `packages/contracts/mvp/MIGRATION.md`: quy tắc migration, idempotency, async checkout, DTO và version.
3. `packages/contracts/mvp/registry.yaml`: owner, consumer, dependency và trạng thái triển khai.
4. `packages/contracts/mvp/TEAM-DECISIONS.md`: phạm vi mở capability và feature gate đề xuất.

Các file `customer-stable`, `customer-draft-planned`, `web-reference/*` và `*.candidate.openapi.yaml` chỉ là provenance/tham khảo. Chúng không được ghi đè public contract ở route trùng nhau. Proto, event schema, Order core facade và service OpenAPI là contract backend/internal; React client không gọi trực tiếp.

## 3. Quy ước migration

| Trạng thái | Ý nghĩa |
| --- | --- |
| `ADAPT` | Đã có màn hình/route tương ứng nhưng method, DTO, header hoặc pagination chưa khớp. |
| `REPLACE` | Luồng hiện tại mâu thuẫn contract mới và phải thay thế. |
| `ADD` | Chưa có operation tại BFF/client. |
| `GATED` | Contract có operation nhưng capability chưa được bật cho vòng Web MVP đầu tiên. |
| `ALIGNED` | BFF/client và mapper đã được chuyển sang contract target trong baseline được pin. |

Đường `/api/*` của Next.js là BFF nội bộ cùng origin và có thể giữ tên thân thiện với UI. Mỗi route đó phải có ánh xạ một-một được kiểm thử tới public contract. DTO wire từ upstream không được trộn trực tiếp với view model camelCase; mapper phải tách rõ hai lớp.

## 4. Ma trận 30 public operations

### 4.1 Cart và Checkout

| Public operation | BFF/UI hiện tại | Trạng thái | Việc phải làm ở giai đoạn triển khai |
| --- | --- | --- | --- |
| `GET /api/v1/cart` — `order_getCart` | `GET /api/cart` | `ADAPT` | Bổ sung server `revision`; map snapshot canonical thay vì chỉ `updatedAt`. |
| `POST /api/v1/cart/items` — `order_addCartItem` | `POST /api/cart/items` | `ADAPT` | Chuẩn hóa SKU, quantity, response revision và PublicError. |
| `PUT /api/v1/cart/items/{item_id}` — `order_updateCartItem` | `PATCH /api/cart/items/{itemId}` | `ADAPT` | Đổi upstream method/payload theo contract; xử lý revision conflict. |
| `DELETE /api/v1/cart/items/{item_id}` — `order_deleteCartItem` | `DELETE /api/cart/items/{itemId}` | `ADAPT` | Giữ ownership phía server và trả snapshot/revision canonical. |
| `POST /api/v1/checkout/quote` — `order_createCheckoutQuote` | `POST /api/checkout/shipping-quotes` | `REPLACE` | Bỏ quote phí `0`; gửi `cart_revision`, `payment_method`, address/address ID và nhận `OrderQuote`. |
| `POST /api/v1/checkout` — `order_acceptCheckout` | `POST /api/checkout/confirm` | `REPLACE` | Chỉ gửi `quote_id`, `cart_revision`, `payment_method`; chuyển idempotency sang header. |
| `GET /api/v1/checkout-operations/{id}` — `order_getCheckoutOperation` | Chưa có | `ADD` | Poll operation, đọc `Retry-After`, dừng ở terminal state và chỉ dùng `order_id` khi thành công. |

### 4.2 Order và Payment

| Public operation | BFF/UI hiện tại | Trạng thái | Việc phải làm ở giai đoạn triển khai |
| --- | --- | --- | --- |
| `GET /api/v1/orders` — `order_listOwnedOrders` | `GET /api/orders` với `page/pageSize` | `ADAPT` | Map sang `limit/cursor/status`; không giả lập total khi upstream không cung cấp. |
| `GET /api/v1/orders/{id}` — `order_getOwnedOrder` | `GET /api/orders/{id}` | `ADAPT` | Chuẩn hóa DTO, Money/status và ownership error. |
| `GET /api/v1/orders/{id}/tracking` — `order_getOrderTracking` | BFF gọi ngầm khi đọc detail | `ADAPT` | Giữ endpoint/mapping riêng để refresh tracking độc lập. |
| `POST /api/v1/orders/{id}/cancel` — `order_cancelOrder` | `POST /api/orders/{id}/cancellations` rồi BFF gọi `/cancel` | `ADAPT` | Thêm version/ETag theo contract và public `Idempotency-Key`. |
| `POST /api/v1/orders/{id}/cancellation-requests` — `order_requestPaidCancellation` | Chưa có | `GATED` | Thêm sau khi paid cancellation policy/provider được chấp nhận. |
| `POST /api/v1/guest-sessions` — `order_createGuestSession` | Cookie cart context tự tạo ở BFF | `GATED` | Không coi cookie local là guest credential backend; triển khai khi guest flow được bật. |
| `POST /api/v1/orders/{order_id}/guest-payment-access` — `payment_issueGuestPaymentAccess` | Custom extension `orders/guest-access/grants` | `GATED` | Thay custom grant bằng public operation khi backend chấp nhận guest payment flow. |
| `DELETE /api/v1/orders/{order_id}/guest-payment-access` — `payment_revokeGuestPaymentAccess` | Chưa có | `GATED` | Thu hồi grant khi thoát/kết thúc flow; chưa bật trong vòng đầu. |
| `GET /api/v1/orders/{order_id}/payment-instructions` — `payment_getSePayPaymentInstructions` | `/api/payments/orders/{id}` dựng từ cookie | `REPLACE` | Lấy hướng dẫn/QR canonical từ backend, không dựng payment ID hoặc amount từ cookie. |
| `GET /api/v1/orders/{order_id}/payment` — `payment_getSePayPaymentStatus` | `/api/payments/orders/{id}/status` suy trạng thái từ expiry cookie | `REPLACE` | Đọc trạng thái đã xác minh từ backend; QR không đồng nghĩa đã thanh toán. |

### 4.3 Profile và Address

| Public operation | BFF/UI hiện tại | Trạng thái | Việc phải làm ở giai đoạn triển khai |
| --- | --- | --- | --- |
| `GET /api/v1/profile` — `profile_getProfile` | `GET /api/customer/profile` | `ALIGNED` | Đã giữ ETag thật, map Profile DTO và hiển thị email chỉ đọc. |
| `PUT /api/v1/profile` — `profile_updateProfile` | `PUT /api/customer/profile` | `ALIGNED` | Đã bỏ email khỏi update và forward `If-Match: "<version>"`. |
| `GET /api/v1/profile/addresses` — `profile_listAddresses` | `/api/customer/addresses` trả array canonical | `ALIGNED` | View model `{items,total}` chỉ được tạo sau mapper phía browser. |
| `POST /api/v1/profile/addresses` — `profile_createAddress` | `POST /api/customer/addresses` | `ALIGNED` | Đã chuẩn hóa code/name, ETag, Location và trạng thái default trả từ server. |
| `GET /api/v1/profile/addresses/{id}` — `profile_getAddress` | `GET /api/customer/addresses/{id}` | `ALIGNED` | Đã có read-owned-address và map 404 qua `PublicError`. |
| `PUT /api/v1/profile/addresses/{id}` — `profile_updateAddress` | `PUT /api/customer/addresses/{id}` | `ALIGNED` | Đã forward strong `If-Match`; đổi mặc định dùng operation riêng. |
| `DELETE /api/v1/profile/addresses/{id}` — `profile_deleteAddress` | `DELETE /api/customer/addresses/{id}` | `ALIGNED` | Đã forward ETag và dùng danh sách canonical backend trả về. |
| `PUT /api/v1/profile/addresses/{id}/default` — `profile_putDefaultAddress` | `PUT /api/customer/addresses/{id}/default` | `ALIGNED` | BFF gọi operation canonical trực tiếp trong production. |
| `PATCH /api/v1/profile/addresses/{id}/default` — `profile_patchDefaultAddress` | `PATCH /api/customer/addresses/{id}/default` | `ALIGNED` | Client/BFF đã hỗ trợ operation; UI hiện dùng PUT intent. |

### 4.4 Catalog

| Public operation | BFF/UI hiện tại | Trạng thái | Việc phải làm ở giai đoạn triển khai |
| --- | --- | --- | --- |
| `GET /api/v1/categories` — `catalog_getCategories` | BFF dùng để dựng `/api/catalog/discovery-config` | `ALIGNED` | Đã có mapper categories canonical; config UI là view model riêng. |
| `GET /api/v1/products` — `catalog_getProducts` | `/api/catalog/products` giữ object pagination | `ALIGNED` | Đã nhận `items/total/page/page_size/total_pages` và bỏ count headers tự dựng. |
| `GET /api/v1/products/search` — `catalog_getProductsSearch` | UI vẫn gọi `/api/catalog/products?q=...`; BFF chọn operation search | `ALIGNED` | Đã map riêng response search; fallback local chỉ chạy ở development. |
| `GET /api/v1/products/{id_or_slug}` — `catalog_getProductsIdOrSlug` | `/api/catalog/products/{slug}` | `ALIGNED` | Đã chuẩn hóa Product/SKU/images/Money và tách wire DTO khỏi view model. |

## 5. Route hiện có nhưng không thuộc 30-operation target

| Route/feature hiện tại | Quyết định |
| --- | --- |
| `/api/checkout/prepare` | UI composition nội bộ; không được tạo checkout identity hoặc giá canonical. Dữ liệu mua hàng phải đến từ Cart/Profile. |
| `/api/checkout/shipping-quotes` | Retire khi `order_createCheckoutQuote` hoạt động. Không giữ fee `0` fallback. |
| `/api/checkout/confirm` | Retire sau khi UI chuyển sang accept + operation polling. |
| `DELETE /api/cart` | Không có trong public target; giữ local-only nếu chỉ phục vụ dev, không gọi production upstream. |
| `/api/payments/orders/{id}/attempts` | Không có trong target; giữ disabled cho đến khi có contract retry payment. |
| Guest challenge/OTP extension | Ngoài public target hiện tại; giữ feature gate và không xem là guest-session contract. |
| `/api/customer/locations` | View-support API riêng; nguồn địa giới cần contract/provider riêng, không gộp vào Profile contract. |
| Auth, Returns, Support, B2B và Admin | Không nằm trong 30 operations của release này; không xóa hoặc tự sửa theo Order MVP. |

## 6. Ranh giới public và internal

| Thành phần | Được phép | Không được phép |
| --- | --- | --- |
| React/browser | Gọi same-origin Next BFF; dùng view model và UI state. | Gọi service URL, proto/gRPC, internal facade, gửi trust headers hoặc lưu service token. |
| Next BFF | Xác minh session, gọi public REST contract, map wire DTO ↔ view model, sanitize PublicError. | Bịa phí/giá/trạng thái; dùng simulator fallback trong production; tin `X-User-ID`, role hoặc permission từ browser. |
| Gateway/provider adapter | Map public `Idempotency-Key` sang provider `X-Idempotency-Key`; map verified principal/customer. | Tạo key mới khi retry cùng lệnh; lộ internal error/PII; đổi `202` thành thành công giả. |
| Backend internal | Dùng proto, event schema, Order core facade theo registry. | Expose trực tiếp các contract internal cho browser. |

## 7. Các quyết định đã khóa cho giai đoạn sau

- `mvp-customer.openapi.yaml` thắng mọi snapshot/candidate ở route trùng nhau.
- BFF giữ mapper rõ ràng; React không phụ thuộc snake_case wire DTO trực tiếp.
- Checkout chuyển hoàn toàn sang quote → accept → operation polling; không duy trì hai luồng production.
- Mutation không tự retry. Khi outcome chưa rõ, giữ nguyên idempotency key và canonical payload.
- `401` là chưa xác thực; `403` là đã xác thực nhưng thiếu quyền.
- Guest, paid cancellation, COD và automatic Completed vẫn feature-gated cho đến khi registry có runtime acceptance tương ứng.
- Contract CI xanh không đồng nghĩa service thật đã sẵn sàng; frontend phải fail closed khi dependency chưa available.

## 8. Kết quả Giai đoạn 2 — lớp contract dùng chung

Lớp contract dùng chung nằm tại `apps/web-user/src/lib/api-contract/`:

| File | Trách nhiệm |
| --- | --- |
| `types.ts` | Wire types cho Catalog, Profile, Cart, Checkout, Order và Payment. |
| `error.ts` | Chuẩn hóa `PublicError`, giữ `requestId`, `retryable`, `operation_id`, `details`. |
| `money.ts` | Kiểm VND, `nanos=0`, integer an toàn và định dạng tiền. |
| `etag.ts` | Strong numeric ETag cho Profile/Address và decimal version cho Order command. |
| `idempotency.ts` | Validate/generate key 16–128 visible ASCII; tách public/provider header. |
| `client.ts` | JSON request client không tự retry mutation; giữ response headers, ETag và Retry-After. |
| `index.ts` | Public exports cho các feature adapter. |

Sau Giai đoạn 4, Catalog/Product và Profile/Address đã dùng lớp này. Cart/Checkout/Order/Payment sẽ được chuyển ở các giai đoạn nghiệp vụ tiếp theo.

## 9. Kết quả Giai đoạn 3 — Catalog và Product

| Public operation | Kết quả frontend |
| --- | --- |
| `GET /api/v1/categories` | BFF kiểm tra response canonical và chỉ map sang `DiscoveryConfig` sau contract boundary. |
| `GET /api/v1/products` | Giữ object pagination `items/total/page/page_size/total_pages`; đã bỏ array và các count header tự dựng. |
| `GET /api/v1/products/search` | Có nhánh DTO/mapper riêng; fallback tìm kiếm chỉ tồn tại ở môi trường development để tương thích thứ tự route Mockoon. |
| `GET /api/v1/products/{id_or_slug}` | BFF trả wire DTO canonical; browser map Product/SKU/images/Money sang view model. |

Các quy tắc đã áp dụng:

- Browser chỉ dùng view model camelCase; wire DTO snake_case nằm trong `lib/api-contract` và mapper feature.
- Giá VND đi qua parser dùng chung, yêu cầu `currency_code=VND`, `nanos=0` và `units` là số nguyên an toàn.
- Tồn kho có ba trạng thái `AVAILABLE`, `UNAVAILABLE`, `UNKNOWN`; `null` không còn bị hiểu sai thành hết hàng.
- Lỗi từ upstream được sanitize về `PublicError`; lỗi BFF tự sinh cũng có đủ `code`, `message`, `requestId` và `details` khi cần.
- Same-origin route `/api/catalog/*` được giữ làm BFF cho UI, còn upstream vẫn ánh xạ tới bốn operation public nêu trên.

## 10. Kết quả Giai đoạn 4 — Profile và Address

- Browser dùng mapper `lib/customer/mvp.ts`; DTO snake_case không đi vào component.
- Profile update chỉ gửi `full_name` và `phone_number`; email hiển thị read-only, date of birth/gender chưa bị ghi ngược ngoài contract.
- Profile và Address mutation dùng strong numeric ETag (`"<version>"`) qua `If-Match`; BFF forward nguyên header và response ETag.
- Address list/delete/default giữ array canonical tại contract boundary; `{items,total}` chỉ là view model phía browser.
- Sửa địa chỉ không còn ghi `is_default`; nếu người dùng chọn mặc định, UI gọi operation default riêng sau update thành công.
- `GET` từng địa chỉ và cả hai method `PUT`/`PATCH` default đã có mapping BFF một-một trong production.
- Tương thích response cũ của Mockoon chỉ chạy ngoài production; production fail closed nếu Profile/Address DTO sai contract.
- Lỗi provider và lỗi BFF được chuẩn hóa về `PublicError`; field errors trong `details.errors` được map về tên trường của form.
