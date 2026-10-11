# Content Service handoff

Task/scope: implement standalone MS-10 editorial CMS inside this directory only.
Branch: feat/content-service. Base/current HEAD before commit:
2291ef5a44dd3a6f7985297903bfec01b325b087 (origin/dev).
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
Docker compose config --quiet PASSED. Docker runtime unavailable: desktop-linux
daemon pipe not found. Docker build/run NOT RUN. go vet ./..., go build ./... and
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

Blockers: external Auth/Gateway verification contract not supplied; Docker daemon
not available. No core PostgreSQL/API implementation blocker remains.
Next: when Docker is available, run docker compose up --build -d and confirm
readiness/public list/admin 401. For deployed admin access, obtain Auth/Gateway
issuer/audience/verification and role-mapping contracts, implement a real Verifier,
then add signature/expiry/audience and role tests. Do not bypass fail-closed auth.
The core implementation is complete; these external verification/integration tasks
remain OPEN. No push or PR.
