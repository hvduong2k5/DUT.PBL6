# Customer Web → Flutter parity

Approved: 2026-10-19. Classification: LARGE / cross-cutting Customer UI feature.

## Objective and scope

Make the Flutter Customer experience consistent with the approved Customer Web: palette, visual hierarchy, screen sequence, customer capabilities, recovery, promotions and purchase/after-purchase flows. Adapt desktop composition to native mobile interaction.

Read-only reference: `origin/feature/ui-web` at `b1ff2918c7297fc93d0cb9a15cc1d0a0ee46b72e`. Mobile baseline: `8e1994a7401a5c4759cfc24149182b5e2148b6be`, branch `feature/mobile-ui`. Remote tips were verified on 2026-10-19. Never change the reference or merge branches.

## Architecture and constraints

Retain Flutter/GoRouter/Cubit and repository interfaces. Extend existing features; add focused features only when a missing Web flow requires them. REST and local fixtures remain separate. ZIP Customer OpenAPI contracts govern HTTP payloads/statuses; Web governs presentation/flow. Neither fixtures nor a UI demo prove real backend processing, email delivery, payments or production policy.

## Roadmap and dependencies

| Phase | Scope | Dependency | Status |
|---|---|---|---|
| P0 | Read Web Customer, inventory screens/states, map contracts and gaps, capture baseline checks | Approved scope | ACTIVE |
| P1 | Shared tokens/components and auth recovery/login/register continuity | P0 mapping | OPEN |
| P2 | Discovery/product/cart/promotions/checkout/payment/confirmation parity | P1 foundation, stable contract mapping | OPEN |
| P3 | Account/profile/address/loyalty/orders/guest tracking/after-sales/support/trace/B2B parity | P2 transaction identity, per-flow mapping | OPEN |
| P4 | Cross-flow regression, Android/scrcpy QA, contract checks, reviewable delivery | All parity rows resolved or explicit external blocker | OPEN |

Detailed plans are written for current implementation only; later phases are refined against the actual code and contract before starting.

## Definition of Done

- Customer Web routes/actions have traceable Mobile counterparts or an explicitly justified platform/contract limitation.
- Recovery and coupon application work through their full states; no dead links or fabricated success.
- Shared palette/CTA/type hierarchy align with Web; mobile layouts support keyboard, scrolling, safe areas and back navigation.
- Stable APIs match supplied contracts; draft/planned capabilities remain explicitly separated from live integration.
- Focused tests, analysis, regression suite and Android build/device QA are recorded honestly.
- Only Mobile branch changes are committed/pushed, with verified remote checkpoint and current handoff.

Current active plan: [P0 audit and first parity slice](active/P0_CUSTOMER_PARITY.md).
