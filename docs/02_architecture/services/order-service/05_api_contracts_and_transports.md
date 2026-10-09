> **Implementation 3.1 — 09/10/2026:** [08_implementation_and_acceptance.md](08_implementation_and_acceptance.md) là nguồn hành vi hiện tại. Acceptance tạo checkout operation, chưa tạo Order; PAID và outbox chờ terminal stock/voucher outcome. Các đoạn v3.0 khác mô tả candidate cần đọc cùng cập nhật này.

# 05 — API contracts, transports và bảo mật

[Chỉ mục](README.md) · [Trước: Use cases](04_usecases_and_saga_orchestration.md) · [Tiếp: Testing](06_testing_and_failure_recovery.md)

## 1. Mức độ hiệu lực

Chương này là **contract candidate v3.0**, không tự thay đổi OpenAPI/protobuf/event schemas dùng chung. `openapi_d2c.yaml` hiện có checkout, lịch sử/chi tiết/cancel/tracking và callback nhưng thiếu COD/finalize/202/idempotency durable của v3.0. Bảng chênh lệch chương 07 là checklist trước integration. Không dùng ví dụ candidate để khẳng định contract hiện tại validate được.

REST giữ `X-Idempotency-Key` theo OpenAPI hiện tại. Header bắt buộc cho checkout/cancel/import/refund có side effect; giới hạn đề xuất UUID hoặc chuỗi ASCII 16..128 ký tự. Các client hiện có format UUID vẫn được hỗ trợ. Bổ sung `If-Match` cho mutation cần version; Guest auth dùng proof riêng, không dùng idempotency key.

## 2. Identity và authorization

Public client dùng bearer JWT hoặc Guest session/proof qua Gateway. Gateway bỏ header internal do client gửi và gắn identity đã xác thực; Order xác minh service caller và scoped context. Metadata `requesting_user_id/cancelled_by_user_id/cashier_staff_id` trong protobuf không tự là bằng chứng quyền. JWT sub là user ID, phải resolve Profile customer ID; không có resolver hợp lệ trả 503 IDENTITY_RESOLUTION_UNAVAILABLE.

| Caller | Quyền |
| --- | --- |
| Customer | List/detail/cancel chỉ order có đúng customer ID |
| Guest | Detail/cancel chỉ order bind proof còn hạn; proof giới hạn read/cancel theo scope |
| Warehouse/Packing | Queue của công đoạn và phạm vi kho được phân quyền |
| Sales Manager | List/detail toàn scope, hold/cancel review; không giả receipt |
| Customer Service | Case-related detail với masking; không tự approve refund |
| Fulfillment/Shipping | Internal detail snapshot cho task/shipment đang được cấp quyền |
| Channel | POS/import scope của terminal/store; không đọc mọi order |
| Profile verifier | Query guest eligibility qua contract tối thiểu, không tải PII toàn bộ |

Không có ownership trả 404 giống không tồn tại để tránh enumeration; thiếu auth 401, có auth nhưng thiếu quyền chức năng 403. Guest proof dùng verifier đáng tin/OTP liên kết order/phone/action, chống replay và brute force, không chỉ so phone trong request. Verification endpoint/issuer là gap G06, chưa hoàn thiện trong repo. Staff quyền + warehouse/assignment được kiểm tra cho mỗi command.

## 3. REST endpoint candidate

| Method / Path | Input chính | Thành công | Ghi chú |
| --- | --- | --- | --- |
| GET `/api/v1/cart` | session/customer | 200 cart + revision | giá dự kiến |
| POST `/api/v1/cart/items` | SKU, qty, expected revision | 200 cart mới | atomic revision |
| PUT `/api/v1/cart/items/{item_id}` | qty, If-Match | 200 | dùng item ID theo OpenAPI hiện tại |
| DELETE `/api/v1/cart/items/{item_id}` | If-Match | 204 | không reserve/release kho |
| POST `/api/v1/checkout/quote` | cart revision, address, carrier/method, voucher | 200 quote | mới; không tạo order |
| POST `/api/v1/checkout` | quote ID, cart revision, method | 201 hoặc 202 | X-Idempotency-Key; duy nhất command tạo D2C |
| GET `/api/v1/orders` | cursor, limit 1..100, status/date | 200 summaries | customer scope, không nhận customer ID tùy ý |
| GET `/api/v1/orders/{id}` | auth/proof | 200 detail + ETag | canonical snapshot |
| GET `/api/v1/orders/{id}/tracking` | auth/proof | 200 shipment projection | chỉ authorized checkpoints |
| POST `/api/v1/orders/{id}/cancel` | reason code, If-Match | 200 hoặc 202 | idempotent, compensation có thể pending |
| POST `/api/v1/orders/{id}/cancellation-requests` | reason, expected version | 202 review reference | paid customer request, không immediate refund |
| GET `/api/v1/admin/orders` | filters, cursor | 200 scoped list | permission `orders:read` |
| GET `/api/v1/admin/orders/queue` | stage, warehouse, SLA filter | 200 actionable queue | stage eligibility + assignment |
| POST `/api/v1/admin/orders/{id}/hold` | reason, expected version | 200 | permission + audit |
| POST `/api/v1/payments/vietqr/callback` | provider raw signed body | provider ACK sau durable receipt | adapter riêng, không bearer khách |

Không mở arbitrary state PATCH. B2B quotation/multi-address/promotion stacking chỉ thêm sau policy review. `GET /orders/{id}` cũng là status resource cho checkout 202; response phân biệt DRAFT/checkout pending và đơn đặt thành công.

## 4. Ví dụ checkout và response

Ví dụ dưới đây là candidate bổ sung quote, giữ header naming hiện có:

```http
POST /api/v1/checkout
Authorization: Bearer <customer-token>
X-Idempotency-Key: 93381d37-e3e4-4cf4-9018-8cd5a76c9011
Content-Type: application/json

{"quote_id":"quote-opaque-reference","cart_revision":12,"payment_method":"VIETQR"}
```

```json
{
  "order_id": "019a0000-0000-7000-8000-000000000001",
  "order_code": "ORD-20261008-000001",
  "status": "PENDING_PAYMENT",
  "version": 2,
  "channel": "D2C_WEB",
  "amounts": {
    "currency": "VND", "subtotal_units": 220000,
    "merchandise_discount_units": 20000,
    "shipping_units": 25000, "shipping_discount_units": 0,
    "final_units": 225000
  },
  "payment": {"method":"VIETQR","status":"PENDING","reference":"OMAMA-000001"},
  "payment_expires_at":"2026-10-08T08:15:00Z",
  "compensation_status":"NONE"
}
```

201 trả Location `/api/v1/orders/{id}`; 202 trả cùng Location và `Retry-After: 2`, status còn DRAFT/PAYMENT_FINALIZING theo tài nguyên đang xử lý. COD trả CONFIRMED_COD và payment UNPAID; không gán paid_at hoặc bank transaction giả. Response list gồm code/time/final/channel/order status/payment summary/SLA flags, không chứa PII chi tiết.

Detail thêm items/address snapshot, authorized timeline, payment summary, stock projection, shipment/packages, `allowed_actions` và operational hold. `allowed_actions` giúp UI nhưng server vẫn kiểm tra guard lúc command tới. ETag từ order version. Cursor ký scope/filter; timestamp ISO-8601 UTC, UI hiển thị Asia/Saigon.

## 5. Lỗi và retry semantics

```json
{
  "error": {
    "code":"IDEMPOTENCY_CONFLICT",
    "message":"Khóa yêu cầu đã được dùng với nội dung khác.",
    "retryable":false,
    "correlation_id":"request-reference",
    "details":[]
  }
}
```

| HTTP | Codes candidate | Hành vi |
| --- | --- | --- |
| 400 / 413 | INVALID_REQUEST / BODY_TOO_LARGE | sửa cấu trúc; reject unknown fields |
| 401 / 403 | AUTH_REQUIRED / PERMISSION_DENIED | không retry bằng đổi customer ID |
| 404 | ORDER_NOT_FOUND / ADDRESS_NOT_FOUND | không tiết lộ ownership |
| 409 | IDEMPOTENCY_CONFLICT, VERSION_CONFLICT, PRICE_CHANGED, INSUFFICIENT_STOCK, CANCELLATION_NOT_ALLOWED, PAYMENT_FINALIZING | reload hoặc thao tác mới khi người dùng chấp nhận; không đổi key để né IN_PROGRESS |
| 422 | INVALID_ADDRESS, VOUCHER_NOT_APPLICABLE, COD_NOT_AVAILABLE, ZERO_AMOUNT_ORDER_UNSUPPORTED | lỗi nghiệp vụ xác định |
| 428 | VERSION_REQUIRED | bổ sung If-Match |
| 429 | RATE_LIMITED | Retry-After |
| 503 | DEPENDENCY_UNAVAILABLE, IDENTITY_RESOLUTION_UNAVAILABLE, CLAIM_VERIFICATION_UNAVAILABLE | retry cùng key sau backoff |

Timeout RPC sau durable acceptance trả checkout 202; 504 chỉ dùng khi gateway deadline thực sự bị vượt và response mất, không diễn giải là remote rollback. Webhook ACK contract theo provider; thiết kế không áp arbitrary JSON error cho provider không hỗ trợ. Correlation ID không chứa token/phone.

## 6. gRPC hiện có và phần bổ sung

| Service và method hiện có trong packages/proto | Cách dùng v3.0 / giới hạn |
| --- | --- |
| Order.CreatePOSOrder | Channel terminal đã xác thực; no marketplace import qua RPC này |
| Order.GetOrderDetail | cần principal service/user trusted metadata; hiện response thiếu lifecycle projections/version/COD fields |
| Order.CancelOrder | chưa có expected version/compensation status; cần additive extension |
| Inventory.ReserveStock / ReleaseReservation | dùng stable key; Release cần reservation_id nên chưa giải quyết unknown ID |
| Catalog.ValidatePriceAndSKU / GetProductVariant | chưa có canonical line snapshot và price version đồng bộ |
| Promotion.ValidateAndLockVoucher / ReleaseVoucher / CalculateDiscount | tên RPC đúng; không dùng ValidateVoucher tưởng tượng |
| Shipping.CalculateShippingFee / CreateShipment / TrackShipment | chưa có quote reference/valid_until dùng cho checkout |
| Profile.GetDeliveryAddress | response thật GetDeliveryAddressResponse, customer required, two-tier additive fields |
| Fulfillment.GetPackingVideoUrl / SubmitPackingEvidence | chưa có ready authorization/cancellation barrier |
| Identity.CheckSpecializedPermission / GetJwksPublicKey | không phải resolver user→customer |

**Proposed Inventory capability**, cần bổ sung proto hoặc protocol tương đương đã chứng minh:

```text
GetReservationByOrder(order_id, command_key)
  -> terminal_state, reservation_id, items_digest, expires_at, version
FinalizeReservation(order_id, reservation_id, command_key, expected_version)
  -> COMMITTED | EXPIRED | RELEASED | UNKNOWN + operation reference
ReleaseReservationByOrder(order_id, reserve_command_key, release_command_key)
  -> RELEASED | COMMITTED | EXPIRED
ReverseCommittedStock(order_id, approval_reference, reversal_key, items_scope)
  -> terminal outcome / query reference
```

Finalize/expire/release được serialize tại Inventory. ReleaseByOrder ghi cancellation tombstone kể cả reserve chưa tồn tại để chặn reserve cũ đến trễ; cùng order/key sau cancelled không reserve lại. Proto hiện có không cam kết tombstone: gap blocker.

Promotion cần query hold by order/key, consume/finalize hold và release-by-order chống hold đến trễ; free-ship input gồm canonical shipping fee + cap. Fulfillment cần authorize-ready generation và cancel/query task terminal outcome. Care cần case eligibility/version barrier. Profile cần verifier query tối thiểu/claim ack; không cho verifier dựa trên arbitrary requesting_user_id.

gRPC errors dùng status codes: InvalidArgument, Unauthenticated, PermissionDenied, NotFound, AlreadyExists cho identity conflict, FailedPrecondition cho transition, Aborted cho version, Unavailable/DeadlineExceeded cho unknown network outcome. DTO ErrorDetail hiện tồn tại chỉ dùng nếu contract cụ thể yêu cầu; không đồng thời trả transport OK và che lỗi hệ thống trong field mà client dễ bỏ qua.

## 7. Kafka: fact, command và phiên bản

Fact mô tả sự kiện đã commit; command là intent chưa biết kết quả. Topics candidate `order.events.v2`, `inventory.commands.v1`, `promotion.commands.v1` và reply events phải có ACL/group/schema riêng. Không dùng OrderCancelled fact làm command release khi Order worker đã release trực tiếp. Chọn duy nhất một adapter gửi command: RPC có stable key hoặc outbox command; không đồng thời dùng cả hai cho cùng effect.

| Event | Thời điểm phát | Hiện trạng |
| --- | --- | --- |
| `vn.omama.order.placed.v2` | PENDING_PAYMENT/CONFIRMED_COD đã durable | candidate, chưa schema |
| `vn.omama.order.paid.v2` | receipt allocated đủ và stock committed | v1 schema có nhưng chưa đủ v3.0 |
| `vn.omama.order.fulfillment_ready.v2` | inventory/voucher finalized, không hold, COD/trả trước đúng policy | candidate; consumer mới bắt buộc |
| `vn.omama.order.cancelled.v2` | cancellation outcome durable | v1 có nhưng resource “đã nhả” chưa đúng khi compensation pending |
| `vn.omama.order.completed.v2` | lifecycle completion barrier thành công | candidate, loyalty policy do Promotion |
| `vn.omama.order.reconciliation_required.v2` | receipt mismatch/late/unmatched | candidate; unmatched dùng aggregate PAYMENT trong outbox |
| `vn.omama.order.marketplace_import_failed.v2` | import rejection durable | candidate |

Payment unmatched không có order ID: outbox dùng `aggregate_type=PAYMENT`, `aggregate_id=payment_id`, `order_id=NULL` và partition key là payment ID. Không tạo order giả để phát event; reconciliation/audit event cùng commit receipt. Khi match sau đó, association/payment mutation có version mới và audit riêng (G12 yêu cầu implementation).

CloudEvents candidate cho **registered customer đã paid**, không chứa Direct PII:

```json
{
  "specversion":"1.0",
  "id":"bdb7155a-3716-4a69-87f2-0a03d4601001",
  "source":"https://omama.vn/services/order-service",
  "type":"vn.omama.order.paid.v2",
  "subject":"order:019a0000-0000-7000-8000-000000000001",
  "time":"2026-10-08T08:02:00Z",
  "datacontenttype":"application/json",
  "traceparent":"00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
  "data":{
    "order_id":"019a0000-0000-7000-8000-000000000001",
    "order_version":4,
    "channel":"D2C_WEB",
    "customer_id":"019a0000-0000-7000-8000-000000000002",
    "shipping_address_id":"019a0000-0000-7000-8000-000000000003",
    "final_paid_amount":{"currency_code":"VND","units":225000,"nanos":0},
    "payment_id":"019a0000-0000-7000-8000-000000000004",
    "stock_status":"COMMITTED",
    "items":[{"sku_code":"MX-GION-500G","quantity":2}]
  }
}
```

`customer_id` nullable cho Guest; shipping snapshot nullable cho POS nhận tại quầy; channel vocabulary `D2C_WEB/MOBILE_APP/SHOPEE/TIKTOK/POS_QUAY/B2B_WHOLESALE` thống nhất trong v3.0, B2B extension cần schema mới. Envelope stable id/source/type; traceparent là extension dự án, không phải mandatory core CloudEvents. Subject reference không phải proof quyền. [CloudEvents 1.0.2 specification](https://github.com/cloudevents/spec/blob/v1.0.2/cloudevents/spec.md).

Không sửa required/type của schema v1 đang phát mà coi backward-compatible. V2 topics/event types cần consumer migration, dual-read có business dedup theo operation/order/version, tránh phát hai bản tạo hai packing tasks. Consumer ready tách khỏi paid trước khi bật COD; legacy paid consumer không được tự commit stock lần hai. Schema v2 phải được bổ sung/validate trước publish; ví dụ trên không validate bằng v1.

Inbound: Fulfillment/Shipping/Inventory/Channel/Profile/Care/Payment adapters kiểm tra source ACL, resource association, event schema và source sequence. Khác topic/key (shipping tracking_code, profile customer_id) không có global order; Inbox + deferred protocol xử lý việc đến lệch thứ tự.

## 8. Security và observability trên transport

TLS/mTLS hoặc authenticated internal token theo deployment đã kiểm chứng; không gửi internal token tới browser. Profile hiện yêu cầu `x-internal-token` gRPC, customer ID đúng, deadline server 2 giây; không tự giả định Profile đã có mTLS. Rate-limit riêng Guest proof/webhook/list; body/raw signature bytes giới hạn. JWKS cache có rotation và fail-closed policy cho key lạ.

Logs chỉ dùng request/order/payment/saga references, state, error code, duration; không dùng order/customer ID làm metric label. Traces W3C qua HTTP/gRPC/Kafka, redact PII. Metrics gồm query/checkout latency theo method/channel, pending/finalizing ages, outbox lag, saga UNKNOWN, reconciliation count, refund UNKNOWN, inbox deferred/DLQ, SLA overdue. Health live chỉ process; readiness kiểm DB và khả năng durable accept. Broker down làm backlog, không tự phủ định DB đã commit.
