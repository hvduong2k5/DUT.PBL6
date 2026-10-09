> **Implementation 3.1 — 09/10/2026:** [08_implementation_and_acceptance.md](08_implementation_and_acceptance.md) là nguồn hành vi hiện tại. Acceptance tạo checkout operation, chưa tạo Order; PAID và outbox chờ terminal stock/voucher outcome. Các đoạn v3.0 khác mô tả candidate cần đọc cùng cập nhật này.

# 06 — Kiểm thử, recovery và truy vết

[Chỉ mục](README.md) · [Trước: Contracts](05_api_contracts_and_transports.md) · [Tiếp: Quyết định/gaps](07_decisions_and_contract_gaps.md)

**Trạng thái:** danh mục yêu cầu gốc v3.0; kết quả runtime 3.1 và phạm vi từng release được ghi tại [báo cáo nghiệm thu](../../../04_testing/order-service/03_implementation_verification.md). Các feature mở rộng chưa triển khai không được đánh dấu PASS.

## 1. Phân lớp code đề xuất

```text
services/order-service/
  cmd/server/                 bootstrap HTTP/gRPC và shutdown
  internal/domain/            Order/Money/guard; không import DB/client
  internal/usecase/           Cart, Quote, Checkout, Receipt, Cancel, Claim, Refund
  internal/saga/              durable process manager, retry/finalize/compensation
  internal/ports/             UnitOfWork, repositories, external capabilities
  internal/repository/        PostgreSQL/Redis adapters; DBTX shared
  internal/transport/         HTTP, gRPC, Kafka; auth/schema/input validation
  internal/provider/          webhook adapter và bank/refund query
  internal/worker/            outbox, pending inbox, timeout, completion, saga
  migrations/                DDL versioned; immutable constraints/backfill
  tests/unit/                 money/FSM/authorization rules
  tests/integration/          DB atomicity, unique, locking, publisher/inbox
  tests/contract/             actual REST/proto/schema compatibility
  tests/fault/                lost ack, crash, lease, TTL race
  tests/load/                 real dependencies, explicit dataset/concurrency
```

Đây là blueprint Go tham khảo theo implementation Profile; lựa chọn ngôn ngữ Order chưa tồn tại trong code. Không biến đường dẫn blueprint thành link file đang có. Unit test pure domain; integration test DB thật; contract test gọi server registered thật; fault test kiểm ledger/references sau restart, không chỉ HTTP status.

## 2. Ma trận test có oracle

| ID | Arrange / Act | Assertion bắt buộc |
| --- | --- | --- |
| ORD-T01 | Customer A query/cancel/address của B; fake internal headers | 404/403 phù hợp, không PII/effect; sub không bị dùng làm customer ID |
| ORD-T02 | Guest biết code + phone nhưng không valid proof; replay proof | không cấp quyền; OTP/proof đúng bind được; expired/replay bị chặn |
| ORD-T03 | 100 concurrent checkout cùng scope/key/body; Redis flush | một checkout operation, một Order sau placement và một reserve business effect; cùng operation response |
| ORD-T04 | cùng key khác body; hai customers cùng key | conflict đầu tiên; scope khác độc lập, không rò response |
| ORD-T05 | SKU duplicate, inactive, changed price, address version changed, ship fee unavailable | không silently reprice; no resource acquisition khi validation fail |
| ORD-T06 | cart free shipping và merchandise discount; overflow/currency/nanos | arithmetic đúng; invalid rejected, tổng line bằng subtotal |
| ORD-T07 | voucher hold thành công rồi reserve rejected | release hold một lần; checkout terminal fail; mọi references truy được |
| ORD-T08 | reserve commit ở Inventory rồi mất reply; Order process crash | durable intent trước RPC; retry/query tìm cùng reservation, không reserve lần hai |
| ORD-T09 | release đến Inventory trước reserve request trễ | tombstone chặn reserve trễ; available cuối đúng, không treo hold |
| ORD-T10 | DB ack mất sau T3 commit; client retry | dedup result/order giữ nguyên, không compensate order thành công |
| ORD-T11 | exact/under/over/multi-small/unknown-reference/late receipt | mọi valid receipt được giữ; chỉ exact policy được allocate; không late PAID |
| ORD-T12 | duplicate provider transaction cùng payload và khác payload | same no-op; altered conflict/security case, không receipt/allocation mới |
| ORD-T13 | receipt vs Order timeout vs Inventory TTL finalize, tại before/equal/after expiry | chỉ terminal stock outcome hợp lệ; không PAID/ready nếu stock expired |
| ORD-T14 | finalize thành công mất reply; stale worker epoch | query stable key ra COMMITTED; không release committed, stale không update |
| ORD-T15 | OrderPaid trước voucher finalize, COD không paid, ready duplicate | packing chỉ sau ready authorization; một task; COD payment UNPAID |
| ORD-T16 | cancel và Fulfillment acceptance đồng thời; delayed ready | một terminal barrier decision; cancelled không bắt đầu packing |
| ORD-T17 | outbox send ack xong crash trước SQL published; nhiều publisher/rebalance | duplicate stable event ID; consumer chỉ một effect; source version guards |
| ORD-T18 | Packing sealed/Shipping delivered đến trước prerequisite | durable DEFERRED; apply sau prerequisite một lần; không bỏ mất checkpoint |
| ORD-T19 | apply inbox effect rồi SQL rollback; Kafka replay | chưa APPLIED nếu rollback; replay thực hiện đúng một lần |
| ORD-T20 | claim same owner/new owner/revoked event/không verifier | no owner overwrite; snapshot giữ nguyên; thiếu verifier không claimed |
| ORD-T21 | hai refunds đồng thời vượt receipt balance; provider ack mất | budget không âm; UNKNOWN giữ nghĩa vụ; query trước submit lại |
| ORD-T22 | partial/full refund, hàng chưa kiểm định, refund fail | financial status đúng; không restock; không tự PROCESSING |
| ORD-T23 | completion job vs active Case tại Care, event lag | barrier hoặc disable completion; không loyalty event sớm/trùng |
| ORD-T24 | query list/queue theo filters, cursor tampering, scopes | pagination ổn định; overdue đúng policy; không hiện task chưa eligible |
| ORD-T25 | marketplace/POS replay khác event ID cùng external sale ID | một order; terminal/store permission; không trừ kho hai lần |
| ORD-T26 | DB/Kafka/Redis/remote unavailable, shutdown midflight, PITR restore | durable nghĩa vụ không mất; backlog/recovery và readiness đúng |

## 3. Failure recovery matrix

| Failure point | Durable state | Recovery | Không được làm |
| --- | --- | --- | --- |
| Validation query timeout | DRAFT/saga STARTED, chưa giữ tài nguyên | retry read hoặc terminal fail có reason | tự đoán giá/phí |
| Voucher/stock RPC timeout | step intent + UNKNOWN | query/retry same key, resolve remote terminal state | coi timeout là chưa giữ gì |
| Stock reserve sau local hủy | cancellation tombstone remote | late reserve rejection; reconcile | reserve lại cùng order cancelled |
| T3 DB commit result unknown | T0/T1 hoặc T3 thật sự committed | đọc dedup/order authoritative rồi quyết định | release chỉ vì SQL client mất ack |
| Receipt callback mất HTTP ack | receipt unique có thể đã committed | provider replay so payload hash | ghi nhận tiền lần hai |
| Inventory expiry thắng finalize | receipt confirmed, stock EXPIRED | cancel/hold, reconciliation/refund approval | phát ready hoặc hồi sinh order |
| Voucher consume unknown | PAID + operational hold | query promotion, retry same key | cho Packing chỉ bằng paid |
| Kafka down | outbox pending, order DB committed | retry publisher, alert oldest age | rollback order đã xác nhận |
| Publisher crash sau broker ack | outbox chưa published | republish same ID | tạo event ID mới |
| Poison/incoming event sớm | inbox REJECTED/DEFERRED + durable record | DLQ/pending replay có audit | mark APPLIED rồi bỏ dữ liệu |
| Release thất bại hết auto retry | cancelled + saga MANUAL_REVIEW | Ops query và resume same intent/key | xóa obligation hoặc tăng tồn thủ công |
| Refund unknown provider result | refund UNKNOWN, amount giữ ngân sách | query statement/provider, human escalation | gửi refund mới với key mới |
| Profile unavailable / guest verifier thiếu | checkout/claim chưa được xác minh | fail closed/503; dữ liệu cũ không thay owner | dùng customer query parameter hoặc fake OTP |
| Restore DB cũ hơn remote | local records thiếu remote outcome mới | full remote reconcile/outbox dedup trước resume | tự retry tất cả RPC không query |

## 4. Traceability đúng mã FR/NFR hiện tại

| Yêu cầu | Thiết kế chịu trách nhiệm | Tests |
| --- | --- | --- |
| FR-05; US-CHK-01..05; EPIC 05/06 | cart revision, quote, canonical snapshot, durable acquisition | T03..10 |
| FR-06; US-PAY-01..03; EPIC 07 | receipts, COD, exact-match, reconciliation/refund ledger | T11..15, T21..22 |
| FR-07; US-ORD-01/02/04/05/06; EPIC 08 | history/detail/cancel/queue/SLA/state guards | T01..02, T13, T16, T24 |
| FR-08/09; EPIC 09 | Inventory ownership, FEFO contract, finalize vs TTL | T07..09, T13..14 |
| FR-10/11/12; EPIC 10/11 | ready barrier, package/shipment projections | T15..18 |
| FR-13/14; EPIC 12/13 | channel/store/external ID uniqueness, POS auth | T25 |
| FR-15; EPIC 14/16 | Case/refund approval, partial financial outcome | T21..23 |
| FR-17/18; EPIC 17 | voucher hold/consume, completion fact, loyalty ngoài Order | T06..07, T15, T23 |
| FR-19/28; EPIC 18/26 | B2B/gifting extension, private snapshot | contract/policy test trước release riêng |
| NFR-01 Performance | measured SLO theo endpoint/tải, keyset/index | load plan, không kết luận từ mock |
| NFR-02 Mobile First | bounded list, allowed actions, pending response | web/mobile E2E khi client có |
| NFR-03 Availability | durable saga, retry và restore/reconcile | T08..10, T17, T26 |
| NFR-04/05 Security/Authorization | trusted principal, scope, service ACL | T01..02, T12, T20, T24..25 |
| NFR-06 Data Integrity | atomic reservation, dedup, local transactions | T03..23 |
| NFR-07 Audit Integrity | history + audit intent, downstream tamper-evident ledger | audit integration riêng + T19 |
| NFR-08 Privacy | encrypted snapshot, no PII fan-out/logs | payload/log scan và auth tests |
| NFR-09 Media Security | private packing evidence qua Fulfillment authorization | URL scope/expiry tests |
| NFR-10 Scalability | index, bounded queues, publisher/worker leases | realistic load/scale/rebalance |
| NFR-11 Observability | trace, counters, backlog, health external dependencies | fault + alert drill |

FR-07 mới là Order Management; NFR-02 là Mobile First, NFR-07 là Audit Integrity, NFR-08 là Privacy. Không giữ mapping sai của v2.0. Local history không tự chứng minh tamper-evident audit chain; phải kiểm MS-18 end-to-end.

## 5. Môi trường và bằng chứng

Dùng PostgreSQL 16/Redis/Kafka thật trong môi trường isolated; provider sandbox và Inventory/Promotion adapter có fault injection/query terminal ledger. Contract consumer tests gọi server thực được registered từ protobuf; stub chỉ kiểm branch logic. Để race test có ý nghĩa, dùng barrier, DB clock và remote terminal assertions thay vì sleep cố định.

Mỗi run ghi commit/schema version, config deadline/TTL, seed, line count/data size, concurrency, faults, latency distribution, SQL ledger counts và remote outcome; scrub PII/secrets. Những test chưa chạy phải đánh `NOT_RUN`, không chuyển PASS vì tài liệu đã mô tả. Xem [kế hoạch nghiệm thu](../../../04_testing/order-service/01_acceptance_and_verification_plan.md).

Go-live gates: G01..G09 và các gaps theo feature trong chương 07 đã đóng; tests MVP pass; no unresolved money/stock integrity issue; restore và reconciliation drill có bằng chứng; provider contract/Guest auth/COD/return policy được phê duyệt. Test tài liệu của đợt này chỉ kiểm link/fence/XML, consistency asset và DDL candidate trong database tạm.
