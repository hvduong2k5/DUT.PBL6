# Customer Mobile parity handoff

Updated 2026-10-20. Goal ACTIVE. Whole Customer parity NOT COMPLETE.

## Git and boundaries

Branch feature/mobile-ui; HEAD before catalog milestone 28c47be2373231a85e8aa4d5fa9c6e34b8364f26, pushed/verified. Resolve resulting checkpoint using git log. Prior checkpoints: 0c2e7df returns, 14d081b orders, b6f4c28 trace, 86dcef2 session, 457eea6 support, eb57d76 Guest, ec8545a payment/account, 1d91c15 purchase, 82fd358 auth. Commit/push Mobile only; no merges/PRs. Read-only Web reference b1ff2918c7297fc93d0cb9a15cc1d0a0ee46b72e using git show origin/feature/ui-web:<path>; no Web checkout/edits performed.

## Implemented

- P0: route inventory/matrix, two-level plans, exact ZIP Customer snapshots/hashes under docs/03_api_specs/frontend-contracts. Decisions in docs/DECISIONS.md. Ancestor historical Stitch review does not govern this approved direct parity task.
- P1: Web palette/type/CTA, login identifier, registration required email/terms/Unicode password rules, auth return destinations; recovery neutral request/link/proof/reset/relogin/expiry/429. Android recovery URI. Reset never auto-logs in.
- P2: explicit SKU, quantity bound 20 from Web; actual images/fallback; available cart selection/selected subtotal; selected-only strict ZIP checkout and safe cart cleanup; two-level locations and saved addresses; voucher discovery/validation; actual order confirmation and payment/QR expiry/status. Mock Checkout creates Orders; no fabricated successful lookup or paid toggle.
- P3 account/session: editable name/email, immutable phone; loyalty actual overview; address CRUD/default/delete/errors; login-return and server capability gates. Native memory-only server cookie jar, no fabricated X-Session-Id, clear on logout/reset. Auth/me restore, 401 refresh, offline retention/local logout cleanup, stale-response guards. Mock has no implicit B2B privileges.
- P3 Guest: neutral challenge/contact/OTP/resend/cooldown/expiry, exact-order navigation; REST only supplied ZIP fields/cookies. Mock denies history/other orders/direct ID/expired/consumed grants and grants only exact newly created checkout. P3 support: context/create/detail/public reply, owned-order selection, retry idempotency, no fake upload/hotline/CSAT; local tickets scope/idempotency.
- P3 trace: manual code and deep link/detail/error/retry, production/expiry/artisan/origin/certificates; mock clearly labelled specimen. Find-product preserves q in catalog route/controller. Shared HTTPS external-media button rejects executable/local/credential schemes and handles launch failure. Added Flutter-published url_launcher 6.3.3 + seven platform packages only; no existing package upgrades. Lock Flutter lower bound becomes 3.44.0, within installed 3.44.9. Official docs checked. Actual generated URL plugin registrant additions are intentional now.

## Open work and contract limits

- Catalog discovery implemented: shared typed PageResult, supported q/page/pageSize, incremental page/dedup/stale/no-progress recovery, loaded-scope category/price/OCOP/confirmed-stock facets and price sorting/filter sheet, explicit loaded/total label, Home→Catalog q/category continuity. Unknown stock labelled updating, detail CTA remains available. Price presets copied from Web; no inferred weight/product-type API fields. Catalog focused 8/8 PASS + analyze PASS; prior purchase/viewport focused 26/26 PASS.

- B2B implemented: verified org/capability gates; request MOQ/date/province/brand/invoice/sample/estimate/recent files; released quote version/items/terms/totals/payment/contact/expiry; consent reset on changed quote and exact ZIP acceptance with uncertain retry-key replay; actual paired mock order is PENDING_PAYMENT. No unsupported notes/upload/draft/internal/document fields. Mock-only verified login 0905999888 (any nonempty legacy password except sai); production uses server projection. Focused B2B 9/9 PASS + analyze PASS; native pending. Server must enforce atomic quote version acceptance because ZIP has no version field; client precheck is not an atomic guarantee.

- After-sales implemented: signed-in return entry from delivered order/account; complete local single-line/quantity/reason/details/evidence URL/bank form→actual mock case; stable REST create adapter/detail and manual case lookup; no fake refund/pickup or draft API. Production context verifies Order, then explicitly unavailable/Support because stable line IDs are absent. Mock fixture issues LOCAL-LINE IDs, enforces quantities/ownership/idempotency and does not retain bank/evidence. Focused returns/auth 12/12 PASS + analyze PASS. Native after-sales pending.

- Order-action slice implemented: server-authoritative cancellation with reason-only payload/retry key/confirmation, guest readonly; real tracking carrier/checkpoints/seal/video; raw SKU/price (no ambiguous multiplication), recipient and financial snapshot, unknown status preservation. Cancellation widget caught Future-returning setState; fixed, focused case PASS. Orders/checkout 14 PASS before final refinements; mixed order/menu other 13 cases PASS. Native action QA pending.

- User confirmed unchanged ZIP: voucher application at POST checkout waits for contract extension; never submit voucher_code or reduce final total. No shipping quote/durable payment QR recovery endpoint.
- P2 OPEN: product reviews/owned-order SKU selection, cart clear confirmation, native discovery/final product detail audit. Catalog advanced global filters remain a documented ZIP limitation, not a fake server feature.
- P3 OPEN: history search/filter/pagination/native QA and production after-sales context gap. Current Web has no reorder action and ZIP no endpoint. Do not fabricate line IDs or silently use drafts. B2B Web source and stable schemas fully read/implemented; notes/upload/document download remain absent from stable contract.
- Support ZIP forbids Web guest contact object and has no stable upload/download API; do not collect unsendable contact, pretend metadata uploaded, or invent file download URLs. Ticket list, loyalty history/redemption and notifications planned/draft. Missing native destinations must not be interpreted as complete.
- P4 full native/cross-flow QA remains OPEN; real backend/email/SMS/payment/credential lifecycle NOT VERIFIED.

## Validation and native state

- Latest full suite at 28c47be: 124/124 PASS. Native hot restart in attach3540 loaded Returns/B2B DI; scrcpy12519142 now Home→Account just clicked (observe fresh state). No new native plugin/build needed for these Dart-only changes; final APK still to rebuild after remaining work.

- Native Guest lookup/tracking now observed end to end: Home→order/contact→neutral challenge→local OTP→actual OM-8921 detail→carrier/tracking code/public checkpoints. No guest cancel action or visible overflow. Phone currently on tracking. Mock-only evidence, no real SMS/cookie backend claim.

- Latest at 14d081b: full 110/110 PASS; APK build PASS 37.6s with source held stable. Installed by adb update (Success), launched Home, no data wipe. New active attach PTY 3540, scrcpy window 12519142 after reopening existing executable. Old session/window invalid. Phone currently Guest lookup, code field focused and dummy OM-8921 just typed; refresh UI before any action. Native update includes URL plugin/current order actions.

- Baseline analyze / 60 tests PASS. Last clean full suite 97/97 PASS (support). Trace/catalog/session focused 11/11 PASS and analyze PASS. New full run 104 PASS / 1 FAIL: added drawer destinations moved login offscreen in old test; ensureVisible + pumpAndSettle corrected it, focused menu PASS and all menu cases passed in later mixed run. Order cancellation widget later found a setState Future bug; fixed, focused case PASS. Full final suite still required. Trace Android build PASS 170.1s with process-local cross-drive Kotlin flags; build began before order-action edits, so rebuild incrementally for final source.
- Prior P1 APK build PASS (321.5s), P2 incremental build PASS (57.6s), installed without wipe. Native observed recovery validation/sent/local link/reset checklist/keyboard/expired proof, Home images, list→SKU→add→cart selection→checkout two-level form. Native support form/options and title/content keyboard scrolling observed; no overflow seen. Support submission/reply, account/Guest/trace full native QA still pending.
- Device secure-storage old-cipher warning pre-existed; fallback allowed mock app. No uninstall/data wipe. Use adb devices to discover current authorized device; no hard-coded ID in source/docs.
- Previous attach PTY 35097 (r/R) and scrcpy window 4130356, process E:/scrcpy-win64-v4.1/scrcpy.exe; verify live before reuse. App still has support source before capability/trace; URL native plugin requires rebuild/install, not reload. Last phone view: support title entered dummy text, content field just focused with keyboard. Node REPL @oai/sky variables scrcpyWindow/scrcpyState; refresh before every action. UI operations local mock only.
- Never commit .kotlin/build/cache/screenshots/logs. Generated registrant headers still only line-ending noise; URL launcher cc/cmake/Swift additions should be reviewed/staged explicitly.

## Exact next action

Checkpoint Catalog and verify Mobile push. Continue history server q/status/page/pageSize using PageResult, then reviews (POST requires owned order_id + exact sku_code, no product_id) and cart clear confirmation. Web Catalog/Review sources and stable list/search/order/review schemas read. Native attach3540 runs B2B checkpoint; reload after Catalog slice. Scrcpy12519142 at Login, local B2B phone0905999888 entered, password field just clicked; refresh focus before typing dummy legacy password. Remaining native: profile/address/support/trace/returns/B2B/purchase/payment/reset. Full suite/build at stable source milestones. Goal ACTIVE.
