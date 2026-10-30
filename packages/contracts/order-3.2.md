> Integration entrypoint: [MVP contract guide](mvp/README.md). Shipping bound quotes now use `omamx.shipping.v2`; v1 retains the Shipping provider field numbers. Provider support is tracked in the MVP matrix.

# Order 3.2 — proposed integration contract

Ngày 15/10/2026. **Trạng thái: đề xuất để các service review/triển khai**, chưa xác nhận provider hỗ trợ. [HLD](https://github.com/hvduong2k5/DUT.PBL6/blob/c753a5201caa88fffa7caf29af5681030138a4ff/docs/02_architecture/services/order-service/00_order_service_hld.md) và [LLD contracts](https://github.com/hvduong2k5/DUT.PBL6/blob/c753a5201caa88fffa7caf29af5681030138a4ff/docs/02_architecture/services/order-service/05_api_contracts_and_transports.md) giải thích quyết định nghiệp vụ. Tài liệu này quy định semantics cho các wire files.

## 1. Phiên bản và capability gate

| Artifact | Thiết kế 3.2 | Runtime đã có |
| --- | --- | --- |
| Shared proto v1 | RPC/field thêm mới, không đổi tag/type/signature cũ | Inventory thật chưa có terminal protocol; simulator chỉ trả outcome cũ |
| `contract_version` | Request và response phải là `order-3.2` | Blank/khác version không đủ điều kiện tích hợp |
| Order OpenAPI | `docs/03_api_specs/order-service.openapi.yaml`, version 3.2.0, proposed | Snapshot `order-service.sandbox-3.1.openapi.yaml` dùng cho sandbox regression |
| Kafka facts | `order.events.v3`, types `vn.omama.order.{name}.v3` | v2 giữ nguyên cho runtime 3.1; v1 có legacy effect consumers |

Client 3.2 phải fail closed với UNIMPLEMENTED, unsupported version hoặc thiếu proof. Không downgrade sang legacy RPC/paid consumer sau timeout. Việc compile proto không chứng minh provider hỗ trợ profile này. Các field mới có default để wire compatibility; default rỗng/0 không phải proof hợp lệ. Provider chưa hỗ trợ phải trả UNIMPLEMENTED/FAILED_PRECONDITION, không giả success.

Voucher, completion, claim, POS, marketplace và refund thật vẫn feature gated theo LLD; các wire proposal ở đây không tự bật feature. Care/provider money chưa có signature hoàn chỉnh vẫn là gate mở, không được thay bằng một chuỗi approval tự khai.

## 2. Identity, keys và canonical hash

TLS + verified service identity/audience + per-method ACL. Principal/actor trong message chỉ là context, không cấp quyền. Inventory mutations chỉ cho Order orchestrator; permit RPC chỉ cho Fulfillment; cancellation barrier chỉ cho Order. Provider query cho Inventory xác minh decision cần ACL đọc đúng mục đích.

Key 16–128 ký tự ASCII hiển thị không có space (`0x21..0x7e`). Scope `(authenticated caller, action, order_id, key)`; ngoài command registry, unique business resource theo `order_id`/generation ngăn key mới tạo effect trùng. Cùng scope/key khác request_hash → ALREADY_EXISTS/IDEMPOTENCY_CONFLICT; không gọi effect. Key/hash/original outcome lưu bền vững, cache không phải registry. Retention response đề xuất 7 ngày, registry tối thiểu 30 ngày; không tái sử dụng order ID, tombstone phải tồn tại đủ để mọi late request không tạo resource mới.

`items_digest`: consolidate theo `sku_code`, sum quantity có kiểm overflow int32; SKU nonempty ASCII, quantity >0. Sort SKU theo byte ASCII. UTF-8 JSON không BOM/space/newline, keys trong từng item là `quantity`, `sku_code`, quantities là decimal integer. SHA-256 lowercase hex của array này. Ví dụ byte input: `[{"quantity":2,"sku_code":"MX-GION-500G"}]`. Digest không chứa warehouse; warehouse phải được validate riêng.

`request_hash`: SHA-256 lowercase hex của UTF-8 JSON business request đã normalize, snake_case field names, object keys sort ASCII, không whitespace, strings JSON escaped, không float; int64 là decimal JSON integer (implementations phải giữ precision). Bỏ `request_hash`, `correlation_id` và `idempotency_key`; giữ `contract_version` và mọi field nghiệp vụ khác. Omit scalar default `""`, false, 0, list rỗng và optional absent; optional có presence vẫn giữ kể cả giá trị 0. Nested messages recurse cùng luật; message empty omit. Reserve items consolidate/sort như trên trước hash; các list khác giữ thứ tự. Không dùng protobuf binary serialization làm canonical hash. Không thay payload/expected_version khi retry cùng command key; query dùng key/action gốc. Audit fields nếu có thuộc input và được hash. Public checkout hash scope riêng theo canonical input đã nhận, không dùng wire hash Inventory thay thế.

## 3. Evidence và lỗi

`CommandEvidence` mô tả quyết định command gốc: action dùng đúng tên RPC, key/hash/result/decision_id/decision_at/resulting_state. APPLIED lần đầu; ALREADY_APPLIED cho replay effect đã áp dụng; REJECTED kèm reason. `decision_id`/decision time gốc không đổi khi replay; resource state/version/observed_at bên ngoài là quan sát hiện tại, có thể đã tiến bước. Query không có command filter có thể không có command evidence; caller cần proof effect thì phải query đúng key/action. Query filter không tìm thấy command trả NOT_FOUND, không suy ra command chưa/không bao giờ chạy. Query không ghi key hay đổi trạng thái.

| Kết quả | gRPC / phản hồi |
| --- | --- |
| Malformed/version không hỗ trợ | INVALID_ARGUMENT hoặc FAILED_PRECONDITION; no effect |
| Auth/ACL fail | UNAUTHENTICATED/PERMISSION_DENIED |
| Key đổi hash | ALREADY_EXISTS, IDENTITY/HASH conflict; no new effect |
| State/expiry từ chối business command | gRPC OK + REJECTED reason + authoritative current resource state |
| Version/resource/digest không khớp | FAILED_PRECONDITION; no effect; caller query/review |
| Resource/command chưa thấy tại thời điểm đọc | NOT_FOUND; không phải terminal absence |
| Lock/transient/unavailable/deadline | ABORTED/UNAVAILABLE/DEADLINE_EXCEEDED; outcome caller UNKNOWN, retry/query same key |

Resource version tăng monotonic trên mỗi state change; observed_at là provider DB clock lúc đọc. Command success dù gRPC OK vẫn phải validate order/resource/digest/warehouse/version/timestamps và contract_version. REJECTED/empty evidence không phải success. Chỉ so sánh version trong cùng resource/service, không so với Order version.

## 4. Inventory

`ReserveStock`: tất cả SKU atomic, một warehouse. Reject TTL <=0 hoặc >15 phút/warehouse không hỗ trợ, không im lặng bỏ input. Response `outcome` bắt buộc trong profile 3.2, kể cả replay có state đã đổi. Stock reject không tạo reservation: status INSUFFICIENT/SKU_* + error/detail, outcome có thể absent; Order không coi đây là success. Same order/key/hash trả resource ban đầu + state hiện tại. Same order khác intent phải conflict. `preferred_warehouse_id` bắt buộc trong MVP một kho; request digest phải đúng items. Field `status=SUCCESS` legacy không cho phép suy ra resource hiện vẫn RESERVED.

Finalize/expire/release serialize cùng reservation lock và Inventory DB clock sau lock. `now < expires_at` mới finalize; equality expired. Replay command thành công xét registry trước expected_version/deadline; không đảo COMMITTED sau hạn. Finalize từ COMMITTED của đúng resource có thể ALREADY_APPLIED; RELEASED/EXPIRED trả REJECTED với state. Reverse từ COMMITTED dùng ledger riêng; không dùng ReleaseReservation để cộng lại stock committed.

Release-by-order trước reserve phải tạo durable tombstone và trả state RELEASED, cancellation_tombstone=true, command proof. reservation_id/expires_at/items_digest/warehouse có thể rỗng chỉ khi chưa có reservation; response vẫn có order_id/version/observed_at. Các expectations đã được caller biết phải được kiểm tra; absence không cho phép bỏ qua mismatch của resource đang có. Registry/order-key lock unique phải serialize cả trường hợp chưa có reservation row. Late reserve cùng order bị reject. NOT_FOUND query không thay thế tombstone ACK.

`ReverseCommittedStock`: MVP full reservation; approval và fulfillment_stopped DecisionReference bắt buộc, bind đúng order/generation/resource version. Inventory query `Fulfillment.GetTaskDecision` bằng order/generation và đối chiếu CANCELLED + decision ID/version, query approval authority theo giao thức được thống nhất. Không trust reference tự khai. ACCEPTED/UNKNOWN/decision không xác minh được → no reversal. Kết quả có state REVERSED, reversal_id và reversed_batches khớp toàn bộ allocation ban đầu. Quantity không cộng hai lần; batch hết hạn/quarantine không trở thành available stock. Refund không gọi reverse tự động. Query-by-order có query_action/key gốc phục hồi reversal mất ACK.

Legacy RPC ReleaseReservation giữ wire signature để các cohort cũ hoạt động; không dùng bool is_released làm proof trong luồng mới.

## 5. Fulfillment và Order permit

1. `Order.GetFulfillmentPermit`: Order lock, kiểm eligibility, hold, generation, digest; bind key/caller/audience=`fulfillment-service`, warehouse, policy, Order version, expiry (đề xuất tối đa 30s). Persist permit hoặc signed evidence có thể lookup; token không tự mang quyền user.
2. `Order.ValidateFulfillmentPermit`: xác minh permit thực sự do Order cấp, payload không sửa, caller/audience/key/generation khớp, chưa expired và hiện còn eligible. Response allowed=false không có permit khi bị từ chối. Không trả allowed=true từ một bản sao dữ liệu do caller gửi.
3. `Fulfillment.AcceptTask`: online validate trước transaction; sau đó lock barrier order/generation, kiểm thời hạn permit tại thời điểm decision, active generation/high-watermark, digest/warehouse. Atomic task + ACCEPTED + command registry + outbox; unique task nghiệp vụ theo Order. Replay trả outcome gốc trước khi kiểm permit đã hết hạn.
4. `CancelTask`: lock cùng barrier, kể cả chưa có task. OPEN/absent → CANCELLED tombstone; ACCEPTED → REJECTED với task identity, không báo đã hủy. `GetTaskDecision` query state và optional command evidence. Quyết định CANCELLED có DecisionReference bền vững cho reversal.
5. Order đặt local hold trước cancel remote. Một accept đã validate permit và đang bay vẫn có thể thắng tại Fulfillment; khi đó cancel bị từ chối. Local hold không phải distributed lock. Hard hold/hủy hoàn tất chỉ sau remote proof.
6. Resume cấp generation mới sau xác minh generation cũ CANCELLED và Order còn eligible, không có task accepted. Fulfillment chỉ mở generation mới bằng permit mới do Order cấp; giữ high-watermark/tombstones chặn generation thấp hơn, không reopen generation cũ. Cancel generation chưa có task vẫn tạo record. Không có task thứ hai nếu Order từng có task ACCEPTED; xử lý task đã nhận thuộc Care/warehouse policy riêng.

GetTaskDecision NOT_FOUND là “chưa thấy”, không phải bằng chứng task không thể xuất hiện. Không reverse stock trên NOT_FOUND hoặc local status.

## 6. Catalog, Shipping và voucher gated

`Catalog.GetCheckoutSnapshot` trả cùng snapshot_id/catalog_version/valid_until cho price/name/weight/packaging/subtotal. SKU unavailable hoặc giá đổi phải có structured error; không ghép ValidatePriceAndSKU với GetProductVariant khác revision thành snapshot. VND units integer, nanos=0. Quantity và subtotal phải khớp request.

Shipping v2 CalculateShippingFee profile 3.2 trả quote_id/quote_version/quote_expires_at/request_hash, bind địa chỉ/weight/carrier/fragile. Cả transfer và COD confirm phải kiểm lại expiry/terms. Order không coi TTL local là validity provider.

Voucher profile 3.2: hold/finalize/query/release dùng cùng registry/hash/expiry/tombstone như stock; HELD→FINALIZED trước PAID, paid fact không consume lần hai. shipping_fee bắt buộc khi tính freeship; discount <= shipping_fee với freeship, tổng discount không quá phần được giảm. Finalized replay giữ nguyên effect; release FINALIZED phải reject. Voucher sau finalize thất bại stock hoặc ngược lại cần reconciliation/compensation policy, không giả atomic transaction. Loyalty/reversal policy chưa có wire đầy đủ nên feature giữ disabled.

## 7. Public REST

OpenAPI target có operation stage/timestamps, independent Order projections, allowed_actions và Cancellation response. Mới accept=202 + Location, order_id=null; replay SUCCEEDED=201, same operation và non-null Order reference; failed replay trả durable error; 410 không giải phóng key. If-Match là decimal version, map đúng expected_version gRPC; không đồng thời nhận một version khác trong body.

Cancel 202=PENDING remote decision/recovery, 200=terminal decision; compensation_status độc lập. Unpaid cancellation fact có thể phát trước stock compensation terminal. Paid request đi Care approval, không tự refund. Poll Order cancellation/compensation projections; không chỉ kiểm lifecycle. Public client không được gửi approval/fulfillment-stopped proof để bypass service ACL.

Provider callback raw bytes/signature là adapter gated; exact signature/header/ACK phải được chọn với provider. Source-event/ready HTTP endpoints của simulator chỉ nằm trong snapshot sandbox, production dùng trusted adapters và proto handshake.

## 8. Events v3

8 schemas trong `packages/events/schemas/order/v3`, ví dụ tương ứng trong `packages/events/examples/order/v3`. Required aggregate_id/version/ordinal, lifecycle/payment/stock. paid bắt buộc stock COMMITTED, allocation_id, stock decision proof, voucher_finalization (null nếu không dùng voucher). ready bắt buộc generation>=1/digest/warehouse; chỉ PAID+CONFIRMED hoặc CONFIRMED_COD+UNPAID. Ready là fact mời xin permit, không phải command tạo task ngay.

Envelope source cố định, subject=`order:<aggregate_id>`, partition key=aggregate_id. Schema kiểm kiểu/pattern; consumer còn phải kiểm equality subject/aggregate và partition key. Dedup CloudEvent id; `(aggregate_id,version,ordinal)` sắp thứ tự facts cùng transaction (paid và ready khác ordinal). Không suy next version luôn liên tục; task acceptance/reversal vẫn dựa remote authority. Retry outbox giữ event ID/time/payload.

`refund_succeeded` có receipt/refund/amount/currency/settlement reference, không gây restock. `reconciliation_required` ở catalog này chỉ cho receipt gắn được Order; unmatched receipts vẫn phải lưu ledger, schema ledger-wide/Finance chưa thống nhất nên không ép chúng có order_id giả hoặc bỏ receipt.

V1/v2 đóng băng. Không thêm required proof vào v2 vì additionalProperties=false và runtime đã phát payload cũ. V3 là version transport, không đổi quyết định nghiệp vụ baseline 3.2.

## 9. Migration và rollback

Consumer v3 trước producer, capability provider trước Order adapter; pin contract commit/SHA hai phía. Route cohort rõ cho từng Order. New-flow Order không phát paid v1/v2 vào effect consumers cũ; không chạy cả finalize RPC và paid-trigger deduction. Legacy Order tiếp tục adapter/cohort tương ứng cho đến drain/reconcile. Rollback disable cohort mới nhưng vẫn chạy recovery/queries cho obligations đã tạo; giữ registry, tombstone, receipts, reversal ledger. Chưa qua provider fault/race suite thì không bật.

## 10. Kiểm chứng

`python packages/contracts/verify_order_design.py` kiểm OpenAPI, examples và negative cases của events. Thêm `--descriptor path --baseline-descriptor path` để kiểm schema protobuf và tag/type/RPC compatibility với baseline đã compile. Compiler và regenerated Go được kiểm bởi CI; đây là static/build evidence, không phải kiểm chứng provider implementation. [Báo cáo đồng bộ](../../docs/02_architecture/services/order-service/11_contract_alignment_report.md) ghi kết quả thực tế và phần còn mở.
