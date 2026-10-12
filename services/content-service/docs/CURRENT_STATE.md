# Content Service handoff

Current task (2026-10-12): fix publish authorization only. Completed domain/usecase
checks requiring CONTENT_APPROVER or ADMIN; the existing APPROVED-only state guard
is preserved. Publish returns 403 for editors, 401 for missing/invalid verified
identity, and 409 for invalid publication state. Other transition errors retain
their existing contract. Auth middleware, repository, migrations and transactions
are unchanged. No other review findings are included.

Intentionally changed: domain/usecase authorization, HTTP publish error mapping,
their tests, PostgreSQL denial regression, README and current plan/handoff.
Verification: go fmt ./..., go test ./..., go test -race ./..., go vet ./...,
go build ./... all PASS in WSL Go 1.22.12. Tests include 35 HTTP role/state cases,
direct domain and usecase denial, unchanged PostgreSQL data on denied publication,
and existing migration/history/optimistic-locking/transaction tests. Real integration
used isolated disposable schemas in the local Docker DB on port 15410; none skipped.
No failing tests. Docker API image was not rebuilt for this fix.
Starting HEAD: 4ceffc2 (Docker recovery documentation). Local fix commit follows
this handoff; use git log -1 for its hash. Next: inspect final diff and commit
locally only. No push/merge/PR. The untracked user archive
services/content-service.rar is intentionally untouched and excluded from staging.

Earlier scope: standalone MS-10 editorial CMS; Docker daemon recovery and
container verification completed on 2026-10-11, with docs-only changes at that time.
Branch: feat/content-service. Base:
2291ef5a44dd3a6f7985297903bfec01b325b087 (origin/dev).
Implementation checkpoint:
3337d3064c47677a5c494e6600932b972ed93d90, committed locally.
HEAD before the Docker verification documentation commit:
a981726bbba69576da2345962183f82a498d01e8. Use git log -1 for final HEAD.
Before resuming, verify HEAD and git status; preserve the untracked user archive.
No push or PR is authorized; no other existing branch/service was edited.

Completed: configuration/pool/lifecycle; SQL up/down; append-only content and
moderation decisions; versioned transactional CRUD/editorial actions; isolated
public snapshot reads; public/admin DTO REST endpoints; fail-closed injectable
auth; Docker/Compose; unit, HTTP and real PostgreSQL integration tests; README.
Intentionally modified area: services/content-service only.

Validation at implementation checkpoint: Go 1.22.12 in WSL;
go mod tidy, go fmt ./..., go test ./... PASSED with real PostgreSQL 16.15 on
loopback port 55410. Domain/config/usecase/HTTP packages passed, and four integration
tests passed (migration up/down/up, constraints, transactions, races, network HTTP).
Docker compose config --quiet PASSED. Initial Docker unavailability was resolved
in the follow-up below; Docker build/run now PASSED. go vet ./..., go build ./... and
go test -race -count=1 ./... PASSED, including real PostgreSQL tests. Vet/build/race
were rerun successfully after final list-query changes. Executable smoke on port
18010: /livez, /readyz, /healthz and empty public list returned 200; admin with a
forged X-User-Role returned 401; SIGINT logged graceful shutdown and exited 0.

Decisions: Profile-compatible dependency versions; test transitive
github.com/rogpeppe/go-internal pinned to v1.12.0 to retain Go 1.22 compatibility.
No automatic toolchain upgrades required. Editorial/public pointers separate;
SEO/media JSONB per immutable revision; permanent slug reservations; historical
URLs return 404 when a new slug is published. Admin production returns 401 until
real verified identity integration; this is an explicit security requirement.

List queries use one joined data query and one count query; list DTOs omit bodies,
detail DTOs include them. First compile exposed a Zerolog pointer-method call;
fixed before passing verification. No failing tests remain.

## Docker verification follow-up

Root cause: backend startup failed on stale AF_UNIX socket reparse points,
first Docker/run/dockerInference, then docker-secrets-engine/engine.sock, with
Windows error 1920 (file cannot be accessed by the system). With the failed backend
stopped, runtime directories containing only sockets were renamed to backups.
Both stale endpoints had to be absent together before restarting successfully.
No factory reset, image/volume deletion, WSL unregister or credential reset was used.
Preserved backups on this workstation:
- %LOCALAPPDATA%/Docker/run.backup-content-service-20261011
- %LOCALAPPDATA%/Docker/run.backup-content-service-20261011-retry
- %LOCALAPPDATA%/docker-secrets-engine.backup-content-service-20261011

Verified Docker Server 29.0.1 linux/amd64, desktop-linux context. Executed:
- docker compose --progress plain up --build -d --wait --wait-timeout 90: PASS.
- HTTP /livez, /readyz, /healthz, public list: 200; missing article: 404;
  admin request with forged identity headers: 401.
- Four public-schema tables present; API user app (uid 100), memory limit 128 MiB;
  DB memory limit 256 MiB. Observed idle use approximately 2.8/37.7 MiB respectively.
- Entire go test -p 2 -count=1 ./... suite: PASS in disposable golang:1.22.12-alpine
  container, source mounted read-only, GOTOOLCHAIN=local, GOMAXPROCS=2,
  CONTENT_TEST_DATABASE_URL targeting db:5432/content_db on pbl6-content_default.
  Includes migration up/down/up, constraints, transactions, races and HTTP integration
  against PostgreSQL 16.15 in the DB container. Test container removed after success.
- Stop only Compose DB: liveness 200; readiness/health 503; public API sanitized 500.
  Start DB: readiness returns 200. Restart API: graceful shutdown logged; healthy again.

Runtime left running for inspection: pbl6-content-api-1 and pbl6-content-db-1,
both healthy. HTTP http://localhost:8010; DB loopback port 15410;
volume pbl6-content_content_data. To stop this project while retaining its data,
run docker compose stop from this service directory.

Remaining blocker: external Auth/Gateway verification contract not supplied.
No Docker or core PostgreSQL/API implementation blocker remains.
Next: for deployed admin access, obtain Auth/Gateway
issuer/audience/verification and role-mapping contracts, implement a real Verifier,
then add signature/expiry/audience and role tests. Do not bypass fail-closed auth.
The core implementation and container verification are complete; external identity
integration remains OPEN. No push or PR.
