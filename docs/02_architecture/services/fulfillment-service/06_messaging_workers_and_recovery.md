# 06 — Messaging, workers và recovery

[Persistence](03_database_and_persistence.md) · [Runbook](../../../06_operations/fulfillment_service_runbook.md). Delivery at-least-once; stable identities và authority guards làm effects idempotent.

## 1. Topics và dispositions

| Topic | Role | Key/dedup | Effect |
| --- | --- | --- | --- |
| order.events.v3 | Consumer fulfillment_ready | aggregate_id/order_id; source+CloudEvent id+hash | Durable ready job; online permit before accept |
| fulfillment.task.events.v1 (candidate mới) | Producer task.accepted | order_id; task/version + event UUID | Order adapter discover exact PACKING_ACCEPTED revision |
| fulfillment.events.v2 | Producer packing.completed | order_id; event UUID + task/version | Shipping handoff and Order seal adapter |
| audit.events.v1 | Producer audit.record | entity_id; audit entry/event UUID | Export local durable audit; no business authority |
| fulfillment.invalid.v1 (proposed private quarantine) | Metadata/reference disposition | original source/id/hash | Diagnose invalid contract/security; not automatic business replay |

`paid`, `placed`, `cancelled` facts có thể cập nhật projection/trigger reconcile nhưng không thay CancelTask/permit protocol. Source event generation/version thấp không làm reopen terminal decision. Source high version không bỏ prerequisites; Order version có thể gap vì không phải version nào cũng phát ready.

Ingress phân loại: VALID→job, DUPLICATE_SAME_HASH→no new job, DUPLICATE_DIFFERENT_HASH→QUARANTINED, TRANSIENT_DB_FAILURE→no offset commit, UNSUPPORTED_SCHEMA→durable quarantine. Nếu source/ACL giả mạo, ghi security metadata tối thiểu; raw payload có thể chứa PII phải mã hóa/giới hạn truy cập/retention, không dump logs.

## 2. Inbox and Kafka offset correctness

T0 lưu source/id/payload_hash/original payload, source version/generation, topic/partition/offset và disposition. `DEFERRED` phải có durable job/reason/next_run, không chỉ in-memory timer. Commit broker offset chỉ sau DB disposition commit; crash trước commit offset → replay dedup safe.

MVP consumer xử lý serial mỗi partition và chỉ commit contiguous persisted offsets. Nếu sau này xử lý parallel, cần partition completion tracker; không commit offset100 khi99 chưa durable. Rebalance revoke không tiếp tục claim jobs mới, finish bounded local Tx hoặc release; business jobs đã durable không phụ thuộc broker assignment. Đừng dùng group rebalance làm quyền ghi resource.

## 3. Transactional outbox protocol

Mỗi business transition tạo immutable event trước commit, cùng tx với resource. Stream key=`order_id` cho business; audit có stream riêng. Unique `(stream_id,stream_version,ordinal)`; seal event ordinal deterministic; event UUID/time/payload không regenerate khi retry.

1. Claim head chưa ACK của mỗi stream bằng `FOR UPDATE SKIP LOCKED`, set lease owner/epoch/expiry trong Tx ngắn.
2. Publish Kafka outside Tx, require configured broker ACK. Không mark sent từ enqueue client buffer.
3. ACK Tx update row bằng event_id+lease owner/epoch, mark published_at. Chỉ claim next stream event khi head có durable ACK disposition.
4. Crash/mất ACK → publish lại original event. DB lease không fence Kafka request cũ, late duplicate có thể tới sau event mới; consumer vẫn phải dedup và kiểm original revision/generation.

Không hứa total order toàn broker sau failover. Unknown broker delivery không sửa event/ID/partition key. Poison event quarantine có explicit operator decision/recovery case; không skip silent làm mất seal obligation. Audit stream lỗi không chặn business stream publisher, local audit vẫn durable.

## 4. Job registry và budgets

| Job kind | Trigger/key | Success oracle | Timeout/UNKNOWN |
| --- | --- | --- | --- |
| READY_ADMISSION | order,generation,source event | AcceptTask original decision | Query same key/action; expired attempt must terminally reject before new key |
| MEDIA_VERIFY | file_id,object_version,scanner_policy | Verified file CAS + checksum/scan evidence | Re-HEAD/scan same object; no overwrite |
| OUTBOX_PUBLISH | event_id | Broker ACK recorded | Duplicate-safe original payload |
| SOURCE_RECONCILE | task_id,source revision,peer | Order adapter authoritative receipt/application | Continue query/recovery, no new source event fabricated |
| SHIPPING_RECONCILE | package_id | Shipping shipment authority | Query package association; no carrier direct mutation |
| SLA_ALERT | task_id,deadline_revision,threshold | Durable dedup notification/audit intent | Does not alter task state |
| MEDIA_CLEANUP | file_id,object_version | Exact object deleted + metadata receipt | Keep DELETE_PENDING on unknown |

Default đề xuất: remote attempt2s, lease15s, heartbeat5s, exponential full jitter cap30s; execution budget10 attempts hoặc15 phút. Stage limit không kéo dài business permit/media expiry. **Reconciliation budget riêng**: tiếp tục query UNKNOWN sau execution exhaustion, đề xuất1 phút→5 phút, escalation sau30 phút và assigned recovery case. MANUAL_REVIEW không đồng nghĩa ngừng query obligation.

Claim bằng DB `next_run_at`, lease epoch monotonic. Stale worker không apply local mutation khi epoch/version sai; external response hợp lệ vẫn có đường append observation/reconcile. Worker epoch không ngăn external side effect đã dispatch; remote stable command key và task uniqueness mới chống duplicate. Queue max inflight và peer bulkhead giới hạn tránh bão retry.

## 5. Order source adapter

Order Core nhận normalized/signed source proofs theo codec riêng. Adapter owner Order, Fulfillment cung cấp exact revision. Không chia sẻ private signing key của Order vào mọi provider. Mapping task accepted/sealed phải giữ source_id/version/kind/original occurred_at/generation/digest/warehouse; seal cần package binding/proof theo G04. Fetch latest khi event cũ tới không hợp lệ.

Accepted discovery chọn **dedicated candidate fact** `vn.omama.fulfillment.task.accepted.v1` trên topic mới `fulfillment.task.events.v1`; [schema](../../../03_api_specs/fulfillment-service/task_accepted.v1.candidate.schema.json). T1 ghi accepted event vào cùng outbox với revision1; Order adapter subscribe topic này và fetch exact GetTaskRevision(task_id,resource_version). Payload không có PII hoặc authority tự set Order state. Không nhét type mới vào topic Shipping completed-v2 để tránh consumer hiện tại skip như invalid event.

Đây là contract mới đề xuất, cần Order/provider sign-off và promote vào shared packages trong G04, chưa phải event hiện có. Seal packing event chỉ gợi lookup exact sealed revision. Adapter xử lý seal tới trước accepted discovery bằng defer và query original accepted checkpoint version1 (bất biến tại admission theo thiết kế này), không synthesize accepted time từ seal event. Nếu source checkpoint thiếu/sai kind thì giữ pending/recovery. Consumer subscribe trước producer; cả hai events giữ identity/time/payload khi retry.

Proof ký sai, source revision mismatch hoặc body đổi cùng event ID → quarantine/recovery. Adapter response timeout → query normalized ingress/operation identity theo agreed contract, không re-sign event với source timestamp mới. Shipping được retry GetShippingSnapshot khi Order chưa apply proof; không chờ Shipping thành công để Order mới nhận seal, tránh circular dependency.

## 6. Failure matrix

| Failure | Durable state còn lại | Recovery |
| --- | --- | --- |
| DB unavailable trước inbox commit | Broker message chưa acknowledged | Retry delivery; no fake disposition |
| Crash sau accept commit trước RPC ACK | ACCEPTED task + registry + outbox | Query key/resource, return original |
| Permit valid rồi Cancel thắng lock | CANCELLED tombstone | Accept reject; Order xử lý cancel proof |
| Scanner timeout | VERIFYING + job | Retry exact object; block required media |
| Seal COMMIT UNKNOWN | May have COMPLETED/package/outbox | Same key under guard; never new seal |
| Kafka down sau seal | COMPLETED + pending outbox | Backoff/recover; UI integration PENDING |
| Order ingress down | Immutable revision/event | Adapter retry/reconcile; Shipping may defer |
| Shipping v2 absent | Gate false trước admission new cohort | Keep recovery old cohort; no v1 fallback |
| Invalid event at partition head | Durable quarantine disposition | Commit only after persistence; route operator |
| Restore DB older than peer | Peer task/shipment may exist | Admission disabled; authoritative reconciliation |

## 7. Observability và alert design

Metrics: admission result/reason; accept-cancel race outcome; ready job age; active blocked count by reason; seal rejects; media verify latency/result; outbox oldest pending; adapter deferred/UNKNOWN age; unresolved recovery cases; SLA breach counts. Low-cardinality labels kind/reason/warehouse nếu warehouse set bounded; không order/task/user/seal ID labels.

Logs structured có request_id, correlation_id, job_id, stage, source_revision and redacted resource references; token/object URL/PII không log. Trace context giữ qua inbox/job/outbox; retry tạo span mới linked original trace, event traceparent pin lúc event creation. Metric exporter/OTLP down không block business Tx. Alert đề xuất oldest outbox>60s, unknown>5m, ready backlog>5m, any digest mismatch/quarantine collision; ngưỡng phải đo và tune.
