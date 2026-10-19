# P0 — Customer audit and implementation entry

## Goal / why

Establish a complete Customer flow map so visual alignment does not hide missing behavior. Produce an evidence-backed order of implementation and start the first coherent slice after baseline verification.

## Dependencies and prior decisions

- User approved implementation and goal/plan creation on 2026-10-19.
- Web `b1ff2918` is the read-only design/flow reference; ZIP is the API authority.
- Mobile baseline `8e1994a7`; initial checkout clean. Current branch is `feature/mobile-ui`.
- Existing `docs/01_requirements`, `02_architecture`, `04_testing` are placeholders on Mobile; create a compact master/phase/handoff system rather than copying Web documentation wholesale.
- Do not follow obsolete Stitch screen-by-screen approval for this already approved parity implementation.

## Scope / non-goals

In: Customer app screens, shared Flutter UI, navigation, corresponding domain/repository contracts, fixtures and meaningful regression tests. Out: Web/Admin edits, backend service implementation, deployments, provider provisioning, dependency upgrades, merges, iOS native execution on Windows.

## Current verified findings

- Mobile has Home/Catalog/Product/Cart/Checkout/Confirmation/Orders/Account/Address/Notifications/Login/Register.
- Recovery route/repository flow is absent. Web uses email request → neutral sent state → magic-link verification → reset proof → new password → normal login.
- Mobile main CTA is gold; Web primary is `#563A26` with white text. Further token differences must be recorded in parity matrix.
- Web additionally exposes promotions, payment, guest order lookup, profile/loyalty, after-sales/support, trace and B2B.
- ADB sees an authorized Android device; existing scrcpy process has a visible phone window. No native test yet.

## Steps

- [x] Verify branch names, clean baseline and current remote tips.
- [x] Read applicable project instructions and discover existing planning structure.
- [x] Preserve supplied Customer contracts with source SHA-256.
- [x] Read Customer flow entrypoints, services and corresponding schemas; create `docs/04_testing/CUSTOMER_PARITY_MATRIX.md` with source paths, gaps, priorities and validation.
- [x] Run baseline Flutter analysis/tests and record pre-existing failures separately.
- [x] Write P1 plan with exact contracts/files, then implement foundation + auth recovery.
- [ ] Validate first slice and record native visual/interaction evidence.
- [ ] Update handoff; create/push a scoped milestone checkpoint on Mobile only.

## Expected areas / contracts

`apps/mobile-app/lib/core/theme`, shared buttons/inputs/navigation; auth domain/adapters/Cubit/pages and router; corresponding tests. Additional feature files are identified per flow before changing them.

Recovery APIs: request `202`, verify `200`, reset `204`; shared safe Error envelope; no account enumeration; reset proof must not become a logged-in session. Handle expired/invalid link, 429, field validation and retries. Coupons must be validated against authoritative subtotal/context, with stale validation cleared when cart context changes.

## UI states / security / risks

Test request/loading/sent/error; token verifying/expired/error/reset/success; password rules/confirmation/visibility; pending disabled; keyboard and back/return target. Keep proofs/tokens ephemeral and out of logs. Do not copy browser cookie/BFF mechanics into native auth or treat mock OTP/email as production.

Risks: Web fixtures may contain policy placeholders; Mobile REST adapters predate ZIP; full parity spans missing features; device evidence must not be extrapolated into backend verification. Record open decisions precisely rather than inventing policies.

## Validation

From `apps/mobile-app`: `flutter analyze`, `flutter test`; targeted flow tests during implementation; `flutter build apk --debug` before device installation. Check scrcpy screenshots and interactions after transitions, including keyboard/scroll/back. Keep screenshots/logs out of Git.

## Acceptance criteria

All Web Customer route groups appear in matrix; current gaps and contract lifecycle are explicit; baseline recorded; first slice has a concrete P1 plan and tests; no unrelated changes or Web modifications.

## Completed / blockers / next

Completed: goal active, branch identification, remote-tip verification, safe switch to Mobile, initial Web palette/recovery reading. Blockers: none yet. Exact next action: preserve ZIP Customer contracts, inventory Web services, read and map auth/cart/checkout contracts and capture baseline checks.
