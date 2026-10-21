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
| P0 | Read Web Customer, inventory screens/states, map contracts and gaps, capture baseline checks | Approved scope | COMPLETE: route and contract inventory |
| P1 | Shared tokens/components and auth recovery/login/register continuity | P0 mapping | VERIFIED LOCAL; P1 acceptance reconciled; provider integration unverified |
| P2 | Discovery/product/cart/promotions/checkout/payment/confirmation parity | P1 foundation, stable contract mapping | IMPLEMENTED; acceptance reconciled; QA-01 payment status interaction open; voucher/actual QR limits explicit |
| P3 | Account/profile/address/loyalty/orders/guest tracking/after-sales/support/trace/B2B parity | P2 transaction identity, per-flow mapping | IMPLEMENTED; acceptance reconciled; native cancellation verified; QA-02 media handoff unverified; production context gaps explicit |
| P4 | Cross-flow regression, Android/scrcpy QA, contract checks, reviewable delivery | All parity rows resolved or explicit external blocker | ACTIVE: source144/144/analyze/APK PASS; P1–P3 review complete; QA-01 next, QA-02 unverified |

Detailed plans are written for current implementation only; later phases are refined against the actual code and contract before starting.

## Definition of Done

- Customer Web routes/actions have traceable Mobile counterparts or an explicitly justified platform/contract limitation.
- Recovery and voucher validation work through their full supported states; checkout application remains explicitly blocked by unchanged ZIP, as agreed by user. No dead links or fabricated success.
- Shared palette/CTA/type hierarchy align with Web; mobile layouts support keyboard, scrolling, safe areas and back navigation.
- Stable APIs match supplied contracts; draft/planned capabilities remain explicitly separated from live integration.
- Focused tests, analysis, regression suite and Android build/device QA are recorded honestly.
- Only Mobile branch changes are committed/pushed, with verified remote checkpoint and current handoff.

Current phase: [P4 validation/delivery](active/P4_CUSTOMER_QA.md). Resume from [CURRENT_STATE](../CURRENT_STATE.md), updated at evidence reconciliation2026-10-21. P1/P2/P3 plans now carry reconciled acceptance records; P4 and the parity matrix distinguish remaining evidence gaps from external contract limits. Do not restart completed slices.

Remaining sequence: QA-01 deterministic payment-status interaction checks→targeted QA-02 disposition→assess whole Definition of Done→Mobile-only final delivery. No broad device rerun, new feature phase or architecture change is implied.
