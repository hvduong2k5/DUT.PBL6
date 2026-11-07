# MS-06 Care Service

Care Service owns registered-customer support tickets and the Care-side workflows for return/refund and verified review. The currently implemented vertical slices contain the platform foundation, internal Media Module, customer Ticket MVP, Agent Queue operations, and SLA monitoring.

Architecture and delivery references:

- [`ms06_care_service_design.md`](../../docs/02_architecture/services/ms06_care_service_design.md)
- [`ms06_care_service_decision_log.md`](../../docs/02_architecture/services/ms06_care_service_decision_log.md)
- [`ms06_care_service_delivery_plan.md`](../../docs/02_architecture/services/ms06_care_service_delivery_plan.md)
- [`ms06_care_service_threat_model.md`](../../docs/02_architecture/services/ms06_care_service_threat_model.md)
- [`care-service.openapi.yaml`](../../docs/03_api_specs/care-service.openapi.yaml)
- [`care_service_runbook.md`](../../docs/05_deployment/care_service_runbook.md)
- [`deploy/kubernetes/README.md`](deploy/kubernetes/README.md)
- [`deploy/observability/README.md`](deploy/observability/README.md)

## Implemented scope

- PostgreSQL 16 private schema for platform, media, and ticket data.
- Transactional idempotency and Kafka outbox.
- S3-compatible direct upload sessions using immutable `file_id` references.
- Upload completion verification, scanner port, and short-lived access URLs.
- Registered-customer ticket create/list/detail/message APIs.
- Permission-protected staff ticket queue/detail, self-claim, internal-note, in-app public-reply, and state-transition APIs with filters, cursor pagination, optimistic concurrency, idempotency, separated private data, SLA updates, histories, metadata-only outbox where appropriate, and audit.
- Duplicate-safe near-breach and breached SLA workers with transactional metadata-only outbox events.
- Immutable upload-session expiry and retry-safe deletion of never-attached orphan objects.
- Prometheus media lifecycle, scan-status, and cleanup-outcome metrics.
- Return persistence/domain foundation with separate case, immutable decision scope,
  returned-goods, and refund-obligation lifecycles; no Return API or external action
  is enabled before the Order/refund contracts are ratified.
- Attachment ownership, purpose, status, and scan checks in the same transaction as ticket writes.
- Health, readiness, Prometheus metrics, structured logs, and optional OTLP traces.
- Per-actor token-bucket rate limiting as a service-side safeguard, with bounded
  in-memory state, `429`/`Retry-After` responses, and rejection metrics. Gateway
  rate limiting remains mandatory across replicas and before trusted authentication.

Not yet implemented:

- assignment/reassignment to other staff or queues and external email/push delivery;
- Order ownership query for linking an order to a ticket;
- Return/refund application workflows and APIs (only schema/domain foundations exist);
- Review workflows;
- production malware-scanner adapter;
- Kafka consumers and inbox processing for external events.

These gaps remain explicit in the delivery plan and are not represented by permissive stubs. In particular, a ticket request containing `order_id` fails closed until Order ownership verification is connected.

## Run locally

Prerequisites: Docker with Compose.

```powershell
$env:CARE_INTERNAL_TOKEN = "replace-with-a-development-secret"
docker compose -f services/care-service/docker-compose.yml up --build
```

Endpoints:

- API: `http://localhost:8006`
- Local S3 API (SeaweedFS): `http://localhost:9010`
- SeaweedFS topology UI: `http://localhost:9011`
- PostgreSQL: `localhost:5438`
- Kafka: `localhost:9092`

Compose waits for Kafka readiness and runs a one-shot initializer for `care.events.v1`
before Care Service starts, so transactional outbox events do not depend on broker
auto-topic creation.

Health checks:

```powershell
Invoke-RestMethod http://localhost:8006/health/live
Invoke-RestMethod http://localhost:8006/health/ready
```

## Trusted gateway headers

The service is not designed for direct public exposure. API Gateway authenticates the public bearer token, then forwards:

- `X-Internal-Token`: server-side shared trust token;
- `X-User-ID`: authenticated immutable user ID;
- `X-Permissions`: comma-separated permissions; each staff operation requires the permission listed below;
- `X-Request-ID`: optional per-request identifier;
- `X-Correlation-ID`: optional identifier shared by all calls/events in one business flow.

Missing or unsafe tracking IDs are replaced with UUIDs and returned in response
headers. Correlation ID is stored with audit/outbox records and emitted as the
CloudEvents `correlationid` extension; request ID remains specific to one HTTP call.

Clients must not be allowed to inject or overwrite these headers at the public edge.

Authenticated API calls are limited per actor and per Care Service replica by
`RATE_LIMIT_PER_SECOND` (default `20`) with `RATE_LIMIT_BURST` (default `40`). This
is defense in depth, not a replacement for a distributed Gateway limit. Health and
Prometheus endpoints are intentionally excluded. Rejected calls return
`RATE_LIMITED`, HTTP `429`, and `Retry-After`.

Current staff read endpoints:

- `GET /api/v1/admin/support/tickets`
- `GET /api/v1/admin/support/tickets/{ticket_id}`
- `POST /api/v1/admin/support/tickets/{ticket_id}/claim` (`care.tickets.claim`)
- `POST /api/v1/admin/support/tickets/{ticket_id}/internal-notes` (`care.tickets.notes.write`)
- `POST /api/v1/admin/support/tickets/{ticket_id}/messages` (`care.tickets.reply`)
- `POST /api/v1/admin/support/tickets/{ticket_id}/transitions` (`care.tickets.transition`)

## Media flow

1. Call `POST /api/v1/media/upload-sessions` with `Idempotency-Key`.
2. Upload bytes directly to the returned URL using the exact required headers.
3. Call `POST /api/v1/media/upload-sessions/{file_id}/complete`.
4. Use the ready `file_id` in a ticket request.

`SCANNER_MODE=noop` is accepted only outside production. Production configuration fails validation until a real scanner mode is implemented.

The service uses `S3_ENDPOINT` for server-side storage operations and `S3_PUBLIC_ENDPOINT` when signing URLs returned to clients. In local Compose these are `care-object-storage:9000` and `localhost:9010`, respectively; production must set the public endpoint to the externally reachable object-storage host.

## Database migrations

The local Compose database mounts migrations into PostgreSQL initialization. The image
also contains `/app/care-migrate` for a dedicated deployment Job. It embeds ordered
up migrations, serializes concurrent runners with a PostgreSQL advisory lock, and
records immutable checksums in `care_schema_migrations`. Application startup never
mutates schema, and the runner never executes destructive down migrations.

## Development commands

When Go is installed:

```powershell
Set-Location services/care-service
go mod tidy
go test ./...
go test -race ./...
go build ./cmd/server
go build ./cmd/migrate
```

PostgreSQL integration tests use a random, isolated schema and clean it up after
the run. With the local Compose database running:

```powershell
$env:CARE_TEST_DATABASE_URL = "postgres://care_user:care_password@localhost:5438/care_db?sslmode=disable"
go test -count=1 -run '^TestIntegration' ./migrations ./internal/repository/postgres
Remove-Item Env:CARE_TEST_DATABASE_URL
```

The integration suite applies both migrations, verifies the transactional ticket,
idempotency, SLA, and outbox writes, then applies the rollback migrations. Normal
`go test ./...` runs skip these database tests unless `CARE_TEST_DATABASE_URL` is set.

