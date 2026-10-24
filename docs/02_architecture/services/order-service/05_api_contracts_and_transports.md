# 05 — Contract và transport, baseline 3.2

[Trước](04_usecases_and_saga_orchestration.md) · [Tiếp](06_testing_and_failure_recovery.md)

## 1. Hiệu lực và wire status

Các bảng target dưới đây đã được cụ thể hóa thành wire proposal ngày 15/10/2026, cần đồng thuận và provider implementation trước rollout. Nguồn: [Order OpenAPI 3.2](../../../03_api_specs/order-service.openapi.yaml), [Inventory proto](../../../../packages/proto/inventory/v1/inventory.proto), [Order proto](../../../../packages/proto/order/v1/order.proto), [Fulfillment proto](../../../../packages/proto/fulfillment/v1/fulfillment.proto), [v3 schemas](../../../../packages/events/schemas/order/v3/paid.event.json) và [semantics profile order-3.2](../../../../packages/contracts/order-3.2.md). [OpenAPI sandbox 3.1](../../../03_api_specs/order-service.sandbox-3.1.openapi.yaml) và v2 events được giữ nguyên để kiểm runtime cũ.

Proto hiện đã thêm command evidence, hash/digest/version, approval/barrier references và reversal trace. Profile opt-in `contract_version=order-3.2` bắt buộc response xác nhận cùng version; caller không fallback nếu thiếu proof. Inventory PR#11 head2b9f179 chỉ thực hiện bốn RPC gốc; simulator Order vẫn chỉ trả outcome cũ. Wire proposal và regenerated Go không đóng implementation gaps.

Không đổi field number/type cũ. Thêm fields/RPC additive hoặc version mới khi semantic breaking; không đổi state string hiện hành thành enum ngay tại cùng tag. HLD code snippets không phải file để generate code.

## 2. Public API

**Order Core addendum 17/10/2026:** [chương12](12_order_core_design.md) và [Core OpenAPI candidate](../../../03_api_specs/order-core.candidate.openapi.yaml) mô tả command resources, hold_id/resume, scoped queries và purpose-bound snapshot views. [Snapshot v1](../../../03_api_specs/order-snapshot.v1.schema.json) là internal payload; public views có redaction riêng. Candidate cần pin version/cohort, không thay route/DTO baseline ngầm.

**Cart/Checkout addendum đang review:** [chương11](11_cart_checkout_design.md) và [OpenAPI candidate](../../../03_api_specs/order-cart-checkout.candidate.openapi.yaml) định nghĩa CartProjection, mutation CAS/replay, selected quote và quote consumption. Các Cart/Quote DTO mới có breaking changes, cần explicit route/version/cohort rollout; chưa thay OpenAPI baseline/sandbox.

**SePay addendum đang review:** [chương 09](09_payment_sepay_vietqr_design.md) và [OpenAPI Payment candidate](../../../03_api_specs/order-payment-sepay.candidate.openapi.yaml) đề xuất route `/api/v1/payments/sepay/webhook`, auth HMAC đúng giao thức SePay, ACK sau observation + work durable, canonical receipt từ API v2 và hai API đọc cho khách. Candidate chưa thay contract/runtime baseline; dòng “Provider callback” bên dưới là baseline tổng quát, amendment SePay được liệt kê rõ ở chương 09 mục 2.

REST target giữ X-Idempotency-Key; key 16..128 ASCII, scope trusted principal/action; body hash canonical. Version mutations dùng expected version theo OpenAPI hiện tại; khi thêm If-Match phải công bố mapping, không dùng hai version khác nhau.

| Endpoint | Kết quả/semantics |
| --- | --- |
| Cart read/mutate | revision atomic, không giữ stock |
| POST /api/v1/checkout/quote | server quote bind owner/method/cart/address, expiry |
| POST /api/v1/checkout mới | 202, Location /api/v1/checkout-operations/{id}, public order_id=null |
| Replay checkout đang chạy | 202 cùng operation |
| Replay SUCCEEDED | 201 cùng Order; không reserve lại |
| Replay FAILED | lưu và trả terminal error cùng operation |
| Terminal response hết retention | 410, operation reference, không reuse key |
| GET /api/v1/checkout-operations/{id} | scope owner, status/stage/error/order_id sau placement |
| GET /api/v1/orders, /{id} | scoped list/detail, lifecycle/payment/stock/hold/allowed_actions |
| POST /api/v1/orders/{id}/cancel | 200 terminal hoặc 202 pending; compensation_status riêng |
| Paid cancellation request | 202 review reference, không tự refund |
| Provider callback | ACK sau receipt durable, theo provider contract |
| Staff queue/hold/resume | permission/scope/version/reason/audit, barrier khi cần |

202 body dùng actual OpenAPI projection; semantic tối thiểu là operation identity/status và nullable order_id. Không dùng GET /orders/{planned_id} làm polling resource trước khi Order tồn tại. UI “đã nhận yêu cầu” khác “đặt thành công”.

| Lỗi | Xử lý |
| --- | --- |
| 400/413 invalid/body size | sửa input |
| 401/403 auth/function denied | không đổi customer_id để retry |
| 404 object absent/not owned | không lộ ownership |
| 409 key/hash/version/price/state conflict | reload/review, không sinh key mới để né unknown |
| 422 feature/policy/zero amount unsupported | definitive reject |
| 429 overload | Retry-After |
| 503/504 mất kết nối/ACK | same key/query; không suy diễn rollback |

## 3. Inventory target protocol

Common mutation context: authenticated caller, order_id immutable UUID, action-scoped idempotency_key, canonical request_hash, correlation/trace, warehouse selection policy. Reserve thêm consolidated items/quantities, items_digest, requested TTL. Finalize/release/reverse thêm expected reservation_id/digest khi đã biết; query-by-order phục hồi khi mất ID. expected_version chỉ dùng guard bước mới, không làm replay command đã thành công bị fail.

Response phải tách **command result** và **current resource state**:
- command_key/action/result: APPLIED, ALREADY_APPLIED hoặc REJECTED với reason;
- order_id, reservation_id (nullable chỉ khi verified tombstone chưa từng reserve), items_digest, warehouse_id;
- state: RESERVED, COMMITTED, RELEASED, EXPIRED; reversal là ledger operation riêng, projection có thể REVERSED;
- resource_version, expires_at, decision_at; cancellation_tombstone, reversal_id/quantity scope khi phù hợp.
- UNKNOWN là quan sát caller khi không nhận được outcome đáng tin; không lưu UNKNOWN như terminal resource state.

| Operation | Guard/effect | Replay/conflict |
| --- | --- | --- |
| ReserveStock | atomically allocate tất cả items; tạo RESERVED với expiry; reject tombstone | same key/hash cùng reservation + current state; changed hash conflict |
| GetReservationByOrder | read authoritative state/digest/version/tombstone | NOT_FOUND chỉ “chưa thấy tại thời điểm đọc”, không chứng minh không có request đang bay |
| FinalizeReservation | dưới row lock: RESERVED và DB now<expiry → COMMITTED; stock+outbox atomic | COMMITTED same resource success; RELEASED/EXPIRED reject có state; khác ID/digest conflict |
| ReleaseReservationByOrder | RESERVED → RELEASED; chưa có reserve → persist tombstone trước ACK | COMMITTED trả state, không nói released; repeated release no extra stock |
| Expire internal | RESERVED và DB now>=expiry → EXPIRED + release quantities + fact atomic | terminal state khác no-op, không đổi COMMITTED |
| ReverseCommittedStock | COMMITTED, verified approval/policy và fulfillment stopped proof; full reservation MVP | stable reversal ID, không cộng hai lần; query được outcome lost ACK |

Reserve và release-by-order phải serialize cả khi reservation row chưa tồn tại: Inventory cần order-key registry unique/row/advisory lock hoặc transaction tương đương. “SELECT reservation FOR UPDATE” trên row không tồn tại là chưa đủ bảo đảm tombstone chặn late reserve. Release tombstone giữ theo key-retention policy; order ID đã cancelled không reserve lại.

Outcome validator tại mọi Order path: đúng order/resource/digest/warehouse + known state + non-stale version + required timestamps. Compensation chỉ close với RELEASED/EXPIRED hoặc valid tombstone; reversal chỉ close với explicit reversal outcome. Legacy IsReleased=true không đủ suy ra actual state. SDK/adapter không dựng fake COMMITTED từ HTTP/gRPC OK.

Single warehouse/TTL15m MVP phải reject unsupported warehouse/TTL; có thể trả actual accepted TTL nếu contract đã cho phép negotiate. Không im lặng bỏ qua input. SKU inactive/insufficient là business error; lock conflict retryable.

## 4. Các provider khác

| Provider / capability target | Request và proof tối thiểu | Điều kiện |
| --- | --- | --- |
| Catalog canonical batch | SKU/quantity, expected price; snapshot_id/version/valid_until + price/name/weight/packaging coherent | không ghép hai response khác revision |
| Shipping quote | address/weight/service, quote_id/fee/valid_until | Order không tự tính phí từ dữ liệu client |
| Promotion hold/query/finalize/release | order/key/hash/totals/shipping fee/cap; hold ID/expiry/version | tombstone chống hold muộn; voucher feature gated |
| Profile address | trusted customer/address/version | ownership tại provider; JWT sub cần mapping |
| Provider money | verified receipt/receiver/currency/reference/transaction ID; statement/query/refund stable key | raw-signature/finality thật, không chỉ QR image |
| Care refund approval | receipt/order/amount/currency/scope/policy/version/actor | signature/service ACL + query decision; không trust arbitrary string |
| Care completion barrier | order/key/expected case version; ALLOWED/DENIED decision ID + query | serialize case-open/complete; post-completion policy rõ |

Tên mới ở bảng là capability thiết kế, chưa phải tuyên bố các service đã có RPC đó.

## 5. Fulfillment ready/cancel protocol

Target capabilities (wire signatures đã có trong proto proposal; semantics cần thống nhất trước tích hợp):
1. Order.GetFulfillmentPermit(order_id,generation,task_request_key): current eligible dưới Order lock → permit bind order/generation/items digest/policy/expiry; service-authenticated, không PII.
2. Fulfillment.AcceptTask(order_id,generation,permit,key): verify permit; serialize cùng barrier record → ACCEPTED/task hoặc CANCELLED/reject; unique business task.
3. Fulfillment.CancelTask(order_id,generation,key): OPEN/chưa có task → CANCELLED tombstone; ACCEPTED → conflict/task state; query cùng key.
4. Fulfillment.GetTaskDecision(order_id,generation): authoritative OPEN/ACCEPTED/CANCELLED + task/version/decision time.
5. Resume: generation mới sau kiểm task/cancel outcome; stale generation không thể reopen.

Wire còn có Order.ValidateFulfillmentPermit để Fulfillment kiểm online permit do Order cấp; không trust dữ liệu permit do caller tự dựng. Proof expiry/audience/request key và generation phải được kiểm; accept/cancel vẫn phân xử tại Fulfillment như mô tả dưới đây.

Order đặt local hold trước cancel RPC. Permit đang bay vẫn có thể được AcceptTask xử lý trước CancelTask; bên thắng tại Fulfillment quyết định kết quả. Không hứa local hold là distributed stop tức thì. Cancel success làm mọi permit cũ mất hiệu lực dù chưa hết expiry. Chuỗi barrier proof dùng cho reversal phải được Inventory xác minh qua trusted service/decision reference; user không được tự điền “cancelled=true”.

## 6. Events và rollout

[Persistence17](17_order_persistence_messaging_design.md) chốt physical outbox/inbox, stream-head publish/failover, offset/rebalance/replay và restore. Generic financial streams có [schema candidate](../../../03_api_specs/order-payment-events.candidate.schema.json) riêng; không sửa closed Order v3 payload để thêm nullable Order hoặc stream sequence.

Topic target order.events.v3, partition key order_id, CloudEvents type vn.omama.order.{name}.v3. Envelope stable id/source/type/time/subject, trace context theo project; data references, aggregate version/ordinal/state/generation, không direct PII. V2 runtime đóng schema bằng additionalProperties=false và thiếu required proof; vì vậy tạo version mới thay vì sửa breaking v2. Version transport v3 vẫn phục vụ baseline thiết kế 3.2.

| Fact | Phát khi | Consumer semantics |
| --- | --- | --- |
| placed | Order transaction đã commit | không phải permission packing |
| paid | verified allocation + stock/voucher finalized | notify/accounting; không trừ kho/consume voucher |
| fulfillment_ready | đủ guards và ready generation | xin permit, accept qua barrier |
| cancelled | local business cancellation đã commit | compensation có thể pending với unpaid; không tự release lần hai |
| completed | completion decision đã apply | loyalty idempotent theo policy |
| reconciliation_required | verified tiền có bất thường | đối soát, không báo tiền biến mất |
| refund_succeeded | verified refund settlement | accounting; không tự restock |

Inventory facts là audit/reconciliation/projections; terminal query là authority khi event trễ. Khi mở provider ledger audit rộng hơn, receipt/refund facts phải hỗ trợ unmatched receipt và có schema được Finance xác nhận; không dùng paid để đại diện mọi khoản tiền nhận.

Rollout: pin contract versions → provider implement/test → adapter consumer tests → deploy consumers hiểu v3 → canary bằng feature/config routing theo cohort Order → enable new producer/path. Với một Order chỉ một commit authority. Không dual-publish v1/v2 paid cho legacy effect consumers của new-flow Order. Drain legacy orders bằng legacy adapter/cohort riêng, reconcile obligations trước tắt; rollback không replay terminal effects hoặc xóa ledger.

## 7. Authentication

TLS/service identity + per-method ACL, Kafka publisher/group ACL; production không tin public X-Internal-Token/header tự khai. Gateway strip spoofable headers; Order verify principal, customer mapping và object scope. Sandbox token/HMAC chỉ dành simulator. Logs không chứa raw callback/PII/token. Reversal/refund permissions tách khỏi public checkout/cancel.
