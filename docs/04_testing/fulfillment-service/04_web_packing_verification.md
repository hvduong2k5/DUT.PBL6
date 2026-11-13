# Web Packing integration verification — 2026-11-05

Baseline `dev@89c6d649e8955050db39e4742da1c720995f337a`, implementation on
`codex/packing-ui-integration`. All data belonged to an isolated local fixture.
No external payment or carrier action was performed.

## Results

| Check | Result |
|---|---|
| Web typecheck / ESLint | PASS |
| Web Vitest | PASS — 46 files, 146 tests |
| Web production build | PASS — 33 static pages plus dynamic routes, including both Packing pages and Fulfillment BFF |
| Identity Gradle test + bootJar with fixture PostgreSQL | PASS — 74 tests, 1 pre-existing opt-in live-provider test skipped; no failures |
| New Identity Fulfillment proxy tests | PASS — 3 tests, no skips |
| Catalog Gradle test + bootJar | PASS |
| Fulfillment Docker image | PASS |
| Real-service E2E through Web Packing BFF | PASS |
| Interactive Chromium Customer → staff Packing | PASS |
| Python / PowerShell syntax and diff whitespace | PASS |

The initial standalone Identity test invocation had no PostgreSQL and failed
its DB integration tests. The first fixture run then exposed a test setup error
(the oversized-body test had no configured Gateway). After configuring the test
target, the second fixture run passed with real PostgreSQL. Failed-run logs were
retained separately; they were not rewritten as successful evidence.

## Real service acceptance

Ran `scripts/test-web-identity.py --fulfillment --keep-running` with Go 1.27.3,
Java 21, Gradle 9.2.0, Node 22 and Docker. Evidence is in the ignored local
`.system_generated/packing-ui-e2e-r2/result.json`.

Identity/Keycloak, Kong, Profile, Catalog, Order, Inventory TLS, Fulfillment,
Shipping, PostgreSQL, Redis, Kafka, versioned RustFS, ffprobe and the ClamAV
engine were real. SePay statements were TEST, carrier FAKE, ClamAV signatures
were the isolated deterministic fixture database. Dart repository/device UI
acceptance was not run in this change.

Packing requests used Web's cookie BFF → Identity's staff proxy → Gateway →
Fulfillment, including binary video upload through the server relay. Acceptance
covered Customer denial, forged browser bearer rejection, untrusted Origin,
actual MP4/SHA256 verification, rejected media, stopped scanner preventing seal,
same-key seal replay and Shipping-outage recovery with exactly one shipment.
The resulting Order reached DELIVERED and appeared in Customer history. The
original stock assertions passed: physical 98, reserved 0, one commit event.

## Browser verification

Using separate Chromium Customer and Employee sessions, signed in via real
Keycloak and created a fresh order from Product → Cart → Checkout. A statement
was generated using only the fixture TEST `/pay` control. On the staff UI:

1. Saw the new READY task and accepted it for the current Employee.
2. Started work and saved a counted quantity matching its committed batch.
3. Saved the required checklist as PASS.
4. Selected and uploaded an actual MP4; observed VERIFIED from the service.
5. Entered parcel measurements and seal; confirmed completion in the dialog.
6. Loaded the immutable task history, including the staff actions through seal.
7. Read the Customer Order page showing PACKED, COMMITTED and its shipment ID.

For this fresh browser-created Order, a read-only DB check confirmed PACKED and
exactly one Shipping row. Browser screenshots are kept locally under
`output/playwright/`, excluded from Git because browser artifacts can contain
session/account details. The fixture's initial read-only result.json reflects
the automated acceptance before this additional browser-created Order.

## Limits

This verifies the local integrated Packing scope, not production readiness or
all admin screens. Actual SePay/GHN, COD, paid cancellation/reversal, multiple
packages, reopen and assigning another Employee remain gated. The launcher
uses the repository's mixed container/native fixture; a standalone all-service
Docker Compose deployment remains separate work.
