# 10 — Roadmap implement và Definition of Done

> **Checkpoint30/10/2026:** Code đã có theo các slice nền tảng; xem [trạng thái từng slice và bằng chứng](../../../04_testing/fulfillment-service/03_implementation_verification.md). Candidate plan vẫn giữ để review scope; không đọc bảng roadmap thành mọi slice/milestone đã được nghiệm thu. Branch hiện tại: `feat/fulfillment-service`.

[Index](README.md) · [Acceptance](../../../04_testing/fulfillment-service/01_acceptance_and_verification_plan.md). Thứ tự là đề xuất theo dependencies, không gán ngày hoàn tất/nhân lực chưa biết.

## 1. Milestones và khả năng demo

| Milestone | Kết quả có thể chứng minh | Chưa tuyên bố |
| --- | --- | --- |
| A — Domain + local DB | Guard/admission races, task/checklist/picks/seal atomicity bằng fixtures verified | Order/Inventory thật, shipping hoặc video safety production |
| B — Peer-backed admission | Ready→real permit/snapshot/COMMITTED allocation→one task | Seal→shipment khi G04/G09 chưa đạt |
| C — Packing→Shipping TEST | Actual verified video, real peer services, carrier fake có nhãn TEST, Order PACKED + one shipment | Production carrier/money, COD/returns/reopen |
| D — Operational release | Auth/storage/restore/backlog/monitoring drills, gated cohort rollout | Khả năng phase2 chưa được test |

Milestone A có test fakes nhưng response profile mang nhãn test, không deploy cổng customer. Không bật adapter fallback khi peer down. TEST SePay/Shipping khác financial production acceptance của Order.

## 2. Vertical slices

| Slice / PR đề xuất | Nội dung commit | Test focus | Dependency/gate |
| --- | --- | --- | --- |
| S0 — Contract review | ADRs, requirement conflicts, peer candidates, named owners | Static verifier, compatibility baseline | G02–13 review |
| S1 — Core/DB skeleton | Domain state, errors, invariants, schema/migrations, query health | Real PG constraints, FSM, digest, rollback/commit UNKNOWN | No mutation cohort |
| S2 — Admission barrier | Registry, permit port, Accept/Cancel/Query, guard/high-water | FUL-T01–08,28–30,37 | Peer ports typed; integration stays false |
| S3 — Staff workflow | Queue, assignment/start/picks/checklist/blocks, authorization | FUL-T09–15,26–28,36,41 | Snapshot/allocation fixtures then real provider |
| S4 — Media | Scoped upload, actual checksum/scan, immutable promotion, private access/audit | FUL-T18–25,36 | G09 storage/scanner |
| S5 — Seal/outbox/revisions | Package/measurements, atomic seal, GetTaskRevision/package proof, exporter | FUL-T15–17,29–34,38 | G04 contract, no fake proof |
| S6 — Peer adapters | Real Order permit/snapshot/source adapter, Inventory allocation, Shipping v2 | Full Order→pack→ship TEST, dedup/races | G03–05/G07–09/G12 accepted |
| S7 — Operations | Ready consumer/jobs, reconcile, SLA, dashboards, restore/runbook | FUL-T31–42, destructive fault tests isolated only | Restore and operator sign-off |
| S8 — Extensions | Rework/multi-package/etc per approved need | New race/proof suites | G06/new contracts; not part of MVP default |

Ba PR đầu nên nhỏ và reviewable: Core/DDL/query; Admission/registry; Staff workflow. Có thể implement media module song song theo team plan nhưng không coi partial source là milestone C. Candidate SQL phải chuyển thành service-owned migrations khi thực sự implement, forward-only expand trước destructive contract.

## 3. DoD áp dụng mọi slice

- Scope/invariant/test IDs, migration và cohort impact ghi trong PR; note contract/runtime/acceptance level.
- No cross-service DB access, no business logic trong generated code, no mutation exposed chỉ bằng role string client.
- Unit tests cho transitions/canonicalization; PG integration cho locks/uniqueness/commit ambiguity. Không chỉ test mock happy path.
- API schema/negative examples, proto compile/wire baseline, actual event payload schema/runtime consumer validation.
- Race suite relevant; formatter/vet/build service; context cancellation and bounded resources.
- Secrets/PII absent docs fixtures/logs; metrics and audit actionable; runbook có recovery identity.
- Fault handling preserves original keys/proofs/obligations; no broad retry with new key.

## 4. Capability flags đề xuất

| Flag | Default | Khi bật |
| --- | --- | --- |
| FULFILLMENT_ADMISSION_ENABLED | false | Real permit/snapshot/allocation + barrier acceptance |
| FULFILLMENT_SEAL_ENABLED | false | Staff/media/pick/source revision prerequisites |
| FULFILLMENT_PACKING_V2_ENABLED | false | Consumer first + verified video policy + peer E2E |
| FULFILLMENT_MEDIA_UPLOAD_ENABLED | false | Immutable storage/checksum/scanner acceptance |
| FULFILLMENT_MEDIA_ACCESS_ENABLED | false | ABAC/access audit/private bucket acceptance |
| FULFILLMENT_CUSTOMER_MEDIA_ENABLED | false | Customer ownership authority accepted |
| FULFILLMENT_REOPEN_ENABLED | false | Future coordinated stop/rework protocol accepted |
| FULFILLMENT_MULTI_PACKAGE_ENABLED | false | Provider/consumer per-package coverage semantics |

Config values đều là proposal names, chưa có loader/runtime. Không dùng toàn bộ flag off để ngừng read/reconciliation cho task cũ. Separate admission switch từ recovery enable; có pinned cohort trên task, không đổi v2→v1 giữa luồng. CLI/admin commands bật gates phải ghi config revision/actor/reason và rollback plan.

## 5. Rollout/rollback

Expand DB → deploy query/recovery and peers → consumers ready → isolated cohort one Order → negative/fault tests → enable new admissions nhỏ. Pin contract and policy per task. Dashboard quan sát duplicate rejection, ready/outbox lag, blocked age, peer snapshot failures và media safety.

Rollback stop admission mới và user mutation mới có risk; giữ provider queries/outbox/reconcile của accepted tasks. Không route existing v2 package vào legacy v1 hoặc paid consumer. Không rollback migration bằng DROP tables có obligations. Partial deploy không được expose SubmitPackingEvidence legacy bypass guards.

## 6. Bàn giao cho người implement

Điểm bắt đầu: read README/master → review G02–05/G07–09 → S1. Sau mỗi slice cập nhật evidence trong docs/04_testing/fulfillment-service và status ở registry local; chỉ gửi đề xuất sửa shared MVP registry khi provider/client PR đã có bằng chứng. Bộ tài liệu này chưa tạo branch/PR/migration runtime hoặc triển khai service.
