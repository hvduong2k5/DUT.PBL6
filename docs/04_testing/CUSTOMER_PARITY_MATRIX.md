# Customer Web / Flutter parity matrix

Updated 2026-10-21. Reference origin/feature/ui-web at b1ff2918 (read-only Git objects). Mobile feature/mobile-ui, starting 8e1994a7. Source paths below are relative to apps/web-user/src. Implementation and native validation are separate. Goal remains ACTIVE while targeted QA and final delivery remain open. Route coverage is distinct from state/visual/provider verification.

| Web reference / group | Flutter counterpart and current implementation | Validation / remaining limit |
|---|---|---|
| app/globals.css; shared layout/auth controls | Web cream/brown/gold/ink, serif headings/sans body, shared CTA and states | Analyze/viewport tests; native core screens observed |
| app/login/login-form.tsx; app/register/registration-flow.tsx | Identifier login, required ZIP email, terms/password checklist, return destination | Automated auth/navigation PASS; native local login→account observed |
| app/account-recovery/recovery-flow.tsx | Neutral request, verified link/restricted proof/reset/success/relogin; expiry/429 | Automated full reset/no auto-login/large-text keyboard PASS; native validation/sent/reset keyboard/expired observed; native successful local reset/relogin observed |
| app/home-experience.tsx; app/products/catalog-flow.tsx | Home search/category continuity; server q/page/pageSize, load more; loaded-scope facets/price sort and filter sheet | Focused pagination/filter/empty/stale/no-progress PASS; native Home, exact search→empty→clear and under200k filter returning3 observed. No unsupported global weight/type/filter queries |
| app/products/[slug]/product-detail-flow.tsx | Explicit SKU including one-option products, quantity bound20, exact price/stock, gallery/certificate/weight/original price, story, unavailable food data + manual trace | REST projection/explicit SKU and viewport PASS; native disabled→selected→add/cart PASS. No inferred ingredients/allergens/batch dates |
| components/product/product-reviews.tsx | Public aggregate/list/paging/replies/media; owned delivered-order and exact snapshot SKU selection; strict ZIP create, verified badge only from response | Focused 4/4 PASS including phone submission/list. Native full owned-order/SKU→comment→submit→aggregate/verified badge observed; keyboard viewport fix native and regression PASS; no invented eligibility/upload API or product_id POST |
| app/cart/cart-flow.tsx | Available-line selection/all/partial/selected subtotal, pending/error, clear confirmation, quantity bounds | Unit/widgets PASS; native selection/add/cart and clear-dialog cancel preserve lines observed |
| app/promotions/promotions-flow.tsx | Voucher list/condition check, stale validation protection, honest checkout limitation | Contract/state tests PASS; native wallet/conditions view observed. Applying voucher to Checkout waits for contract extension per user |
| app/checkout/checkout-flow.tsx; services/checkout-service.ts | Selected-only checkout, province→ward, saved address, exact ZIP body/header/retry key, safe selected-only cleanup | Payload/state/phone transaction PASS; native guest two-level address→actual COD order85,000 PASS. Saved-address autofill and VietQR pending order85,000 also observed. Keyboard/subtotal fixes validated; see P4 |
| app/payment/[orderId]/payment-flow.tsx; confirmation | Actual order/result, pending QR/expiry/read status, COD explicitly unpaid; no fake paid toggle | Automated expired/invalid context PASS; native created COD→payment→same order detail observed. Native VietQR order OM-82047142/countdown/explicit missing QR observed; Check status result needs fresh observation. Mock supplies no QR URL; real QR/payment not verified; no payment reissue endpoint |
| app/account/profile/profile-flow.tsx | Editable name/email, immutable phone, conflict/reload and return login | Contract/state/widget PASS; native profile/save acknowledgement observed |
| app/account/addresses/address-book-flow.tsx | Two-level CRUD/default/delete confirmation, checkout continuity | Automated payload/mutation PASS; native list/default/edit/save→list observed |
| app/account/loyalty/loyalty-flow.tsx | Actual overview/tier/points/conversion, no fake history/redemption | Automated PASS; native overview→promotions observed. Transactions/redemption remain draft/planned |
| app/account/orders/orders-flow.tsx; detail | Server q/status/page/pageSize, load more/empty recovery, exact raw status/price/totals/SKU, actual timeline, server cancellation | Focused orders15/15 PASS; native newly created order detail + Guest tracking observed. Native history status-empty→clear and delivered detail observed; native empty reason/keyboard/confirm→cancelled/action hidden observed. Stale history on return fixed; installed APK pending filter→empty→clear→cancelled badge verified. No unsupported year/reorder query/API |
| app/track-order/page.tsx | Neutral challenge/contact→OTP/resend/expiry→exact scoped order; no unrestricted ID grant | Automated ownership/replay/expiry PASS; native full OM-8921 challenge/OTP→detail→tracking observed. Real SMS/server cookies not verified |
| after-sales request/case flows | Full local form/create/case; stable REST create/detail and manual existing-case lookup | Focused returns/auth12/12 PASS incl phone. Production create context cannot obtain stable line_id: explicit unavailable + Support. Native exact-line form/keyboard/create→LOCAL-RET REQUESTED/refund0 observed with fake bank fixtures |
| support request/ticket flows | Typed context/create/detail/public reply, owned-order choice, idempotent retry | Automated create→reply/ownership PASS; native owned-order creation→actual local ticket→public reply/composer cleared observed. No guest contact object/upload/inbox invented |
| app/trace/[qrCode]/page.tsx | Manual entry/deep link/detail, actual fields and safe public HTTPS media, find-product search | Focused trace/media tests PASS; native entry/validation/labelled sample/detail/disclaimer→exact search→empty/clear observed. Local specimens labelled, no fake certificate association |
| app/b2b/b2b-request-flow.tsx; released quote | Verified actor/org/capability guards, request MOQ/date/province/brand/invoice/sample/estimate, quote terms/version/expiry/consent/acceptance→unpaid order | Focused9/9 PASS incl phone. Native request→receipt and released quote→consent→accept→unpaid reference order observed. No unsupported notes/upload/docs; server atomic version acceptance still required by unchanged ZIP |
| No matching Web notification route | Existing local notifications preserved | Native feature only; ZIP notifications planned, not live integration |
| Session/capabilities/cookies across groups | Startup identity/401 refresh/offline retention; protected401 recovery invalidates stale actor; explicit retry without write replay; native server-cookie jar scoped and memory-only | Auth focused21/21 PASS, cookie/session contract tests PASS. Real backend lifecycle not verified |

## API authority and differences

Exact supplied ZIP snapshots/hashes: ../03_api_specs/frontend-contracts/SOURCE.md. Stable/draft/planned distinctions preserved. ZIP requires login identifier, registration email and two-level shipping without district. Review POST uses order_id/sku_code/rating/comment/media_urls. Checkout forbids voucher_code, item prices and unrelated fields. No invented shipping quote, payment reissue, uploads, after-sales line IDs, ticket inbox, loyalty transactions/redemption, notifications, advanced product filters or B2B document fields.

## Evidence and next step

Cancellation slice2026-10-21: analyze PASS110.5s, full144/144 PASS, focused order actions/history10/10 PASS, debug APK build166.0s PASS. New regressions reproduce stale history before fix and pass after navigation-return reload; query/filter preserved. APK installed update Success, no uninstall/wipe. Physical original OM-82047142 cancel observed; after mock process reset, replacement LOCAL-B2B-001 pending-filter→cancel→back0/0empty→clear→cancelled badge verified. No real backend/payment/provider proof.

## P1–P3 acceptance review — 2026-10-21

Reviewed source/checkpoint e86f82ba834e3f148f3be7523b1f41ddda483685, clean worktree and matching origin before documentation changes. No browser/computer-use calls, app source edits, dependency changes or new test/build run in this audit. The144/144/analyze/APK results above belong to that unchanged source and recorded checkpoint, not a fresh execution here. Existing full-suite log still ends144 passed.

Read-only Web page inventory has24Customer pages. Every group has Mobile routes wired in app_router.dart/AppRoutes:

| Phase | Web page coverage | Evidence reviewed | Engineering disposition |
|---|---|---|---|
| P1 | 3: login/register/recovery | Shared theme/CTA vs Web globals.css; auth REST; recovery/session/cart-continuity tests; recorded recovery/login native | VERIFIED LOCAL; real email/password/session integration unverified |
| P2 | 8: Home/products list/detail/cart/checkout/confirmation/payment/promotions | Strict checkout adapter, selected cart/paging/product/review/voucher tests, recorded purchase/review/filter native | IMPLEMENTED, main flows locally verified; QA-01 OPEN; voucher checkout waits for contract |
| P3 | 13: profile/addresses/loyalty/orders list/detail/Guest track/returns request/case/B2B request/quote/Support request/ticket/trace | Router, returns blocking context, media launcher, service/session/ownership tests and P4 native records | IMPLEMENTED, main flows locally verified; QA-02 UNVERIFIED; production return context and provider limits explicit |

This page count proves coverage of destinations, not pixel equality or every action/state. Mobile-only Account, notifications, manual trace/case lookup and separate tracking adapt navigation; reviews live inside Product. No missing destination found in the24page mapping.

Contract integrity: original ZIP SHA-256 and all3imported reference hashes recomputed and match SOURCE.md. Snapshot contents unchanged. Stable checkout source excludes voucher_code/district/item prices. Returns stable context lacks line_id and blocks instead of calling draft eligibility. Shared primary#563a26/surface#fcf9f4/gold#d4a24c/ink/muted/border/error match Web tokens; typography uses the existing serif/sans adaptation. No new full-screen visual equality claim.

### Remaining targeted evidence and next step

| ID | Precise gap | Available evidence | Smallest useful next action |
|---|---|---|---|
| QA-01 | Payment Check status result, loading, failure/retry and returned paid/cancelled transitions have no direct widget/native interaction proof | PaymentPage._check reads OrdersRepository; only PaymentPage widget test checks expired QR/no fake paid; previous native click outcome unconfirmed | Add deterministic widget cases for pending/paid/cancelled/error→retry and duplicate pending taps; change source only for a reproduced bug. This should precede any further device loop |
| QA-02 | Trace external-media OS handoff/playback unverified | URL policy tests plus launcher/error handling source; native trace detail/search completed | If a valid approved media fixture exists, one targeted Android handoff observation; otherwise retain NOT VERIFIED. Do not infer playback or acquire external media to manufacture evidence |

External/integration limits stay separate: actual voucher checkout field needs contract extension (user accepted waiting); stable return line-ID/eligibility context missing; atomic B2B acceptance remains server responsibility; live email/SMS/cookies/payment/provider and actual QR not verified. Draft/planned uploads/inbox/loyalty-redemption/notifications/B2B documents are not silently promoted. No source defect was confirmed by this review; QA-01 is missing evidence, not a claim that status checking is broken.

P1–P3 plan reconciliation is COMPLETE for this audit. P4/whole goal remains ACTIVE, with QA-01 as exact next task. No general native rerun or complete feature rewrite is required.
