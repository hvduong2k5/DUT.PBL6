# 08 — Tích hợp và khả năng mở rộng

[Gaps](09_decisions_and_contract_gaps.md) · [Contract appendix](../../../03_api_specs/fulfillment-service/README.md). Matrix này phân biệt observed source và target; không là biên bản sign-off của peer owners.

## 1. Provider/consumer matrix

| Peer | Fulfillment cần | Source đã thấy tại SHA manifest | Công việc còn cần |
| --- | --- | --- | --- |
| Order | Permit+online validation; immutable packing inputs; apply source proof; shipping snapshot | Durable Order/Core/auth; proto declares permit/shipping; gRPC handler chưa implement các method đó | Implement owning RPCs/adapter; exact package binding; E2E acceptance |
| Inventory | COMMITTED allocation exact reservation, quantities/batch/expiry/quality; eligibility checkpoint | Reserve/release/stock/FEFO và paid-v1 stock deduction legacy | Terminal protocol3.2 và packing allocation/validation provider; no double deduction |
| Shipping | Packing completed v2 consumer, snapshot query, one shipment/package; v2 revisions | v1/v2 server, bound quotes/revisions và cohort-aware v2 events; consumer validates video/task/generation | Run with real Order snapshot+Fulfillment event; don't infer full E2E from code |
| Identity/Gateway | Staff/service identities, audience fulfillment-api, permissions/warehouse membership | Keycloak principal+customer Order claims; gateway conventions vary by service | Dedicated staff/delegation profile + ACL/mTLS acceptance |
| Profile | Optional employee/compliance authority | Profile/address/Identity consumer code | Chỉ add staff compliance gate nếu requirement owner chốt; không dùng shipping address book làm packing source |
| Care | Authorized video lookup, case entitlement, rework approvals future | Ticket/media modules; return/refund foundations; Order ownership and return workflows pending | Delegation API/purpose and case authority; no assumed refund approval runtime |
| Audit | Async audit export/checkpoint | Schema audit.record.v1 | Codec/chain/checkpoint acceptance, không block business vì Audit server down |
| Notification | SLA/packing notification intents | Chưa có runtime trên refs đã audit | Consumer feature gate, no sync email trong seal Tx |
| Traceability | Batch provenance references | Batch/QR backend code | Link batch IDs qua Inventory/Traceability API; không lấy QR story làm allocation authority |
| Catalog | Packing text/fragile flags via Order snapshot | Public/admin Catalog implementation | Không query live Catalog để rewrite purchased lines |
| Channel/Promotion/Finance/Analytics | Future events/policy/reporting | Chưa có core runtime tương ứng trên audit refs | Contract/version/capability review trước bật |

## 2. Packing inputs — đề xuất đóng khoảng trống dữ liệu

Permit chỉ có digest/warehouse/generation/policy reference, không có line list, checklist definitions, deadline hoặc allocation. Ready event cũng không chứa đủ thông tin. Vì vậy cần hai peer read providers, không được fill dummy data:

**Order.GetPackingSnapshot (candidate):** input order/generation/contract_version; authenticated caller Fulfillment. Output immutable snapshot ID/version/schema, lines theo order_line_id/SKU/qty/name/packaging/fragile, items_digest, warehouse, reservation_id, policy_version/checklist/evidence policy, channel, packing_due_at và deadline_revision. Snapshot không cần customer PII/price. Response binds original readiness generation; latest order detail không thay thế snapshot khi replay.

**Inventory.GetCommittedPackingAllocation (candidate):** input order/reservation/generation/digest/warehouse. Output state COMMITTED, allocation_version, observed_at, original command association, batch slots theo order line hoặc deterministic SKU-to-line mapping, qty, expiry instant, current quality/eligibility revision và warehouse/location. Inventory proto ReservationOutcome hiện chưa trả committed allocation chi tiết; GetBatchFEFODetails chọn FEFO theo stock hiện tại không phải allocation đã commit.

**Inventory.ValidatePackingAllocation (candidate):** input task/order/reservation/allocation_version/pick slots/checkpoint expected binding. Output allowed/reason, binding IDs/version, validation_id/observed_at/valid_until, per-batch eligibility and quantity facts. Check no expired/quarantine at validation checkpoint. Sau proof provider có thể recall/quarantine; cần block/recall workflow và giới hạn checkpoint được mô tả rõ, không giả lease là distributed lock. Khi hard real-time stop là yêu cầu, gate G08 đòi protocol fencing riêng.

Candidate proto nêu shape và normalized values; chuyển sang owning service namespaces sau review và giữ field compatibility. Caller không gọi một facade trả allowed=true chỉ từ request copy.

## 3. Order source proof và package binding

Shared GetTaskRevision có order/task/version/generation/kind/seal/time/digest/warehouse, thiếu package_id/parcel. Shipping GetShippingSnapshot đòi Order kiểm package thật. G04 đề xuất additive exact `GetPackageRevision` (package_id, task_version) hoặc append fields package_id/parcel/checksum theo wire review; không sửa tag/type cũ. Payload phải từ immutable seal revision.

Order adapter ownership: verify provider exact revision → normalize/sign theo Core codec → submit trusted ingress → durable association/receipt. Accepted discovery chọn dedicated candidate task.accepted.v1 topic/schema theo chương06, persist ở T1 và consumer deploy trước producer; nếu seal tới trước thì defer và fetch original accepted checkpoint. Source event không được cấp Order signing key hoặc bypass ACL. Accepted schema mới vẫn cần review/promote vào shared contracts, không coi đã có runtime.

Shipping call có thể đến trước Order apply seal; response not-eligible/transient/defer thì retry same package. Order không đợi shipment-created mới bind seal, tránh vòng lặp. Shipping consumer hiện retry dependency nhưng invalid event có thể bị commit-and-skip theo code: provider phải validate schema/runtime stricter trước publish, peer cần acceptance quarantine/invalid handling riêng, không tuyên bố lossless replay đã có.

## 4. Extension design và migration impact

| Extension | Giữ ổn định | Cần thêm | Gate |
| --- | --- | --- | --- |
| Multiple package/task | Task/order identity, old package revision/event | package lines qty coverage, per-package seal/evidence, task completion aggregate, Order snapshot per package | One-shipment/package và partial state semantics |
| Multi-warehouse | Existing generation/guard/tombstone | Order fulfillment_group_id, Inventory per-group resources, group barrier, partial cancel/return | Unique task/order phải migrate cùng provider protocol |
| Rework after seal | Original revision/event/seal và audit | ReworkOperation, stop/fencing Shipping+Order, replacement package lineage | No in-flight create/handover UNKNOWN |
| Marketplace/B2B | All admission checks | Channel/policy snapshot, custom packing checklist/lead time; Order remains owner | Source channel cannot bypass stock/payment |
| POS pickup | Snapshot/stock authority | Pickup fulfillment policy and handover proof; may skip carrier explicitly | Order POS capability accepted |
| Optional/no-video shipping | Current v2 frozen | New event/profile and Shipping consumer accepted, evidence mode explicit | Không gửi empty key/fake checksum vào v2 |
| Compliance gate | Staff current identity and warehouse | Profile qualification read contract+freshness/revocation policy | Không mặc định customer API là HR authority |
| Notifications/analytics | Business event version, no PII | Async subscribers, read projections/materialized views | Không thêm runtime dependency vào seal |

Policy extension không dùng `map<string,any>` để nhét business command tùy ý. Tách typed versioned packing instructions và known extension namespace; consumer reject required capability chưa hiểu. Backward-compatible optional fields chỉ khi semantics cũ vẫn đúng. Breaking required fields →version mới/cohort mới.

## 5. Contract rollout checklist

1. Chốt artifact/version, named owner, provider/client PR và acceptance tests; pin manifest SHA hai phía.
2. Additive shared proto/events trước, regenerate với toolchain pinned; static wire checker không thay runtime proof.
3. Deploy peer read authority và Fulfillment provider/query/recovery trước mutation cohort.
4. Deploy Order/Shipping consumers v2 và source adapter; test replay before producer enable.
5. Isolated test cohort with one Order end-to-end; double scan/seal/replay, cancel races, wrong warehouse and lost ACK.
6. Enable new admission incrementally; existing obligations pinned to original protocol. Rollback close new admissions, keep queries/outbox/reconcile alive.

Không merge/publish/bật gates ở lượt tài liệu. Registry acceptance chưa được tự đổi vì candidate artifacts parse thành công.
