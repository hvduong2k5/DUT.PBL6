# Customer Mobile parity handoff

Updated 2026-10-20. Goal ACTIVE; implementation ready for final Android QA, whole task not complete.

## Authority and Git

Only feature/mobile-ui may be modified/committed/pushed. No merge/PR. Current implementation HEAD718159eddb1d0132018b62e4e3e8fea9524783fd, verified on origin. Later docs checkpoint identified by git log. Web read-only b1ff2918c7297fc93d0cb9a15cc1d0a0ee46b72e via git show origin/feature/ui-web:<path>. ZIP snapshots/hashes: docs/03_api_specs/frontend-contracts/SOURCE.md. User chose unchanged ZIP, including no checkout voucher field.

## Resume

Read AGENTS, MASTER_PLAN, P4_CUSTOMER_QA plus relevant P2/P3 plan. Inspect branch/HEAD/status. f5b9acf: shared input unfocus/flexible checkout subtotal/focus regression/matrix. Current milestone: Vietnamese Material/Cupertino/Widgets SDK localization, required intl0.20.3→0.20.2 SDK pin, calendar action focus prevents date keyboard restoration. No other branch edits.

## Implemented

- Shared Web palette/type/CTA; login/register/return destination and required ZIP email; recovery neutral email request/link/proof/reset/relogin/expiry/429, never auto-login.
- Explicit SKU/quantity, product availability/images/fallback; catalog supported server paging/search with loaded-scope category/price/OCOP/stock/sort/filter sheet; cart available selection/subtotal, selected-only strict two-level checkout and safe cleanup; promotions list/validate; actual confirmation/payment/QR/expiry/status.
- Profile/name/email, immutable phone, loyalty overview, address CRUD/default/confirmation; current-session restore/refresh/capabilities and native scoped memory-only server cookies.
- Guest challenge/OTP/exact-order grant; actual orders/history server search/status/paging/cancel/tracking/seal/media, precise status and no fabricated future timeline.
- Support context/create/detail/public reply/owned order, manual trace/deep link/detail and safe HTTPS media links.
- Returns full local form/create/case and stable REST adapter/detail; production context explicitly stops at missing stable line IDs and offers Support.
- Verified B2B request/context/quote/public terms/version/consent/acceptance with expiry/changed quote/idempotent uncertain replay; actual unpaid paired local order.
- Reviews public aggregates/list/paging, owned delivered-order and exact SKU selection, strict ZIP POST, server verified badge, retry/idempotency/capability; explicit local OM-8860 canonical delivered purchase fixture.
- New session slice revalidates identity after protected 401; no automatic mutation replay; invalid session removes stale actor, offline retains tokens. Auth endpoints excluded, concurrent failures share recovery.

## Constraints / limits

Never invent APIs, line IDs or proof of live provider behavior. Checkout voucher application waits for extension; validation never reduces final total. No stable return eligibility/line-ID projection, uploads, ticket inbox, loyalty history/redemption, notifications, payment reissue or B2B documents. Global advanced product filters absent from ZIP; local facets labelled loaded scope. No food fields in stable product; show unavailable and manual trace, no fake batch association. Server must enforce atomic quote version acceptance (ZIP has no version field). No actual backend/email/SMS/payment/cookie/provider lifecycle verified. Mock fixtures are local only.

## Evidence

Baseline60. Current localization milestone analyze PASS, full142/142 PASS and APK build69.6s PASS with stable source. B2B focused3/3 PASS includes localized calendar and focus return; first test attempts exposed real focus restoration and were fixed with persistent calendar action FocusNode, no skipped assertions. f5b9acf analyze/full141/build172.1s PASS; prior b8ff216 build35.3s installed. Native refocus fix exposed hidden27px subtotal overflow; fixed without disabling checks. No diagnostic hooks remain.

Native observed: recovery validation/sent/reset keyboard/expired; Guest full challenge/OTP→OM-8921 detail→tracking; support form/keyboard. At b8ff216 explicit SKU→add/cart; clear confirmation cancel preserves line; guest checkout Huế/Thuận Hòa→actual COD order85,000→payment explicitly unpaid→same detail. Local B2B login→account, profile save acknowledgement, loyalty→promotions, address list/default/edit/save→list observed. B2B verified org/request/date picker observed. Remaining native QR, support submit/reply, trace, returns, discovery/filter/reviews/history actions. Successful local recovery/reset/relogin and B2B acceptance now observed below. Matrix records partial honestly.

Native B2B now observed end-to-end: date/MOQ20/request→YC-LOCAL-2 receipt; existing released BG-LOCAL-001 public terms/totals, disabled accept before consent→consent→accept→receipt→LOCAL-B2B-001 detail PENDING_PAYMENT, total4,180,000 equals quote. No real transaction. New APK718159e installed update Success, no wipe. Native recovery custom URI/mock-token→valid reset→all checklist/confirmation→local success requiring login→relogin→account observed. Mock accepts nonempty passwords, so this proves UI flow only, not backend credential verification. Vietnamese calendar labels/month/day/Huỷ observed, OK returns selected date with no keyboard. Phone currently B2B form, selected2026-10-27. Attach26739 waiting/connecting; verify before reuse.

## Runtime and next action

SDK Flutter3.44.9/Dart3.12.2. Build process-local GRADLE_OPTS kotlin.incremental=false/compiler.execution.strategy=in-process for E/C cache split. ADB D:/AS_Sdk/platform-tools/adb.exe; device reconnected after user confirmed USB. scrcpy E:/scrcpy-win64-v4.1/scrcpy.exe; active window68230, sky/scrcpyWindow/scrcpyState in node_repl (reset after user turn, reinitialize if missing). Prior window9373924/attach97559 may be invalid. Do not print screenshot data URLs. No uninstall/wipe. Generated Linux/Windows registrant headers are line-ending-only noise, never stage.

Exact next: continue native P4: reviews, support submit/reply, trace, returns, discovery/filter/history actions and QR. New APK installed, recovery/relogin/calendar/B2B native completed above; do not repeat. Current scrcpy68230 B2B form; back→account, Home/product for reviews with logged-in local verified fixture0905999888. No new source edits since718159e. Reconcile final plans/matrix/handoff. Goal remains ACTIVE, do not mark complete at checkpoint.

