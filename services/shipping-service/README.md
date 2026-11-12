# MS-12 Shipping Service

Shipping Service owns Shipment, waybill and tracking data. The MVP foundation includes PostgreSQL persistence, a transactional outbox, Kafka publication, HTTP and gRPC APIs, and a deterministic fake carrier. A feature-gated GHN staging adapter supports quote preview, order creation and authenticated status callbacks without exposing GHN-specific payloads to Order Service.

## Run locally

```powershell
Set-Location services/shipping-service
$env:SHIPPING_ENV = "local"
$env:SHIPPING_INTERNAL_TOKEN = "shipping-local-development-secret"
$env:SHIPPING_FAKE_WEBHOOK_SECRET = "fake-carrier-webhook-secret"
docker compose up --build
```

Health endpoints:

- `GET /health/live`
- `GET /health/ready`

The gRPC API listens on `localhost:9012` and requires `authorization: Bearer <SHIPPING_INTERNAL_TOKEN>` metadata.
For production, set `SHIPPING_ENV=production` and provide a TLS certificate and key through
`SHIPPING_GRPC_TLS_CERT_FILE` and `SHIPPING_GRPC_TLS_KEY_FILE`. This matches the Order client,
which verifies Shipping's TLS certificate using its configured CA and server name. The local
Compose profile is explicitly insecure for local development only.

Shipping registers both gRPC v1 and v2 on the same listener by default; set
`SHIPPING_V2_ENABLED=false` only for a deployment that has not migrated its callers to v2.
The v2 API persists bound quotes with their canonical request hash, immutable expiry and
monotonic quote version. It also serves exact, proof-bearing shipment revisions created after
the revision migration; older rows are deliberately not backfilled from current state. The
event cohort is pinned on each Shipment: legacy callers publish only `shipping.events.v1`, while
gRPC v2 and Packing v2 shipments publish proof-bearing events to `shipping.events.v2`. Runtime
rollout remains gated until owning-service and consumer acceptance is complete.

The employee read-only admin endpoints require an Identity-issued RS256 JWT in
`Authorization: Bearer <token>`. Shipping validates the issuer, audience, signature, expiry,
employee principal type and `SHIPMENT_VIEW#GLOBAL` grant. The current Shipping read model has
no warehouse ownership field, so warehouse- or department-scoped grants are not treated as
global access. Configure `SHIPPING_OIDC_ISSUER` and `SHIPPING_OIDC_AUDIENCE` (default
`omama-api`); `SHIPPING_OIDC_JWKS_URL` is optional and defaults to the Keycloak realm JWKS URL.

Service-to-service HTTP/gRPC operations continue to use the separate `SHIPPING_INTERNAL_TOKEN`.
Webhook authentication remains provider-specific; an internal token is not accepted as employee
authentication on the admin routes.

## Implemented endpoints

The REST contract is documented in [`shipping-service.openapi.yaml`](../../docs/03_api_specs/shipping-service.openapi.yaml), with access and response-state notes in [`shipping-api-state-matrix.md`](../../docs/03_api_specs/shipping-api-state-matrix.md). These routes are internal/provider endpoints, not public Kong APIs. Errors include `requestId` and the matching `X-Request-Id` response header.

- `POST /api/v1/internal/shipping/quotes`
- `POST /api/v1/internal/shipments`
- `POST /api/v1/internal/shipments/{shipmentID}/cancel`
- `POST /api/v1/internal/shipments/{shipmentID}/label-url`
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

Cancel a shipment through `POST /api/v1/internal/shipments/{shipmentID}/cancel` with internal
authentication and a provider-neutral `reason_code`: `PICKUP_DELAY`, `OUT_OF_STOCK`,
`RECIPIENT_CHANGED_MIND`, or `OTHER`. Cancellation is allowed only before carrier handover.
GHN timeout and retryable failures are reconciled through order detail; Shipping never sends a
blind second cancellation request.

Generate a short-lived print URL through
`POST /api/v1/internal/shipments/{shipmentID}/label-url` with body `{"format":"A5"}`.
Supported formats are `A5`, `80X80`, and `52X70`. Shipping stores only the stable
`label_reference`; provider print tokens and URLs are returned to the authenticated caller and
are never persisted. The returned `expires_at` includes a safety margin before GHN's token expiry.

Configure the GHN Developer Portal only after the public endpoint is reachable. Add the custom
header `X-GHN-Webhook-Secret` with the same value as `SHIPPING_GHN_WEBHOOK_SECRET`. A missing
local shipment returns `503` so GHN retries a callback that raced the shipment transaction;
invalid secrets and malformed provider payloads are rejected with non-retryable `4xx` responses.

## Order and Fulfillment integration

Order owns the packing-to-shipment handoff. It consumes the proof-bearing Fulfillment packing
event, verifies the exact Fulfillment revisions, persists a durable handoff, and calls Shipping's
gRPC v2 `CreateShipment`; Order also consumes `shipping.events.v2` and verifies exact Shipping
revisions. Shipping therefore does not consume Fulfillment events or call Order's
`GetShippingSnapshot`. This avoids two services independently creating a shipment for the same
package. Shipping v2 is enabled by default for the Order integration; configure the same service
credential on both sides: Order's `ORDER_SHIPPING_TOKEN_FILE` must
contain the value configured as Shipping's `SHIPPING_INTERNAL_TOKEN` (minimum 32 characters).
Outside explicit sandbox loopback, Order verifies Shipping's gRPC TLS certificate using its
configured CA and server name, so Shipping production deployments must configure its gRPC
certificate and key.

## Local verification

```powershell
go mod download
go test ./...
go vet ./...
go build ./cmd/server
```
