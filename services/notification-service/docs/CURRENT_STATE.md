# Notification Service — current handoff

Updated: 2026-11-06 (Asia/Saigon). Classification LARGE/ARCHITECTURAL, **planning-only task**.

## Git / authority

- Active branch: `codex/notification-service`.
- Base / HEAD before planning checkpoint: `a24492b63c364f93ad19d880798137e554898c3f` (local `origin/dev`). Local planning checkpoint follows this base; obtain its exact SHA with `git log -1 --format=%H -- services/notification-service` rather than embedding a self-referential commit hash here.
- Original branch was `feature/mobile-ui` at `edb858d7a2ef0cc894ef6f872128d44c9917b828`; no changes to its tracked files. New branch was created with `git switch --no-track -c codex/notification-service origin/dev`.
- User explicitly forbids **merge/push**. No fetch, merge, push, PR, remote deployment, provider send or credential operation. No remote checkpoint is claimed.
- Existing unrelated untracked `docs/MOBILE_MVP_DISCUSSION_REFERENCE.md` must remain untouched. Entry SHA256 `4E7E0337E6E53A55D9C9B8516DCFE793C27A8C503E29D0E85E9F8212D70CDFC7`.

## Completed work

1. Read applicable instructions and Mobile handoff only to preserve branch/task boundaries, then moved planning to new dev-based branch.
2. Narrow audit of EPIC 28, MS-17 boundaries, event docs/schema/examples, Order/Shipping producer code, Profile contract/auth, Identity targeted source, relevant service branch references. Exact commits/evidence in SOURCE_AUDIT.
3. Designed Email-first MVP with full EPIC traceability; Go layered architecture, private PostgreSQL + Kafka + recoverable Redis hint, provider ports, versioned templates/policies, operations/auth/privacy.
4. Defined candidate API/event/recipient contracts, schema/dedup/lease/UNKNOWN/callback/retention semantics, integration gates and failure test matrix.
5. Created stable master N0–N7, detailed current/next N0/N1 plan, and copy-paste implementation prompt.

## Intentional modified area

Only `services/notification-service/`: AGENTS, README, docs/PROJECT_SPEC, ARCHITECTURE, SOURCE_AUDIT, CONTRACTS, DATA_AND_DELIVERY, DECISIONS, VALIDATION, CURRENT_STATE, IMPLEMENTATION_PROMPT, docs/plans/MASTER_PLAN and docs/plans/active/N0_N1_CONTRACT_FOUNDATION (all Markdown). Existing src/.gitkeep preserved. No root planning files/shared contract registry changed.

## Validation

- Source evidence reviewed at pinned local revisions; remote freshness NOT VERIFIED (no fetch).
- Documentation validation PASS: 13 Markdown files; all relative document links resolve; `git diff --cached --check` clean; staged-path allowlist contains only `services/notification-service/**`. Git LF/CRLF conversion warnings reflect repository settings, not whitespace-check failures.
- Unrelated Mobile reference SHA256 rechecked unchanged. No tracked edits outside Notification; final status after the local checkpoint should retain only that pre-existing untracked file.
- Go tests/analyze/build, DB/Kafka/Redis/Compose, provider and native UI tests: **NOT RUN — no implementation in this task**.
- Known runtime failures: none observed/tested for Notification. Availability of Docker/toolchain/backend has not been probed; old Mobile environment observations are not reused as current Notification evidence.

## Decisions / blockers

- All new docs/source stay service-local; no mixing Mobile or sibling service progress trackers.
- Existing catalog is inconsistent with producer versions; pin adapters, do not assume v1 everywhere.
- Order/Shipping events lack complete recipient Email/audience. Recipient contract and private permissions must be agreed; no direct peer DB access.
- OTP owner is Identity/IdP; Invoice producer/artifact contract not implemented on base. These remain real integration gates.
- Provider, auth permissions, exact toolchain/dependency versions, retention, consent and production SLO need scoped decisions G01–G12. Proposed baseline is not deployed/ratified merely because documented.

## Exact next action

User reviews the plan or gives another model [IMPLEMENTATION_PROMPT](IMPLEMENTATION_PROMPT.md). With implementation authorization, perform N0.1–N0.4 in [active plan](plans/active/N0_N1_CONTRACT_FOUNDATION.md), resolve only gates needed for the chosen slice, then N1 foundation. Without that authorization, do not start Go source or real-provider work. Do not continue the former Mobile roadmap from this branch.
