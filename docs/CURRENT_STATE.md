# Customer Mobile parity handoff

Updated 2026-10-21 after one bounded cancellation step. Goal ACTIVE; implementation and local QA progressed, whole task not complete.

## Scope, branch and checkpoint

- Work and push only `feature/mobile-ui`; no merge, PR or changes to other branches.
- Resume HEAD: `d6acb9addcefece29271b4c69de523c567f8a7e7`, verified on origin before this step. New cancellation source/test/document checkpoint is identifiable with `git log -1`; this file records its parent rather than a self-referencing commit hash.
- Customer Web reference: read-only `origin/feature/ui-web` at `b1ff2918c7297fc93d0cb9a15cc1d0a0ee46b72e`, read through Git objects.
- API ZIP authority/hashes: `docs/03_api_specs/frontend-contracts/SOURCE.md`. User chose unchanged ZIP: voucher validation is implemented; applying a voucher at checkout waits for contract extension.
- Current phase: `docs/plans/active/P4_CUSTOMER_QA.md`. Roadmap: `docs/plans/MASTER_PLAN.md`; coverage: `docs/04_testing/CUSTOMER_PARITY_MATRIX.md`.

## Completed implementation

Shared Web colors/type/CTA and phone layouts; login/register/recovery/link/reset/relogin; session restore/refresh/capabilities and protected-401 identity recovery without automatic mutation replay; catalog search/pagination/loaded-scope filters; explicit SKU/gallery/product data; selectable cart/clear confirmation; strict selected-only checkout/saved address/promotions; actual order confirmation/payment status; account/profile/addresses/loyalty; history/cancel/tracking/Guest OTP; Support creation/public replies; manual trace/deep links; local returns/create/case with stable REST adapter; verified B2B request/released quote/consent/acceptance; public reviews and owned delivered-order/SKU submission.

Native QA found and fixed keyboard refocus, checkout subtotal overflow, Vietnamese calendar labels/date-action focus and review comment viewport/caret visibility. 2026-10-21 source slice: orders_page.dart awaits detail navigation and reloads history from repository while mounted, preserving current query/filter. order_actions_test.dart adds two cancellation-return regressions. No API, design tokens or dependencies changed.

## Latest verification

Current cancellation source: `flutter analyze --no-pub` PASS (110.5s); full `flutter test --no-pub` **144/144 PASS**; debug APK build **PASS (166.0s)**. Focused order actions/history **10/10 PASS**. New regressions fail against old navigation (stale badge and stale pending-filter result), pass after fix, preserve query/filter and reopen cancelled detail. Initial test harness used generic pageBack incompatible with custom header; corrected to actual Quay lại button before proving regression. Earlier review keyboard regression remains in full suite.

Current cancellation APK installed with update Success, no uninstall/data wipe. Process-local fixtures reset on relaunch. Used released mock quote only to prepare replacement unpaid LOCAL-B2B-001 for cancellation QA; no real transaction. Native: pending filter1order→detail/reason/confirm→Đã hủy/action hidden→back keeps pending filter and shows0/0 empty→clear restores5cards with replacement marked Đã hủy. This verifies current installed APK, not a clean-install test. Phone ends on history, all filter selected.

All following native evidence uses local mock fixtures; it does not prove real backend/email/SMS/payment/provider behavior:

- Recovery neutral request/validation/expiry, valid mock link/reset/checklist/keyboard/success requiring login, then relogin/account. Mock accepts nonempty passwords, so real credential validation/revocation is unverified.
- Guest challenge/OTP→OM-8921 detail→tracking; explicit SKU→cart, clear-dialog cancel preserves line, guest Huế/Thuận Hòa checkout→COD85,000→unpaid payment→same order detail.
- Profile save acknowledgement, loyalty→promotions, address list/default/edit/save. Vietnamese calendar selection returns without reopening keyboard.
- B2B MOQ20/date2026-10-27→YC-LOCAL-2 receipt; released BG-LOCAL-001 consent→accept→LOCAL-B2B-001 PENDING_PAYMENT4,180,000, matching quote.
- Review owned OM-8860/exact Hộp500g→comment→submit→public5/5 aggregate/count1/verified badge; hot-reloaded keyboard fix automatically reveals caret without manual scroll.
- Trace empty validation→labelled local specimen→batch/artisan/origin/disclaimer→find-product preserves query→empty→clear restores4; price filter under200k returns3.
- History pending-payment filter empty→clear restores4; delivered OM-8874→return exact line/max2/reference110k→quality details/example HTTPS evidence/fake bank fixtures→LOCAL-RET REQUESTED, refund0 and no confirmed transfer.
- Support ORDER_HELP + owned OM-8874→local ticket→public reply visible and composer cleared; no real CSKH transmission/upload.
- Logged-in checkout saved-address fills Huế/Thuận Hòa, VietQR order **OM-82047142 /85,000 /PENDING_PAYMENT**. Payment shows countdown and explicit missing QR because mock supplies no QR URL. Historical Check status click has no newly verified result. OM-82047142 was cancelled during this step before APK replacement reset the mock process. No actual QR/payment verified.

## Constraints and unresolved work

No fabricated endpoints, line IDs, food/certificate association, QR or payment success. Stable contract lacks checkout voucher field, production return eligibility/line-ID projection, uploads, Support inbox, loyalty history/redemption, notifications, payment reissue and B2B documents. Global advanced catalog filters remain loaded-scope only. Server must enforce atomic quote-version acceptance; ZIP has no request version field. These external limits remain explicit, not silently completed.

Native local order cancellation now observed: OM-82047142 empty reason blocked, reason/keyboard/confirmation→Đã hủy, action hidden. History retained stale pending status on return; fixed by awaiting detail navigation and reloading current query/filter from repository. Two regressions reproduce the defect before fix and pass after fix, including pending-filter empty recovery. Current APK physical replacement-order flow also verifies refresh/empty/clear/cancelled badge. Final phase/matrix acceptance reconciliation remains OPEN. The goal must not be marked complete merely because this checkpoint is pushed.

## Exact next action

This session is limited to cancellation plus its directly observed stale-history defect. Do not start final acceptance reconciliation in this session. After validation/checkpoint, stop as requested.

Next session: read AGENTS/handoff/P4, verify branch/HEAD/status/remote, then reconcile P1–P3 acceptance records and final parity matrix against recorded native evidence and contract limits. Preserve QR/status-read limitations: previous Check status click has no newly verified result, no mock QR URL and no provider payment proof. Only fix verified issues; assess full Definition of Done before completing goal.

## Runtime hints

Flutter3.44.9/Dart3.12.2 executable: `E:/.vscode/Mobile/flutter/bin/flutter.bat`. Process-local GRADLE_OPTS: `-Dorg.gradle.project.kotlin.incremental=false -Dorg.gradle.project.kotlin.compiler.execution.strategy=in-process` for E-project/C-cache split; no persistent build setting change.

ADB `D:/AS_Sdk/platform-tools/adb.exe`; scrcpy `E:/scrcpy-win64-v4.1/scrcpy.exe`; last authorized LG API29 device LMV409N70de1ce3. Current scrcpy window328264; rediscover before reuse. Old attach did not connect and was stopped. No active attach is assumed. No uninstall/wipe. APK: `apps/mobile-app/build/app/outputs/flutter-apk/app-debug.apk`. Do not commit caches, temporary screenshots/logs or build output.
