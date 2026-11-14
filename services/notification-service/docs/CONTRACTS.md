# Integration contracts — candidate, không phải API đã có

Tất cả endpoint/DTO Notification trong file này là **PROPOSED v0.1**. N0 sẽ chốt rồi viết machine-readable OpenAPI/JSON Schema trong `services/notification-service/contracts/`; không sửa shared contract registry hoặc giả rằng Gateway đã route. Source pins ở SOURCE_AUDIT.

## 1. Event onboarding matrix

| Nguồn | Wire thực tế/tham chiếu | Chọn cho Notification | Dependency/gate |
|---|---|---|---|
| Order core | `order.events.v3`, `vn.omama.order.placed.v3`, `.paid.v3`, `.cancelled.v3`, `.completed.v3`; key order_id, subject `order:<id>`, source `https://omama.vn/services/order-service` | N3 ưu tiên placed/paid/cancelled; completed chỉ nếu owner yêu cầu. Mỗi type có purpose riêng | Code publisher có, runtime enable chưa xác minh; thiếu approved contact/content resolver. Pin schema hash và exact fixture |
| Order sandbox | `order.events.v2`, type `.v2`; app Emit hiện tạo version/status nhưng không giống mọi optional field v3 | Adapter riêng chỉ khi có consumer requirement; default disabled | Không dual-subscribe cùng Order/cohort; không cast v2 thành v3 |
| Order legacy | `order.events.v1` từ catalog cũ | Deferred | Không áp policy core lên legacy mà chưa owner approval |
| Shipping | `shipping.events.v2`, `vn.omama.shipping.shipment.{handed_over,delivered,delivery_failed}.v2`; key shipment_id, subject `shipment_id:<id>` | N5; Email shipping có thể thêm sau N3 bằng scope riêng | Có resource_version/ready_generation; resolver Order/audience; stale/coalescing policy. v1 không tự suy ra revision |
| Fulfillment | `fulfillment.events.v2`, schema `packages/events/schemas/fulfillment/v2/packing_completed.event.json`; publisher ở ref Fulfillment được audit | Optional phase sau | Pin actual type/source/subject/key từ schema+producer ở phase entry; không dùng tên `PackingCompletedEvent` trong HLD để deserialize |
| Finance invoice | `finance.events.v1`, `vn.omama.finance.invoice.issued.v1`; key order_id | N3 gate invoice | Schema có, service implementation không có trong base; private artifact/audience missing |
| Identity OTP | Chưa có dispatch contract xác nhận | Trusted private command adapter, disabled | Owner chốt Notification hay IdP gửi; không subscribe UserRegistered để tự gửi OTP |
| Care/Profile/Inventory | Care SLA metadata events; Profile compliance; Inventory expiry | N7 | Pin actual event names/schema từ owning branch khi phase bắt đầu. Không coi topic catalog là payload đã được kiểm chứng |
| Marketing/Promotion | Audience/campaign/consent owner chưa chốt | N6+, disabled | Không tự biến Order contact thành Marketing consent |

Consumer group candidate: `notification-dispatch-v1-<env>`; topics là explicit allowlist, không regex toàn broker. Producer auth từ Kafka ACL/service identity; `source` trong JSON chỉ là một kiểm tra bổ sung, không chứng minh danh tính producer. N0 cần producer→topic→type→cohort→purpose matrix được owner duyệt.

Decoder phải kiểm giới hạn kích thước/depth, UTF-8/JSON, specversion, type/source/subject/key relation, data required fields/types/version, schema pin và allowed cohort. Unknown topic/type/version → durable quarantine + alert, không gửi. Schema validation dùng exact version; optional additive fields theo schema, không globally reject mọi extra field phá backward compatibility. Normalized `order-source.event.json` dành cho Order ingress không được nhận như producer CloudEvent.

## 2. Recipient/content contract còn thiếu — không bịa endpoint

Port ứng viên `ResolveNotificationContext` (chưa phải HTTP/gRPC endpoint triển khai):

Input: authenticated requesting service; source service/event ID/type; aggregate ID/version/cohort; approved purpose; audience `BUYER|GIFT_RECIPIENT`; optional opaque recipient ref.

Output bắt buộc: stable recipient key (issuer-scoped customer/guest-order-audience ref), contact ref + encrypted-transfer destination over TLS, contact provenance/version/permission evidence, locale, policy revision, audience, hide-price flag, expiry và minimal allowlisted template variables. Order nguồn xác nhận quan hệ với resource; Profile chỉ có email optional không thay cho chứng minh sở hữu/audience. ID principal và customer không mặc định bằng nhau.

- Guest dùng stable source-issued ref scoped đúng Order/audience; không hash email rồi coi là global customer.
- Một contact bị đổi sau khi request tạo: retry giữ snapshot đã được phép, nhưng phải re-check revocation/suppression/expiry ngay trước send. Không tự chuyển sang người nhận mới. Contact correction cần source intent/revision mới được duyệt.
- Resolver timeout/unavailable → `WAITING_CONTEXT` và bounded retry; denied/missing address → `SUPPRESSED` hoặc terminal validation failure với reason riêng; không fallback email giả.
- Template data không hydrate bằng cách scrape public Order API hoặc đọc DB nguồn. Giá/thuế dùng Money contract gốc, không float tính lại. Hide-price projection loại bỏ giá ở server trước render.
- N0 cần owner quyết định: source phát private Dispatch command (khuyến nghị khi có contact/gift/OTP) hay Notification gọi resolver least-privilege. Nếu cả event và command cùng tồn tại, phải dùng cùng canonical intent identity, chỉ một route tạo delivery.

## 3. Internal intake API ứng viên

`POST /api/v1/internal/notification-requests`

Headers: authenticated service credential; `Idempotency-Key` 1–128 ký tự ASCII allowlist; `X-Request-ID` optional kiểm tra định dạng. Không tin `producer` từ body để cấp quyền. HTTPS/private network; body cap candidate 64 KiB, schema strict cho command.

```json
{
  "source_event": {
    "id": "11111111-1111-4111-8111-111111111111",
    "source": "order-service",
    "type": "vn.omama.order.placed.v3",
    "aggregate_id": "22222222-2222-4222-8222-222222222222",
    "aggregate_version": 2,
    "cohort": "order-core-1"
  },
  "purpose": "ORDER_CONFIRMATION",
  "audience": "BUYER",
  "recipient_ref": "order:22222222-2222-4222-8222-222222222222:buyer",
  "locale": "vi-VN",
  "context_ref": "opaque-owner-context-reference"
}
```

Không cho caller chọn arbitrary HTML, sender, provider, template path, callback URL, required channel hoặc mandatory flag. Server ánh xạ purpose→classification/template/channel theo approved policy. Locale missing dùng default vi-VN chỉ khi template policy chốt; unsupported locale có explicit fallback hoặc reject, không ngầm gửi ngôn ngữ khác.

Response lần đầu **202** + `Location`:

```json
{"request_id":"33333333-3333-4333-8333-333333333333","status":"ACCEPTED","duplicate":false}
```

ACCEPTED ở request nghĩa DB đã commit. Cùng authenticated producer + idempotency key + semantic fingerprint → **200**, original request_id, duplicate=true, current non-sensitive status; không render/send lại. Cùng key nhưng body khác → **409 IDEMPOTENCY_CONFLICT**. Khác key nhưng cùng canonical intent → trả existing request nếu fingerprint phù hợp; conflict được ghi riêng. Network timeout/commit unknown: client giữ key để query/retry; service không sinh request mới khi chưa biết DB commit outcome.

`GET /api/v1/internal/notification-requests/{id}`: chỉ producer sở hữu request và scope được cấp; 200 summary request/delivery statuses, masked contact, attempt counts; không raw vars/OTP. Foreign ID trả 404 không tiết lộ tồn tại.

Errors chung: JSON `{"code":"...","message":"safe text","request_id":"...","retryable":false}`. 400 malformed, 401 missing/invalid credentials, 403 authenticated unauthorized purpose/action, 404 out-of-scope/not found, 409 idempotency/state conflict, 413 body too large, 415 media type, 422 semantic invalid/expired/unsupported purpose, 429 quota với Retry-After, 503 durable store/dependency unavailable. Không trả 202 nếu chưa persist được.

## 4. OTP/invoice command extensions — gated

OTP contract riêng chỉ Identity identity được gọi: challenge ref, opaque subject/contact ref, purpose, expires_at và encrypted sensitive payload/reference. Nếu secret-reference thì resolver ACL chỉ cho Notification đọc một challenge; không đặt secret URL có thể fetch tùy ý. Identity sở hữu resend identity/TTL/anti-enumeration/account lookup/rate limit. Ciphertext request fingerprint dùng semantic HMAC của secret tại trusted ingress, không hash ciphertext ngẫu nhiên gây false conflicts. Không gửi raw OTP qua shared domain topic hoặc giữ sau expiry. Nếu Keycloak đã gửi email cho flow đó: route Notification disabled.

Invoice: invoice ref/revision, order/legal-entity/audience binding, allowed artifact access. V1 đề xuất link được Finance cấp quyền/expiry; attachment binary deferred. Không tự fetch arbitrary invoice_pdf_url: allowlisted HTTPS origin, không redirect private/local network, size/type/timeout bounds nếu phase sau cần download. Link hết hạn tại send phải refresh qua authorized owner API, không tạo public URL.

## 5. Operations API ứng viên

| Method/path | Permission ứng viên | Semantics |
|---|---|---|
| `GET /api/v1/admin/notification-requests` | `notification:read` + resource scope | Filter source/status/channel/time range; cursor keyset (created_at,id), limit 1–100; masked data; query bounds |
| `GET /api/v1/admin/notification-requests/{id}` | như trên | Timeline metadata; separate delivery/request status |
| `POST /api/v1/admin/notification-requests/{id}/replays` | `notification:replay` + scope | Idempotency-Key, reason, expected_version; chỉ replay pre-send/quarantine/retry-exhausted đủ điều kiện; giữ intent/provider key |
| `POST /api/v1/admin/notification-requests/{id}/resends` | `notification:resend` + scope | Deferred by default; intentional new send, child request/resend sequence, fresh expiry/consent check, audit; phải có approval policy |

Không reset attempts/status trực tiếp qua PATCH. UNKNOWN không replay/resend nếu chưa có evidence resolution; DELIVERED/ACCEPTED không tự resend. Perm names chưa có trong Identity IAM registry: dùng interface/fake trong tests, thật fail-closed cho tới owner mapping. Authorize usecase trước repository access; current employee status/resource scope từ trusted authority, JWT valid không đủ chứng minh quyền hiện tại. 401/403/404/409 không gộp tất cả thành 500.

Templates MVP là files versioned + registry metadata deploy, review qua code review. Admin template/routing CRUD/preview/publish UI/API deferred; không tạo unaudited live editing. Replay/resend audit atomically gắn actor/reason/from-state/new command ID, không body/contact.

## 6. Callback API ứng viên

`POST /api/v1/webhooks/notifications/{provider}`. Public gateway routing chưa được thêm. Đây là provider-authenticated endpoint, không đòi user JWT; verify provider signature trên **raw body** trước JSON parse và persist, timestamp/replay window theo đúng provider contract, key rotation allowlist. Không dùng HMAC tự đặt cho real provider. Fake local có protocol riêng, không nhận production credentials.

Internal normalized callback: provider account, provider event ID, provider message ID/stable key, normalized status, occurred_at, sanitized code. Dedup `(provider, account, event_id)` + body fingerprint. HTTP 200/204 sau durable receipt; same event different bytes → conflict ledger/alert, không ghi đè; auth fail 401/403, oversize 413, DB unavailable 503 cho provider retry theo contract.

Callback trước Send response: persist `UNMATCHED`, correlation worker ghép lại theo provider message ID/stable key; response đến muộn không hạ trạng thái. Callback không chứng minh recipient đọc; open pixel/read tracking ngoài MVP và cần privacy review.

## 7. Outgoing status/audit

Local outbox trước, shared topic/schema mapping sau N0 approval. Candidate notification status topic `notification.events.v1` và type `vn.omama.notification.delivery.updated.v1` **chưa có trong shared registry**, disabled publish default. Chỉ IDs/status/reason/timestamps/source reference, không contact/payload. Audit adapter phải tuân owner `audit.events.v1` schema đã pin; không tự phát event tự xưng audit-compliant chỉ vì có tên topic. Nếu audit publish chưa nối, local obligation vẫn retained và remote audit acceptance ghi NOT VERIFIED.
