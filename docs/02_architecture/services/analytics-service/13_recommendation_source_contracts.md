# 13 — Dữ liệu nguồn cho Recommendation Engine

**Hợp đồng đề xuất v1 · 07/11/2026.** Pilot dev ưu tiên owner xuất file theo lịch, cùng mẫu vận hành đã dùng cho Supply. API snapshot ở mục 3.2 là phương án mở rộng. [Chương 12](12_recommendation_engine_design.md) xác định thuật toán và scope sử dụng.

## 0. Đánh giá khả thi từ code và phần phải xác minh

Đây là đánh giá tĩnh của checkout `feat/analysis-service`. Cần source fixture và chạy đối soát trên DB owner trước khi khẳng định producer sẵn sàng. Schema có trường dữ liệu chỉ chứng minh khả năng trích xuất, còn độ đầy đủ của dữ liệu dev phụ thuộc môi trường.

| Nguồn | Mức công việc ước lượng | Bằng chứng và giới hạn |
| --- | --- | --- |
| Catalog metadata | Nhẹ đến vừa | Category/SKU/packaging/weight/channel price đã có. CheckoutSnapshotService đã có query chọn giá hiệu lực và đổi gram. Cần một query/export adapter nhất quán và test Product bị ẩn. |
| Inventory sellability | Vừa, cần owner xác nhận | Terminal reserve ghi allocation và tăng reserved theo batch, commit/release cập nhật batch/item. Đã có SQL xuất lô Supply. Cần đối soát các đường chạy thật và sửa phạm vi kho ở đường đọc nếu dùng API. |
| Order basket anonymous | Nhẹ đến vừa | orders/order_items/order_milestones có sẵn, SQL Supply đã join cùng các bảng. Có thể xuất basket theo đơn trong transaction read. |
| Order principal cho mua cá nhân | Vừa, mở sau khi chứng minh namespace | orders có owner_kind/principal_id và commercial snapshot được kiểm tại placement. Cần fixture chứng minh khớp principal Identity; chưa đạt thì trường Customer để null. |
| fallbackRank | Chưa thấy lưu trong Catalog | Pilot dùng config SKU/rank tại MS-11 được Catalog duyệt, có policy version. Trường source có thể null. |
| eligibilityReason / identityMappingVersion | Metadata mới của exporter | Sinh từ policy export đã nghiệm thu và cấu hình loại dữ liệu test. Bản ghi chưa chứng minh mapping giữ principal null. |
| Hành vi/consent | Phần triển khai mới ở MS-11 và Web | Owner cung cấp phiên Customer hiện hành, còn collector, history, policy và model do Analytics/Web thực hiện. |

Mức công việc mô tả phạm vi kỹ thuật tương đối. Đợt đầu giữ toàn bộ contract record và kiểm checksum, đồng thời triển khai transport file nhỏ trước. Các ngưỡng freshness 60/120 giây của Inventory cần đo trực tiếp với job và DB owner, rồi mới bật route trên dev.

## 1. Bằng chứng hiện có và phần cần bổ sung

| Owner | Đã quan sát trong repo | Cần cung cấp thêm |
| --- | --- | --- |
| Catalog | Model có category ID, Product/SKU version, `packagingType`, khối lượng và giá kênh. API public có list/detail nhưng detail adapter Web còn thiếu một số metadata | Snapshot nhất quán theo scope có trạng thái công khai, metadata chuẩn hóa, giá D2C và fallback rank |
| Inventory | Có kho trên `inventory_items`, batch/reservation và API đọc stock. Handler gRPC hiện gọi use case bằng danh sách SKU, chưa truyền `warehouse_id` | Snapshot bán được theo đúng kho và lô, giới hạn độ mới, completeness cho cả SKU bằng 0 |
| Order | Có `order_items`, milestone `COMPLETED`, snapshot nghiệp vụ có channel/owner và cơ chế giải mã ở owner | Export basket theo đơn/kho/cohort, Customer mapping đã xác minh, revision và coverage toàn cửa sổ |
| Supply trong MS-11 | Có bản xuất `order_day`, `inventory_day`, `inventory_batches`, revision và inbox sync | Mapping scope D2C tới mã kho Supply, đọc 14 ngày đủ coverage trước khi dùng bán ròng |
| Identity / Profile | Web có `OMAMA_SESSION`, Identity profile và mapping Customer | Xác nhận principal namespace dùng xuyên suốt, trạng thái Customer hiện hành và mapping khách từ Order |
| Storefront | Luồng chọn SKU, thêm giỏ và phiên Customer đã có | Consent cá nhân hóa, events theo allowlist và ngữ cảnh Recommendation |

[Catalog controller](../../../../services/catalog-service/src/main/java/com/hvduong/catalog/api/CatalogHttpController.java) · [Catalog variant model](../../../../services/catalog-service/src/main/java/com/hvduong/catalog/domain/entity/ProductVariant.java) · [Inventory handler](../../../../services/inventory-service/internal/presentation/grpc/handler.go) · [Order snapshot](../../../../services/order-service/internal/domain/core/snapshot.go) · [Supply v1](09_supply_analytics_v1.md)

## 2. Scope và định danh

Scope ban đầu ánh xạ một kênh D2C và một kho nguồn. Ví dụ cấu hình dev mang tính minh họa.

```json
{
  "scopeId": "d2c-hue",
  "channel": "D2C_WEB",
  "warehouseId": "a7277eb6-0fd2-4850-b640-169bc446387e",
  "supplyWarehouseId": "warehouse-supply-pilot",
  "orderCohort": "order-core-1",
  "mappingVersion": "scope-v1"
}
```

`scopeId` đến từ cấu hình server cho Web, source caller và MS-11. `warehouseId` là UUID owner xác nhận. `supplyWarehouseId` chỉ dùng nối vào Supply đã có. Thay mapping tạo scope/version mới và rebuild trước khi phục vụ.

SKU là mã canonical của Catalog, giữ nguyên case và phải khớp chính xác ở Inventory/Order. Importer cách ly SKU lạ, ghi số bản ghi không map được và chặn model nếu source completeness không còn chứng minh được. Product/variant/Order/principal ID là UUID namespace của owner. Customer ID của Profile chỉ được chuyển thành Identity principal qua mapping được xác minh.

## 3. Giao thức snapshot chung

Mỗi owner tạo dataset trong môi trường của mình, bằng script/command được owner quản lý và DB role chỉ đọc. Analytics đọc bản xuất đã publish. Dataset và payload dùng cùng ngữ nghĩa cho transport file hoặc HTTP.

| Owner | Dataset | Kỳ |
| --- | --- | --- |
| Catalog | `catalog` | Toàn bộ Product/SKU trong catalog D2C của scope |
| Inventory | `sellability` | Trạng thái của toàn bộ SKU được quản lý tại kho scope |
| Order | `completed_baskets` | Cửa sổ 90 ngày, từ `from` bao gồm tới `to` loại trừ |

### 3.1. Transport file cho pilot dev

Owner bổ sung một query/export command cho dataset của mình. Có thể dùng publisher helper chung để chuẩn hóa JSONL/manifest, tính checksum và publish atomic. Owner giữ SQL/schema knowledge, credential DB và scope allowlist trong môi trường của mình.

```text
/run/recommendation-exports/
  catalog-service/<snapshotId>/records.jsonl
  catalog-service/<snapshotId>/manifest.json
  inventory-service/<snapshotId>/records.jsonl
  inventory-service/<snapshotId>/manifest.json
  order-service/<snapshotId>/records.jsonl
  order-service/<snapshotId>/manifest.json
```

Exporter chụp dữ liệu bằng một transaction read nhất quán, ghi thư mục staging, fsync data rồi ghi manifest sau cùng và rename atomic trên cùng filesystem. Worker chỉ đọc thư mục đã publish có manifest hợp lệ, từ chối symlink và path nằm ngoài root cấu hình. Quyền ghi giới hạn theo folder owner, MS-11 mount read-only. Truyền file giữa các host cần job vận chuyển được cấu hình riêng trước khi nghiệm thu.

Manifest dùng các trường trong mục 3.2, thêm `transport=FILE_JSONL` và `dataFile=records.jsonl`. `pageCount` để null ở transport này. records.jsonl áp canonical format/digest của mục 3.3. Pilot giới hạn file 64 MiB và count theo budget chương 12, vượt mức giữ source cũ và báo lỗi. Reader đọc streaming để kiểm byte/count/digest trước publish.

Owner state lưu bền revision counter, khóa một publisher cho mỗi scope/dataset và cấp revision tăng dưới lock. Sau crash có thể bỏ trống revision nhưng không tái sử dụng revision cũ. Cùng snapshotId và digest là duplicate, khác digest là conflict. Xóa/khởi tạo lại owner state yêu cầu đối soát revision với manifest MS-11 trước khi mở lại job.

Lịch vẫn là Catalog 300 giây, Inventory 60 giây, Order 900 giây. Giữ cùng timestamp/checksum/completeness và nguyên tắc full replacement của mục 3.3. Tái sử dụng mẫu owner publisher và read-only inbox từ [Supply publisher](../../../../scripts/publish-analytics-supply.py) và [Supply sync](../../../../services/analytics-service/src/analytics_service/supply_sync.py), bổ sung parser/schema riêng cho bulk Recommendation.

### 3.2. HTTP snapshot khi mở rộng transport

API dưới đây phục vụ đợt mở rộng cần truy xuất từ xa và chia trang. Chỉ triển khai khi transport file đã gây hạn chế vận hành được đo và source owner chốt scope thay đổi.

```http
POST /internal/analytics/recommendation-snapshots
Idempotency-Key: <UUID của lần đồng bộ>
X-Internal-Token: <credential export do owner cấp>
Content-Type: application/json
```

```json
{
  "schemaVersion": "1",
  "dataset": "completed_baskets",
  "scopeId": "d2c-hue",
  "channel": "D2C_WEB",
  "warehouseId": "a7277eb6-0fd2-4850-b640-169bc446387e",
  "from": "2026-08-09T00:00:00Z",
  "to": "2026-11-07T00:00:00Z"
}
```

`from`/`to` chỉ có ở Order, là timestamp UTC và khoảng cách tối đa 90 ngày. Catalog/Inventory từ chối các trường kỳ. Owner cấu hình token export riêng, ánh xạ token sang allowlist scope/dataset. Source caller chỉ được đọc phạm vi đã cấp. Body có field lạ hoặc dataset sai owner trả `400`.

POST trả `202` với `snapshotId`, `state=BUILDING` và `Location`. Cùng idempotency key và cùng body trả lại job cũ, khác body trả `409`. Mỗi scope/dataset chỉ có một job BUILDING. Caller gặp job đang chạy nhận `409 EXPORT_IN_PROGRESS` và tham chiếu job đó.

```http
GET /internal/analytics/recommendation-snapshots/{snapshotId}
GET /internal/analytics/recommendation-snapshots/{snapshotId}/pages?cursor=<opaque-token>
```

Status GET trả `BUILDING`, `READY` hoặc `FAILED`. Owner giới hạn thời gian BUILDING tối đa 5 phút, sau đó ghi FAILED/EXPORT_TIMEOUT và giải phóng job scope. Khi READY, manifest có các trường sau.

| Trường | Kiểu | Ý nghĩa |
| --- | --- | --- |
| `schemaVersion`, `dataset`, `source` | string | Phiên bản và owner |
| `snapshotId` | UUID | Định danh bất biến của bản xuất |
| `sourceRevision` | int64 dương | Số tăng bền theo owner/dataset/scope, cấp tại thời điểm capture |
| `scopeId`, `channel`, `warehouseId` | string | Scope đã được owner đối chiếu |
| `sourceAsOf` | UTC timestamp | Mốc read snapshot nhất quán |
| `window` | object hoặc null | `from`/`to` của Order |
| `coverageStatus` | `COMPLETE` | Đã xuất toàn bộ tập đủ điều kiện trong scope/kỳ |
| `policyVersion` | string | Quy tắc lọc nghiệp vụ của owner |
| `recordCount` | integer >= 0 | Tổng record |
| `pageCount` | integer >= 0 hoặc null | Tổng trang HTTP; null với FILE_JSONL |
| `dataSha256` | 64 ký tự hex thường | Digest toàn bộ canonical records theo thứ tự |
| `publishedAt`, `expiresAt` | UTC timestamp | Mốc publish bản hoàn chỉnh và deadline mặc định sau 24 giờ |
| `coverageEvidenceRef` | string | Tham chiếu tới count/đối soát tại owner |

Mỗi trang có `snapshotId`, `pageNumber` bắt đầu từ 0, `records`, `pageSha256` và `nextCursor`. Trang đầu bỏ cursor. Trang tối đa 500 record và 2 MiB. Cursor gắn scope, snapshot và trang kế tiếp. `410 SNAPSHOT_EXPIRED` yêu cầu tạo snapshot mới. Caller tạo job mới sau lỗi capture/retry không thể tiếp tục.

### 3.3. Tính nhất quán và checksum

Owner capture trong transaction read nhất quán và materialize dữ liệu bất biến trước khi READY. Phân trang đọc tập đã materialize, giữ nguyên `sourceAsOf` và thứ tự trong mọi lần retry.

Record có `recordKey` ASCII duy nhất. Catalog/Inventory dùng `sku:<skuCode>`, Order dùng `order:<orderId>`. Sort record theo `recordKey` tăng dần theo byte UTF-8. Canonical JSON dùng object keys sắp xếp đệ quy, UTF-8 không escape Unicode, không khoảng trắng, integer cho quantity/money và một LF sau mỗi record. Mảng đã chuẩn hóa trước khi hash, đặc biệt Order sort item theo SKU. `pageSha256` hash concatenation canonical records trong trang. `dataSha256` hash cùng stream canonical records của mọi trang. Snapshot rỗng có checksum SHA256 của chuỗi byte rỗng; `pageCount=0` với HTTP và null với FILE_JSONL.

Importer kiểm count, page digest, data digest, recordKey unique, scope và required fields trước khi cập nhật pointer. Snapshot mới hơn chỉ publish nếu `sourceRevision` cao hơn và `sourceAsOf` không lùi. Cùng snapshot ID/digest là duplicate. Cùng ID hoặc revision nhưng khác digest là conflict cần owner xử lý. Tương lai tối đa 5 phút chờ đồng bộ clock, vượt mức thì quarantine.

Snapshot là full replacement của dataset/scope/window. Omission có hiệu lực sau khi đã chứng minh COMPLETE. Catalog/Inventory record bị bỏ khỏi snapshot mới lập tức mất quyền tham gia gợi ý. Model Order 90 ngày được dựng lại theo tập mới để áp correction và evict đơn hết cửa sổ. Snapshot incomplete chỉ giữ ở staging.

Owner giữ idempotency metadata và snapshot sẵn sàng tối thiểu 24 giờ. Analytics giữ journal digest lâu hơn thời gian retry, theo chương 15. Source job chạy bằng role chỉ đọc dữ liệu nguồn và chỉ ghi storage bản xuất riêng.

## 4. Catalog — snapshot thương mại

Mỗi record là một SKU cùng dữ liệu product cần phục vụ. Export gồm SKU hiện hữu trong catalog D2C của scope, kể cả SKU vừa bị ẩn để phép lọc rõ ràng.

| Trường | Bắt buộc | Ngữ nghĩa |
| --- | --- | --- |
| `skuCode`, `productId`, `categoryId` | Có; category có thể null | Khóa Catalog canonical |
| `productVersion`, `skuVersion`, `priceVersion` | Có; price có thể null | Version dữ liệu thương mại từ owner |
| `publiclyVisible`, `skuActive` | Có | Kết quả quy tắc approval/listing của Catalog |
| `slug`, `name`, `thumbnailUrl` | Có khi visible | Nội dung đã được duyệt để hiển thị |
| `priceVnd` | Có thể null | Integer VND dương nếu đang bán; giá của `D2C_WEB` tại cutoff |
| `weightGrams`, `packagingType` | Có thể null | Khối lượng nguyên chuẩn hóa và mã bao bì canonical |
| `fallbackRank` | Có thể null | Integer dương, do Catalog duyệt theo scope/placement |
| `sourceUpdatedAt` | Có | Mốc thay đổi record tại owner |

```json
{
  "recordKey": "sku:MX-DEO-450G",
  "skuCode": "MX-DEO-450G",
  "productId": "10000000-0000-4000-8000-000000000001",
  "categoryId": "20000000-0000-4000-8000-000000000001",
  "productVersion": 7,
  "skuVersion": 3,
  "priceVersion": 4,
  "publiclyVisible": true,
  "skuActive": true,
  "slug": "me-xung-deo",
  "name": "Mè xửng dẻo",
  "thumbnailUrl": "https://cdn.omama.vn/products/mx-deo.jpg",
  "priceVnd": 95000,
  "weightGrams": 450,
  "packagingType": "HOP_GIAY_KRAFT",
  "fallbackRank": 1,
  "sourceUpdatedAt": "2026-11-07T00:00:00Z"
}
```

Catalog phải giải quyết giá hiệu lực ở cutoff và conversion `KILOGRAM → GRAM` bằng số học chính xác. Giá/mapping thiếu làm SKU không đủ điều kiện phục vụ. Thumbnail dùng ảnh placeholder đã cấu hình ở Web nếu Catalog chưa có ảnh hợp lệ. Category thiếu vẫn có thể vào fallback nhưng không tham gia điểm cùng danh mục.

Fallback rank áp dụng cùng danh sách cho ba placement trong pilot. Source có thể xuất null vì chưa có bằng chứng field này được lưu trong Catalog. MS-11 nhận file policy `scopeId/skuCode/rank/policyVersion` do Catalog duyệt, áp rank cấu hình trước rank source và pin policy digest vào generation. Thứ tự null đứng cuối, tie-break bằng SKU. Product có nhiều variant được gộp ở MS-11. Source giữ `packagingType` như mã nghiệp vụ và danh sách mã/version để hai SKU có thể so sánh.

## 5. Inventory — khả dụng theo kho

| Trường | Ý nghĩa |
| --- | --- |
| `skuCode`, `warehouseId` | SKU và kho đúng scope |
| `skuActive` | Trạng thái SKU cho phép bán tại kho |
| `sellableUnits` | Integer >= 0 đã loại reserved, batch bị chặn và hết hạn |
| `eligibilityValidUntil` | Deadline cần kiểm lại tính khả dụng |
| `observedAt` | Bằng sourceAsOf, thời điểm quan sát nhất quán SKU/lô/reservation |

```json
{
  "recordKey": "sku:MX-DEO-450G",
  "skuCode": "MX-DEO-450G",
  "warehouseId": "a7277eb6-0fd2-4850-b640-169bc446387e",
  "skuActive": true,
  "sellableUnits": 18,
  "eligibilityValidUntil": "2026-11-07T00:02:00Z",
  "observedAt": "2026-11-07T00:00:00Z"
}
```

Source tính theo `inventory_items.warehouse_id` rồi join batch của đúng SKU. Schema hiện key SKU toàn cục, batch tham chiếu SKU. Dùng quan hệ này để xác định kho, đồng thời kiểm các invariant dữ liệu trước khi xuất.

`sellableUnits` lấy tổng `physical_qty − reserved_qty` của batch ACTIVE/NEAR_EXPIRY, `exp_date > businessDate`, chỉ khi inventory item cho phép bán. Đối chiếu lượng tổng với inventory item và allocations theo cùng transaction. Batch-level reserved chưa được owner chứng minh đầy đủ thì export FAILED để chặn suy ra sellability sai.

`eligibilityValidUntil = min(sourceAsOf + 120 giây, đầu ngày Việt Nam kế tiếp)`. Ngày `exp_date` là ngày đầu batch hết quyền bán theo hợp đồng Supply. Record SKU không bán được vẫn có mặt với số 0 và lý do ở evidence nội bộ. SKU bị bỏ khỏi snapshot hoàn chỉnh mới nhận trạng thái UNKNOWN ở MS-11 và bị loại khỏi kết quả.

## 6. Order — basket đã hoàn tất

Snapshot lấy mọi đơn có milestone COMPLETED trong kỳ được yêu cầu và cohort/kho/kênh đúng scope. Owner xác nhận completeness từ DB của mình, đối chiếu số distinct order và số SKU lines. Basket anonymous có thể lấy bằng join orders/order_items/order_milestones tương tự SQL Supply. Cột channel và owner trong projection Order chỉ được dùng sau khi đối chiếu cohort với placement và commercial snapshot. Khi cần kiểm sâu, owner dùng hàm đọc/giải mã hiện có của Order.

Pilot basket có thể bắt đầu với customerPrincipalId/identityMappingVersion null. Sau khi fixture Customer chứng minh quan hệ principal Identity, exporter bổ sung các trường này từ owner_kind/principal_id và policy mapping được duyệt. Lịch sử lượt xem/thêm giỏ cá nhân hóa vẫn được MS-11/Web thu độc lập theo consent.

| Trường | Ngữ nghĩa |
| --- | --- |
| `orderId`, `orderVersion` | Khóa và version Order |
| `cohort`, `channel`, `warehouseId` | Scope nghiệp vụ |
| `completedAt` | Mốc hoàn tất có thẩm quyền |
| `eligible`, `eligibilityReason` | Owner xác nhận đơn được dùng hoặc lý do bị rút |
| `customerPrincipalId` | UUID Identity đã mapping hoặc null cho Guest/chưa chứng minh mapping |
| `identityMappingVersion` | Version mapping, bắt buộc khi principal có giá trị |
| `items` | Tập SKU distinct cùng `quantityAtCompletion` nguyên dương |

```json
{
  "recordKey": "order:30000000-0000-4000-8000-000000000001",
  "orderId": "30000000-0000-4000-8000-000000000001",
  "orderVersion": 12,
  "cohort": "order-core-1",
  "channel": "D2C_WEB",
  "warehouseId": "a7277eb6-0fd2-4850-b640-169bc446387e",
  "completedAt": "2026-11-06T08:00:00Z",
  "eligible": true,
  "eligibilityReason": null,
  "customerPrincipalId": null,
  "identityMappingVersion": null,
  "items": [
    {"skuCode": "MX-DEO-450G", "quantityAtCompletion": 2},
    {"skuCode": "TRA-SEN-100G", "quantityAtCompletion": 1}
  ]
}
```

Basket đang được dùng rồi bị owner xác nhận sai được xuất `eligible=false` hoặc bỏ khỏi bản replacement COMPLETE tiếp theo. Analytics loại nó khỏi tập thống kê và dựng generation mới. Cùng order ID xuất hiện hai lần trong một snapshot là conflict.

Đơn test và đơn không có căn cứ cohort D2C được loại tại owner theo policy version. Pilot thực tế cần danh sách dữ liệu test được nhận diện rõ trong cấu hình dev và evidence. Backfill từ cohort legacy chỉ mở khi có adapter và đối soát riêng.

`customerPrincipalId` chỉ có trong truyền tải nội bộ và storage owner được bảo vệ. Worker MS-11 chuyển ngay sang subject HMAC, kiểm consent và bỏ principal thô trước khi lưu projection. Purchase cá nhân chỉ tạo khi `completedAt >= grantedAt`, trong 30 ngày và consent còn hiệu lực. Aggregate basket dùng order ID và SKU phục vụ thống kê scope.

## 7. Identity, Profile, consent và hành vi

| Bên cung cấp | Cần xác nhận hoặc triển khai |
| --- | --- |
| Identity | `/api/v1/users/me` từ session/Bearer hiện hành trả `principalId`, `principalType=CUSTOMER`, `provisioningStatus=ACTIVE`; cùng namespace với principal trên Order |
| Profile | Mapping Customer ID ↔ principal ID có version nếu dữ liệu Order còn dùng Customer ID khác namespace |
| MS-11 | Consent PERSONALIZATION có version, `grantedAt`, `revokedAt`, pseudonym key version và privacy epoch |
| Web | Hiển thị opt-in rõ ràng, gửi VIEW_ITEM/ADD_TO_CART sau kiểm consent; SKU và thời điểm, không gửi subject do client tự khai |

Web gửi ADD_TO_CART sau khi Cart API thành công. Đây là implicit feedback có mức tin cậy của client, có rate limit và cap đóng góp. PURCHASE chỉ được tạo từ Order. Identity gặp lỗi tạm thời trả unavailable để retry, phiên Employee bị từ chối cho API cá nhân hóa.

Guest claim hoặc merge tài khoản chưa nằm trong policy liên kết lịch sử của pilot. Khi cần mở, Profile phải cung cấp effective time và version mapping trước khi rebuild dữ liệu cá nhân. [Identity contract](../../../../packages/contracts/identity-3.3.md) · [BFF credential hiện có](../../../../apps/web-user/src/lib/auth/server-credential.ts)

## 8. Checklist bàn giao cho từng owner

| Owner | Bằng chứng nghiệm thu cần giao |
| --- | --- |
| Catalog | Manifest/JSONL fixture, conversion khối lượng, giá đúng kênh, Product bị ẩn, giá thiếu và fallback config được duyệt |
| Inventory | Query đúng kho, reserved theo batch, lô hết hạn/quarantine, SKU bằng 0, hết hạn tại biên ngày và đối chiếu lượng bán được |
| Order | Export 90 ngày, count basket khớp DB, correction đơn, test cohort, snapshot owner mapping và fixture Guest/Customer |
| Identity/Profile | Một Customer active, một Employee, phiên hết hạn và bằng chứng namespace principal của Order |
| Web/Marketing | Consent riêng cho cá nhân hóa và đo GA4, event đúng điểm phát, dãy hiển thị/chọn/add-to-cart theo placement |
| MS-11 | Duplicate/conflict/revision, completeness, freshness và build/replay cùng input digest cho cùng output |

Transport pilot là bulk JSONL/manifest do owner publish theo lịch. HTTP job/cursor là phương án mở rộng cùng contract dữ liệu. Inbox Supply theo file cung cấp metric Supply riêng; Analytics nối hai nguồn bằng scope mapping đã cấu hình. [Persistence và lifecycle](15_recommendation_persistence_and_delivery.md)
