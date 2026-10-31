# MS-12 Shipping Service

Shipping Service owns Shipment, waybill and tracking data. The MVP foundation includes PostgreSQL persistence, a transactional outbox, Kafka publication, HTTP and gRPC APIs, a `PackingCompletedEvent` consumer and a deterministic fake carrier. A feature-gated GHN staging adapter supports quote preview, order creation and authenticated status callbacks without exposing GHN-specific payloads to Order Service.

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
monotonic quote version. It also serves exact, proof-bearing shipment revisions created after
the revision migration; older rows are deliberately not backfilled from current state. The
event cohort is pinned on each Shipment: legacy callers publish only `shipping.events.v1`, while
gRPC v2 and Packing v2 shipments publish proof-bearing events to `shipping.events.v2`. Runtime
rollout remains gated until owning-service and consumer acceptance is complete.

Authenticated endpoints accept either `Authorization: Bearer <token>` or `X-Internal-Token`.

## Implemented endpoints

- `POST /api/v1/internal/shipping/quotes`
- `POST /api/v1/internal/shipments`
- `POST /api/v1/internal/shipments/{shipmentID}/handover`
- `GET /api/v1/admin/shipping/shipments`
- `GET /api/v1/admin/shipping/shipments/{shipmentID}`
- `GET /api/v1/orders/{orderID}/tracking`
- `POST /api/v1/webhooks/logistics/fake`
- `POST /api/v1/webhooks/logistics/ghn` (only active when GHN is configured)

The fake webhook requires `X-Shipping-Signature: sha256=<hex hmac>` calculated from the exact request body and `SHIPPING_FAKE_WEBHOOK_SECRET`.

## GHN staging adapter

GHN remains disabled by default. Configure sandbox credentials locally or through deployment
secrets, never in source control:

```powershell
$env:SHIPPING_GHN_ENABLED = "true"
$env:SHIPPING_GHN_BASE_URL = "https://dev-online-gateway.ghn.vn"
$env:SHIPPING_GHN_TOKEN = "<staging-token>"
$env:SHIPPING_GHN_SHOP_ID = "<shop-id>"
$env:SHIPPING_GHN_FROM_DISTRICT_ID = "<pickup-district-id>"
$env:SHIPPING_GHN_FROM_WARD_CODE = "<pickup-ward-code>"
$env:SHIPPING_GHN_WEBHOOK_SECRET = "<random-secret>"
```

Quote requests use GHN's non-mutating order preview endpoint so the internal address contract
can remain provider-neutral. Shipment creation uses the immutable internal `shipment_id` as
GHN `client_order_code`. The returned print label is stored as the stable reference
`GHN:<tracking_code>`; an expiring print URL must be generated only when requested.
If creation times out or returns a retryable provider error, Shipping performs one bounded
lookup through GHN `detail-by-client-code`. A matching order is recovered instead of creating
another carrier order; a definitive provider `4xx` is never retried or reconciled. Because the
detail response omits `total_fee`, recovery refreshes it with a non-mutating preview and falls
back to the checkout quote if that preview is temporarily unavailable.

Configure the GHN Developer Portal only after the public endpoint is reachable. Add the custom
header `X-GHN-Webhook-Secret` with the same value as `SHIPPING_GHN_WEBHOOK_SECRET`. A missing
local shipment returns `503` so GHN retries a callback that raced the shipment transaction;
invalid secrets and malformed provider payloads are rejected with non-retryable `4xx` responses.

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
