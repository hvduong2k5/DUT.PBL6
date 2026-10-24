# Order-service — implementation ADRs

Quyết định và gap gặp khi coding được ghi tại đây, bên cạnh các ADR thiết kế OD ở [chương07](../../services/order-service/07_decisions_and_contract_gaps.md). Mỗi entry ghi bối cảnh, lựa chọn, hệ quả, evidence và phần còn mở. Accepted không đồng nghĩa provider/production capability đã bật.

| ADR | Trạng thái / phạm vi |
| --- | --- |
| [001 — Pure Core và legacy cohort](adr_001_pure_core_and_legacy_cohort.md) | Accepted, implemented foundations trong PR1 scope |
| [002 — Digest và canonical snapshot profiles](adr_002_digest_and_canonical_snapshot_profiles.md) | Accepted, sửa gap design/fixture trước integration |
| [003 — Decision clock, proof boundary và milestone authority](adr_003_decision_clock_proofs_and_milestones.md) | Accepted, domain guards implemented; persistence/provider gates còn mở |
| [004 — Expand migration và cohort fencing](adr_004_expand_migration_and_cohort_fencing.md) | Accepted, migration004/legacy fences; target traffic chưa active |
| [005 — SQL retry và commit UNKNOWN](adr_005_sql_retry_and_commit_unknown_recovery.md) | Accepted, real PostgreSQL retry/identity recovery tests |
| [006 — Encrypted hydration và atomic PLACE](adr_006_encrypted_hydration_and_atomic_placement.md) | Accepted, Core repository/T3/AEAD và immutable outbox persisted |
| [007 — SePay sandbox capture trên Cloudflare](adr_007_sepay_sandbox_capture_on_cloudflare.md) | Accepted, URL test/HMAC/encrypted receipts; chưa nối Payment ledger |
| [008 — Verification contract theo cohort](adr_008_verification_contract_bound_to_cohort.md) | Accepted, sửa runtime checker dùng đúng sandbox schema; isolated project/output |
| [009 — SePay Payment runtime và capability gates](adr_009_sepay_payment_runtime_and_capability_gates.md) | Implemented TEST slice: webhook/API/ledger/finalize/read/manual controls; target checkout, provider proof, publisher và owning authority acceptance còn mở |

Evidence cho lượt này: [Core foundations verification](../../../04_testing/order-service/04_order_core_foundations.md). Các PR sau phải bổ sung ADR cho quyết định mới hoặc entry superseding nếu thay quyết định đã pin; không sửa lịch sử để gọi gap đã được giải quyết mà chưa có evidence.

PR2 evidence: [Persistence foundation verification](../../../04_testing/order-service/05_persistence_foundation.md), có real PostgreSQL + isolated Redis/Kafka regression, phân biệt target persistence với legacy messaging/provider simulator acceptance.
