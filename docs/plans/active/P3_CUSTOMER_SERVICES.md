# P3 — Account, ownership and Customer service flows

## Goal / dependency

Close missing Web Customer destinations and protect transitions between account, orders and customer service. P2 core automated checks pass (83 tests); native QA and discovery limitations are still tracked, so independent P3 implementation must not mark P2 complete prematurely or rewrite its contracts.

## Scope / prior decisions

Keep the supplied ZIP unchanged, including stable/draft/planned distinctions. User confirmed voucher submission waits for contract extension. Match Customer Web presentation; do not copy its unsupported legacy request fields or placeholder contact numbers. Changes stay on Mobile branch only.

## Current evidence

- Web profile supports name/email changes; phone is read-only. ZIP GET/PUT `/profile` uses existing Customer projection and `{full_name,email}` only.
- Loyalty stable overview has customer_id/current_points/tier/points_to_vnd_rate; history/redemption are not stable.
- ZIP address omits district; Mobile account model/form still exposes it and must be aligned.
- Web guest lookup uses order/contact → neutral challenge → OTP → exact-order access. Contract relies on server-issued guest_session/guest_order_grant cookies, not a client-invented order grant. Native HTTP currently has no cookie manager, so this is a verified transport gap.
- Catalog advanced filters and legacy category_id/page_size query parameters in Web are not described in ZIP's parameter list. Do not invent server filter guarantees; track contract limits and use supported pageSize/q.

## Steps / expected areas

- [x] Extend existing transport with native memory-only server-cookie handling (no hand-written cookie parser); retain bearer tokens and a Flutter-web-safe conditional adapter. Existing dependencies have no cookie jar. Use standard small Dio cookie packages after checking primary docs; do not persist guest proofs to files or fabricate cookies. Clear native cookies on logout/reset.
- [x] Profile/loyalty domain, REST/mock, Cubits and pages; safe login return and server errors. Update account navigation and current-user display after saved profile.
- [x] Two-level account address form/payload/default/delete behavior and saved-address checkout continuity.
- [x] Current-session restore/refresh and capability projection/account guards. Network failure keeps tokens; invalid session clears; stale identity/capability responses cannot regrant access. B2B route-specific entry guard remains pending with B2B implementation.
- [x] Guest lookup challenge/expiry/resend/OTP/grant and exact-order navigation; preserve only server-issued authority, never direct-ID authorization. Mock ownership denies history/other orders/expired grants. Native and real-cookie integration QA still pending.
- [x] Order read/cancel/tracking: exact SKU/raw price and server totals, recipient snapshot, unknown status, carrier/seal/video/checkpoints, retry/confirmation/server-authoritative cancel. Guest remains read-only. Web reference has no reorder action/ZIP endpoint; no invented reorder API.
- [ ] Order history search/filter/pagination reconciliation and native order-action QA.
- [x] Support context/create/detail/public messages. No fake upload/contact extension/hotline/CSAT. Automated create→ticket→reply, idempotency and ownership checks PASS; native form/keyboard inspected, native submission pending.
- [x] Trace manual-code entry/detail/error/retry, deep link, find-product search, HTTPS-only external media action. Local certificate fixture explicitly labelled; video launch/native build QA pending.
- [ ] After-sales create/detail with drafts separated. Stable Order lacks line_id required by creation; draft eligibility lines have no defined item shape. Do not invent IDs or promote draft calls.
- [ ] B2B context/quote request/released detail/version acceptance with idempotency and correct capability/ownership.
- [ ] Focused tests, complete regression, native scrcpy QA; matrix/handoff/checkpoint.

## UI / security / failures

Loading/empty/error/retry and separate 401/403; expired/429 challenge/proof; pending/disabled mutations; field errors/revision conflict; keyboard/back and long content. Safe messages preserve requestId. No internal notes/storage references/binary uploads in customer projections. Draft upload and policy workflows stay explicit and do not imply live storage capability.

## Validation / DoD

Payload contract tests and state/navigation tests for each new flow, plus analyze/full suite/Android. Test guest scope and cookie retention/expiry/logout at transport boundary. DoD only when corresponding Web route rows have working mobile destinations or precise accepted contract limitations, with automated/native evidence separated.

## Handoff / next

Status: ACTIVE. Account/address, Guest/support/session/trace and order actions implemented. Trace APK build PASS 170.1s. New full suite found offscreen Drawer test; fixed by scroll+settle, focused menu PASS. Orders/checkout 14/14 PASS before final recipient/cancel-widget refinements; subsequent cancellation widget caught setState returning Future, corrected and focused case PASS (remaining 13 mixed order/menu cases had passed). Analyze PASS before that closure correction. Full final suite/native QA still required. Next: checkpoint orders, native APK refresh and after-sales/B2B/history/discovery/reviews.
