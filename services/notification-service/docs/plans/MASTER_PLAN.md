# Notification Service — master plan

Updated 2026-11-06. **LARGE/ARCHITECTURAL; current task planning-only.** Branch `codex/notification-service`, pinned base dev `a24492b63c364f93ad19d880798137e554898c3f`. Scope boundary và authority ở [AGENTS](../../AGENTS.md). No merge/push/PR. Stable spec/architecture ở docs cấp trên; execution details chỉ active plan/current state.

## Roadmap và dependencies

| Phase | Mục tiêu / deliverables | Dependencies / gates | Definition of Done | State |
|---|---|---|---|---|
| N0 | Audit và chốt baseline, event/recipient/auth/provider contracts, candidate OpenAPI/schema/fixtures | User giao implementation; G01–G03; gates khác được gán owner/status | Source pins, approved baseline hoặc local-only gates, contract tests định nghĩa rõ, no ambiguity trong core interfaces | **DESIGN PACKAGE COMPLETE; RATIFICATION OPEN** |
| N1 | Go foundation, config, private DB migrations, domain/ports, atomic intake/idempotency/auth seams | N0 baseline; G01 toolchain; local fake scope có thể độc lập G04–G10 | Module build/test/vet; isolated DB concurrency/rollback tests; health/ready; protected routes fail closed | NOT STARTED |
| N2 | Templates/policy, recipient snapshot, reliable delivery worker, fake provider, callbacks, suppression, audit outbox | N1; DATA_AND_DELIVERY ratified, G06 defaults local | Duplicate/UNKNOWN/crash/callback ordering/privacy gates pass; local capture email, no real send | NOT STARTED |
| N3 | Owner-integrated Email slices: Order → OTP → Invoice; operator query/replay; provider adapter | N2; per-slice G02–G10, G11 scope | Correct source/audience/private contact/artifact, authorized APIs; each slice reports fake/local/real status separately | NOT STARTED / EXTERNAL GATES OPEN |
| N4 | Full Email hardening, failure/load tests, runbook, isolated packaging, release evidence | N3 accepted scope; G06/G09/G10 | Required tests not skipped; restart/replay/key/privacy checks; full Email MVP or explicit reduced scope acceptance | NOT STARTED |
| N5 | SMS/Zalo + Shipment routing/order/coalescing/fallback | N4 and own provider/contact policy | No duplicate fallback after ambiguous send; stale transition protection; provider acceptance separate | FUTURE |
| N6 | Push Mobile/Web, endpoints, Notification Center, preferences/Marketing gates | N4; identity/client/consent contracts | Device ownership/current auth/read semantics/quiet hours; clients implemented only separately authorized | FUTURE |
| N7 | Internal Alerts, staff audiences, ack/resolution/escalation | N4; source conditions and current RBAC/resource scopes | Correct scopes/revocation, duplicate-free escalation, no source-state ownership violation | FUTURE |

N5–N7 không bắt buộc tuần tự giữa nhau; tất cả dựa trên N4 reliability core, chỉ chạy khi có scope và owners tương ứng. Không thêm kênh để né blocker Email. Không tự triển khai mọi phase chỉ vì xuất hiện trong master.

## Thứ tự slice thực dụng

N1 durable intake trước → N2 fake Email với deterministic failures → N3 Order Email với producer fixtures/resolver → chốt OTP và Invoice ownership → provider sandbox verification → N4 acceptance. Nếu OTP/Finance chưa sẵn sàng: báo partial local deliverable, giữ pending dependencies; user quyết định giảm scope chứ model không tự tuyên bố toàn EPIC 28 xong.

## Whole-service DoD

Theo PROJECT_SPEC traceability; provider-independent domain; no source rollback; durable dedup không hứa exactly-once external; UNKNOWN/replay safe; auth/privacy/consent/versioned contracts; proportional real DB/Kafka tests và mode-specific evidence; all source/docs isolated; no undocumented cross-service writes; handoff đủ tiếp tục. Phase 2 FR chỉ DONE sau N5–N7, không sau N4.

## Current phase / next step

[N0/N1 detailed plan](active/N0_N1_CONTRACT_FOUNDATION.md). Audit/design đã được viết, chưa ratify architecture hoặc implementation. Next: user review/giao triển khai, sau đó thực hiện N0 gate triage và contract fixtures, rồi N1. Xem [CURRENT_STATE](../CURRENT_STATE.md) để biết Git và validation hiện tại.
