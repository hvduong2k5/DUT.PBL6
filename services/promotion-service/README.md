# MS-07 Promotion Service

Promotion Service owns campaigns, coupons, voucher holds and redemption records. This initial
foundation provides a Go service process, PostgreSQL migrations, liveness/readiness endpoints,
and a local PostgreSQL Compose stack. The voucher domain and transactional hold repository now
implement discount calculation, availability checks, hold creation, finalization and release.
The existing `packages/proto/promotion/v1/promotion.proto` remains the integration contract.

## Run locally

```powershell
Set-Location services/promotion-service
$env:PROMOTION_INTERNAL_TOKEN = "promotion-local-development-secret"
docker compose up --build
```

The service listens on HTTP `localhost:8017` and gRPC `localhost:9017`. Readiness checks
PostgreSQL at `GET /health/ready`; liveness is `GET /health/live`. PostgreSQL is exposed on
`localhost:5447`. gRPC calls require `authorization: Bearer <PROMOTION_INTERNAL_TOKEN>` metadata.

## Admin coupon API

- `GET /api/v1/admin/coupons?limit=50&offset=0`
- `POST /api/v1/admin/coupons` creates a coupon in `DRAFT` state.
- `PUT /api/v1/admin/coupons/{couponID}` edits a draft. Published coupons are immutable; create
  a new coupon if the offer must change.
- `POST /api/v1/admin/coupons/{couponID}/publish` publishes now or schedules for `starts_at`.
- `POST /api/v1/admin/coupons/{couponID}/pause` pauses an active/scheduled coupon and requires a reason.
- `GET /api/v1/promotions/active?limit=50&offset=0` lists public, active, in-window coupons with remaining total uses.

Admin coupon calls require an Identity-provisioned, Keycloak-signed RS256 employee JWT. Promotion verifies the signature
with its configured public key, issuer, `omama-api` audience, expiry, `principal_type=EMPLOYEE`,
and the signed `permission_grants` claim. `GET /api/v1/admin/coupons` requires
`PROMOTION_VIEW#GLOBAL`; create/update/publish/pause require `PROMOTION_MANAGE#GLOBAL`. Roles and
browser-supplied `X-User-*` headers do not grant access. Configure `PROMOTION_STAFF_JWT_ISSUER`,
`PROMOTION_STAFF_JWT_AUDIENCE` (default `omama-api`) and `PROMOTION_STAFF_JWT_PUBLIC_KEY_PATH`;
mount the matching Identity/Keycloak RSA public key read-only at that path. Until these values are
configured, admin routes fail closed with `503 AUTH_UNAVAILABLE`. Keep the key public only (never
mount a signing private key). gRPC and service-to-service loyalty routes continue using their
separate internal tokens. Do not expose service-internal routes directly to untrusted networks.
A lifecycle worker advances scheduled coupons and ends expired ones;
all manual/automatic status changes are audited. A voucher expiry worker marks overdue `HELD`
resources as `EXPIRED` in batches using PostgreSQL time, incrementing the resource version. It
serializes with finalize/release through row locks; the Order command remains authoritative for
its own outcome. Order 3.2 release-by-order also writes a durable `RELEASED` tombstone when no
hold exists yet, preventing a late voucher lock from attaching to an order that is already being
cancelled. `GetVoucherHoldByOrder` is read-only: it returns the current resource without storing a
query key or advancing expiry state. Callers can optionally provide the original command action,
idempotency key and request hash to retrieve its durable command evidence.

## Configuration

- `PROMOTION_HTTP_ADDRESS` (default `:8017`)
- `PROMOTION_GRPC_ADDRESS` (default `:9017`)
- `PROMOTION_DATABASE_URL` (local default targets port `5447`)
- `PROMOTION_INTERNAL_TOKEN` (required)
- `PROMOTION_STAFF_JWT_ISSUER` (required to enable staff coupon administration)
- `PROMOTION_STAFF_JWT_AUDIENCE` (default `omama-api`)
- `PROMOTION_STAFF_JWT_PUBLIC_KEY_PATH` (path to the Identity/Keycloak RSA public key PEM)
- `PROMOTION_CUSTOMER_PROFILE_URL` (Profile's authenticated customer-self endpoint for customer Loyalty reads)
- `PROMOTION_ORDER_EVENTS_ENABLED` (default `false`; enables the Order v3 paid consumer and Promotion outbox publisher)
- `PROMOTION_KAFKA_BROKERS` (comma-separated broker addresses; required when the consumer is enabled)
- `PROMOTION_ORDER_EVENTS_TOPIC` (default `order.events.v3`)
- `PROMOTION_ORDER_CONSUMER_GROUP` (default `promotion-order-v3`)
- `PROMOTION_ORDER_EVENTS_DLQ_TOPIC` (default `promotion.order.v3.dlq`)
- `PROMOTION_EVENTS_TOPIC` (default `promotion.events.v1`)
- `PROMOTION_LOYALTY_ORDER_TOKEN` (optional; enables internal Order earn/redemption commands)
- `PROMOTION_LOYALTY_REFUND_TOKEN` (optional; enables internal refund conversion command)

Database migrations run at startup and can also be applied explicitly with `go run ./cmd/migrate`.

The Order v3 event integration is opt-in. A paid event with `voucher_finalization: null` is recorded
as consumed and produces no voucher event. A proof-bearing event must reference the matching
FINALIZED hold and its decision ID before `voucher.used` is inserted into the transactional
outbox. Invalid paid events are sent to the configured DLQ; the source offset is committed only
after DLQ publication succeeds. The outbox publisher uses a short database lease, Kafka keyed by
`customer_id`, and exponential retry. Delivery is at least once, so consumers should deduplicate by
CloudEvent ID. Keep the feature disabled until Order emits the finalization proof for voucher
checkouts and both services are configured against the same Kafka cluster.

## Loyalty ledger foundation

`GET /api/v1/loyalty/points` is the customer-facing contract endpoint. It requires a customer Bearer
token and calls Profile's authenticated `GET /api/v1/profile` endpoint to verify the active customer
and resolve the canonical Profile customer ID before reading the ledger. It never accepts a customer
ID from the caller's path, query, body, or identity headers. Configure
`PROMOTION_CUSTOMER_PROFILE_URL` to Profile's customer-self endpoint (for example
`http://profile-service:8080/api/v1/profile`). Missing configuration, malformed Profile identity,
or an upstream outage fails closed; an inactive customer receives `403`.

`GET /internal/v1/loyalty/customers/{customerID}/points` provides the current wallet fields, and
`GET /internal/v1/loyalty/customers/{customerID}/ledger?limit=50&offset=0` provides a paginated history.
Both require `X-Internal-Token` and are intended for the trusted Gateway only; do not expose internal
routes directly to customer networks. Ledger entries are append-only and idempotent by request key and
source. The active development policy `loyalty-dev-v1` uses 100 basis points for purchase earning,
floor rounding, one point per VND for redemption/refund conversion, BRONZE as the base tier, a
15-minute redemption hold and no point expiry. These are provisional defaults for development.

The active policy is readable at `GET /internal/v1/loyalty/policy/active`. Internal commands are
separately gated: `PROMOTION_LOYALTY_ORDER_TOKEN` enables Order purchase earning and redemption
hold/finalize/release routes; `PROMOTION_LOYALTY_REFUND_TOKEN` enables refund conversion. Purchase
earn commands require a completion decision and policy version. Refund conversion commands require
an approved refund decision, explicit consent reference, amount and policy version. These routes do
not verify remote proofs themselves; only trusted Order/Care integrations may call them. Automatic
Order event consumption and automatic Refund processing remain disabled until their producer
contracts and credentials are accepted.

## Current data foundation

The first migration establishes promotion and coupon definitions, order-scoped voucher holds,
finalized redemption usage, command idempotency records and a transactional outbox. Coupon
redemption limits are checked in a transaction while the coupon row is locked; a cached counter
is never the authority for availability. The gRPC adapter implements the current Promotion v1
source contract, including the feature-gated Order 3.2 voucher-hold RPCs.
