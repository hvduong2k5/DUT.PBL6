# Customer Mobile — current handoff

Updated 2026-10-31. Current task: four-step MVP contract migration planning; execute M1 analysis only. Whole parity/migration goal ACTIVE, not complete.

## Branch / source / authority

- Work/commit/push only feature/mobile-ui; initial HEAD3f206027dc8de3b992bfc327aeb9dcb27db9a86f and worktree clean. Current documentation checkpoint identified by git log -1.
- Read-only dev target7f22226c7b2f4363d3ba376b715feb0e63f69355, fetched by explicit user permission. No checkout/merge/cherry-pick, no changes to dev/Web, no PR.
- Target is docs/03_api_specs/mvp-customer.openapi.yaml at that commit, release mvp-2026-10-26; README/MIGRATION/registry/TEAM-DECISIONS explain source priority and pending implementation acceptance. Read with git show pinned-ref:path.
- ZIP snapshots remain historical, unchanged. For overlapping routes MVP is the new migration target; this planning step does not migrate runtime adapters or activate capabilities. Features absent from target are not silently removed or declared supported.
- Existing Web visual reference b1ff2918; retain shared colors/type/components. No redesign in M1.

## Plans / progress

Master: docs/plans/MASTER_PLAN.md. Active: docs/plans/active/M1_MVP_CONTRACT_GAPS.md.

1. M1 contract gaps: **ANALYSIS COMPLETE**, G1–G3 owning-team decisions **OPEN**.
2. M2 cart/address/API models + quote review: NOT STARTED; detailed plan at entry after relevant decisions.
3. M3 async checkout-operation and recovery: NOT STARTED; detailed plan at entry.
4. M4 payment/order actions/cancellation and migration QA: NOT STARTED; detailed plan at entry.

## M1 findings / proposed defaults (not approved policy)

- G1: quote accepts cart_revision but no selected line IDs/items. Current Mobile selectedCart submits a subset under old ZIP. Recommend full-cart quote for MVP or explicit partial-checkout contract; never temporarily delete unselected lines. Team must confirm quote scope and post-placement cart semantics. Do not change checkbox behavior before this decision.
- G2: public target requires canonical province_code/ward_code; no geography lookup. Direct Profile validation/suggestion endpoints are not a public lookup. Current Mobile uses /locations/checkout plus name-only DTOs, wrapper list and no If-Match. Need canonical data source/version and saved-vs-inline policy. Saved-address-only fallback requires review; no suitable address means recoverable blocked checkout, not guessed codes.
- G3: order list returns items/next_cursor but no input cursor/limit/filter/search contract; provider spec also lacks it. Tracking response is unconstrained object. Need typed pagination/tracking spec before real load-more/timeline. Do not invent queries or infer delivery timestamps. Current Mobile pageSize/page/total parser and old tracking cannot be treated as compatible.
- Active M1 records evidence positions, four decision requests/owners, UI fallback proposals and dependency gates. No communication sent to team, no backend approval claimed. Contract integration target is not proof provider deployed.

## Modified areas / validation

This step changes only AGENTS, CURRENT_STATE, DECISIONS, MASTER_PLAN and new M1 detailed plan. No Flutter source/mock/test/API snapshot changes, no dependency installs, no browser/computer-use calls.

PASS: parsed pinned public/provider YAML and8targeted static assertions confirming quote scope/revision/geography absence/list-cursor omission/tracking placeholder/address version requirement. This is local schema inspection, not full contract CI/provider/runtime verification. git diff --check recorded before checkpoint. No Flutter test/analyze/APK rerun for documentation-only work.

Baseline source is still e86f82b (3f20602 was documentation): recorded144/144 tests, analyze PASS, APK166s PASS, installed update and native cancellation-return refresh verified. Those results are historical and do not validate MVP compatibility. Detailed baseline native evidence remains P4_CUSTOMER_QA.md and CUSTOMER_PARITY_MATRIX.md.

## Exact next action

Stop after M1 documentation checkpoint. User/team review G1–G3 in active plan. Once relevant answers exist, open detailed M2 analysis for actual files/model boundaries/UX/tests, not a broad UI rewrite. Record changed assumptions and target provenance before implementation. Until then the contract decision gate remains OPEN; M1 analysis is done but backend gaps are not solved.

Old QA-01 payment tests must be re-scoped to new payment contract in M4; QA-02 external-media handoff remains unverified/outside M1. No automatic scrcpy loops. Native runtime handles from October21 may be stale; rediscover only when a later targeted test needs them.
