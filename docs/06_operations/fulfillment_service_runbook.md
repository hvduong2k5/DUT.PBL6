# Fulfillment operations runbook — thiết kế mục tiêu

> **MVP hiện tại:** đọc [hướng dẫn tích hợp và xử lý retry](../../packages/contracts/mvp/fulfillment-integration.md). Theo dõi thêm `fulfillment_shipping_handoffs` thuộc Order; không xóa yêu cầu gốc hoặc đổi idempotency key khi Shipping timeout. Bộ recovery admin đầy đủ bên dưới vẫn là thiết kế mục tiêu.

Ngày30/10/2026. [Deployment](../05_deployment/fulfillment_service_deployment.md) · [Test plan](../04_testing/fulfillment-service/01_acceptance_and_verification_plan.md). Chưa có CLI/HTTP admin recovery runtime; các bước chỉ định hành động và evidence cần triển khai, không phải endpoint đã tồn tại.

## 1. Triage theo authority

Luôn ghi incident ID, actor, timestamps UTC, order/task/package IDs, generation, original command key/action/hash, source revision, peer SHAs/config và outcome đã biết. Không log JWT/presigned URL/customer PII/video bytes. Giữ một assignee và next-check deadline cho mỗi obligation UNKNOWN.

| Symptom | Kiểm tra trước | Hành động | Không làm |
| --- | --- | --- | --- |
| Ready backlog, no task | Inbox/job durable, permit/snapshot/allocation capability, Order hold/generation | Close new admissions nếu peer outage; retry/reconcile original attempt | Tạo READY task tay hoặc bypass permit |
| Cancel timeout | GetTaskDecision original key/action + current state | UNKNOWN giữ obligation; query/retry same request | Suy NOT_FOUND thành stopped/restock |
| Cancel REJECTED/ACCEPTED | Task identity/warehouse/physical progress | Escalate Order/warehouse; local block nếu authorized | Báo hủy/hoàn kho hoàn tất |
| Staff không seal được | assignee/version/checklist/picks/blocks/media/fresh validation | Resolve đúng source guard, refresh authoritative input | Override all_checked hoặc đổi batch tự do |
| Seal timeout/COMMIT unknown | Original command registry/package/revision/outbox | Retry same key/body under guard; show pending | New seal/package/key |
| COMPLETED nhưng chưa có vận đơn | Outbox ACK, Order adapter exact seal, Order snapshot readiness, Shipping association | Retry/reconcile same package/consumer, record peer fault | Direct carrier create hoặc switch v1 |
| Media stuck VERIFYING | Exact object version/hash, scanner job/lease/policy | Retry actual verify; keep required-media blocked | Mark VERIFIED từ browser checksum |
| Video cannot view | Permission/purpose/case, object state/version, URL TTL | New authorized access request; audit failure | Public bucket hoặc share long-lived URL |
| Batch recalled after checkpoint | Inventory source revision, task/package/shipment state | Block work, quality/recall case with owners | Assume checkpoint proof forbids all later recalls |

## 2. Decision trees

### 2.1 Admission UNKNOWN

1. Query original `(caller,AcceptTask,order,key)` on primary authority and resource `(order,generation)`.
2. Original APPLIED/ALREADY_APPLIED → use returned task identity, verify bindings; no new accept.
3. Original REJECTED → handle exact reason. If expired attempt, only new eligible generation/attempt policy after terminal original outcome, never mutate old bytes.
4. Query NOT_FOUND/current OPEN → result is still observational; retry original AcceptTask so guard serializes with any in-flight commit. If original request expired, expect durable rejection, retain original response.
5. Query unavailable/ambiguous → continue scheduled query, operator case and circuit admission new orders if backlog grows. Do not purge registry/job.

### 2.2 Seal done locally but handoff pending

1. Confirm exact PACKING_SEALED revision and original event ID/payload hash; ensure physical seal matches package.
2. Outbox pending/broker UNKNOWN → retry original event, preserve partition key/time/ID. ACKed does not prove Shipping applied.
3. Order not PACKED → inspect accepted+sealed source adapter receipt and package binding; recover exact revisions, not latest state.
4. Shipping snapshot not eligible → retry/defer until authoritative Order apply, or escalate mismatch case. Never use current Profile address.
5. Shipping has shipment/package association → reconcile projection, do not create another shipment. Missing query capability → pending manual investigation, not presumed absence.

## 3. Security/media incident

Close new media upload/access admission as needed; preserve read/verification evidence. Revoke compromised credentials via owning platform, restrict bucket/topic; issued presigned GET may remain usable until expiry, document impact. Pin object version/hash and access issuance audit; avoid downloading video into logs/chat. Don't replace original file to conceal tampering. Create integrity block/recovery case for affected packages; coordinate Care/Order/Shipping according to physical progress.

Cleanup failures keep DELETE_PENDING with exact object version; retry delete and store receipt. Never recursive-delete shared bucket/prefix. Legal/case hold overrides normal retention; deletion eligibility checked under media lock.

## 4. Backlog and worker failure

Signals: oldest outbox, ready job age, peer UNKNOWN age, media verify latency, lease expiry, active blocked/SLA breaches. Triage by one stream/job; avoid global “mark all sent/done”. Scale within DB/peer capacity only after checking retry storm/poison payload. MANUAL_REVIEW jobs still schedule reconciliation; execution budget exhaustion must not stop financial/physical obligation queries.

Lease owner crash: next worker claims after DB expiry and increments epoch. Old response may still be valid external proof; append observation/reconcile but no stale local write. DB fencing alone cannot retract Kafka/RPC sent before crash. Duplicate event/response expected, validate source identity/hash/revision.

## 5. Backup/restore drill

1. Declare restore mode, disable admission/seal/new effect dispatch/publisher; preserve already-running request inventory and peer associations. Isolate restored database from production mutation workers.
2. Restore DB to target point and exact object manifest; don't reset Kafka offsets/keys casually. Compare pending/accepted guards/task/package IDs against Order and Shipping authority, including resources newer than backup.
3. Reconcile original commands/outcomes; recreate recovery metadata only from verified peer evidence, retain immutable originals and mark restore provenance. Unknown cannot become not-found terminal by operator choice.
4. Verify videos/checksums/versions, registry/tombstone uniqueness, outbox late duplicates and source adapter revisions. Run FUL-T37 plus relevant tests.
5. Named operator+reviewer sign-off and documented residual obligations; enable queries/recovery first, publisher next as safe, new admission last. Keep existing cohort protocol.

## 6. Safe operator capability design

Future recovery APIs create/query recovery operations with permission, reason, assignee, evidence refs and expected version. Allowed: request reconciliation, retry same job identity, record verified external observation, resolve own-source block via policy. Forbidden: PATCH task/stock/payment state directly, delete guard, alter original event/revision, invent CANCELLED proof, resend carrier/refund using fresh key. Every action writes durable audit and produces a reviewable result.

## Outbox dead letters

After 10 recorded Kafka publish failures, a delivery is retained in
`fulfillment.outbox` with `dead_lettered_at` and `last_error_code`.
Migration 000004 also quarantines unpublished records that already reached
10 attempts. Apply it with the publisher stopped to avoid an in-flight
delivery during the migration. Later deliveries in the same stream wait
for recovery. Other streams continue. Inspect the worker log by event ID and correct the Kafka issue
before clearing `dead_lettered_at`, `last_error_code`, and
`consecutive_failures` for that event and setting `next_run_at` to the
current time. Preserve the event ID and payload on replay.
