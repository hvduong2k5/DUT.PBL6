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

The existing provider contract is documented in [`shipping-service.openapi.yaml`](../../docs/03_api_specs/shipping-service.openapi.yaml). Gateway/auth rollout is tracked in the shared [`admin-stable.openapi.yaml`](../../docs/03_api_specs/admin-stable.openapi.yaml) and [`admin-api-state-matrix.md`](../../docs/03_api_specs/admin-api-state-matrix.md). Only admin shipment reads and the GHN callback are registered at Kong; internal commands, order tracking and the fake callback are not public. Errors include `requestId` and the matching `X-Request-Id` response header.

- `POST /api/v1/internal/shipping/quotes`
- `POST /api/v1/internal/shipments`
- `POST /api/v1/internal/return-shipments` (Care only after employee case approval)
- `GET /api/v1/internal/return-shipments/{returnShipmentID}`
- `POST /api/v1/internal/shipments/{shipmentID}/cancel`
- `POST /api/v1/internal/shipments/{shipmentID}/label-url`
- `POST /api/v1/internal/shipments/{shipmentID}/handover`
- `GET /api/v1/admin/shipping/shipments`
- `GET /api/v1/admin/shipping/shipments/{shipmentID}`
- `GET/POST /api/v2/admin/shipping/shipments/{shipmentID}/assignment` (separate opt-in flag)
- `GET /api/v2/admin/shipping/shipments/{shipmentID}/assignment/history` (separate opt-in flag)
- `GET /api/v1/orders/{orderID}/tracking`
- `POST /api/v1/webhooks/logistics/fake`
- `POST /api/v1/webhooks/logistics/ghn` (only active when GHN is configured)

The fake webhook requires `X-Shipping-Signature: sha256=<hex hmac>` calculated from the exact request body and `SHIPPING_FAKE_WEBHOOK_SECRET`.

### Customer return pickup (MVP)

Care approves or rejects a return case; Shipping does not make that decision. After approval,
Care can create a separate return-shipment through the internal endpoint using a stable
`Idempotency-Key`. The customer address is the pickup, the warehouse is the destination, COD is
always zero, and the shop pays the carrier fee. This MVP deliberately supports the deterministic
`FAKE` carrier only; GHN return pickup is unsupported until its mapping and sandbox behavior are
implemented and verified. Any future carrier adapter must honor the stable idempotency key or
reconcile by a stable client reference before retrying a create. Return records and outbox events are separate from outbound delivery
shipments, and their events use `shipping.return.events.v1` (partition key `return_shipment_id`).
Shipping does not approve Care cases, issue refunds, or change Order state. Return event
consumers remain pending integration acceptance.
## Company delivery implementation plan

Company fleet delivery is a separate `INTERNAL` carrier, not a rename of `FAKE` and not an
operator override for GHN shipments. Implementation proceeds in separately tested tasks:

1. **Implemented foundation:** opt-in provider, configured flat rate/estimated transit time,
   stable shipment references, existing create/cancel/idempotency lifecycle and independent tests.
2. **Implemented storage/application foundation:** persistent driver assignment, immutable
   assignment/reassignment audit, actor-scoped idempotency and concurrency guards. Opt-in REST
   assignment endpoints now validate signed dispatcher permissions and live Identity state.
3. **Implemented opt-in backend:** driver JWT permission and assignee-bound acceptance/progress,
   manual reasons, atomic shipment/tracking/revision/outbox writes and retry protection.
   Verified photo evidence, live Identity provisioning and end-to-end consumer acceptance remain pending.
4. **Pending:** connect actual employee Web BFF/workbench and register only approved public routes.
5. **Pending:** company-delivery return workflow, integration with Order/Care, then joint Coolify
   acceptance for GHN and INTERNAL. A returned outbound parcel and a Care-approved new pickup
   remain distinct resources/workflows; neither directly executes a refund.

Keep `SHIPPING_INTERNAL_DELIVERY_ENABLED=false` until workbench acceptance. When explicitly
testing the foundation, set `SHIPPING_INTERNAL_DELIVERY_FEE_VND` (0..1,000,000,000) and
`SHIPPING_INTERNAL_DELIVERY_ETA` (positive Go duration, at most 720h). No tariff is silently
selected. The example 25,000 VND / 24h is a development example, not an approved production rate.
The provider accepts `STANDARD`, creates `INT-<shipment UUID>` tracking codes, and does not
invent a driver, pickup confirmation, proof, printable URL or delivered state. Nonzero COD is
rejected until collection/reconciliation policy is implemented. No public INTERNAL webhook is
accepted. Existing GHN/fake behavior and their credentials are unchanged.

Assignment APIs enforce `SHIPMENT_ASSIGN#GLOBAL` for dispatch. `SHIPMENT_EXECUTE#GLOBAL`
is required for driver acceptance/progress, separate from existing shipment read permission.
Driver action authorization must also match the stored assignee; a GLOBAL action grant alone
does not permit manipulating another driver's shipment. These grants are not seeded in Identity
by this task and driver workbench is not accepted yet. Start with prepaid/no-COD delivery; require reason for failure and
an approved immutable proof reference before declaring proof-bearing delivery complete.

Assignment storage uses `internal_deliveries` for current state and
`internal_delivery_assignment_events` for append-only snapshots (migration 000010). Initial
assignment expects version 0; reassignment requires the current assignment version and a reason.
Only prepaid INTERNAL shipments in CREATED/READY_FOR_HANDOVER can be assigned; the same driver
with a new key is rejected as a no-op, and reassignment after handover is not yet supported.
The repository locks the shipment row and commits assignment plus audit in one transaction.
The assignment version is distinct from the shipment proof revision: assigning a driver does
not invent a tracking event, change shipment status or publish an Order lifecycle event.
Exact retries return their original snapshot even after a later reassignment; changed payloads
or reuse against another shipment conflict. Actor IDs come from the verified employee principal,
never a client-supplied override; the opt-in API checks live Identity actor and driver eligibility.
Stored UUIDs alone do not prove that a driver is an active employee.

Run the real PostgreSQL assignment suite with `SHIPPING_TEST_DATABASE_URL` pointing to a test
database and `go test -race ./internal/repository/postgres -run TestPostgresInternalDeliveryAssignments`.
It creates/drops its own random schema and verifies migration replay, concurrent assignment,
immutable retries, carrier/status/COD restrictions and transaction rollback on injected failure.

### Internal driver progress

POST `/api/v2/admin/shipping/shipments/{shipmentID}/progress` shares the opt-in assignment
flag and requires SHIPMENT_EXECUTE#GLOBAL plus live ACTIVE EMPLOYEE validation. Payload:
`{ "assignment_version": 1, "expected_revision": 4, "status": "IN_TRANSIT", "reason": "Started route" }`.
Idempotency-Key is required; actor comes only from JWT. Under a shipment lock, the current
driver and accepted current assignment are checked, then the shipment revision is compared.
Warehouse handover must already exist. Stages are HANDED_OVER → IN_TRANSIT → OUT_FOR_DELIVERY
→ DELIVERED/DELIVERY_FAILED; failed deliveries may retry OUT_FOR_DELIVERY or RETURNING → RETURNED.
No skips, repeated new actions or terminal rewrites. Acceptance/reassignment cannot bypass this.

Migration 000012 stores immutable actor/action receipts. Status, tracking, receipt, applicable
exact-revision proof and existing versioned event outbox commit in one transaction. The existing
Kafka worker publishes these events; no new Order protocol is introduced. Order's adapter uses
created/handed-over/delivered/delivery-failed v2 proofs; intermediate/return events keep their
existing semantics and do not imply automatic refund or Care return-pickup completion.
Reason is a private manual attestation, not verified photo evidence; settlement_verified remains
false. GHN/FAKE cannot use these employee actions. No new Kong route/UI is enabled yet.

### Driver acceptance foundation

Migration 000011 persists acceptance against an immutable assignment version. Only the
assigned driver can accept before handover; stale versions and duplicate new actions are
rejected. Exact retries return the original receipt even after reassignment or handover.
Reassignment requires a new acceptance: old receipts never authorize a new assignment.
Acceptance does not change shipment status/revision or publish a delivery event to Order.
POST `/api/v2/admin/shipping/shipments/{shipmentID}/accept` is enabled by the same opt-in
assignment flag. It requires SHIPMENT_EXECUTE#GLOBAL plus resource ownership checked under
the database lock, an ACTIVE EMPLOYEE validated via Identity `/api/v1/users/me`, an
Idempotency-Key and JSON `{ "expected_version": 1 }`. Actor/driver overrides are rejected.
USER_VIEW is not required for the driver's self-service action. GLOBAL is the token scope,
not an ownership bypass; SHIPMENT_ASSIGN/SHIPMENT_VIEW alone cannot accept a delivery.
Identity must provision SHIPMENT_EXECUTE before rollout. No Kong route or driver UI yet;
driver UI and deployment acceptance remain pending. PostgreSQL tests cover
ownership, versioning, replay, reassignment and handover boundaries.

### Dispatcher assignment API

Enable `SHIPPING_INTERNAL_ASSIGNMENTS_ENABLED=true` only with internal delivery enabled and
`SHIPPING_IDENTITY_BASE_URL` set to the Identity origin. HTTPS is required except loopback HTTP
in local mode. The flag defaults to false and assignment routes are not registered at Kong yet.
The existing UI workbench still needs a BFF projection/authentication adapter.

Assignment GET/current/history require `SHIPMENT_ASSIGN#GLOBAL`. POST additionally requires
`USER_VIEW#GLOBAL`, because the existing Identity provider protects `GET /api/v1/admin/users/{id}`
with USER_VIEW. Shipping forwards the verified caller's bearer, not a shared token or actor
header. Each request validates ACTIVE EMPLOYEE state and matching user_id/sub at `/api/v1/users/me`.
POST also checks the target's current ACTIVE EMPLOYEE state through that admin detail endpoint.
No fresh Identity permission list replaces signed JWT grants. Directory timeout, malformed data,
principal mismatch or redirect fails closed with 503; ineligible target is 422; insufficient
directory access or inactive actor is 403. Exact retries still require current eligibility.
POST body is `{driver_id, expected_version, reason}` plus a 16..128 printable ASCII
`Idempotency-Key`; unknown fields such as actor_id are rejected. History returns `{items: [],
total: 0}` when empty. Actor/driver UUIDs are Identity principal IDs, not Keycloak subjects.

Identity provisioning of SHIPMENT_ASSIGN, affected role assignments and real JWT acceptance
remain rollout work. No Identity migration, frontend code or permission bypass is introduced by
this task. The HTTP/PostgreSQL suite verifies signed RS256 tokens against a test JWKS and an
explicit HTTP fixture matching Identity's existing DTOs; it is not a live Identity deployment test.

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

By default, Order owns the packing-to-shipment handoff. It consumes the proof-bearing Fulfillment packing
event, verifies the exact Fulfillment revisions, persists a durable handoff, and calls Shipping's
gRPC v2 `CreateShipment`; Order also consumes `shipping.events.v2` and verifies exact Shipping
revisions. In this default mode, leave `SHIPPING_ORDER_GRPC_ENDPOINT` unset so Shipping does not
also consume Fulfillment events or call Order's `GetShippingSnapshot`. Shipping v2 is enabled
by default for the Order integration; configure the same service
credential on both sides: Order's `ORDER_SHIPPING_TOKEN_FILE` must
contain the value configured as Shipping's `SHIPPING_INTERNAL_TOKEN` (minimum 32 characters).
Outside explicit sandbox loopback, Order verifies Shipping's gRPC TLS certificate using its
configured CA and server name, so Shipping production deployments must configure its gRPC
certificate and key.

The merged F-41 integration also supports an opt-in Shipping-owned handoff. To use it, set
`ORDER_SHIPPING_OWNS_PACKING_HANDOFF=true` on Order (which disables Order's shipment-create
worker and enables its authenticated `GetShippingSnapshot` RPC). Configure
`SHIPPING_ORDER_GRPC_ENDPOINT`, `SHIPPING_ORDER_INTERNAL_TOKEN` (matching
`ORDER_SHIPPING_CALLER_TOKEN`), `SHIPPING_ORDER_CA_FILE` and `SHIPPING_ORDER_SERVER_NAME` on
Shipping. Set `SHIPPING_PACKING_V2_ENABLED=true` to additionally consume proof-bearing
`fulfillment.events.v2`; the legacy `fulfillment.events.v1` consumer is also registered.
Never run both handoff owners for the same deployment or publish a package to both packing
topics. The consumer retains the provider-neutral snapshot and package idempotency checks;
outbound shipment creation, employee JWT reads and Care return pickups share the same
Shipping implementation in either mode. TLS is required by configuration for packing v2.

## Local verification

### Gateway and owning-service deployment

1. Create the local shared network once: `docker network create omamx-network` (skip if it exists).
2. Start Shipping with `docker compose -f docker-compose.yml -f docker-compose.gateway.yml up -d --build`.
   Only the application joins the external network under DNS `shipping-service`; the database and
   Kafka remain on the private Compose network. For Coolify set `SHIPPING_GATEWAY_NETWORK=coolify`.
   A production application still needs `SHIPPING_ENV=production` and mounted gRPC TLS files;
   the local stack is not a production secret/TLS provisioning mechanism.
3. Load `services/api-gateway/config/kong.yml` into the gateway. The public admin provider routes
   are `GET /api/v2/admin/shipping/shipments` and `GET /api/v2/admin/shipping/shipments/{id}`;
   Kong preserves their full path; Shipping serves authenticated v2 aliases of its provider reads.
   The legacy camelCase v1 UI contract is
   not replaced by these snake_case provider routes. The public callback is
   `POST /api/v1/webhooks/logistics/ghn`. The DB-less local gateway mounts
   it; the DB-backed production gateway requires the deployment owner to apply it to Kong's DB.
   Committing YAML does not update an already-running gateway. Admin reads forward the employee
   bearer unchanged; GHN must supply `X-GHN-Webhook-Secret`. Kong never injects an internal token.
4. Deploy Identity V23, sync affected principals using the existing IAM sync-permissions endpoint
   (`forceRelogin=false`), then refresh JWTs. Required grant is `SHIPMENT_VIEW#GLOBAL` and audience
   `omama-api`. V22 is reserved by Promotion: deploy that seed before V23 in the shared
   Identity migration stream. If Shipping must deploy first, coordinate a new higher version
   for the still-unreleased Promotion seed; do not enable out-of-order migrations blindly or
   rewrite applied Flyway history. Existing JWTs are not changed by a SQL migration.
5. Keep one packing-handoff owner as described above. With Order-owned handoff, configure
   Order's Shipping endpoint/CA/token, shared Kafka broker and `shipping.events.v2`. With F-41
   Shipping-owned handoff, additionally configure the dedicated snapshot credential and Order's
   ownership flag. Fulfillment remains the packing-proof producer in both modes.
6. Care's approved-return caller is implemented on its own branch, not present in this checkout.
   It calls the private `POST /api/v1/internal/return-shipments`, sends the Shipping service token
   and a stable `Idempotency-Key`; it does not call Kong or trigger a refund through Shipping.

The Web admin Shipping BFF still targets port 4030 and expects camelCase UI projections plus
session credentials. Shipping returns its snake_case provider read model and verifies a JWT.
Do not point that BFF at this service without an explicit authentication/projection adapter;
this backend/gateway task does not claim the admin UI or company-driver workbench is connected.
Customer checkout and tracking continue through Order, not an unprotected Shipping endpoint.

The `Shipping integration gates` workflow runs regression guards, race/unit tests, real
PostgreSQL transport/outbox tests and Kong config validation. Locally run
`python scripts/verify-shipping-integration.py` with the MVP requirements installed.
Set `SHIPPING_TEST_DATABASE_URL` to a **disposable/test** PostgreSQL database to enable the
isolated-schema gRPC/HTTP persistence test. It uses a FAKE carrier and verifies persisted outbox
rows, not live Kafka delivery or real Order/Care processes. Full Web/Identity/Order/Fulfillment/
Shipping acceptance remains in `scripts/test-web-identity.py --fulfillment`; GHN production
acceptance and deployed employee token checks remain deployment checks. The live staging
verification below is separate from CI's deterministic fake-carrier acceptance.

### Live GHN staging verification (2026-11-20)

Verified against `https://dev-online-gateway.ghn.vn` using local Shipping, PostgreSQL and Kafka,
with a temporary HTTPS tunnel restricted to the exact GHN callback path. No production order
or real customer was used; the successful test order had zero COD and was cancelled afterwards.

- Token/shop lookup succeeded; the configured shop was active and its pickup district/ward
  matched local configuration. Preview through Shipping returned a fee of 53,900 VND for a
  500 g test parcel.
- Shipment creation returned 201. Replaying the same create key returned the same shipment
  and carrier tracking code; PostgreSQL contained exactly one row for that test order.
- A5 print-token generation returned a short-lived staging URL; fetching its page returned
  200 with non-empty HTML. Neither the print token nor its URL is recorded here.
- GHN itself delivered the create callback through the tunnel; Shipping persisted an applied
  `ready_to_pick` event and advanced to `READY_FOR_HANDOVER`.
- Cancellation through Shipping returned 200. Replaying its action key did not increase the
  revision. GHN order detail independently confirmed `cancel`; GHN's actual cancellation
  callback was also persisted. Replaying the cancellation event identity returned 200 /
  `DUPLICATE`, with one event row and no revision increase.
- Created/status-changed outbox rows were marked published to the local Kafka v1 topic. This
  carrier smoke test used synthetic order/package identifiers, not Fulfillment proof; it is
  not evidence of a live GHN-to-Order v2 consumer run. The full internal v2 path is covered by
  the separate fake-carrier acceptance workflow.
- Missing/incorrect webhook secrets returned 401; a correct secret with invalid payload
  returned 400. The temporary tunnel returned 404 for admin and internal command paths.

One earlier create attempt failed because GHN's staging warehouse-routing service timed out
(`WAREHOUSE_NOT_FOUND`). A client-reference lookup confirmed no order existed for that attempt
before a separate test case was created successfully. This is not proof of automatic recovery
from every ambiguous provider failure. A cancelled order was found by carrier order code even
when its client-reference lookup returned not found; use authoritative carrier detail when
investigating cancellation.

Still outside this staging smoke test: physical pickup/delivery/failure/return transitions,
live GHN plus the full Order/Fulfillment stack, production gateway/TLS/secrets deployment, and
GHN customer-return pickup (the return MVP remains FAKE-only). A temporary tunnel URL is not a
deployment endpoint: disable the staging portal callback after the tunnel is stopped, or replace
it with the deployed callback URL before resuming tests. Never commit webhook/API secrets.

```powershell
go mod download
go test ./...
go vet ./...
go build ./cmd/server
```
