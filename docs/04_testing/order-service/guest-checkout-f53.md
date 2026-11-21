# F-53 guest checkout implementation and acceptance

Base: `dev` at `bc48ea340fee511d7aeb86993cbdcaee2c0b7d39`.
Work branch: `codex/guest-checkout-f53`.

## B-G0 trust boundary

- Keycloak RS256 and guest-session HS256 validators are selected using token
  metadata, verified independently and never retried through another validator.
- Guest sessions have a dedicated key, issuer, audience and JWT type. Session
  activity is checked against the existing persisted/revocable guest store.
- A default-deny route matrix separates customers, guests, employees and exact
  order payment grants. Existing ownership and finance MFA checks remain in place.
- Anonymous issuance accepts an empty body/object only, persists one session and
  enforces a Redis quota (10 per IP/hour by default). An invalid request, disabled
  issuance flag or exhausted quota creates no session.
- The session repository can run independently of SePay. Grant minting/reads
  still require the configured SePay payment runtime and its existing ES256 key.
- The BFF gives any present customer credential precedence, including malformed
  or expired credentials. It never falls back to a guest cookie after failure.
- Guest cookies are host-only, httpOnly, SameSite=Lax, Secure in deployed HTTPS,
  and issued lazily for cart/checkout actions. A page view or empty cart GET does
  not create a session. Tokens are not returned by the Web session endpoint.
- Sandbox authentication, route behavior and legacy payment rejection status
  remain compatible. Keycloak errors use the shared safe Error envelope.

### Configuration

All five `GUEST_*_ENABLED` flags and `NEXT_PUBLIC_MVP_GUEST_ORDER_ENABLED`
default to `false`. The order identity compose overlay passes the flags through.
`GUEST_SESSION_SECRET` must contain at least 32 bytes and differ from
`ORDER_JWT_SECRET`.

With `WEB_AUTH_MODE=identity-bff`, set `GUEST_ORDER_UPSTREAM_URL` to the direct
gateway `/api/v1` base. The customer `/customer-api` session proxy is not an
anonymous issuance path. Missing guest routing configuration fails closed.

For per-client quotas behind proxies, configure explicit
`GUEST_TRUSTED_PROXY_CIDRS` in Order and the matching Kong trusted proxy chain.
The BFF can forward a single client IP from `WEB_GUEST_CLIENT_IP_HEADER` only
when that header is overwritten by the trusted storefront ingress. Without
proxy configuration the quota uses the direct peer address. Never trust an
arbitrary incoming client IP header or all internet addresses.

### Local checks

Use disposable PostgreSQL 16, Redis 7 and Kafka. The service tests create isolated
database schemas; set `ORDER_CORE_TEST_DATABASE`, `ORDER_TEST_DATABASE`,
`ORDER_TEST_SIM_DATABASE`, `ORDER_TEST_REDIS` and `ORDER_TEST_KAFKA`.

```text
go vet ./...
go test -count=1 ./...
ORDER_GUEST_TEST_KONG=true go test ./internal/transport -run TestGuestAnonymousGatewayIssuanceAUTH11 -count=1 -v
npm run verify:mvp
npm run build
```

The opt-in AUTH-11 test uses the repository's actual Kong 3.9 configuration,
Order HTTP handler and PostgreSQL, with no Keycloak or Identity service and no
customer credential. It verifies exactly one persisted session and no-store.
AUTH-12 through AUTH-16 cover input overrides, principal isolation, flag-off,
quota exhaustion and cross-validator rejection. Finance persistence regressions
retain MFA, receiver/connection scope and approval/executor separation checks;
STEP-07 additionally rejects a Customer finance write with a guest cookie present.

The full Web fixture is `python scripts/test-web-identity.py`. On this checkout
the Gradle wrapper JAR is absent, so use its `--gradle` override with an installed
Gradle 9.2 and `--java` with Java 21. This does not require source changes in
Identity or Keycloak. The fixture uses real Keycloak, Kong, Identity, Profile,
Catalog, Inventory, Shipping, PostgreSQL, Redis and Kafka; payment statements and
packing are TEST and the carrier is FAKE.

### B-G0 result (2026-11-21)

PASS: `go vet ./...`; the complete Order test suite with real PostgreSQL,
Redis and Kafka; AUTH-11 through real Kong; AUTH-12..16 and STEP-07;
Web `verify:mvp` (155 tests after the final credential checks); production build;
and the complete `test-web-identity.py` fixture (four accounts, registered
checkout/TEST payment/shipping/history, login/refresh/logout and persona gates).
All four frontend contracts lint successfully. Existing example warnings remain.

An initial Kong regex syntax error was corrected and verified with
`KONG_DATABASE=off kong config parse`. A transient Windows memory exhaustion
interrupted a later linker invocation; the affected checks passed again with
`GOMEMLIMIT=256MiB`, `GOMAXPROCS=2` and `go test -p 1`.

## B-G1 guest cart and COD

Implemented independent cart/COD admission flags, guest inline-address preparation,
credential-owned order reads and the order-code/tracking handoff. Cart mutation
quota is 60/session/minute. PostgreSQL admission locks the session and permits
at most 3 unpaid pending orders, counting in-flight operations; concurrent keys
cannot bypass it and existing idempotent results can replay at quota.

Tests cover keycloak-mode guest COD over HTTP through CONFIRMED_COD persistence,
a fourth-order rejection, replay at quota, eight concurrent admission keys,
cart flag-off and 60/61 mutation requests. The simulator address seam is unchanged.

The existing phone-OTP extension remains the tracking provider. The UI stops
submissions after five attempts and hides the local fixture OTP in production.
The external provider must enforce its own five-attempt limit; the UI is not a
server-side security boundary. Verify that provider during live acceptance.

`PROOF-COD-01` is pending the audited rc.2 deployment and a real logged-out order.

## B-G2 guest VietQR

The payment BFF now mints an existing order-bound ES256 grant using the active
session and stores it in an httpOnly cookie scoped to that exact order's payment
routes. Polling sends only the grant upstream; it never forwards the guest
session credential or identity cookies. Customer credential presence still wins,
and a failed customer validation cannot switch to the grant. An expired browser
grant cookie can be renewed lazily without exposing its token in JavaScript.

Verified SePay code: HMAC-SHA256 over timestamp plus raw payload, +/-300 second
window, deduplicated persisted observations, canonical SePay API rows, then
transactional allocation and inventory finalization. No browser action can mark
an order paid. Existing late-money reconciliation behavior is unchanged.

Local tests include guest keycloak-mode VietQR quote/checkout, exact-order grant
reads, rejection of a session as a grant, rejection on another order, signed
webhook plus TEST statement settlement, duplicate webhook replay with one receipt,
one allocation and one FIRST_PAID milestone, and actual timeout followed by late
money with no allocation/fulfilment. BFF tests verify cookie scope, no JSON token
exposure, grant-only polling and invalid customer-session precedence.

`PROOF-VQR-01` remains pending the approved live SePay transfer after deployment.

## B-G3 cart merge on Customer login

`POST /api/v1/cart` is a Customer-only merge operation already covered by the
Identity customer proxy allowlist. The guest JWT is a separate source field;
Customer authorization never switches to that credential. No Identity or
Keycloak source changes were needed. `GET /api/v1/cart?merge_key=...` reads the
Customer-bound immutable result without an active guest session.

Migration 009 adds `storefront_carts` and an immutable `guest_cart_merges` ledger.
Only participating carts convert from Redis. Shared PostgreSQL scope locks cover
conversion, subsequent mutations and core checkout cleanup. Converted scopes
remain SQL-authoritative with the merge flag off; expiration acts as an empty
tombstone instead of exposing a stale Redis cart. The merge commits target cart,
source clearing, per-SKU result and source-session revocation together.

Quantities sum, cap at the existing Order limit of 999/SKU, and retain at most
100 SKUs with existing target lines first. Catalog snapshots replace all retained
prices; unavailable/deleted SKUs are dropped with explicit reasons. Normal carts
use batch validation, with bounded individual checks only for an unavailable
batch. The Customer revision increases once. Same key/source/customer returns
the original ledger result even after revocation, without calling Catalog again.

The Web attempts merge after verified Customer login or registration, derives a
stable key from the httpOnly source cookie, clears only that cookie on success,
and displays merged/adjusted/dropped outcomes in cart. Failure preserves cookies,
keeps login successful and offers a retry. No source cookie is a no-op. Results
from a prior identity are not shown after logout.

Local tests cover x2+x3=x5, replay after revocation, fresh prices, a deleted SKU,
cross-Customer isolation, quantity cap, concurrent retries, a forced final-step
failure rolling back every SQL write, sticky SQL authority with flag off, and
checkout cleanup both clearing the converted cart and preserving a later edit.
BFF tests cover source/Customer credential separation, cookie preservation on
failure, stable retry key and login without a guest cookie.

The nullable COD payment deadline mismatch in the existing Web consumer was
fixed separately in `5cdf8181`, with a versioned pinned contract correction and
regressions for both guest and registered COD.

Run `python scripts/test-web-identity.py --guest-checkout` to extend the complete
real-stack fixture with logged-out COD/VietQR and a real OIDC login/Identity-proxy
merge. The fixture transfers remain TEST; `PROOF-MERGE-01` and the live COD/VietQR
proofs still require Muse audit, Bao approval, rc.2 deployment and live evidence.

## Release acceptance

B-G1 must not start before the B-G0 regression gate passes. B-G1, B-G2 and B-G3
each require their named live proof in the supplied brief. Local synthetic/test
transfers are not evidence of a real SePay transfer or a live storefront order.

The rc.2 pick and deployment require Muse's audit and Bao's go. Restart Kong after
container recreation before running the live proofs. No release branch pick,
deployment or live payment has been performed by this worktree.
