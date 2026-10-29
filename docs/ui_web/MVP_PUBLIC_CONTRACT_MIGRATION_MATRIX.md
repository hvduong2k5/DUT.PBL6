# MVP Public Contract — Frontend migration matrix

> Giai đoạn 1–9 hoàn tất: đã chốt baseline, tạo lớp contract dùng chung, chuyển Catalog/Product, Profile/Address, Cart, Checkout, Order và Payment sang public MVP contract; hoàn thiện Identity/quyền; sau đó hardening capability/runtime và lập readiness handoff.

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
| `GET /api/v1/cart` — `order_getCart` | `GET /api/cart` | `ALIGNED` | Đã giữ server `revision` và map snapshot canonical; không tạo `updatedAt` giả. |
| `POST /api/v1/cart/items` — `order_addCartItem` | `POST /api/cart/items` | `ALIGNED` | Đã gửi `sku_code`, `quantity`, `revision`; response và lỗi dùng contract chung. |
| `PUT /api/v1/cart/items/{item_id}` — `order_updateCartItem` | `PUT /api/cart/items/{itemId}` | `ALIGNED` | Đã dùng PUT, gửi revision và hiển thị riêng lỗi conflict `409`. |
| `DELETE /api/v1/cart/items/{item_id}` — `order_deleteCartItem` | `DELETE /api/cart/items/{itemId}` | `ALIGNED` | BFF xóa theo ownership rồi đọc lại snapshot canonical cho UI. |
| `POST /api/v1/checkout/quote` — `order_createCheckoutQuote` | `POST /api/checkout/quote` | `ALIGNED` | Đã bỏ quote phí `0`; gửi `cart_revision`, `payment_method`, address/address ID và nhận `OrderQuote` canonical. |
| `POST /api/v1/checkout` — `order_acceptCheckout` | `POST /api/checkout` | `ALIGNED` | Chỉ gửi `quote_id`, `cart_revision`, `payment_method`; `Idempotency-Key` nằm ở header và được giữ ổn định cho cùng lệnh. |
| `GET /api/v1/checkout-operations/{id}` — `order_getCheckoutOperation` | `GET /api/checkout/operations/{id}` | `ALIGNED` | Đã poll operation theo `Retry-After`, dừng ở terminal state và chỉ dùng `order_id` khi `SUCCEEDED`. |

### 4.2 Order và Payment

| Public operation | BFF/UI hiện tại | Trạng thái | Việc phải làm ở giai đoạn triển khai |
| --- | --- | --- | --- |
| `GET /api/v1/orders` — `order_listOwnedOrders` | `GET /api/orders` | `ALIGNED` | BFF trả `items/next_cursor` canonical; search/status/year chỉ lọc response hiện tại ở browser, không gửi query ngoài contract. |
| `GET /api/v1/orders/{id}` — `order_getOwnedOrder` | `GET /api/orders/{id}` | `ALIGNED` | Đã map `OrderOrder`, giữ lifecycle/payment/stock/cancellation độc lập và dùng `allowed_actions` làm authority UI. |
| `GET /api/v1/orders/{id}/tracking` — `order_getOrderTracking` | `GET /api/orders/{id}/tracking` | `ALIGNED` | Refresh tracking độc lập; UI không suy diễn checkpoint/carrier vì baseline hiện chỉ khai báo response là object. |
| `POST /api/v1/orders/{id}/cancel` — `order_cancelOrder` | `POST /api/orders/{id}/cancel` | `ALIGNED` | Đã gửi decimal Order version qua `If-Match`, public `Idempotency-Key`, body `{active,reason}` và giữ kết quả `200/202`. |
| `POST /api/v1/orders/{id}/cancellation-requests` — `order_requestPaidCancellation` | Chưa có | `GATED` | Thêm sau khi paid cancellation policy/provider được chấp nhận. |
| `POST /api/v1/guest-sessions` — `order_createGuestSession` | Cookie cart context tự tạo ở BFF | `GATED` | Không coi cookie local là guest credential backend; triển khai khi guest flow được bật. |
| `POST /api/v1/orders/{order_id}/guest-payment-access` — `payment_issueGuestPaymentAccess` | Custom extension `orders/guest-access/grants` | `GATED` | Thay custom grant bằng public operation khi backend chấp nhận guest payment flow. |
| `DELETE /api/v1/orders/{order_id}/guest-payment-access` — `payment_revokeGuestPaymentAccess` | Chưa có | `GATED` | Thu hồi grant khi thoát/kết thúc flow; chưa bật trong vòng đầu. |
| `GET /api/v1/orders/{order_id}/payment-instructions` — `payment_getSePayPaymentInstructions` | `GET /api/payments/orders/{id}/instructions` | `ALIGNED` | Đã lấy persisted VietQR intent, receiver, amount, content, QR và expiry canonical; không dựng dữ liệu từ cookie. |
| `GET /api/v1/orders/{order_id}/payment` — `payment_getSePayPaymentStatus` | `GET /api/payments/orders/{id}/status` | `ALIGNED` | Đã đọc financial projection canonical và poll theo `next_poll_after_seconds`; frontend không suy trạng thái từ countdown. |

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
| `/api/checkout/prepare` | Đã loại khỏi BFF. Browser composition dữ liệu từ Cart/Profile/Address/locations và không tạo checkout identity hoặc giá canonical. |
| `/api/checkout/shipping-quotes` | Đã loại khỏi runtime; thay bằng `POST /api/checkout/quote`, không giữ fee `0` fallback. |
| `/api/checkout/confirm` | Đã loại khỏi runtime; thay bằng accept + operation polling. |
| `DELETE /api/cart` | Đã bỏ route ngoài contract; nút dọn giỏ là UI composition gọi tuần tự operation xóa từng dòng. |
| `/api/payments/orders/{id}/attempts` | Đã loại khỏi runtime vì public target không có operation tạo/retry Payment intent. |
| Guest challenge/OTP extension | Ngoài public target hiện tại; mặc định tắt bằng `ORDER_GUEST_EXTENSION_ENABLED=false` và không xem là guest-session contract. |
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

Sau Giai đoạn 8, toàn bộ operation Customer đang mở trong 30-operation target đã dùng lớp contract chung. Các operation Guest, paid cancellation và capability chưa được nghiệm thu vẫn giữ trạng thái `GATED`.

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

## 11. Kết quả Giai đoạn 5 — Cart

- Cart browser mapper nhận `OrderCart { revision, items }`; mutation không còn dựa vào `updatedAt` hoặc count/tổng do mock tự khai.
- Add/update gửi revision canonical; update đổi từ `PATCH` sang `PUT`, conflict `409` được giữ nguyên cho UI xử lý.
- `DELETE /api/cart` ngoài contract đã bị loại bỏ. “Dọn giỏ hàng” là composition của các operation xóa dòng đã có contract.
- Delete item được BFF đọc lại Cart sau khi xóa vì response delete trong contract chưa đặc tả snapshot cụ thể.
- `item_id` là bắt buộc ở mapper UI để mutation không dùng nhầm `sku_code` làm identity dòng giỏ.
- Các field enrichment không có trong `OrderItem` như ảnh, trạng thái tồn và giá có thể vắng mặt không bị bịa thành dữ liệu authoritative; UI ghi rõ sẽ kiểm tra lại khi Checkout.
- Cookie context và `X-Session-Id` tự khai đã bị loại khỏi Cart production. Production yêu cầu credential xác thực; tương thích response Mockoon cũ chỉ chạy ngoài production.
- BFF và client dùng `PublicError`; mutation không tự retry.

## 12. Kết quả Giai đoạn 6 — Checkout

- Luồng runtime đã chuyển hoàn toàn sang `quote → accept → operation polling`; ba BFF route ánh xạ một-một tới public contract được pin.
- Quote từ server là nguồn canonical cho tạm tính, phí vận chuyển, giảm giá, tổng thanh toán và thời hạn; frontend không còn tự dựng phí `0` hoặc chốt số tiền từ dữ liệu hiển thị.
- Accept chỉ gửi `quote_id`, `cart_revision`, `payment_method`; cùng một quote giữ cùng `Idempotency-Key`, mutation không tự retry bằng khóa mới khi kết quả chưa rõ.
- BFF giữ nguyên semantics `201`/`202`, `Location` và `Retry-After`; client chỉ dùng `order_id` sau khi operation ở trạng thái `SUCCEEDED`.
- `FAILED` và `MANUAL_REVIEW` được hiển thị riêng. Operation chưa terminal có thể tiếp tục poll bằng chính `operation_id`, không tạo lại lệnh Checkout.
- Các cookie/grant tự khai của Checkout cũ, `X-Session-Id`, QR/payment claim cục bộ và guest credential giả đã bị loại khỏi luồng production.
- Bước chuẩn bị màn hình chỉ composition dữ liệu Cart, Profile, Address và locations ở browser; mọi quyết định thương mại vẫn thuộc quote backend.
- Mockoon mobile hiện tại chưa có ba operation quote/accept/poll theo target. Frontend vì vậy fail closed cho đến khi backend hoặc mock contract mới được cung cấp, thay vì quay về luồng cũ không tương thích.

## 13. Kết quả Giai đoạn 7 — Order

- List/detail dùng trực tiếp `OwnedOrdersResponse` và `OwnedOrder`; BFF không còn map DTO Mockoon cũ hoặc dựng customer/total/payment/shipping state giả.
- Order, Payment, Inventory, cancellation và compensation được trình bày độc lập. UI lấy quyền hành động từ `allowed_actions`, không suy `canCancel` từ status.
- Snapshot sản phẩm/địa chỉ là optional đúng contract; khi vắng mặt UI hiển thị trạng thái thiếu dữ liệu thay vì nối lại Catalog/Profile hiện tại.
- Tracking được tách thành operation riêng. Vì baseline chỉ đặc tả response `object`, frontend chỉ xác nhận endpoint khả dụng và không tự đặt tên field carrier/checkpoint.
- Cancel gửi `{active:true, reason}`, decimal Order `version` qua `If-Match` và public `Idempotency-Key`; cùng payload/version giữ cùng key khi outcome chưa rõ.
- Response cancel `PENDING` chỉ là intent bền vững, không bị UI đổi thành “đã hủy”. Sau response xác định, detail được đọc lại để lấy `cancellation_status` và `compensation_status` canonical.
- Paid cancellation (`REQUEST_CANCELLATION`) vẫn gated; UI không gọi operation Care review khi capability chưa được nghiệm thu.
- Guest challenge/OTP cũ được giữ như extension có feature gate server-side và tắt mặc định. Public Order target vòng đầu yêu cầu bearer đã xác minh.
- List contract trả `next_cursor` nhưng chưa khai báo cursor request parameter. UI ghi rõ còn trang nhưng không tự phát minh query; đây là contract gap cần backend/contract owner bổ sung trước khi làm phân trang tiếp.
- Mockoon mobile hiện tại chưa trả đầy đủ `OrderOrder`; runtime mới fail closed cho đến khi backend hoặc mock contract target sẵn sàng.

## 14. Kết quả Giai đoạn 7B — Payment

- Payment page đọc `PaymentPaymentStatus` trước; chỉ gọi `PaymentPaymentInstructions` khi backend trả `can_pay=true`.
- VietQR amount, receiver, account, transfer content, payment code, QR URL và expiry đều đến từ persisted intent. Cookie `oma_payment_access`, secret ký local và payment ID dựng từ Order ID đã bị loại bỏ.
- UI dùng `display_state`, `payment_status`, `order_status` và `intent_status` độc lập; countdown chỉ hiển thị expiry, không tự đổi trạng thái.
- Automatic polling tuân theo `next_poll_after_seconds` và dừng khi backend trả `0`; thao tác kiểm tra thủ công chỉ đọc projection, không tạo Payment attempt.
- Allocated/refunded amount, reconciliation flag, support reason/reference và verification deadline được hiển thị từ dữ liệu tài chính canonical.
- BFF chỉ chấp nhận UUID Order và hai GET route tương ứng, yêu cầu principal đã xác minh, forward credential tới public gateway và trả `Cache-Control: no-store`.
- Endpoint retry `/attempts` và POST route đã bị xóa vì không tồn tại trong target; frontend không phát minh mutation để gia hạn expiry hoặc tạo intent mới.
- Guest payment grant issue/revoke và COD vẫn gated. Local signed cookie cũ không còn được xem là credential Guest hợp lệ.
- QR URL phải là HTTPS và toàn bộ response được kiểm tra enum, UUID, amount, date-time, version và invariants trước khi đi vào UI.
- Mockoon Payment cũ không có hai response canonical này; runtime fail closed cho đến khi backend hoặc mock contract target sẵn sàng.

## 14A. Kết quả Giai đoạn 8 — Identity và quyền

- `FE-AUTH-01`: BFF chỉ lấy access token từ cookie HttpOnly `oma_access_token`; không còn nhận `Authorization`, `X-User-ID`, role, permission hoặc toàn bộ cookie do browser tự cung cấp để chuyển tiếp upstream.
- Trước khi cấp capability Customer, BFF gọi `/auth/me` bằng token server-side để backend/gateway xác minh principal. Response sai contract hoặc Identity không khả dụng đều fail closed.
- `FE-AUTH-02`: `401` được giữ cho phiên thiếu/hết hạn và đưa UI về đăng nhập; `403` được giữ cho principal đã xác thực nhưng thiếu mapping/quyền và hiển thị access-denied riêng.
- `FE-AUTH-03`: `CUSTOMER` dùng Customer features khi có `customer_id`. `EMPLOYEE` chỉ kế thừa Customer features khi Identity/backend trả `customer_id` đã liên kết; role `CUSTOMER` do browser khai báo không đủ để mở quyền.
- `FE-AUTH-04`: UI lấy role/permission/capability từ projection đã xác minh để ẩn/hiện. Mọi BFF mutation/read bảo vệ vẫn gọi lại Identity/capability projection; UI không phải authority.
- Principal session giữ riêng `principalType`, role, permission và `customerAccess`; staff permission không bị trộn với Customer capability.
- Chi tiết boundary, contract gap và acceptance criteria nằm tại [`MVP_PUBLIC_IDENTITY_AUTHORIZATION.md`](./MVP_PUBLIC_IDENTITY_AUTHORIZATION.md).

## 15. Kết quả Giai đoạn 9 — Integration hardening và readiness

- Tổng số operation trong target: **30**; frontend đã căn chỉnh **26** operation đang mở và giữ **4** operation Guest/paid cancellation ở trạng thái `GATED`.
- COD, Voucher và Guest Order được khóa mặc định bằng capability gate dùng chung. UI không hiển thị capability bị khóa; BFF Checkout vẫn kiểm tra lại và trả `403 FEATURE_NOT_ENABLED` nếu client gọi trực tiếp.
- Feature gate chỉ khóa/mở việc phát hành capability; không cấp quyền. Khi gate được bật, Identity/mapping/capability check của Giai đoạn 8 vẫn chạy trước nghiệp vụ được bảo vệ.
- Header đưa người dùng đã đăng nhập tới lịch sử Order. Người chưa đăng nhập chỉ thấy Guest Order khi capability này được bật; mặc định được hướng tới đăng nhập.
- `.env.example` chỉ còn các upstream/config thực sự được source hiện tại sử dụng; các URL chia nhỏ theo mock feature cũ đã bị loại khỏi cấu hình chuẩn.
- Các script `dev:cart`, `dev:checkout`, `dev:orders`, `dev:payment` vẫn là fixture lịch sử cho từng lát cắt. Chúng không đại diện cho public target 30-operation và không được dùng làm bằng chứng E2E/acceptance.
- Frontend readiness, ma trận trạng thái, dependency runtime, contract gap và checklist nghiệm thu được chốt tại [`MVP_PUBLIC_FRONTEND_READINESS.md`](./MVP_PUBLIC_FRONTEND_READINESS.md).
- Contract CI xanh chỉ xác nhận schema/wire. Luồng thật chỉ được coi là sẵn sàng sau khi public gateway, Identity và các provider owning service có runtime acceptance và E2E evidence.

## 16. Kết quả Giai đoạn 10 — Contract regression và release verification

- Thêm một cổng kiểm tra thống nhất `npm run verify:mvp` cho Web User: typecheck → lint → test.
- Vitest đã được cấu hình alias `@` giống TypeScript và chạy một worker không isolate để ổn định trên Windows sandbox.
- Bổ sung regression test chứng minh BFF chỉ nhận access token từ cookie tin cậy, bỏ qua browser `Authorization`.
- Bổ sung regression test cho `401`, Employee thiếu/có Customer mapping và capability `403` sau khi principal được xác minh.
- Đồng bộ assertion Profile email read-only và availability ba trạng thái của Catalog/Product Detail với contract hiện tại.
- Kết quả local ngày 2026-10-30: typecheck PASS, ESLint PASS, **26 test files / 87 tests PASS**.
- Đây là frontend regression evidence, chưa thay thế live-service E2E hoặc owning-service runtime acceptance. Báo cáo đầy đủ nằm tại [`MVP_PUBLIC_RELEASE_VERIFICATION.md`](./MVP_PUBLIC_RELEASE_VERIFICATION.md).
