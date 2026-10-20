# Customer Mobile parity handoff

Updated 2026-10-20. Goal ACTIVE; implementation ready for final Android QA, whole task not complete.

## Authority and Git

Only feature/mobile-ui may be modified/committed/pushed. No merge/PR. HEAD before native keyboard checkpoint b8ff216472375c23c13f4ab6a690b02c31a6273c, verified on origin. Resolve resulting checkpoint with git log. Web read-only b1ff2918c7297fc93d0cb9a15cc1d0a0ee46b72e via git show origin/feature/ui-web:<path>. ZIP snapshots/hashes: docs/03_api_specs/frontend-contracts/SOURCE.md. User chose unchanged ZIP, including no checkout voucher field.

## Resume

Read AGENTS, MASTER_PLAN, P4_CUSTOMER_QA plus relevant P2/P3 plan. Inspect branch/HEAD/status. Native keyboard checkpoint: shared AppTextField onTapOutside unfocus; flexible checkout subtotal row; focus regression assertion; reconciled parity matrix. Prior product/session/cart slice committed b8ff216. Source remains stable during APK build9810. No other branch edits.

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

Baseline60. Current keyboard patch analyze PASS and full141/141 PASS; focused checkout3/3 PASS incl focus assertion. Build9810 still running; source held stable. Prior b8ff216 analyze/full141/build35.3s PASS, installed. Native refocus fix exposed hidden27px subtotal overflow after layout moved into view; fixed without disabling checks. No temporary diagnostic test hook remains.

Native observed: recovery validation/sent/reset keyboard/expired; Guest full challenge/OTP→OM-8921 detail→tracking; support form/keyboard. At b8ff216 explicit SKU→add/cart; clear confirmation cancel preserves line; guest checkout Huế/Thuận Hòa→actual COD order85,000→payment explicitly unpaid→same detail. Local B2B login→account, profile save acknowledgement, loyalty→promotions, address list/default/edit/save→list observed. B2B verified org/request/date picker observed. Remaining native full successful reset/relogin, QR, support submit/reply, trace, returns, B2B accept, discovery/filter/reviews/history actions. Matrix records partial honestly.

Latest native issue: date picker Material labels English. Installed SDK flutter_localizations pins intl0.20.2; lock currently0.20.3. After build9810 completes, add SDK localization dependency + Vietnamese delegates/locale (required intl patch downgrade, no other upgrade), add localized picker test, validate/build again. Phone currently B2B calendar OK just clicked; observe fresh.

## Runtime and next action

SDK Flutter3.44.9/Dart3.12.2. Build process-local GRADLE_OPTS kotlin.incremental=false/compiler.execution.strategy=in-process for E/C cache split. ADB D:/AS_Sdk/platform-tools/adb.exe; device reconnected after user confirmed USB. scrcpy E:/scrcpy-win64-v4.1/scrcpy.exe; active window68230, sky/scrcpyWindow/scrcpyState in node_repl (reset after user turn, reinitialize if missing). Prior window9373924/attach97559 may be invalid. Do not print screenshot data URLs. No uninstall/wipe. Generated Linux/Windows registrant headers are line-ending-only noise, never stage.

Exact next: finish build9810 without source edits, then Vietnamese Material localization with focused test as above; native B2B date/request/quote/accept and remaining P4. Checkpoint Mobile only and verify push. Fix verified defects and reconcile final matrix/plans/handoff. Goal remains ACTIVE, do not mark complete at checkpoint.
