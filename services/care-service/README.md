# MS-06 Care Service

Care Service owns registered-customer support tickets and the Care-side workflows for return/refund and verified review. The currently implemented vertical slice contains the platform foundation, internal Media Module, and customer Ticket MVP.

Architecture and delivery references:

- [`ms06_care_service_design.md`](../../docs/02_architecture/services/ms06_care_service_design.md)
- [`ms06_care_service_decision_log.md`](../../docs/02_architecture/services/ms06_care_service_decision_log.md)
- [`ms06_care_service_delivery_plan.md`](../../docs/02_architecture/services/ms06_care_service_delivery_plan.md)
- [`care-service.openapi.yaml`](../../docs/03_api_specs/care-service.openapi.yaml)

## Implemented scope

- PostgreSQL 16 private schema for platform, media, and ticket data.
- Transactional idempotency and Kafka outbox.
- MinIO/S3 direct upload sessions using immutable `file_id` references.
- Upload completion verification, scanner port, and short-lived access URLs.
- Registered-customer ticket create/list/detail/message APIs.
- Attachment ownership, purpose, status, and scan checks in the same transaction as ticket writes.
- Health, readiness, Prometheus metrics, structured logs, and optional OTLP traces.

Not yet implemented:

- agent queue, assignment, internal-note, transition, and SLA workers;
- Order ownership query for linking an order to a ticket;
- Return/refund workflows;
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
- MinIO API: `http://localhost:9010`
- MinIO Console: `http://localhost:9011`
- PostgreSQL: `localhost:5438`
- Kafka: `localhost:9092`

Health checks:

```powershell
Invoke-RestMethod http://localhost:8006/health/live
Invoke-RestMethod http://localhost:8006/health/ready
```

## Trusted gateway headers

The service is not designed for direct public exposure. API Gateway authenticates the public bearer token, then forwards:

- `X-Internal-Token`: server-side shared trust token;
- `X-User-ID`: authenticated immutable user ID;
- `X-Permissions`: comma-separated permissions for future staff routes;
- `X-Request-ID`: optional correlation ID.

Clients must not be allowed to inject or overwrite these headers at the public edge.

## Media flow

1. Call `POST /api/v1/media/upload-sessions` with `Idempotency-Key`.
2. Upload bytes directly to the returned URL using the exact required headers.
3. Call `POST /api/v1/media/upload-sessions/{file_id}/complete`.
4. Use the ready `file_id` in a ticket request.

`SCANNER_MODE=noop` is accepted only outside production. Production configuration fails validation until a real scanner mode is implemented.

## Database migrations

The local Compose database mounts the initial migration into PostgreSQL initialization. Production must run migrations as a dedicated deployment job before rolling out the service; application startup does not mutate schema.

## Development commands

When Go is installed:

```powershell
Set-Location services/care-service
go mod tidy
go test ./...
go test -race ./...
go build ./cmd/server
```

