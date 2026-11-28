# Central test infrastructure

Order and Fulfillment runtime/test topologies are maintained here rather than
inside service directories. Service owners keep exactly `Dockerfile` (local,
prebuilt Linux artifacts) and `Dockerfile.coolify` (independent source build).
The shared codegen/race toolchain is `../go-tools.Dockerfile`; its pinned protobuf
versions are unchanged. These fixtures do not configure production credentials.

Run from the repository root:

```powershell
./infra/scripts/build-go-artifacts.ps1 -Service order-service
docker compose -f infra/docker/stacks/order-test.compose.yml up -d --build
./services/order-service/scripts/test.ps1

./infra/scripts/build-go-artifacts.ps1 -Service fulfillment-service
docker compose -f infra/docker/stacks/fulfillment-local.compose.yml up -d --build
./services/fulfillment-service/scripts/test.ps1
```

`order-connected.compose.yml` and `order-identity.compose.yml` overlay the central
Order test topology. Connected payment/tunnel credentials remain private and
are only used when explicitly requested; ordinary tests use synthetic payment
statements. The Fulfillment test recipe uses a separate project and disposable
PostgreSQL/Kafka fixtures. Stop the same project without deleting data volumes.

Build the shared tool image with `--build-arg SERVICE_PATH=services/order-service`
or `services/fulfillment-service`, using repository root as build context.
Local artifact builds accept `-Architecture arm64` when required by the host.
