# MS-04 Order Service (Go, sandbox 3.1)

Async checkout operations, durable PostgreSQL sagas/idempotency/financial ledger, Redis Cart/quote, Kafka inbox/outbox; stateful dependencies and provider ledger in a separate PostgreSQL database. Public HTTP and registered Order/Inventory gRPC contracts are exercised by release gates.

## Start

From repository root, with Docker Desktop running:

```powershell
docker compose -f services/order-service/docker-compose.test.yml up -d --build
```

API: `http://127.0.0.1:18004`; readiness `/readyz`; metrics `/metrics`. Simulator is `http://127.0.0.1:18104`; its controls require the synthetic internal token in `.env.example`. Order gRPC port 19004; Inventory gRPC port 19104. Compose uses dedicated databases/volumes and no fixed `container_name` that could replace Profile.

Only sandbox adapters are implemented. `APP_MODE=production` is refused. Existing Profile HTTP can be selected with `PROFILE_URL` and its own `PROFILE_INTERNAL_TOKEN`; the default stack uses the Profile simulator. No production provider/Identity/FEFO/OTP/loyalty integration is claimed.

## Run release gates

```powershell
./services/order-service/scripts/test.ps1 -StartStack
```

This builds a pinned Go/protoc toolchain image and runs formatting/vet/race tests against isolated real infrastructure. Postman collections stay local and are ignored by Git; if the Order collection is available locally, the script also runs it without cloud event upload (Postman CLI required). Otherwise Postman is reported as NOT_RUN. Reports go to `docs/04_testing/order-service`.

Go-only gates:

```powershell
docker build -f services/order-service/Dockerfile.tools -t omamx/order-go-tools:local services/order-service
docker compose -f services/order-service/docker-compose.test.yml run --rm tests
```

Go harnesses create/drop their own synthetic PostgreSQL schemas and use Redis DB1, not the running sandbox API's Cart DB0. Domain-only builds outside this environment show integration NOT_RUN; the release command sets required infrastructure variables.

Validate local OpenAPI, event payloads, current documentation links and protobuf drift:

```powershell
python -m pip install -r services/order-service/scripts/requirements-verification.txt
python services/order-service/scripts/verify-contracts.py
```

The contract checker can create and cancel a synthetic placement fixture when a fresh checkout has no sandbox Orders; it does not depend on the local Postman collection. Use `--fresh-fixture` to exercise this path explicitly.

Regenerate protobuf without touching Profile generated files:

```powershell
docker run --rm --mount 'type=bind,source=E:\DUT.K1N4\PBL,target=/workspace' omamx/order-go-tools:local sh scripts/generate-proto.sh
```

## API flow

1. Obtain a sandbox token through authenticated simulator `/test/token`, or a Guest session through Order `/api/v1/guest-sessions`.
2. Mutate owned Cart with revision; server issues encrypted quote via `/api/v1/checkout/quote`.
3. POST checkout with key returns 202 and `/api/v1/checkout-operations/{id}`. Same key/body replays; altered payload conflicts. Terminal response expiry returns 410 without key reuse.
4. Worker canonicalizes and reserves, then atomically creates Order. Query operation for the real Order ID.
5. Provider simulator can record money with callback disabled; statement reconciliation discovers it. Paid and ready facts wait for terminal prerequisites.
6. Order self-cancel, staff queue/hold, trusted source events, approved cancellation/refund and recovery use guarded commands. No arbitrary state PATCH.

Synthetic provider callbacks use timestamp-bound HMAC, a receiver reference and immutable receipt IDs. These are simulator contracts. Payment/refund HTTP timeout leaves UNKNOWN; provider query and stable operation keys resolve it.

## Data and operations

Forward migrations run under a database lock and version table. Destructive down migration is refused. Commercial snapshots and audit records have immutable triggers; application workflows never truncate ledgers or delete idempotency registry keys. PII input/snapshots use AES-GCM with an explicit 32-byte key. Backup/key rotation and production credentials require a separate rollout.

[Implementation details](../../docs/02_architecture/services/order-service/08_implementation_and_acceptance.md) · [REST](../../docs/03_api_specs/order-service.openapi.yaml) · [Dependency contracts](../../docs/03_api_specs/order-sandbox-dependencies.openapi.yaml) · [Verification](../../docs/04_testing/order-service/03_implementation_verification.md).
