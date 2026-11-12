# Shipping HTTP API State and Access Matrix

This matrix describes the currently implemented HTTP adapter in
[`shipping-service.openapi.yaml`](shipping-service.openapi.yaml). It is not a decision to
publish these routes through Kong.

| Route group | Authentication / intended caller | Success and empty state | Error and retry behavior | Exposure status |
|---|---|---|---|---|
| `/health/*` | No auth; platform probes | `200` when process/database is healthy | Readiness returns `503` when database is unavailable | Operational only |
| `/api/v1/internal/*` | Shared `SHIPPING_INTERNAL_TOKEN`; trusted service caller | Quote `200`; shipment creation `201`; actions `200` | `400` malformed input or missing/invalid `Idempotency-Key`, `401` missing/invalid token, `404` missing shipment, `409` transition/idempotency conflict, `413` body exceeds 1 MiB, `415` missing/unsupported JSON media type on JSON-body routes, `422` validation, `502` carrier failure | Internal only; Order's canonical integration remains versioned gRPC |
| `/api/v1/admin/shipping/*` | Identity-issued employee RS256 JWT; issuer and audience validated; requires `principal_type=EMPLOYEE` and `SHIPMENT_VIEW#GLOBAL` | Shipment `200`; list returns `200` with `items: []` when empty | `400 INVALID_PAGINATION` for invalid/out-of-range/repeated page values or overflow; `400 INVALID_STATUS_FILTER` for an invalid or repeated status; `400 INVALID_FILTER` for repeated carrier/orderId; `401` missing/invalid JWT, `403` missing global shipment permission, `404` missing shipment, `500` internal failure | Employee read-only API; no warehouse/department filtering is supported yet |
| `/api/v1/orders/{orderID}/tracking` | Shared internal token; backend/BFF caller | `200`; no shipments returns `shipments: []`; only applied events and no internal notes | `400` blank order ID, `401` invalid token, `500` internal failure | Not a public customer route; no Kong route is implied |
| Fake carrier webhook | HMAC-SHA256 in `X-Shipping-Signature` using exact body | `200` accepted or duplicate callback | `400` malformed body/payload, `401` invalid signature, `404` unknown shipment, `413` body exceeds 1 MiB, `415` missing/unsupported JSON media type after signature validation, `500` processing failure | Local/test carrier only |
| GHN webhook | Constant-time comparison of `X-GHN-Webhook-Secret` | `200` accepted or duplicate callback | `404` when disabled, `400` malformed payload, `401` invalid secret, `413` body exceeds 1 MiB, `415` missing/unsupported JSON media type after secret validation, `503` shipment not committed yet (retry), `500` unexpected processing failure | Provider callback only; endpoint must be reachable over HTTPS in deployment |

## Common error shape

All HTTP errors return `{ "code": "...", "message": "...", "requestId": "..." }` and
also include the matching `X-Request-Id` response header. Messages are safe summaries and
must not include provider credentials, raw webhook bodies, or internal exception details.

Shipment creation, cancellation, and handover require a 16-128 character printable ASCII
`Idempotency-Key`. Repeated cancellation and handover requests return the current shipment
state without repeating the state transition. Shipment creation replays return the original
shipment only when the key and request payload match; reusing a key with a different payload
returns `409 IDEMPOTENCY_KEY_REUSED`. The request fingerprint is persisted with the shipment.
Cancellation and handover keys are persisted per shipment/action and bound to a canonical request
hash. Matching retries return the current shipment state; reusing a key for a changed request or
using a new key for an already-bound action returns `409 IDEMPOTENCY_KEY_REUSED`.
Shipments created before the fingerprint migration have no stored fingerprint and fail
closed with the same conflict on create replay; reconcile those legacy requests before
retrying them with a new key.

## Explicitly not covered

- Customer-facing shipment tracking behind customer authentication and order ownership checks.
- Any new Kong route, public hostname, or gateway policy.
- The separate gRPC v1/v2 contracts and Kafka event schemas.
