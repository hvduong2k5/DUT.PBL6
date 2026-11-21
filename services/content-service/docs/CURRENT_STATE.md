# Content Service handoff

Current task (2026-11-21): address the PR #52 review by hvduong2k5 (code items only).

Changed, all inside services/content-service:
- Admin boundary: RS256 staff-token verifier (`middleware/jwt.go`) wired in
  `cmd/server`, fail-closed when `CONTENT_JWT_ISSUER`/`CONTENT_JWT_PUBLIC_KEY` are
  unset. Authority comes only from signed `permission_grants` (GLOBAL scope);
  role names and ADMIN carry no authority. Non-staff valid tokens are 403.
  Per-action permissions `CONTENT_ARTICLE_{VIEW,CREATE,EDIT,SUBMIT,APPROVE,REJECT,PUBLISH,HIDE}`.
- Flat error envelope `{code,message,requestId,errors?}` everywhere; request ID in
  context; every invalid transition is 409 `INVALID_STATE`.
- `Idempotency-Key` (optional, honored when present) for admin POSTs, stored in
  `idempotency_keys` (migration 000002).
- `restore` action (hide -> publish the same approved snapshot) and an append-only
  `audit_events` trail for create/edit/submit/approve/reject/publish/hide/restore.
- Repository only inserts new revisions and moves status with a compare-and-set;
  PUBLISHED status is immutable (trigger); unique violations map to 409.
- Public reads: `Cache-Control: public, max-age=60, s-maxage=300` + ETag/304;
  admin and errors `no-store`.
- Config for timeouts/pool/body size; panic recovery logs the stack.
- Migration runner (`cmd/migrate`, embedded SQL, `schema_migrations`, advisory lock),
  container entrypoint that migrates before start, `Dockerfile.coolify`.
- README/AGENTS.md updated; AGENTS.md no longer forbids contract updates.

Review items NOT done here (need decisions or are outside this directory):
contract and state-matrix updates in docs/03_api_specs, architecture/requirements
doc sync, Kafka event catalog, `.github/workflows` (CI and cd-deploy),
`docs/06_operations/cd-coolify.md`, IAM permission seed for `CONTENT_*`, and the
MVP features of review section 3 (Product SEO, media, scheduling, preview,
redirect/sitemap, archive, full SEO render, metrics). The review's claim that the
list query selects `body` is not accurate for this branch: only the `q` filter reads it.

VERIFICATION STATUS (Go 1.22.12 portable, Windows): go mod tidy (no further change),
go vet ./..., go build ./..., gofmt (ignoring CRLF) and go test -count=1 ./... PASS;
go test -race ./internal/... PASS. Unit and HTTP tests include real RS256 token
verification, error envelope, caching, idempotency and the hide/restore/audit flow.
NOT VERIFIED: every test in ./tests/integration is SKIPPED because no PostgreSQL was
available (Docker/WSL cannot start on this machine). Migration 000002, the migrate
runner, the repository write path (append-only revisions, status compare-and-set,
audit rows), the PostgreSQL idempotency store and the Docker images have never run.
Before merging run: CONTENT_TEST_DATABASE_URL=... go test -count=1 ./tests/integration
against a disposable database, and `docker build` both Dockerfiles.

Previous task (2026-10-12): publication restricted to approvers/admins by role
(superseded by per-permission grants above).

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
