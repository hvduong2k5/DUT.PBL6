# Traceability Service MS-03

Implementation in progress on `feat/traceability-service`. See the
[development plan](../../docs/02_architecture/services/traceability-service/02_development_plan.md).

Current checkpoint: **P7 review candidate with Docker smoke, race tests and runbook**.

Start here: [reviewer runbook](../../docs/04_testing/traceability-service/02_reviewer_runbook.md).
From this directory, run `./scripts/dev.ps1 -Action Init`, then `-Action Start`
and `-Action Smoke`. Use `-Action Start -Relay` to enable real broker delivery.
Versioned migrations and an explicit migration CLI now exist, with real PostgreSQL
tests for database guards and runtime grants. Atomic command dedup and the
create/read/edit Draft repository now have real PostgreSQL tests. Domain checks
cover certificates, scope, expiry, evidence and source snapshot validation.
Sandbox identity, explicit source fixture, application permissions, list/create/
get/edit/new revision/submit/review/publish/suspend/archive routes and anonymous
public QR are wired. Business events are persisted atomically in outbox; the
optional broker relay and expiry worker are installed. `/readyz` checks schema/mode/checksum and effective runtime
role privileges; startup fails before listening if these or the verifier/source
files are missing. `/livez` indicates liveness. Integration startup remains refused.
Signed HTTP/PostgreSQL, real isolated broker tests, Linux race tests and container
lifecycle smoke pass. The final R1 requirement/evidence audit is recorded in
`docs/04_testing/traceability-service/04_final_r1_audit.md`; R2 peer integration
is not complete. No production or full EPIC completion is claimed.

Requirements: Go 1.25 or later; PostgreSQL 16 and Docker for the subsequent real
database tests. Set variables from `.env.example` in your process environment;
the server does not implicitly load `.env` or assume a sandbox mode.

```powershell
go test ./...
go vet ./...
go build ./...
go run ./cmd/server
```

Without `TRACEABILITY_TEST_DATABASE_URL`, PostgreSQL tests explicitly skip; a green
default test command alone does not prove database behavior. The
server validates `APP_MODE`, `HTTP_ADDR`, `DATABASE_URL`, `PUBLIC_WEB_BASE_URL`
and bounded shutdown settings before opening a listener. Errors do not echo
database credentials. No test signing keys or real credentials are committed.

Compose starts only the isolated DB by default. Set `TRACEABILITY_DB_PASSWORD`
to a local secret and run `docker compose up -d db`. The optional API profile
also requires `TRACEABILITY_RUNTIME_DATABASE_URL`; do not use the migrator role
for the API. Sandbox containers require explicit isolated-container configuration
and publish only on host loopback. No destructive reset is part of startup.

The sandbox API additionally requires `SANDBOX_JWT_PUBLIC_KEY_FILE` (one Ed25519
PKIX PUBLIC KEY PEM, `kid=sandbox-local`) and `SANDBOX_FIXTURE_FILE` (explicit
synthetic source JSON; example in `fixtures/batches.example.json`). The API never
receives a private signing key. `cmd/sandbox` provides init/tokens/smoke, preserves
existing credentials on reuse, and writes private local files under ignored
`.sandbox`. Compose mounts only the public key and fixture read-only.

Sandbox tokens use EdDSA, issuer `urn:omama:traceability:sandbox`, audience
`traceability-service:sandbox`, UUID `sub`, integer `iat`/`exp`, lifetime <=15m,
and an explicit `permissions` array. Optional `nbf` must already be valid.
This issuer has no token revocation/introspection integration; it is only a test
identity, not the shared Identity Service contract. Client role/user headers are
ignored. Permissions use `traceability.*`; create/edit also require
`traceability.read_internal` because responses include profile history.
`traceability.read_evidence` independently controls evidence references on reads
and every command response, including replay. No ADMIN bypass exists.

Implemented routes: POST `/api/v1/admin/trace/records`, GET
`/api/v1/admin/trace/records/{id}`, PATCH
`/api/v1/admin/trace/records/{id}/revisions/{revision}`. Writes require JSON and
`X-Idempotency-Key`; PATCH replaces full Draft content and requires
`expected_version`. Create verifies Product/SKU/Batch against the explicit fixture;
unknown or mismatched sources do not silently fall back to another source.

Also implemented: GET the records collection with `page_size`/`cursor`, POST
`records/{id}/revisions` with `from_revision`, POST revision commands
`submit`/`approve`/`reject`/`publish`, POST profile `suspend`/`archive`, and
anonymous GET `/api/v1/trace/{qr_code}`. Every write requires expected_version
except create; approval/rejection/suspension/archive need a non-empty reason.
Withdrawing an existing working revision requires a reason and preserves all
source/withdrawn contributors. There is no restore operation.

Approve does not publish. Publish rechecks the source outside the DB transaction,
then rechecks version, certificate eligibility and source hash under the aggregate
lock. Public pointer, local audit and PENDING event commit together. New Drafts do
not alter the published snapshot. Public reads enforce expiry directly and return
404 before first publication, or a safe 410 reason after suspension/archive/expiry.
No provider is called during a scan. Public `verified_at` is the OCOP manual check;
`source_verified_at` is the frozen dependency snapshot check time, not a claim that
the external source was queried on every scan. Optional artisan/ATTP/video remain null.

Event topic `traceability.events.v1` and types remain a service-local draft pending
consumer agreement. Isolated broker delivery is verified; shared peer consumer
compatibility and Audit ingestion are not verified.

`OUTBOX_RELAY_ENABLED=false` is the sandbox default; events stay PENDING and
`traceability_relay_enabled` reports 0. Enabling it requires explicit `KAFKA_BROKERS`;
bad configuration fails startup, and broker failure never switches the relay off.
Compose uses `kafka:9092` inside the network; host-native broker clients use
`127.0.0.1:19093`. Start only this test broker with
`docker compose --profile broker up -d --wait kafka`. Provision the topic with
`docker compose exec kafka rpk topic create traceability.events.v1 --brokers 127.0.0.1:9092 -p 1 -r 1`
before enabling delivery (the writer does not auto-create topics).

Relay uses synchronous broker acknowledgement, one event per claim, a fresh
30-second lease token, bounded publish, retry from 1 second up to 1 minute, and
10 total attempts. A crashed last attempt becomes FAILED after lease expiry.
FAILED events block later events only for their own profile. Investigate metrics/
logs and reconcile consumer effects before an operator requeues a failed event;
there is no public retry endpoint or automatic bypass of failed predecessors.
Delivery is at least once: a broker ack followed by a crash before local Ack can
deliver the same event ID/payload twice. Consumers must dedup and compare versions;
version gaps are normal because not every administrative command emits an event.

`EXPIRY_WORKER_ENABLED=true` scans bounded pages every 30 seconds. Due certificate/
review_due observations, local audit and an eligibility event commit together,
with a unique cutoff key preventing duplicate observations. Root stays ACTIVE;
read-time guards keep protecting scans when the worker is late/disabled. The worker
uses a reserved local system actor, never a client-supplied identity.

`GET /metrics` exposes only numeric operational stats and fixed route templates:
request count/duration, relay delivery/error, outbox pending/failed/oldest age,
dependency result counts and expiry observations/errors. It is available on the
sandbox listener without a token; no evidence, raw path, actor or broker errors
are labels. Broker degradation is observed here without disabling public reads.

To run broker tests, set `TRACEABILITY_TEST_KAFKA_BROKERS=127.0.0.1:19093` and the
isolated PostgreSQL test DSN, then run `go test -count=1 -run TestKafka -v ./internal/infrastructure/postgres`.
These tests provision/use `traceability.events.v1` in the explicitly configured
test broker. Never point test settings at a shared peer or production broker.

Run migrations explicitly using the migrator DSN in `DATABASE_URL` and
`APP_MODE=sandbox` (or `integration` for a separate integration database):

```powershell
go run ./cmd/migrate
```

An administrator must provision a separate unprivileged LOGIN role with a local
secret, no memberships, and no schema/table ownership. Then run
`go run ./cmd/migrate -runtime-role trace_runtime` to grant the required DML.
This command does not create roles or handle passwords. The API must use this
runtime role rather than the migrator. Production mode is unsupported.

For database tests, set `TRACEABILITY_TEST_DATABASE_URL` to the isolated test
database's migrator DSN, then run `go test -count=1 ./migrations` or `go test ./...`.
The harness creates a random `trace_test_*` schema per test, uses only that schema
in search_path, and drops only its own schema afterward. The test role needs
CREATE ROLE to verify runtime privilege isolation. Never point the test DSN at a
peer or production database. Migration down is exposed only to the disposable
schema test helper; there is no destructive CLI command.

For aggregate coverage across application code exercised by integration tests,
run `go test -count=1 '-coverpkg=./...' '-coverprofile=coverage.out' ./...`, then
`go tool cover '-func=coverage.out'`. Without the test DSN this cannot verify the
database/HTTP scenarios.

See the [verification report](../../docs/04_testing/traceability-service/01_verification.md)
for actual checks and outstanding gates.
