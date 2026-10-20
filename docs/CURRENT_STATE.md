# Customer Mobile parity handoff

Updated 2026-10-20. Goal ACTIVE; implementation ready for final Android QA, whole task not complete.

## Authority and Git

Only feature/mobile-ui may be modified/committed/pushed. No merge/PR. Current committed HEAD 5bf0b3093e89f24522086e9b3a78715a2d2d8886 (reviews), verified on origin. Web read-only b1ff2918c7297fc93d0cb9a15cc1d0a0ee46b72e via git show origin/feature/ui-web:<path>. Supplied ZIP snapshots/hashes: docs/03_api_specs/frontend-contracts/SOURCE.md. User chose unchanged ZIP, including no checkout voucher field.

## Resume

Read AGENTS, MASTER_PLAN, P4_CUSTOMER_QA plus relevant P2/P3 plan. Inspect branch/HEAD/status. Current uncommitted slice: confirmed cart clear, mid-session recovery, friendly logistics labels, local order SKU preservation, product gallery/certificate/weight/original price and explicit SKU selection, corresponding tests/docs. Source remains stable during validation/build. No other branch edits.

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

Baseline 60 tests. Final current source analyze PASS, full 141/141 PASS, APK build PASS 35.3s (source held stable). Earlier reviews/cart/session 139/139 PASS + build 56.4s; product catalog/reviews/viewport 32/32 PASS. Projection assertion was corrected to compare Money BigInt; final full suite includes it. Native final-source QA still pending.

Native previously observed recovery validation/sent/reset keyboard/expired, product add/cart/checkout form, support keyboard, Guest full challenge/OTP→OM-8921 detail→tracking. Remaining native: successful reset/relogin, full purchase/payment, account/address/loyalty, support submit/reply, trace, returns, B2B, discovery/reviews. Latest phone runs old B2B source; current scrcpy12519142 at Login with local fixture0905999888 and dummy legacy. Refresh UI before every input. Native screenshots not committed.

## Runtime and next action

SDK Flutter3.44.9/Dart3.12.2. APK build process-local GRADLE_OPTS kotlin.incremental=false/compiler.execution.strategy=in-process for E/C cache split; no persisted build changes. ADB path D:/AS_Sdk/platform-tools/adb.exe; discover device. scrcpy E:/scrcpy-win64-v4.1/scrcpy.exe; sky in node_repl, scrcpyWindow/scrcpyState; do not print screenshot data URLs. Attach3540 may disconnect after install; verify before reuse. No uninstall/wipe. Generated Linux/Windows registrant headers are line-ending-only tool noise, never stage.

Exact next: checkpoint intentional Mobile source/tests/docs and verify push, install new APK, perform native QA through scrcpy. Fix verified defects, rerun relevant checks, reconcile parity matrix/P1-P4/handoff and deliver only after acceptance work is handled or precise contract gaps documented. Goal remains ACTIVE.
