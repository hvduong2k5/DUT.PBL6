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
- [x] Order history supported server search/status/page/pageSize, count/load-more/retry/stale guards, empty-query recovery and precise raw payment/packing labels. No uncontracted year query. Native history QA pending.
- [x] Support context/create/detail/public messages. No fake upload/contact extension/hotline/CSAT. Automated create→ticket→reply, idempotency and ownership checks PASS; native form/keyboard inspected, native submission pending.
- [x] Trace manual-code entry/detail/error/retry, deep link, find-product search, HTTPS-only external media action. Local certificate fixture explicitly labelled; video launch/native build QA pending.
- [x] After-sales local form/create→case and stable REST detail/create adapter. Production form stops at documented missing line-ID context and offers Support. No implicit draft calls, fake upload, refund completion or retained bank details. Backend creation remains an open contract gap.
- [x] B2B context/request/released quote/acceptance with server actor/capability/VERIFIED guards, MOQ/date/province/branding/invoice/sample, estimates/recent requests, complete public terms/totals/payment/contact, expiry/changed-content consent reset, uncertain-POST key replay and unpaid order reference. Backend atomic version enforcement remains a server responsibility under unchanged ZIP.
- [ ] Focused tests, complete regression, native scrcpy QA; matrix/handoff/checkpoint.

## UI / security / failures

Loading/empty/error/retry and separate 401/403; expired/429 challenge/proof; pending/disabled mutations; field errors/revision conflict; keyboard/back and long content. Safe messages preserve requestId. No internal notes/storage references/binary uploads in customer projections. Draft upload and policy workflows stay explicit and do not imply live storage capability.

## Validation / DoD

Payload contract tests and state/navigation tests for each new flow, plus analyze/full suite/Android. Test guest scope and cookie retention/expiry/logout at transport boundary. DoD only when corresponding Web route rows have working mobile destinations or precise accepted contract limitations, with automated/native evidence separated.

## Handoff / next

Status: ACTIVE. Account/address, Guest/support/session/trace and order actions implemented. Trace APK build PASS 170.1s. New full suite found offscreen Drawer test; fixed by scroll+settle, focused menu PASS. Orders/checkout 14/14 PASS before final recipient/cancel-widget refinements; subsequent cancellation widget caught setState returning Future, corrected and focused case PASS (remaining 13 mixed order/menu cases had passed). Analyze PASS before that closure correction. Full final suite/native QA still required. Next: checkpoint orders, native APK refresh and after-sales/B2B/history/discovery/reviews.

2026-10-20 verification at 14d081b: full 110/110 PASS; APK build PASS 37.6s. Source was held stable during this build. Native update/QA next; previous scrcpy window closed, rediscover device/window instead of reusing handles.

## Current task — after-sales counterpart

- Keep stable create/detail payloads in a dedicated ReturnsRepository, separate from UI/bank/evidence inputs. Request follows Web single-line/quantity/reason/details/evidence URLs/refund-bank form; no upload service or resolution types absent from ZIP.
- Production context must explicitly explain missing stable line_id/options and offer Support; do not send a draft eligibility request or synthesize line IDs from SKU/index. Local mock may issue its own clearly labelled fixture line IDs and exercise full form→created case, including ownership/idempotency/quantity.
- Stable detail shows only returned case/order/status/reason/amount/date. No fabricated pickup/messages/refund completion. Approved is not refunded. Manual case-ID entry/deep link allows existing cases without a draft list API.
- Validate bank number/holder and HTTPS evidence URLs without persisting/logging them. UI keyboard/scroll/pending/errors/back and unknown status are required. Add REST payload, ownership/retry and navigation checks; current source files under features/returns, app wiring and order/account entrypoints only.

After-sales implementation checkpoint: analyze PASS; focused returns/auth 12/12 PASS (including narrow-phone form→created case, no implied refund, exact REST payload, no draft call, ownership, quantity/idempotency and retry-key changes). Native after-sales NOT RUN. Next: B2B context/quote flow; both Web source files and request/accept schemas have been read, but quote detail schema and remaining markup still need inspection. ZIP accept body has confirmation/idempotencyKey only, no quoteVersion; do not invent version fields/headers or claim a client-checked atomic acceptance guarantee.

## Current task — verified B2B workspace

Web request and quote UI fully read; stable context/create/detail/accept schemas read. Add typed B2B repository/Cubits and mobile sections for verified organization, purpose/date/province, MOQ quantities, brand options, invoice/sample, estimate/recent requests; quote released version/items/terms/totals/payment/contact/expiry and confirmation→actual order reference. No internal draft notes or unsupported request notes/file upload. Use actor B2B and feature-specific capabilities plus server organization verification; server always owns authorization. Local fixture must be explicit at mock adapter only, never infer production rights from phone/name.

Acceptance body remains exact ZIP confirmation/idempotencyKey. Re-fetch public quote before accepting and require same version/terms/financial content; if changed, clear agreement and require reread. This client precheck cannot replace atomic version enforcement on server; document that limitation and never add uncontracted quoteVersion/If-Match. Expired/not-acceptable/403/409 remain recoverable; acceptance is not payment and must not auto-mark order paid. Tests: guards, strict payloads, pending/retry, expired/changed version, duplicate acceptance, navigation. Next action: implement this slice without source changes outside mobile/auth fixture wiring/docs.

B2B checkpoint: focused 9/9 PASS, including narrow-phone request→quote→accept, unverified gate, exact request/accept bodies, quantity/idempotency, changed version, uncertain replay, capability revoked during precheck and expiry. Analyze PASS. Mock B2B login fixture 0905999888 with any nonempty legacy password except sai; no real account. Native B2B NOT RUN. Next: history/discovery pagination/search/filter and product reviews/cart clear, then full/native QA.

History checkpoint: server q/status/page/pageSize, total/page_size, incremental recovery and stale guards, precise paid/packing labels; no Web-only year query. Removed synthetic future/confirmation timeline inferred from payment; actual creation/recorded events and separate tracking remain. Focused orders 15/15 PASS and analyze PASS. Native/history/full final QA pending; next reviews + cart clear then P4.
