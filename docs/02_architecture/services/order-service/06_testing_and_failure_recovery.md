# 06 — Verification và recovery, baseline 3.2

[Trước](05_api_contracts_and_transports.md) · [Tiếp](07_decisions_and_contract_gaps.md)

## 1. Mức bằng chứng

[Report3.1](../../../04_testing/order-service/03_implementation_verification.md) ghi local sandbox results ngày09/10; không đổi thành evidence3.2. Lượt thiết kế này kiểm consistency/link/diagram syntax structure, không claim runtime tests mới. Unit/mock kiểm local logic; sandbox kiểm orchestration với simulator; integration gate bắt buộc provider thật cùng wire/version/config.

## 2. Acceptance matrix

| ID | Scenario | Oracle bắt buộc | Scope |
| --- | --- | --- | --- |
| T01 | fake identity/owner/address/internal headers | denied, không PII/effect | core |
| T02 | Guest session và claim proof/replay | session scoped; claim fail nếu thiếu verifier | claim gated |
| T03 | concurrent same key/body; Redis loss | một operation/Order/stock effect | core |
| T04 | same key changed body hoặc khác principal | conflict hoặc scope độc lập | core |
| T05 | giá/metadata/address/fee đổi | coherent snapshot, không silently reprice | core |
| T06 | money overflow/free shipping/zero amount | invariant và policy reject đúng | promotion gated |
| T07 | voucher hold rồi reserve fail | release terminal đúng một effect | voucher gated |
| T08 | reserve commit lost ACK/crash | same resource, không duplicate hold | core real Inventory |
| T09 | cancel trước reserve chưa có row | tombstone serialize chặn late reserve | core real Inventory |
| T10 | SQL placement commit lost ACK | query result, không compensate Order đã thành công | core |
| T11 | exact/under/over/late/unmatched/multiple | mọi verified receipt giữ, only exact auto allocate | payment |
| T12 | duplicated/altered transaction | same no-op, altered conflict | payment |
| T13 | payment/timeout/finalize/expiry trước-bằng-sau hạn | một legal stock outcome, no premature ready | core real Inventory |
| T14 | finalize ACK lost/stale worker | query COMMITTED, epoch guard, no release | core |
| T15 | voucher unknown; COD unpaid; duplicate ready | no ready trước prerequisites, COD không paid giả | per feature |
| T16 | accept vs cancel, permit/event trễ, hard hold | remote barrier một winner, cancelled không task mới | real Fulfillment |
| T17 | outbox crash/rebalance; message N lỗi rồi N+1 | stable duplicate, không mất N | real Kafka |
| T18 | checkpoint out-of-order/stale | durable DEFERRED, source query, không state lùi | Shipping/Fulfillment |
| T19 | inbox transaction rollback/replay | effect và APPLIED atomic | core |
| T20 | claim owner conflict/revoke | no overwrite, explicit policy | claim gated |
| T21 | concurrent refund/unknown ACK | budget không âm, key stable/query | real provider |
| T22 | partial/full refund và stock return | financial truth, không auto restock/restart | payment/returns |
| T23 | case-open vs completion/callback lag | Care serialized decision, post-complete policy | real Care |
| T24 | queue/cursor/hold scope + generation resume | đúng scope/SLA, old permit không reopen | core |
| T25 | POS/marketplace business ID replay | unique channel/store/sale và payment evidence | extension gated |
| T26 | DB/Kafka/remote outage, shutdown, PITR | obligations tồn tại, reconcile before resume | deployment |
| T27 | success payload state rỗng/UNKNOWN/RESERVED/sai ID/digest | compensation/finalize/cancel không close sai | all adapters |
| T28 | COMMITTED release; repeated reverse/batch expired | no fake release, one reversal, no sale of invalid batch | real Inventory |
| T29 | production missing broker/auth/capability | fail startup/readiness, không log-and-publish giả | deployment |
| T30 | cohort v1/v2 legacy và v3 target rollout/rollback | một authority/order, no double deduction/task | migration |
| T31 | COD canonical quote hết hạn trước placement | revalidate approved policy hoặc compensate/review, không stale terms | COD |
| T32 | response retention hết, obligation còn UNKNOWN | không key reuse hoặc xóa obligation/tombstone | retention |

## 3. Failure recovery

| Điểm lỗi | Durable state | Recovery |
| --- | --- | --- |
| Validation unavailable | operation accepted, chưa acquire | bounded retry; policy reject rõ |
| Reserve/hold lost ACK | step intent/key | authoritative query/replay, không key mới |
| DB commit unclear | registry + operation/Order có thể đã commit | read authoritative trước compensation |
| Remote expired sau receipt | receipt/allocation trace | hold/reconciliation/refund review, no ready |
| Voucher failure sau stock committed | stock proof + job | no PAID, barrier/reversal/refund review |
| Cancellation release unknown | cancelled + compensation obligation | query/tombstone proof; UI pending |
| Barrier ACK lost | cancel intent/key/generation | query Fulfillment, không chỉ nhìn Order state |
| Reverse success ACK lost | reversal obligation/key | query same reversal, không cộng lần hai |
| Refund unknown | budget reserved | provider query/statement, không new refund key |
| Kafka unavailable | outbox chưa published | retry/alert oldest age |
| Malformed event | durable quarantine/DLQ | audited replay; no unsafe offset advance |
| Auto retries exhausted | MANUAL_REVIEW + hold/reference | Ops query/resume same key, không xóa nghĩa vụ |
| DB restore cũ hơn provider | checkpoint/history | reconcile ledger trước enable workers |

Ops cần dashboard theo operation/order/key/remote decision. Manual resolution phải có permission/reason/evidence/audit; không generic status patch hoặc SQL sửa available để “khớp số”.

## 4. Performance và release gates

Acceptance latency: đầu request→durable202. Placement completion: acceptance→operation terminal. Payment finalization: verified receipt→PAID hoặc definitive reconciliation. Target đề xuất: acceptance p95<=200ms, placement p95<=3s, receipt→finalization p95<=5s khi dependencies healthy. Workload kiểm chứng ban đầu: 20 checkout/s trong10 phút sau warm-up2 phút, 5 SKU/order, dataset100.000 Orders và10.000 SKU; tài nguyên máy/network/cấu hình phải ghi vào report. Đây là workload mục tiêu cho lần nghiệm thu, chưa chạy/chưa chứng minh đạt. Chạy riêng hot-SKU/contention/100-SKU và fault tests; không loại business failures khỏi báo cáo error rate. NFR cũ <200ms toàn critical path cần được chủ yêu cầu xác nhận lại phạm vi đo: baseline tách acceptance khỏi completion, không dùng202 nhanh để tuyên bố toàn checkout đạt200ms. Không cộng các P95 riêng thành P95 toàn luồng hoặc dùng simulator latency làm production promise.

Evidence mỗi run: SHAs cả hai service, schema/proto version, dataset/concurrency/line count, TTL/deadline, fault point, stock/receipt/refund ledger trước-sau và raw result đã scrub. Race dùng deterministic barrier/clock boundary, không dựa sleep ngẫu nhiên.

Integration release: T01/03/04/05/08/09/10/11/12/13/14/17/19/27/28 pass với Inventory/provider thật; packing cần T16/18, COD T15/31, refund T21/22, completion T23; migration T30; deployment readiness/restore T26/29. Feature không đủ dependency giữ disabled. Không yêu cầu hoàn tất mọi feature mở rộng để nghiệm thu slice core.

## 5. Traceability

FR05 checkout → T03–10; FR06 payment → T11–15/21–22; FR07 Order lifecycle → T01/13/16/24; FR08–09 Inventory/FEFO → T08–09/13–14/28; FR10–12 fulfillment/shipping → T15–18; FR13–14 channels → T25; FR15 returns → T21–23; FR17–18 promotion → T06–07/15/23; FR19/28 B2B/gifting → extension tests riêng.

NFR performance/availability cần workload/restore evidence; integrity cần ledger assertions; auth/privacy cần negative access và payload/log inspection; audit cần independent anchor policy khi yêu cầu chống privileged deletion; observability cần alert drill. Requirement labels tham chiếu bộ yêu cầu hiện tại, không suy từ số lượng tests thành coverage hoàn chỉnh.
