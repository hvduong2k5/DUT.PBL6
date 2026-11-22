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

Set `NEXT_PUBLIC_MVP_GUEST_ORDER_ENABLED` before building Web; the BFF and Order
flags are server configuration. Registered Customer COD retains its independent
`NEXT_PUBLIC_MVP_COD_ENABLED` gate.

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

Real Catalog COD retains the `dev` default policy `cod-collection-1.0`;
`ORDER_COD_POLICY_REFERENCE` is an optional versioned override. Guest COD still
requires its independent feature flag. Placement
snapshots only the amount to collect and leaves payment UNPAID. The previous
blanket rejection mixed placement with the later Finance settlement requirement;
the completion evidence gate remains enforced. No paid/settled proof is created.

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

Successful merge responses forward their validated original JSON bytes, so
large int64 Catalog versions cannot be rounded by BFF serialization. Guest cart
mutation rechecks revocation on its existing transaction connection; a real
PostgreSQL/Redis regression with a one-connection pool verifies successful
mutation and rejection after revocation without nested pool acquisition.

The nullable COD payment deadline mismatch in the existing Web consumer was
fixed separately in `5cdf8181`, with a versioned pinned contract correction and
regressions for both guest and registered COD.

Run `python scripts/test-web-identity.py --guest-checkout` to extend the complete
real-stack fixture with logged-out COD/VietQR and a real OIDC login/Identity-proxy
merge. The fixture transfers remain TEST; `PROOF-MERGE-01` and the live COD/VietQR
proofs still require Muse audit, Bao approval, rc.2 deployment and live evidence.

## Final local regression (2026-11-21)

Implementation HEAD: `5b45d5f7`. The full Web fixture passed on `db3fcdd5`;
the subsequent backend change reuses the same SQL transaction connection and
is covered by the single-connection integration regression and final Go suite.
Web source is unchanged between these revisions.

| Gate | Result |
| --- | --- |
| `go vet ./...` | PASS |
| `go test -p 1 -count=1 ./...`, real PostgreSQL/Redis/Kafka, `ORDER_GUEST_TEST_KONG=true` | PASS |
| Web `npm run verify:mvp` | PASS: 50 files, 169 tests; typecheck and lint |
| Web `npm run build` | PASS |
| All four required frontend OpenAPI files and the pinned MVP Customer contract | PASS; existing warnings remain |
| `python scripts/test-web-identity.py --guest-checkout` | PASS, four accounts, actual OIDC/Identity proxy and service stack |

The full fixture verifies guest COD, guest VietQR with order-grant-only polling
and TEST settlement, Customer COD, Customer VietQR through delivery/history,
merge x2+x3=x5, dropped-SKU reasons, replay after revocation, result retrieval,
login without a source cookie, refresh/logout, owner isolation and persona gates.
It asserts exactly five orders, four stock-commit events, physical/batch stock
95 and reserved/batch-reserved stock 0. Fulfillment/packing and SePay statements
are TEST; the carrier is FAKE. Optional Flutter client runtime was not run.

The local fixture result is recorded in the ignored
`.system_generated/web-identity/result.json`; detailed logs are in
`E:/DUT.K1N4/stage/f53-final-web-identity.log`, `f53-final-go-tests.log`,
`f53-final-web-gates.log` and `f53-final-web-build.log`. Use the fixture's
`--keep-running` option to inspect its disposable SQL state before cleanup.

One previous full Go run reported `ORDER_DEADLINE_GUARD` in
`TestSePayObserveAllocateFinalizeReplay`. Five immediate focused repetitions
passed, as did the persistence package on the clean full rerun. The cause was
not established; no payment/deadline guard was changed. The failed run is
preserved locally in `E:/DUT.K1N4/stage/f53-go-tests-deadline-guard.log`.

## Integration with current dev (2026-11-22)

PR #164 integrates `dev` at `6971c2f5` through merge commit `f1fded3c`.
All 15 conflicted files were reconciled, including semantic integration:

- Customer requests retain `withCustomerAuth` renewal, rotated response cookies,
  refresh-only credential precedence and `customerCommerceUpstreamUrl` routing.
  Invalid Customer renewal never switches to a guest credential.
- Principals retain both F-53 kinds and current IAM/permission grant fields.
  Current storefront styling, recommendations and frozen order views are kept.
- COD retains the current `dev` default policy `cod-collection-1.0`; the optional
  server override and independent Guest COD gate remain available. COD UI
  deadlines remain null, matching the current read model.
- Converted SQL carts retain F-35 live Catalog display enrichment on reads and
  mutations. Enrichment never changes the immutable stored merge outcome.
- The provider contract is corrected/versioned at 3.2.1, the public generator
  retains version 1.1.0-integration, and the design verifier pins the new provider
  version. Regenerating the public artifact is now deterministic.

PASS: Web typecheck/lint, 255 tests across 63 files, production build; five
frontend/MVP OpenAPI lint checks; `mvp/verify.py` and `verify_order_design.py`
including OpenAPI, wire baselines, registry, proof and negative guards.
PASS: `go vet ./...` and the complete `go test -p 1 -count=1 ./...` with
real PostgreSQL/Redis/Kafka and `ORDER_GUEST_TEST_KONG=true` on the integrated
branch. The [Order CI gate on `44eb2034`](https://github.com/hvduong2k5/DUT.PBL6/actions/runs/45000410088/job/137140482890)
also passed the complete suite with the race detector, contract/runtime-payload
verification, code generation/formatting and sandbox image build.
The first integrated real-stack run passed guest COD/VietQR, exact-order grant
polling, OIDC cart merge/replay/revocation, registered COD/VietQR through delivery,
account isolation, and exact Inventory totals. Its final staff projection check
then failed with 403: the new F-53 principal-kind gate did not admit the current
`dev` admin-detail route, although the existing staff gate did. The failed log is
preserved at `E:/DUT.K1N4/stage/f53-merge-web-identity.log`.

Commit `06f5a16a` shares the existing employee route allowlist between both gates,
retaining admin detail and cancellation requests. Application permission and
warehouse checks remain unchanged. App/transport tests pass, including principal
boundary and route-allowlist regressions. The full guest-enabled fixture is rerun
with output at `E:/DUT.K1N4/stage/f53-merge-web-identity-final.log`; its final status
and the latest CI results are recorded on [PR #164](https://github.com/hvduong2k5/DUT.PBL6/pull/164).
The ignored `.system_generated/web-identity/result.json` is written only after
all fixture assertions succeed. The hosted Web CI additionally runs real
Fulfillment video processing and the Flutter client; local guest verification
uses TEST Fulfillment and SePay statements, a FAKE carrier and no external money.

Two earlier local full-suite runs exposed timing assumptions in test fixtures:
a single refund scheduler tick was assumed to reach UNKNOWN, and a foreground
checkout tick treated a stale epoch from a competing worker as fatal. Commit
`777b240e` waits for the asserted refund state and retries only `ErrLease` while
still requiring a successful checkout. Financial effect counts, terminal-state
assertions and all production lease/deadline guards are unchanged. SQL display
enrichment and the refund lost-ACK scenario also passed five focused repetitions.

The initial CI race run detected test fixtures writing guest authentication
configuration after Kafka workers had started. Commit `44eb2034` configures
those fixtures before `Start()`; the complete race suite passed afterward.
No production synchronization or race checks were weakened.

The GitHub `dev` ruleset requires at least one approving review, Code Owner
review where applicable, resolved review threads and linear target history.
Repository auto-merge is disabled. These requirements remain separate from
resolving textual conflicts and running local tests.

## Release acceptance

B-G1 must not start before the B-G0 regression gate passes. B-G1, B-G2 and B-G3
each require their named live proof in the supplied brief. Local synthetic/test
transfers are not evidence of a real SePay transfer or a live storefront order.

The rc.2 pick and deployment require Muse's audit and Bao's go. Restart Kong after
container recreation before running the live proofs. No release branch pick,
deployment or live payment has been performed by this worktree.
