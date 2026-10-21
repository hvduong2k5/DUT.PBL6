# P3 — Account, ownership and Customer service flows

## Goal / dependency

Close missing Web Customer destinations and protect transitions between account, orders and customer service. P2 core implementation/main-flow local QA is recorded; targeted payment QA-01 remains open. Preserve completed contracts. Current source e86f82b passed144tests/analyze/debug APK; no P3 source change is needed for this reconciliation.

## Scope / prior decisions

Keep the supplied ZIP unchanged, including stable/draft/planned distinctions. User confirmed voucher submission waits for contract extension. Match Customer Web presentation; do not copy its unsupported legacy request fields or placeholder contact numbers. Changes stay on Mobile branch only.

## Current evidence

- Web profile supports name/email changes; phone is read-only. ZIP GET/PUT `/profile` uses existing Customer projection and `{full_name,email}` only.
- Loyalty stable overview has customer_id/current_points/tier/points_to_vnd_rate; history/redemption are not stable.
- ZIP address omits district; current Mobile account model/form/payload use province/ward and saved-address continuity.
- Web guest lookup uses order/contact → neutral challenge → OTP → exact-order access. Contract relies on server-issued guest_session/guest_order_grant cookies, not a client-invented order grant. Native memory-only scoped cookie adapter is implemented/tested; real server cookie lifecycle remains unverified.
- Catalog advanced filters and legacy category_id/page_size query parameters in Web are not described in ZIP's parameter list. Do not invent server filter guarantees; track contract limits and use supported pageSize/q.

## Steps / expected areas

- [x] Extend existing transport with native memory-only server-cookie handling (no hand-written cookie parser); retain bearer tokens and a Flutter-web-safe conditional adapter. Standard Dio cookie packages were added for this transport boundary; do not persist guest proofs to files or fabricate cookies. Clear native cookies on logout/reset.
- [x] Profile/loyalty domain, REST/mock, Cubits and pages; safe login return and server errors. Update account navigation and current-user display after saved profile.
- [x] Two-level account address form/payload/default/delete behavior and saved-address checkout continuity.
- [x] Current-session restore/refresh and capability projection/account guards. Network failure keeps tokens; invalid session clears; stale identity/capability responses cannot regrant access. B2B route-specific actor/capability/verification guards implemented and tested.
- [x] Guest lookup challenge/expiry/resend/OTP/grant and exact-order navigation; preserve only server-issued authority, never direct-ID authorization. Mock ownership denies history/other orders/expired grants. Native local challenge/OTP→detail/tracking observed; real-cookie integration remains unverified.
- [x] Order read/cancel/tracking: exact SKU/raw price and server totals, recipient snapshot, unknown status, carrier/seal/video/checkpoints, retry/confirmation/server-authoritative cancel. Guest remains read-only. Web reference has no reorder action/ZIP endpoint; no invented reorder API.
- [x] Order history supported server search/status/page/pageSize, count/load-more/retry/stale guards, empty-query recovery and precise raw payment/packing labels. No uncontracted year query. Native history/filter/empty recovery and cancellation-return refresh observed.
- [x] Support context/create/detail/public messages. No fake upload/contact extension/hotline/CSAT. Automated create→ticket→reply, idempotency and ownership checks PASS; native create→ticket→public reply/composer reset observed.
- [x] Trace manual-code entry/detail/error/retry, deep link, find-product search, HTTPS-only external media action. Local certificate fixture explicitly labelled; native trace input/detail/search observed; OS external-media launch not verified (QA-02).
- [x] After-sales local form/create→case and stable REST detail/create adapter. Production form stops at documented missing line-ID context and offers Support. No implicit draft calls, fake upload, refund completion or retained bank details. Backend creation remains an open contract gap.
- [x] B2B context/request/released quote/acceptance with server actor/capability/VERIFIED guards, MOQ/date/province/branding/invoice/sample, estimates/recent requests, complete public terms/totals/payment/contact, expiry/changed-content consent reset, uncertain-POST key replay and unpaid order reference. Backend atomic version enforcement remains a server responsibility under unchanged ZIP.
- [x] Focused tests, full regression/analyze/debug APK, main-flow local Android QA; matrix/handoff/Mobile checkpoint.
- [ ] QA-02: actual OS external-media handoff is not verified. URI safety is tested; do not claim video playback or provider availability.

## UI / security / failures

Loading/empty/error/retry and separate 401/403; expired/429 challenge/proof; pending/disabled mutations; field errors/revision conflict; keyboard/back and long content. Safe messages preserve requestId. No internal notes/storage references/binary uploads in customer projections. Draft upload and policy workflows stay explicit and do not imply live storage capability.

## Validation / DoD

Payload contract tests and state/navigation tests for each new flow, plus analyze/full suite/Android. Test guest scope and cookie retention/expiry/logout at transport boundary. DoD only when corresponding Web route rows have working mobile destinations or precise accepted contract limitations, with automated/native evidence separated.

## Acceptance reconciliation — 2026-10-21

Status: IMPLEMENTED / VERIFIED LOCAL MAIN FLOWS with explicit contract/integration limits; QA-02 external-media handoff remains unverified.

- Web13page group has routed counterparts: profile/addresses/loyalty/orders list/detail, Guest track, after-sales request/case, B2B request/quote, Support request/ticket, trace detail. Account/manual trace/case lookup and tracking are additional Mobile entrypoints/adaptations, not absent Web routes.
- Tests: customer_account_test.dart/profile_navigation_test.dart/addresses_test.dart; session_capabilities_test.dart/mid_session_recovery_test.dart/session_cookies_test.dart; guest_orders_test.dart/guest_navigation_test.dart; order_actions_test.dart/orders_pagination_test.dart; support_test.dart/support_navigation_test.dart; trace_test.dart; returns_test.dart/returns_navigation_test.dart; b2b_test.dart/b2b_navigation_test.dart. Included in current144/144 suite, analyze/APK PASS.
- P4 records physical account/profile/addresses/loyalty, Guest ownership/tracking, order history/cancel, Support creation/reply, trace, local return form/case and verified B2B request/acceptance/unpaid reference. Cancellation native regression on current installed APK confirmed pending filter empties on return and all-list badge is cancelled.
- Trace tests check allowed HTTPS URLs and reject executable/local/credential URLs. PublicHttpsButton calls launchUrl in externalApplication mode and handles failure. This does not prove Android media handoff or playback. Keep QA-02 explicitly unverified; no blanket device rerun required.
- Returns REST context intentionally blocks production creation because stable Order has no line_id projection; Support recovery remains available. Mock owns its fixture line IDs; never infer real IDs from SKU/index. Stable create/detail adapter exists, does not prove production eligibility/refund. Bank/evidence inputs are not logged/persisted.
- B2B preflight refresh/fingerprint/consent reset is a client check; unchanged ZIP confirmation/idempotencyKey body cannot enforce atomic quote-version acceptance. Server must enforce it. Acceptance is not payment; no extra version fields/headers/doc uploads.
- No live backend/SMS/cookie/payment/provider integration verified; draft loyalty history/redemption, uploads, notifications, Support inbox and B2B documents remain unavailable. Native notification entry remains local-only.

Next P3/P4 action: track QA-02 separately from implementation and external contract limits; prefer one targeted media-handoff check only if testable fixture/provider exists. Do not repeat completed service flows or turn missing external evidence into fabricated success.
