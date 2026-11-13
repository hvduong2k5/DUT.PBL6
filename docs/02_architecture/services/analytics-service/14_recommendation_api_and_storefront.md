# 14 — Recommendation API, consent và Storefront

**Hợp đồng v1 · 07/11/2026.** [Candidate OpenAPI](../../../03_api_specs/analytics-recommendation.candidate.openapi.yaml) và [JSON Schema khớp runtime](../../../03_api_specs/analytics-recommendation.v1.schema.json) đã được thêm vào contract tests. [Thuật toán](12_recommendation_engine_design.md) · [Hợp đồng nguồn](13_recommendation_source_contracts.md) · [Runtime thực tế](15_recommendation_persistence_and_delivery.md)

## 1. API đọc gợi ý

```http
GET /api/v1/products/recommended
```

| Query | Quy tắc |
| --- | --- |
| `placement` | Bắt buộc, một trong `product_detail`, `cart`, `home` |
| `productId` | UUID bắt buộc với `product_detail` |
| `skuCode` | Tùy chọn với `product_detail`, phải thuộc product mốc và đang công khai/active |
| `skuCodes` | Query lặp, tối đa 20 SKU distinct với `cart`; rỗng được phép |
| `limit` | Integer 1–6, mặc định 6 |

`product_detail` dùng related content. `cart` dùng basket pairs. `home` dùng cá nhân hóa khi phiên/consent cho phép, nếu không có credential thì dùng fallback. Query không thuộc placement, field lạ, product ID sai định dạng và SKU quá dài trả `400 INVALID_RECOMMENDATION_QUERY`. SKU canonical giới hạn 64 ký tự `[A-Za-z0-9_.-]` bắt đầu bằng chữ hoặc số, đối chiếu Catalog để áp ngữ nghĩa nguồn.

```http
GET /api/v1/products/recommended?placement=product_detail&productId=10000000-0000-4000-8000-000000000001&skuCode=MX-DEO-450G&limit=6
GET /api/v1/products/recommended?placement=cart&skuCodes=MX-DEO-450G&skuCodes=TRA-SEN-100G
GET /api/v1/products/recommended?placement=home
```

Scope lấy từ cấu hình Storefront ở BFF/Gateway và allowlist của MS-11. Đợt đầu mỗi deployment Web có một scope. Service không dùng query kho tùy ý để chọn dữ liệu. Mapping scope có version và được pin trong source projection, generation và response.

### 1.1. Response

```json
{
  "recommendationId": "40000000-0000-4000-8000-000000000001",
  "placement": "product_detail",
  "scopeId": "d2c-hue",
  "generationId": "50000000-0000-4000-8000-000000000001",
  "algorithmVersion": "related_content_v1",
  "strategy": "RELATED",
  "reasonCodes": [],
  "asOf": "2026-11-07T00:00:00Z",
  "items": [
    {
      "productId": "10000000-0000-4000-8000-000000000002",
      "skuCode": "MX-GION-450G",
      "slug": "me-xung-gion",
      "name": "Mè xửng giòn",
      "thumbnailUrl": "https://cdn.omama.vn/products/mx-gion.jpg",
      "priceVnd": 95000,
      "rank": 1,
      "algorithmVersion": "related_content_v1",
      "reasonCode": "SIMILAR_ATTRIBUTES"
    }
  ]
}
```

`recommendationId` là request/decision ID dùng đo và trace. `generationId` là generation chính đang dùng, có thể null nếu chỉ có fallback source mà model chưa được publish. `asOf` là timestamp cũ nhất trong các source snapshots đang dùng để render kết quả đó. `rank` bắt đầu 1, liên tục theo thứ tự mảng.

| `strategy` | Ý nghĩa |
| --- | --- |
| `RELATED` | Có mục related content |
| `BOUGHT_TOGETHER` | Có ít nhất một mục từ pair model |
| `PERSONALIZED` | Có mục từ item-to-item hoặc category affinity của Customer |
| `FALLBACK` | Toàn bộ kết quả từ danh sách Catalog đã duyệt |
| `EMPTY` | Nguồn còn hiệu lực và xác nhận không có mục để trả |

Kết quả có thể trộn mục thuật toán chính và fallback. Mỗi item ghi algorithm/reason riêng, phần top-level có `PARTIAL_FALLBACK` nếu đã điền bổ sung. Reason cho fallback gồm `MODEL_NOT_READY`, `STALE_MODEL`, `NO_SELECTED_SKU`, `INSUFFICIENT_BASKETS`, `NO_QUALIFIED_PAIR`, `STALE_ORDER_SOURCE`, `NO_HISTORY`, `PERSONALIZATION_NOT_ENABLED` và `PRIVACY_REBUILD`.

Với `NO_SELECTED_SKU`, strategy vẫn có thể RELATED vì cùng danh mục đã đủ chọn ứng viên. Worker ghi scope/cutoff/feature version trong manifest nội bộ. Response công khai chỉ trả metadata tối thiểu và thẻ sản phẩm đã được lọc.

### 1.2. Lỗi và HTTP

| HTTP / code | Khi nào | Client |
| --- | --- | --- |
| `400 INVALID_RECOMMENDATION_QUERY` | Query sai kiểu/placement, SKU không thuộc product hoặc quá giới hạn | Sửa request, không retry tự động |
| `404 PRODUCT_NOT_FOUND` | Product mốc không có trong catalog công khai hiện hành | Ẩn dãy và để trang chi tiết xử lý trạng thái product |
| `401 UNAUTHENTICATED` | Credential đã gửi cho home nhưng sai/hết hạn | Cập nhật phiên rồi có thể gọi lại dạng guest |
| `403 CUSTOMER_REQUIRED` | Employee hoặc principal chưa active dùng API cá nhân | Ẩn phần cá nhân |
| `429 RATE_LIMITED` | Vượt budget | Tôn trọng `Retry-After` |
| `503 RECOMMENDATION_SOURCE_UNAVAILABLE` | Catalog/Inventory thiếu hoặc quá hạn | Ẩn dãy |
| `503 RECOMMENDATION_TEMPORARILY_UNAVAILABLE` | PostgreSQL hoặc serving data gặp lỗi | Ẩn dãy |
| `503 IDENTITY_UNAVAILABLE` | Cần xác thực credential nhưng Identity tạm lỗi | Giữ trạng thái phiên, ẩn dãy |

Error body dùng `{code, message, requestId}` như Analytics hiện có. Mọi response có `X-Request-ID` và `Cache-Control: private, no-store`. API giữ cache nội bộ được version hóa theo chương 15. Tránh cache ở CDN khi dùng cùng route cho Customer và guest.

Cart rỗng trả `200 EMPTY`. Cart chứa SKU không còn công khai vẫn được loại khỏi context phục vụ; nếu hết anchor thì dùng fallback hợp lệ. Lookup product/SKU thực hiện từ projection nội bộ đã kiểm freshness.

## 2. Consent PERSONALIZATION

```http
GET /api/v1/recommendations/consent
PUT /api/v1/recommendations/consent
```

Cả hai yêu cầu Customer active từ Identity. GET trả state và ETag theo `consentVersion`, mặc định disabled/version 0 khi chưa có record. PUT dùng `If-Match` với version hiện hành và `Idempotency-Key` UUID.

```json
{
  "enabled": true,
  "policyVersion": "personalization-v1"
}
```

```json
{
  "enabled": true,
  "consentVersion": 1,
  "policyVersion": "personalization-v1",
  "grantedAt": "2026-11-07T00:00:00Z",
  "revokedAt": null
}
```

Policy version phải khớp version server đang cho phép. Chưa gửi `If-Match` trả `428 CONSENT_PRECONDITION_REQUIRED`, version sai trả `412 CONSENT_VERSION_CONFLICT`. Web GET lại state trước khi thử thao tác mới. Cùng key/body trả state hiện tại cùng kết quả duplicate và không tạo lại thay đổi. Key/body khác trả `409 IDEMPOTENCY_CONFLICT`.

Chuyển disabled → enabled tăng version và đặt `grantedAt=serverNow`. enabled → disabled tăng version, đặt `revokedAt`, tăng privacy epoch và tạo tác vụ xóa dữ liệu cá nhân trong cùng transaction. Gửi lại cùng state giữ nguyên mốc đồng ý. Đồng ý lại sau khi rút tạo lần grant mới và history bắt đầu từ lần mới.

Recommendation consent được ghi bền tại MS-11. UI có thể đồng bộ consent đo GA4 bằng cơ chế Marketing riêng. Khi policy cá nhân hóa đổi, consent policy cũ không còn đủ để xử lý history cho policy mới.

## 3. API hành vi first-party

```http
POST /api/v1/recommendations/events
```

Nhận một event mỗi request, tối đa 4 KiB, Customer active và consent đang bật. Tối đa 120 request/phút/principal, đồng thời giới hạn theo IP ở Gateway.

```json
{
  "eventId": "60000000-0000-4000-8000-000000000001",
  "eventType": "ADD_TO_CART",
  "skuCode": "MX-DEO-450G",
  "occurredAt": "2026-11-07T00:01:00Z",
  "consentVersion": 1,
  "recommendationId": "40000000-0000-4000-8000-000000000001"
}
```

`eventType` chỉ nhận VIEW_ITEM/ADD_TO_CART. `recommendationId` tùy chọn, là ngữ cảnh tương tác với dãy đã hiển thị. `eventId` UUID và field lạ bị từ chối. Không có trường subject trong request. Server suy ra principal từ phiên và chuyển sang subject key.

- `occurredAt` phải nằm trong 24 giờ vừa qua, từ `grantedAt` hiện hành trở đi. Tương lai tối đa 5 phút được clamp về `receivedAt` thành `effectiveOccurredAt` cho feature và ghi clock-skew flag, vượt ngưỡng trả `400 INVALID_EVENT_TIME`. Digest dedup dùng timestamp client đã chuẩn hóa UTC, giữ nguyên khi retry, cùng consentVersion và các field payload. `receivedAt` và thời gian sau clamp được lưu riêng.
- SKU phải tồn tại trong projection Catalog. Event có consentVersion cũ hoặc đã revoke trả `409 CONSENT_VERSION_CONFLICT`. Employee trả `403`.
- Dedup key là subject key/eventId, digest tính trên payload chuẩn hóa. Cùng digest trả `200 {status: DUPLICATE}`, khác digest trả `409 EVENT_ID_CONFLICT`.
- Event mới được commit vào PostgreSQL trước `202 {status: ACCEPTED}`. Revoke và ingest đồng bộ bằng lock consent row để event không được chèn sau lần revoke đã commit.
- ADD_TO_CART được Web phát sau Cart API thành công. Vì đây là tín hiệu từ client, feature áp cap theo subject/SKU/ngày và không dùng nó để chứng minh một giao dịch mua.

Web gửi trong nền, retry tối đa ba lần với cùng eventId cho lỗi transport/5xx, có jitter. `4xx` xử lý theo code. Event queue chỉ tồn tại trong memory của tab, được xóa khi mất consent hoặc đăng xuất.

## 4. Xác thực, scope và quyền DB

MS-11 bổ sung Customer auth dựa vào Identity session/Bearer đã được cấu hình cho môi trường. Tái sử dụng logic xác minh phiên và namespace hiện có ở Web/Identity, tách policy Customer khỏi `PilotAuth` staff của Dashboard. Request home có credential phải được xác minh trước khi truy cập profile.

BFF chuyển credential bằng helper hiện có, chỉ forward cookie `OMAMA_SESSION` hoặc Bearer theo mode và Origin cấu hình server. Gateway loại header danh tính/scope từ client. MS-11 chỉ tin scope do cấu hình route và deployment cung cấp. GET liên quan sản phẩm và cart context có thể dùng public vì context chỉ gồm SKU công khai.

Consent/event write đi qua BFF cùng origin. GET consent thành công đặt cookie CSRF `oma_rec_csrf` nếu chưa có, giá trị ngẫu nhiên 32 byte base64url, `SameSite=Strict`, path `/` và tuổi tối đa 24 giờ. Dùng `Secure` trên HTTPS, còn dev localhost HTTP áp cơ chế cookie session của Web. Cookie được Web đọc để gửi `X-CSRF-Token` khi PUT/POST và xóa khi đăng xuất. BFF so khớp token/cookie và Origin chính xác với public Web origin, whitelist đường dẫn và giới hạn body. MS-11 cũng kiểm Origin allowlist cho cookie-auth mutation, phiên/consent, body size và rate limit. Bearer request dùng xác thực token của môi trường.

Role API Recommendation được SELECT projection/serving và chỉ INSERT/UPDATE các bảng consent/event/decision cần thiết. Role sync/worker ghi staging, feature và generation. Migration role riêng tạo bảng và cấp quyền. Cấu hình `migrate.py` hiện cấp SELECT cho role API tổng quát, vì vậy quyền ghi mới phải được giới hạn theo bảng và theo role mới.

## 5. Gateway và BFF

| Route Web | Upstream MS-11 |
| --- | --- |
| `GET /api/recommendations` | `GET /api/v1/products/recommended` |
| `GET/PUT /api/recommendations/consent` | Consent API |
| `POST /api/recommendations/events` | Event API |

Kong bổ sung exact path `GET /api/v1/products/recommended` có ưu tiên hơn `/api/v1/products` của Catalog. Hai route consent/events đi dưới prefix `/api/v1/recommendations`. Kiểm tra method, dấu slash cuối và query để route Catalog không nhận nhầm request gợi ý. Chốt `recommended` là slug dành riêng trước khi mở.

BFF gọi MS-11 qua base URL/Gateway được cấu hình riêng, timeout 1 giây, `redirect: manual` và `cache: no-store`. Không proxy qua Catalog handler hiện đang hiểu `/products/{slug}`. Client nhận error envelope thống nhất với `requestId`.

Rate limit Gateway cho query là 60 request/phút/IP trên dev. API còn áp 60 request/phút/subject cho home đã xác thực, 120 event/phút/subject và 10 consent write/phút/subject. Rate limit nhiều replica đặt ở Gateway hoặc shared limiter, local memory limiter chỉ dùng test một process.

## 6. Tích hợp ba placement

| Màn hình | Request và render | Trạng thái thiếu dữ liệu |
| --- | --- | --- |
| Trang chi tiết | Sau load product, gửi productId và SKU đang được chọn. Thay SKU hủy request cũ. Hiển thị “Sản phẩm tương tự” | Fallback vẫn có thể hiển thị với thẻ đúng Catalog |
| Giỏ hàng | Gửi tập SKU giỏ hiện hành, debounce 200 ms và hủy request cũ khi giỏ đổi | Không render khi giỏ rỗng; fallback ghi nhãn “Gợi ý thêm cho bạn” |
| Trang chủ | Sau xác định trạng thái phiên, gọi home với credential hiện hành | Guest/chưa đồng ý/history rỗng hiển thị “Sản phẩm nổi bật” |

Dùng component dãy chung với loading độc lập, tối đa sáu thẻ, lazy image và placeholder. Khi rỗng/lỗi/timeout, dãy được ẩn. Trang sản phẩm và giỏ tiếp tục luồng riêng. Kết quả chỉ render nếu vẫn khớp context request hiện hành.

Thẻ dẫn tới slug công khai, mang SKU gợi ý làm ngữ cảnh hiển thị. Trang chi tiết vẫn yêu cầu chọn rõ SKU theo policy UI hiện có. Nút thêm giỏ nếu có phải xác nhận SKU/số lượng và đi qua Cart service.

## 7. GA4 và đánh giá hiệu quả

Storefront phát `view_item_list` khi ít nhất 50% dãy nằm trong viewport liên tục 1 giây, một lần mỗi recommendationId trên trang. `select_item` phát khi khách chọn thẻ. `item_list_id` dùng giá trị ổn định theo placement, ví dụ `rec_product_detail_v1`. `items[].item_id` là skuCode và `index` là vị trí. Quy ước visibility là lựa chọn pilot của dự án. [GA4 Ecommerce](https://developers.google.com/analytics/devguides/collection/ga4/ecommerce)

Lưu provenance recommendationId/placement/SKU trong memory phiên Web sau click. Khi khách thêm đúng SKU trong cùng phiên, gắn provenance vào event add-to-cart và decision trace được phép thu thập. Chỉ giữ provenance tối đa 30 phút. `recommendationId` là giá trị trace, không đăng ký thành custom dimension có cardinality lớn.

Consent đo GA4 được Marketing kiểm trước khi gửi. Request gợi ý ngữ cảnh có thể chạy khi consent đo lường bị tắt. Subject key, Customer ID, email, địa chỉ và token được giữ trong các luồng nội bộ có quyền. URL đo lường được lọc thông tin nhận dạng. [Chính sách dữ liệu GA4](https://support.google.com/analytics/answer/6366371)

Chỉ số chính là tỷ lệ phiên đủ điều kiện có thêm ít nhất một SKU gợi ý vào giỏ. Theo dõi riêng lượt thấy, click, render coverage, fallback, lỗi và latency. Pilot dev kiểm đúng tracking và thuật toán bằng fixture đã đối soát. A/B trên traffic thật chỉ bật sau khi Marketing chốt cỡ mẫu, kỳ đo và tiêu chí lựa chọn.
