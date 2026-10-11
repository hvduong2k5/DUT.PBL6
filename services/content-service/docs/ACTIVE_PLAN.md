# Current implementation

Goal: a runnable PostgreSQL-backed service with the requested public/admin APIs.
Prior evidence: Profile uses cmd/server, domain, usecase, repository, Chi transport,
pgxpool, UUIDs, env configuration, Testify and multi-stage Docker. Inventory wraps
mutations in transactions and locks rows. Catalog checks DRAFT/REJECTED -> PENDING
-> APPROVED/REJECTED and uses compare-and-swap versions. Avoid forwarded-header
auth, silent transaction fallback, and returning stale objects after transitions.

Contracts: create starts version 1; all mutations require positive expected version.
PATCH uses optional fields and creates a new DRAFT revision; pending review is
immutable. Approval/rejection requires ADMIN or CONTENT_APPROVER; editorial actions
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
- [ ] Docker image build and Compose runtime smoke: daemon unavailable, NOT RUN.

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
Next: validate Docker image/Compose when a daemon is available; separately obtain
the real Auth/Gateway verification contract before enabling deployed admin access.
Current detailed handoff and evidence: CURRENT_STATE.md.
