# Notification Service — operating contract

## Authority and isolation

- User follow-up 2026-11-06 authorizes implementation following the plan. Execute the local foundation and later phases when their entry criteria pass; owner/provider integration gates remain explicit. No real sends or production operations are implied.
- Branch: `codex/notification-service`, created from local `origin/dev` snapshot `a24492b63c364f93ad19d880798137e554898c3f`. **NO MERGE, NO PR.** User follow-up 2026-11-06 explicitly authorizes committing the testing handoff and pushing this branch, including its existing planning/foundation commits. No other branch push is authorized. Checkpoint commits may include only intentional files in this directory.
- All task source, plans, contracts, tests and evidence belong under `services/notification-service/`. Do not edit root `docs/`, `packages/`, `infra/`, `.github/`, apps or sibling services without separate authorization. Read reference branches using `git show <pinned-sha>:<path>`.
- Do not copy Mobile `CURRENT_STATE`, master/phase plans or AGENTS instructions into this service. Root documents are reference sources, not this service's progress tracker.
- Preserve unrelated untracked `docs/MOBILE_MVP_DISCUSSION_REFERENCE.md`. Do not stage it, move it or delete it.
- No real sends, provider accounts, credentials, deployments, shared migrations, new cloud resources, fetch or dependency upgrades are implied by this plan.

## Resume order

1. Applicable ancestor instructions and this file.
2. `docs/CURRENT_STATE.md`, `docs/plans/MASTER_PLAN.md`, current active plan.
3. `docs/DECISIONS.md`, relevant rows in `docs/SOURCE_AUDIT.md` and `docs/CONTRACTS.md`.
4. `git branch --show-current`, `git rev-parse HEAD`, `git status --short`; then narrowly inspect relevant code.

## Implementation rules for the later authorized task

- Keep domain/usecase/ports separate from Chi, pgx, Kafka, Redis and provider SDKs. Do not import sibling service `internal` packages.
- Source events decide business triggers; Notification applies approved dispatch policies. Never update Order/Payment/Shipment states or validate OTPs.
- No external I/O inside SQL transactions. Durable inbox/job before Kafka acknowledgment. Stable provider idempotency key, fenced leases and explicit UNKNOWN outcome.
- Do not claim exactly-once external delivery. Do not automatically retry ambiguous sends without a safe provider mechanism.
- Auth fail-closed; authorize usecases before repository/provider access. Test identities/providers cannot activate in production.
- Never log contacts, OTPs, device tokens, body, signed URLs or auth headers. Real provider acceptance/delivery needs separate evidence.
- Update the active plan and local handoff at milestones. Future phases need entry review; do not mark all EPIC 28 complete after an Email demo.
- Planned validation from this module: `go test ./...`, `go vet ./...`, `go build ./...`; `go test -race -count=1 ./...` on supported toolchain. Real Postgres/Kafka tests must explicitly report skips and use isolated resources. These commands do not run until a Go module exists.
