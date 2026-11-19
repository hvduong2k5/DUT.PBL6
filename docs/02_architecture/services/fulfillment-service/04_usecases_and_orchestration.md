# 04 — Use cases và orchestration

[State](02_state_machine_and_lifecycle.md) · [Persistence](03_database_and_persistence.md) · [Contracts](05_api_contracts_and_transports.md). Mọi pseudocode là target, chưa có runtime. Prefix `FUL-UC` dùng cho traceability.

## 1. FUL-UC01 — Nhận ready observation và accept

Actor: ready coordinator/worker có stable service identity `fulfillment-service`; đây là caller namespace duy nhất cho AcceptTask trong profile MVP. Order cấp permit và gọi cancellation/query, không gửi staff create-task command. Input event có schema/source/subject/key đúng và stock COMMITTED theo Order v3; event vẫn chỉ là wake-up.

```text
consumer:
  verify envelope/schema/source/partition; hash exact payload
  T0: insert inbox(source,id,hash) and ensure one ready job(order,generation,event)
  commit Kafka offset only after T0 commit; same id/different hash -> quarantine

ready worker:
  use durable attempt identity; do not generate new key each retry
  get Order packing snapshot, Inventory committed allocation, checklist policy
  validate order/generation/digest/warehouse/reservation and full line quantities
  persist attempt request key, permit request and original peer associations
  get Order permit(task_request_key); persist exact AcceptTask request/hash
  call AcceptTask using that immutable request

AcceptTask:
  authenticate caller + ACL; normalize/hash; inspect original registry (primary)
  if matching durable outcome: return it after current ACL check
  online ValidateFulfillmentPermit; load required input proofs before SQL lock
  T1: insert/lock Order guard; re-read registry and admission
      replay first; check existing ever-accepted task and binding
      reject cancelled/stale generation or ineligible/expired permit durably
      check DB decision clock, policy/input versions, warehouse/digest equality
      create task, package draft, lines, allocation rows, checklist snapshot
      write ACCEPTED and guard accepted_task_id
      append task revision1(PACKING_ACCEPTED), history, audit, outbox, registry
  return TaskDecision with original command evidence and current resource
```

Preloaded proofs bind immutable request; snapshot/allocation missing → no admission, retry/defer with reason. Online validate timeout → UNKNOWN of validation, no accepted decision; caller still query if command may already have been dispatched. Do not hold DB lock across RPC. A rejection outcome can create an OPEN admission row/registry without task; it cannot fabricate CANCELLED proof.

Worker có thể invoke use case nội bộ với trusted system context hoặc qua gRPC từ worker deployment bằng chính Fulfillment service identity. Fulfillment vẫn thực hiện online validate và snapshot/Inventory checks. Không trust snapshot embedded trong caller JSON. Nếu sau này cho thêm caller Order vào AcceptTask, phải mở capability/profile và bổ sung disambiguation originating caller cho command query trước; scope registry không được đổi ngầm. Missing provider trả capability-unavailable, không tạo task bằng placeholders.

## 2. FUL-UC02 — Cancel trước admission

Actor duy nhất mutation: Order orchestrator service. Input `(order_id,generation,key,request_hash,reason,contract_version)`; no staff proxy trực tiếp.

```text
T2 lock Order guard -> admission -> command registry
same scope/key/hash -> replay
guard ever-accepted at another generation -> FAILED_PRECONDITION, no new tombstone
current ACCEPTED -> store REJECTED(TASK_ALREADY_ACCEPTED), preserve task
current CANCELLED -> store ALREADY_APPLIED cancellation outcome
absent/OPEN -> CANCELLED with original decision_id/time and version+1
persist registry, audit and high-watermark before ACK
```

`GetTaskDecision` có filter `command_key + command_action` cùng xuất hiện hoặc cùng vắng. Command filter không thấy → NOT_FOUND, không tạo record. Resource query có thể trả latest resource version; filtered evidence giữ original decision/time. Inventory kiểm DecisionReference bằng authenticated lookup, không tin boolean staff gửi. Cancel REJECTED không được biến thành refund/reverse success.

## 3. FUL-UC03 — Queue, assignment và SLA

Queue chỉ gồm admitted tasks trong warehouse scopes actor. Default filter workflow READY/ASSIGNED/IN_PROGRESS; BLOCKED là tab rõ lý do; COMPLETED là history. Fields: task/order display ID, source channel, quantity/line count, admitted_at, due_at, priority_reason, assignee, workflow, active block summary, local version và integration summary. Không đưa payment amount/address/phone vào queue khi không cần.

AssignSelf expected_version; T3 lock task và kiểm READY, permission, warehouse, staff active then set assignee=verified actor. Hai staff tranh task → một winner, một412, không last-write-wins. Manager reassign có permission riêng và reason; reassignment khi đang IN_PROGRESS chưa hỗ trợ mặc định, phải block task và xử lý handover có audit trước; không đổi assignee giữa lệnh seal.

SLA: snapshot `packing_due_at`, source deadline/version, policy và lead time. Priority sort: overdue trước, rồi earliest due_at; tie-break admitted_at/task_id. Missing due_at không coi urgent giả, hiển thị `SLA_UNAVAILABLE` và route manager, không tự bịa hạn. Update từ Order phải có newer source deadline revision/authority; lưu previous/new và reason, invalidate cursor assumptions. SLA breach job idempotent theo `(task_id,deadline_revision,threshold)`; không tự seal/cancel vì quá hạn.

## 4. FUL-UC04 — Picking theo allocation

Input mỗi slot: `order_line_id,batch_id,counted_quantity,allocation_version,expected_task_version`. Batch barcode resolve bằng Inventory mapping, không dùng raw barcode làm batch authority. Worker cung cấp allocation item: reservation_id/allocation_version, warehouse, SKU/batch ID/code, allocated qty, expiry instant/date policy, quarantine/eligibility revision và location reference nếu có.

T3 kiểm assignee và IN_PROGRESS, slot thuộc task/line/allocation, counted>=0 và <=allocated. Counted=0 được phép ghi nhận chưa lấy trong slot đã tồn tại; không tạo allocation quantity0. Mutation **set counted total**, không tăng +1 vô điều kiện; retry cùng key không nhân số lượng. Device muốn incremental scans phải gán scan operation UUID và map qua server dedup, ngoài MVP API set-total.

Tổng counted theo SKU/line phải đúng snapshot trước seal; batch qty<=allocated, không lấy bù từ batch khác dù Inventory GetBatchFEFODetails gợi ý. SKU sai/batch expired/quarantine → record issue/block, no stock mutation. Reallocation chưa có accepted provider workflow: tạo recovery case với Inventory, giữ task BLOCKED; không tự sửa allocation snapshot. Batch expiry dùng `valid_until` do Inventory chuẩn hóa theo policy/timezone; local UI không tự diễn giải cuối ngày HSD.

## 5. FUL-UC05 — Checklist và blocks

Required steps đề xuất: SKU identity, quantity, batch/expiry check, package integrity, fragile protection nếu cần, seal verification và evidence nếu policy yêu cầu. Chính sách chọn required set; definition text/version persist tại admission. API update `item_code,result=PASS|FAIL,note` theo task version; actor/time server-set. Không cho thêm item PASS tùy ý hoặc gửi `all_checked=true` thay item results.

FAIL có thể tạo QUALITY block theo policy, revision+history. Resolve block cần reason và authority của source: Inventory-block do Inventory proof mới; integrity-block cần review; operator discrepancy manager/assignee theo policy. Không cho staff clear financial/order hold từ endpoint local. Resolve all blocks kiểm lại quyền, actual allocation và required media, sau đó phục hồi resume_state; không tự đánh PASS các checklist.

## 6. FUL-UC06 — Seal và hoàn tất

Input immutable command: task ID, expected_version, package ID, normalized unique seal, parcel measurements, selected VERIFIED evidence IDs. Canonical REST hash bao expected version/body/action/task; JWT/trace IDs/idempotency header không thuộc business hash. Permission luôn được kiểm trước replay.

1. Đọc original registry; nếu command đã áp dụng trả original seal receipt và current view riêng nếu cần. Không revalidate permit đã hết hạn để phủ nhận task từng accepted.
2. Load authoritative Inventory validation cho đúng allocation and task, freshness window; query package/Order packing constraints nếu policy cần. Bind proof to input task version. Thiếu capability →503 CAPABILITY_UNAVAILABLE, chưa có seal.
3. T4 canonical lock order; replay lại; kiểm expected_version, assignee, IN_PROGRESS, no active blocks; input digest/warehouse/allocation still match.
4. Check full required checklist PASS; sum picks exactly required per line/SKU; batch validation and expiry at DB decision time. Proof freshness đề xuất≤5s, never claim global quarantine atomicity.
5. Validate parcel bounds/fragile policy, package belongs task, seal chưa dùng; media VERIFIED+safe+linked đúng package, immutable object version/checksum. Integrated-v2 requires at least one primary video; choose deterministically `primary_evidence_id`.
6. Set package SEALED and task COMPLETED at same DB timestamp. Append task version `n+1` revision `PACKING_SEALED`; persist original task/parcel proof, history/audit and event `packing.completed.v2`; write command receipt. Commit once.
7. Return200 local completion, package/task version and integration PENDING. No sync call to carrier/Order mutation after commit as a condition of local success.

Nếu video optional local mode: có thể hoàn tất task với no media theo policy, nhưng handoff hiện tại phải fail gate trước admission nếu yêu cầu chạy integrated-v2. Không seal local rồi mới phát hiện không thể bàn giao do thiếu policy prerequisites. Sandbox domain-only không được báo “sẵn sàng giao thực tế”.

Event chứa measurements thực; Order giữ shipping fee/recipient đã chốt. Fee/parcel discrepancy vượt tolerance là recovery case, không sửa event đã publish hoặc âm thầm chọn carrier khác. Seal physical xảy ra trong thế giới thực không có DB rollback: UI chỉ hướng dẫn dán/xác nhận sau checklist, nếu DB commit uncertain giữ nhãn và query command, không cấp seal khác.

## 7. FUL-UC07 — Media upload và xem bằng chứng

Create session theo task/package, purpose PACKING_VIDEO, declared type/size/checksum; server sinh object key. Direct upload qua URL scoped method/key/version conditions. Complete request chỉ enqueue verify nếu object upload xong; worker HEAD/scan/checksum rồi CAS VERIFIED. `CompleteUpload=202` không nghĩa video verified. Chi tiết threat model và immutable object ở chương07.

Access URL: actor+purpose/case+warehouse verification; read media metadata; append durable access audit; sign GET exact object/version với TTL≤15 phút. Task has no video →200 `has_video=false` khi authorized; unauthorized →403/404 theo policy không lộ existence. Storage unavailable →503, không trả has_video=false để che lỗi. Care không có quyền thay object/link chỉ vì được xem.

## 8. FUL-UC08 — Exact source revisions cho Order

`GetTaskRevision(task_id,resource_version)` lookup đúng PK, never synthesize from current task. Trả PACKING_ACCEPTED/PACKING_SEALED đúng original `occurred_at,order_id,task_id,ready_generation,items_digest,warehouse_id,seal_code` khi applicable. Revision của assignment/pick/checklist có kind TASK_UPDATED nội bộ; không được dùng làm PACKING_SEALED proof.

Order adapter map `PACKING_ACCEPTED` → Core ingress kind tương ứng và `PACKING_SEALED` → kind tương ứng. Event packing.completed.v2 chỉ có seal; accepted discovery dùng candidate task.accepted.v1 riêng theo chương06, không giả Order đã thấy accepted chỉ từ T1 commit. Codec ký/normalized bytes do Order giữ; event JSON trực tiếp không tương đương signed Core proof. G04 chặn khi accepted-event contract/source adapter/ingress chưa nghiệm thu.

## 9. FUL-UC09 — Reopen/rework (thiết kế mở rộng, disabled)

FR-PACK-20 được giữ trong backlog acceptance. MVP endpoint trả FEATURE_DISABLED. Reopen sau seal cần Order hold + Shipping stop/cancel authority đúng package/shipment version; prove chưa handover hoặc đã stop bằng protocol có fencing với create đang bay. `shipment NOT_FOUND` hoặc local “chưa giao” không đủ vì outbox cũ có thể tới muộn.

Thiết kế future: durable ReworkOperation, barrier chặn old package ở Shipping trước tạo package replacement; package cũ VOID nhưng revision/event gốc không xóa, seal không tái dùng; assignment/checklist mới theo policy; link evidence originals và reason; Order source adapter nhận rework lineage với version mới. Có UNKNOWN thì pending reconciliation, không mở lại task tự động. Không thêm transition COMPLETED→READY trước khi wire/proofs được chấp thuận.

## 10. Các lỗi vận hành và hướng xử lý cho UI

| Code | UI action | Điều cấm |
| --- | --- | --- |
| VERSION_MISMATCH | Refresh task; giải thích đã thay đổi | Auto retry với version mới nhưng cùng key cũ |
| TASK_ALREADY_ASSIGNED | Show current assignment | Ghi đè assignee |
| BATCH_INVALID / PICK_MISMATCH | Block/resolution flow | Thay batch hoặc tăng stock từ UI |
| CHECKLIST_INCOMPLETE | Highlight exact required items | Cho nút override không audit |
| MEDIA_NOT_VERIFIED | Poll file status/retry verification | Tin browser upload progress100% |
| CAPABILITY_UNAVAILABLE | Show pending dependency/retry later | Fallback provider fake |
| OUTCOME_UNKNOWN | Poll original command/task receipt | Generate new seal/package/key |
| TASK_ALREADY_ACCEPTED khi cancel | Escalate Order/warehouse workflow | Báo hủy/hoàn kho thành công |
