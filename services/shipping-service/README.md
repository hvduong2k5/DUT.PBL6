# MS-12 Shipping Service

Shipping Service owns Shipment, waybill and tracking data. The MVP foundation includes PostgreSQL persistence, a transactional outbox, Kafka publication, HTTP and gRPC APIs, a `PackingCompletedEvent` consumer and a deterministic fake carrier. Real GHN/Viettel Post adapters remain integration steps that require provider credentials.

## Run locally

```powershell
Set-Location services/shipping-service
$env:SHIPPING_INTERNAL_TOKEN = "shipping-local-development-secret"
$env:SHIPPING_FAKE_WEBHOOK_SECRET = "fake-carrier-webhook-secret"
docker compose up --build
```

Health endpoints:

- `GET /health/live`
- `GET /health/ready`

The gRPC API listens on `localhost:9012` and requires `authorization: Bearer <SHIPPING_INTERNAL_TOKEN>` metadata.

Shipping v1 remains the default provider contract. Set `SHIPPING_V2_ENABLED=true` to register
the `omamx.shipping.v2.ShippingService` API on the same gRPC listener while retaining v1.
The v2 API persists bound quotes with their canonical request hash, immutable expiry and
monotonic quote version. The capability remains rollout-gated until packing v2, historical
shipment revisions and v2 events are implemented.

Authenticated endpoints accept either `Authorization: Bearer <token>` or `X-Internal-Token`.

## Implemented endpoints

- `POST /api/v1/internal/shipping/quotes`
- `POST /api/v1/internal/shipments`
- `POST /api/v1/internal/shipments/{shipmentID}/handover`
- `GET /api/v1/admin/shipping/shipments`
- `GET /api/v1/admin/shipping/shipments/{shipmentID}`
- `GET /api/v1/orders/{orderID}/tracking`
- `POST /api/v1/webhooks/logistics/fake`

The fake webhook requires `X-Shipping-Signature: sha256=<hex hmac>` calculated from the exact request body and `SHIPPING_FAKE_WEBHOOK_SECRET`.

## PackingCompleted integration

Shipping consumes `vn.omama.fulfillment.packing.completed.v1` from `fulfillment.events.v1`. The event contains package measurements but no customer PII. Shipping calls Order Service `GetShippingSnapshot` for the immutable checkout recipient, COD amount, shipping fee and selected carrier/service, then creates one shipment per `package_id`.

The consumer is disabled when `SHIPPING_ORDER_GRPC_ENDPOINT` is empty. Enable it when Order Service exposes the contract:

```powershell
$env:SHIPPING_ORDER_GRPC_ENDPOINT = "localhost:9004"
$env:SHIPPING_ORDER_INTERNAL_TOKEN = "order-local-development-secret"
```

The legacy `fulfillment.events.v1` consumer remains active. To opt a rollout cohort into the
proof-bearing packing contract, set `SHIPPING_PACKING_V2_ENABLED=true`. Shipping then also
consumes `vn.omama.fulfillment.packing.completed.v2` from `fulfillment.events.v2`, verifies the
Fulfillment task proof and requires its `ready_generation` to match Order's immutable shipping
snapshot. Do not publish the same package to both topics.

## Local verification

```powershell
go mod download
go test ./...
go vet ./...
go build ./cmd/server
```
