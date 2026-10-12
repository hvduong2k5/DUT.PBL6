# Current implementation

Goal: a runnable PostgreSQL-backed service with the requested public/admin APIs.

Current follow-up (2026-10-12): restrict publication to CONTENT_APPROVER/ADMIN in
domain and usecase; preserve APPROVED-only publication and return HTTP 409 for
invalid publication state, 403 for editors, 401 for missing/invalid identity.
Scope excludes other review findings, repository/transaction changes and auth adapters.
Acceptance: role/state matrix, direct business-layer denial, unchanged persisted
article on denied publication; go test, test -race, vet and build all pass.
Status: implemented and verified. All four commands passed in WSL Go 1.22.12,
including real Docker PostgreSQL tests; 35 HTTP role/state cases passed. Only the
publish authorization gates and publish-specific HTTP state mapping changed.
Next: local commit only; no push, PR, auth adapter or other review findings.

Prior evidence: Profile uses cmd/server, domain, usecase, repository, Chi transport,
pgxpool, UUIDs, env configuration, Testify and multi-stage Docker. Inventory wraps
mutations in transactions and locks rows. Catalog checks DRAFT/REJECTED -> PENDING
-> APPROVED/REJECTED and uses compare-and-swap versions. Avoid forwarded-header
auth, silent transaction fallback, and returning stale objects after transitions.

Contracts: create starts version 1; all mutations require positive expected version.
PATCH uses optional fields and creates a new DRAFT revision; pending review is
immutable. Approval/rejection/publication requires ADMIN or CONTENT_APPROVER; editorial actions
require ADMIN, CONTENT_EDITOR or CONTENT_APPROVER. Rejection requires a reason.
Public reads use only the visible published pointer. Hiding preserves snapshots.
Every content snapshot and decision records actor/time. All writes are atomic.

Steps / acceptance:
- [x] Configuration/startup/pool/health/shutdown with bounded timeouts.
- [x] SQL up/down, relational ownership constraints, unique slug reservations.
- [x] Domain transitions and transactional repository with optimistic locking.
- [x] DTO REST handlers; pagination, stable sorting, query/category/status filters.
- [x] Injectable verifier, fail-closed default, tests cannot bypass authorization.
- [x] Unit + HTTP + opt-in real PostgreSQL integration, migration up/down/up.
- [x] fmt/test/vet/build/race and executable smoke; Compose configuration validation.
- [x] Final scope review: only services/content-service, no push/PR/other branch edits.
- [x] Docker image build and Compose runtime smoke: passed after daemon recovery.

Expected areas: cmd/server, internal/{config,domain,usecase,repository,
infrastructure/postgres,transport/http}, migrations, tests, Docker/readme/env.
Risks: slug races, stale version races, draft leaking through public joins,
publishing unapproved content, DB errors escaping HTTP, unbounded bodies/queries.
Validation: go fmt ./...; go test ./...; go vet ./...; go build ./...;
CONTENT_TEST_DATABASE_URL enables isolated-schema migration/HTTP/concurrency tests.
No external infrastructure or data migration is in scope.

Completed: all core APIs and required editorial behavior implemented and tested.
Additional decision: list queries return summaries without bodies in one joined
data query plus a count query, keeping memory and DB roundtrips bounded.
Next: obtain the real Auth/Gateway verification contract before enabling deployed
admin access. Docker image/Compose verification is complete.
Current detailed handoff and evidence: CURRENT_STATE.md.

## Docker verification follow-up

User authorized daemon repair and container verification. Scope: recover local
Docker Desktop without resetting existing images/volumes, build this service's
image, run its Compose project and verify health, public/admin HTTP and PostgreSQL.
Observed blocker: stale Windows AF_UNIX socket reparse points in Docker/run and
docker-secrets-engine cause backend startup to fail with Windows error 1920.
Runtime directories are renamed to backups; persistent Docker data is untouched.
- [x] Recover the desktop-linux daemon and confirm its server API (29.0.1).
- [x] Build image and start dedicated pbl6-content Compose project.
- [x] Verify HTTP, migration, non-root runtime and container health/resource limits.
- [x] Run database integration checks against container PostgreSQL; record results.

Results: API and DB healthy; all Go tests passed inside Go 1.22.12 Alpine against
PostgreSQL 16.15. Database outage yields readiness/health 503, liveness 200 and
sanitized public error 500; restarting DB restores readiness 200. API restart
logs graceful shutdown. Containers remain running on localhost:8010/15410.
