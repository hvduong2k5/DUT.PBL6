# MS-04 Order Service (Go, sandbox 3.1)

Target Core foundations đã được thêm tại [internal/domain/core](internal/domain/core/) theo [Roadmap PR1](../../docs/02_architecture/services/order-service/18_order_implementation_roadmap.md). Package thuần nghiệp vụ chưa thay sandbox routes/persistence. [Implementation ADRs](../../docs/02_architecture/adr/order-service/README.md) ghi các gap/quyết định khi coding; [verification](../../docs/04_testing/order-service/04_order_core_foundations.md) tách domain tests khỏi real-stack integration chưa chạy.

Async checkout operations, durable PostgreSQL sagas/idempotency/financial ledger, Redis Cart/quote, Kafka inbox/outbox; stateful dependencies and provider ledger in a separate PostgreSQL database. Public HTTP and registered Order/Inventory gRPC contracts are exercised by release gates.

## Start

From repository root, with Docker Desktop running:

```powershell
docker compose -f services/order-service/docker-compose.test.yml up -d --build
```

API: `http://127.0.0.1:18004`; readiness `/readyz`; metrics `/metrics`. Simulator is `http://127.0.0.1:18104`; its controls require the synthetic internal token in `.env.example`. Order gRPC port 19004; Inventory gRPC port 19104. Compose uses dedicated databases/volumes and no fixed `container_name` that could replace Profile.

Only sandbox adapters are implemented. `APP_MODE=production` is refused. Existing Profile HTTP can be selected with `PROFILE_URL` and its own `PROFILE_INTERNAL_TOKEN`; the default stack uses the Profile simulator. No production provider/Identity/FEFO/OTP/loyalty integration is claimed.

Order 3.2 now has a [proposed wire contract](../../packages/contracts/order-3.2.md), [target OpenAPI](../../docs/03_api_specs/order-service.openapi.yaml) and event schemas v3. Regenerated Go includes the added fields/stubs; application mappings and real provider capabilities remain pending. Sandbox regression uses the frozen [3.1 OpenAPI](../../docs/03_api_specs/order-service.sandbox-3.1.openapi.yaml) and unchanged v2 events. Missing 3.2 proof must never be synthesized or silently downgraded by an adapter.

Static design-contract checks (no sandbox stack or external API calls):

```powershell
python -m pip install -r services/order-service/scripts/requirements-verification.txt
python packages/contracts/verify_order_design.py --compile-proto
```

## Run release gates

Target persistence foundation tests chỉ cần real PostgreSQL16 qua `ORDER_CORE_TEST_DATABASE`: `go test -race ./internal/persistence ./tests/persistence`. Test tự tạo/drop schema UUID, không dùng dữ liệu schema runtime. [Migration004](migrations/000004_core_foundation.up.sql) là expand có cohort fences; [PR2 evidence](../../docs/04_testing/order-service/05_persistence_foundation.md) phân biệt target SQL boundary với legacy full-stack regression. Target public routes/worker/publisher chưa được switch.

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

**SePay implementation24/10/2026:** opt-in TEST adapter, canonical API ledger, finalization, Guest grants và manual controls được mô tả trong [SEPAY_TEST.md](SEPAY_TEST.md). Khi bật SePay, new legacy VIETQR checkout được chặn tới khi target executor/Inventory3.2 đạt gate. Các bước bên dưới vẫn mô tả default legacy sandbox; chúng không chứng minh SePay end-to-end đã nghiệm thu. [ADR009](../../docs/02_architecture/adr/order-service/adr_009_sepay_payment_runtime_and_capability_gates.md) ghi các quyết định và dependency còn thiếu.

1. Obtain a sandbox token through authenticated simulator `/test/token`, or a Guest session through Order `/api/v1/guest-sessions`.
2. Mutate owned Cart with revision; server issues encrypted quote via `/api/v1/checkout/quote`.
3. POST checkout with key returns 202 and `/api/v1/checkout-operations/{id}`. Same key/body replays; altered payload conflicts. Terminal response expiry returns 410 without key reuse.
4. Worker canonicalizes and reserves, then atomically creates Order. Query operation for the real Order ID.
5. Provider simulator can record money with callback disabled; statement reconciliation discovers it. Paid and ready facts wait for terminal prerequisites.
6. Order self-cancel, staff queue/hold, trusted source events, approved cancellation/refund and recovery use guarded commands. No arbitrary state PATCH.

Synthetic provider callbacks use timestamp-bound HMAC, a receiver reference and immutable receipt IDs. These are simulator contracts. Payment/refund HTTP timeout leaves UNKNOWN; provider query and stable operation keys resolve it.

## Data and operations

Forward migrations run under a database lock and version table. Destructive down migration is refused. Commercial snapshots and audit records have immutable triggers; application workflows never truncate ledgers or delete idempotency registry keys. PII input/snapshots use AES-GCM with an explicit 32-byte key. Backup/key rotation and production credentials require a separate rollout.

[Implementation details](../../docs/02_architecture/services/order-service/08_implementation_and_acceptance.md) · [Sandbox REST](../../docs/03_api_specs/order-service.sandbox-3.1.openapi.yaml) · [Dependency contracts](../../docs/03_api_specs/order-sandbox-dependencies.openapi.yaml) · [Verification](../../docs/04_testing/order-service/03_implementation_verification.md).
