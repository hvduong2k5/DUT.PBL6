# 03 — Persistence và transaction, baseline 3.2

> **Phạm vi áp dụng MVP 1 — 15/10/2026:** đọc [quyết định giao bản đầu](13_mvp_delivery_decisions.md). Chỉ khách đăng nhập, VietQR chuyển khoản qua SePay Payment Gateway; không COD/Guest/voucher, không phương thức phụ, hoàn tiền thủ công có kiểm soát và dừng tự động tại DELIVERED. Các nhánh mở rộng dưới đây không bật trong MVP. Implementation sandbox của đợt đầu được ghi ở [báo cáo](14_mvp_implementation_and_adr_report.md); baseline 3.2 chưa được nghiệm thu tích hợp production.

[Trước](02_state_machine_and_lifecycle.md) · [Tiếp](04_usecases_and_saga_orchestration.md)

## 1. Runtime schema và target

PostgreSQL là source of truth. [Migrations 3.1](../../../../services/order-service/migrations/000001_order.up.sql) và [schema snapshot](order_schema_candidate.sql) mô tả runtime sandbox. Bảng dưới dùng tên runtime thật; field target chưa có được ghi rõ, không coi tài liệu là migration đã áp dụng.

| Bảng | Trách nhiệm và constraint |
| --- | --- |
| checkout_operations | input/canonical encrypted, planned_order_id unique, status/stage, lease/epoch, reservation reference/expiry |
| idempotency_records | PK(scope,kind,key), hash, operation FK, response retention |
| orders / order_items | unique operation/order code, snapshot immutable, quantity/money CHECK, PK(order,sku), version |
| receipts / allocations | unique(provider,transaction_id); nullable matched Order; allocation unique receipt và Order cho MVP exact |
| reconciliation_cases | receipt reference/reason, trạng thái xử lý riêng |
| jobs | unique command key, stage/status/attempts/lease/epoch |
| command_registry | scope/kind/key/hash + stored outcome cho commands |
| cancellation_requests | request/reviewer/outcome; target bổ sung barrier proof reference |
| refunds | unique approval/operation key, amount/state/provider ref |
| order_history | unique(order,version), actor/source/reason/time |
| outbox / inbox | stable event ID/version/ordinal; inbox source/event/hash + durable outcome |
| source_projections | unique(order,source,resource), last applied version |
| audit_records / quarantine | append-only audit; durable rejected-message reference/hash |
| target resource operations | per-action key/hash/digest, last outcome/version/decision time, barrier evidence, next query time |

Không có cross-database FK; remote IDs là references phải validate qua port. Target resource operations có thể nằm trong job payload đã kiểm schema hoặc bảng riêng, nhưng phải truy vấn được theo obligation và không chỉ nằm memory.

```mermaid
erDiagram
    checkout_operations ||--o| orders : places
    checkout_operations ||--o{ idempotency_records : replays
    orders ||--|{ order_items : snapshots
    orders ||--o{ jobs : coordinates
    orders ||--o{ order_history : records
    orders ||--o{ allocations : receives
    receipts ||--o| allocations : allocates
    receipts ||--o{ refunds : funds
    receipts ||--o| reconciliation_cases : reconciles
    orders ||--o{ source_projections : observes
```

## 2. Local transaction boundaries

| Tx | Atomic writes | Sau commit |
| --- | --- | --- |
| T0 acceptance | operation + dedup + encrypted input + planned ID + audit | trả 202; worker claim |
| T1 canonical | validated snapshot + deadline + acquisition intent/key | voucher/stock call |
| T2 remote result | identity/digest/version/outcome + next stage | step tiếp/query nếu UNKNOWN |
| T3 placement | Order/items/history/placed outbox + operation SUCCEEDED | client query thấy Order |
| T4 receipt | verified receipt + allocation hoặc reconciliation + finalize job | gọi Inventory/Promotion |
| T5 finalize | resource proof + PAID/history/paid outbox + ready job | publish ready khi đủ guard |
| T6 cancel pending | cancelled/history/outbox + release intent | release-by-order |
| T6b cancel committed | hold + cancel intent; sau terminal barrier/reverse ghi cancelled | refund approval/execution độc lập |
| T7 event consume | inbox + effect/history/outbox hoặc DEFERRED/REJECTED durable | commit Kafka offset |
| T8 refund budget | lock receipt, reserve obligation, approval/key | provider refund/query |
| T9 refund result | verified settlement + summary/history/outbox | accounting/notify |

Không giữ SQL transaction trong network call. Kết quả commit SQL bị mất ACK: đọc registry/operation/Order trước khi quyết định compensate.

## 3. Locks, fencing và time

Lock hierarchy trong Order: Order → receipts theo ID → jobs/refunds. Unmatched receipt không quay lại lock Order cùng transaction; match ở workflow riêng theo đúng thứ tự. Cross-row refundable tính dưới receipt lock. Mọi mutation guard/version/history phải atomic; rows affected=0 phải xử lý conflict.

Worker claim SKIP LOCKED, lease default 10s/epoch tăng; chỉ owner+epoch còn hạn được ghi outcome. Remote stable key không đổi theo epoch. Lease local không chặn RPC cũ còn bay: provider tombstone/idempotency giải quyết.

Decision clock lấy sau row lock; Inventory dùng DB clock của Inventory. Không giả định đồng hồ hai DB tuyệt đối giống nhau; final authority là remote outcome. Deadline local lấy min và safety margin để giảm payment sát hạn, không bảo đảm finalize nếu network delay.

## 4. Idempotency và retention

Canonical hash gồm scope/action, immutable planned order/resource, sorted consolidated items, warehouse/method và policy có ảnh hưởng effect. Cùng key/hash → cùng effect; khác hash → conflict. Query trả current resource state riêng với original command result để tránh trả stale RESERVED như hold còn sống.

Checkout response retention 7 ngày; terminal replay sau đó 410 + operation reference. Registry giữ tối thiểu 30 ngày và không auto reuse key trong MVP. Nghĩa vụ chưa terminal, receipt/refund uniqueness và remote cancellation tombstone phải giữ vượt mọi retry/replay window. Khi chưa giới hạn được lifetime request cũ, không xóa tombstone. Compaction/PII deletion cần policy riêng; không xóa ledger để “dọn cache”.

## 5. Outbox / inbox

Outbox effect + event cùng transaction, stable ID, aggregate version + ordinal, partition key order_id. Publisher chỉ advance sau broker ACK; crash trước mark published → duplicate cùng ID. Không có broker thật thì production fail readiness, không log-only rồi mark published.

Inbox dedup theo source/event ID + payload hash, kết hợp business resource/version. Event thiếu prerequisite → DEFERRED + payload bền vững; worker query source và apply sau. Poison event → durable quarantine/DLQ intent trước ACK. Message N lỗi transient không được commit N+1 cùng partition vượt N, trừ khi N đã vào durable retry queue. Không claim exactly-once xuyên PostgreSQL/Kafka.

## 6. Data operations

Redis: cart revision/TTL30d, encrypted preview TTL5m, read cache; mất Redis không mất accepted input/obligation. Không cache ownership/money quyết định như nguồn chuẩn.

Index: operation/job run_after+lease; pending payment expiry; customer keyset(created_at,id); queue(stage,sla_due_at,id); receipt uniqueness; outbox aggregate sequence. SLA theo state entry/policy, không hard-code created_at cho mọi stage.

Migrations expand/backfill/validate; không xóa ledger ở rollback. Restore drill phải reconcile Inventory/provider mới hơn backup trước resume. Key rotation giữ khả năng decrypt snapshots đang cần; local audit hash chain chưa chứng minh chống tail deletion bởi privileged actor. PITR/HA/retention là gate vận hành, chưa được chứng minh bởi tài liệu này.

## 7. Delta persistence MVP 1

[Migration 000004](../../../../services/order-service/migrations/000004_mvp_payment.up.sql) đã bổ sung payment_attempts với order_id unique, merchant_invoice unique theo provider/environment, amount/VND, expiry nội bộ, provider_order_id nullable, provider_order_id nullable; tiến độ đối soát nằm trong sepay_scan. T3 lưu Order và attempt/ý định chuẩn bị thanh toán cùng transaction. Chống trùng receipt theo provider + environment + transaction ID.

Refund thêm execution_mode=MANUAL_BANK_TRANSFER, người nhận việc, mã giao dịch ngân hàng/bằng chứng, người kiểm tra và thời điểm xác nhận. T8 giữ budget trước khi người thực hiện chuyển khoản; T9 chỉ thành công sau đối chiếu bằng chứng. UNKNOWN giữ budget và quyền xử lý, không tự gửi lại tiền. Không giả lập manual proof thành kết quả SePay refund. Xem [quyết định MVP](13_mvp_delivery_decisions.md).

Implementation dùng bảng manual_refunds riêng với payee/evidence mã hóa, để không đưa nghĩa vụ manual vào worker refund tự động. Budget tính cả refunds lịch sử và manual_refunds; hai ledger không cho phép hoàn vượt khoản thu. Xem [ADR-003](adr/ADR-003-manual-refund.md).
