# Customer Mobile parity handoff

Updated 2026-10-20 at user-requested checkpoint. Goal ACTIVE; implementation and local QA progressed, whole task not complete.

## Scope, branch and checkpoint

- Work and push only `feature/mobile-ui`; no merge, PR or changes to other branches.
- Source HEAD: `f4e0a085b55b0ed2ac97d59bb3531bfd0b898512`, already verified on origin. The following documentation checkpoint is identifiable with `git log -1`; this file deliberately records its parent rather than a self-referencing commit hash.
- Customer Web reference: read-only `origin/feature/ui-web` at `b1ff2918c7297fc93d0cb9a15cc1d0a0ee46b72e`, read through Git objects.
- API ZIP authority/hashes: `docs/03_api_specs/frontend-contracts/SOURCE.md`. User chose unchanged ZIP: voucher validation is implemented; applying a voucher at checkout waits for contract extension.
- Current phase: `docs/plans/active/P4_CUSTOMER_QA.md`. Roadmap: `docs/plans/MASTER_PLAN.md`; coverage: `docs/04_testing/CUSTOMER_PARITY_MATRIX.md`.

## Completed implementation

Shared Web colors/type/CTA and phone layouts; login/register/recovery/link/reset/relogin; session restore/refresh/capabilities and protected-401 identity recovery without automatic mutation replay; catalog search/pagination/loaded-scope filters; explicit SKU/gallery/product data; selectable cart/clear confirmation; strict selected-only checkout/saved address/promotions; actual order confirmation/payment status; account/profile/addresses/loyalty; history/cancel/tracking/Guest OTP; Support creation/public replies; manual trace/deep links; local returns/create/case with stable REST adapter; verified B2B request/released quote/consent/acceptance; public reviews and owned delivered-order/SKU submission.

Native QA found and fixed keyboard refocus, checkout subtotal overflow, Vietnamese calendar labels/date-action focus and review comment viewport/caret visibility. Latest source edits are review presentation + regression test. No source changes after f4e0a08; this checkpoint updates task documentation only and removes generated header line-ending noise.

## Latest verification

Source f4e0a08: `flutter analyze --no-pub` PASS; full `flutter test --no-pub` **142/142 PASS**; debug APK build **PASS (37.4s)**. Focused review tests 4/4 PASS include a 400px keyboard/caret visibility regression. No need to rerun unchanged source checks for this documentation checkpoint.

Phone has localization APK718159e installed as an update without data wipe, then review fix hot-reloaded successfully. f4e0a08 APK was built but not installed after hot reload. Distinguish this physical source evidence from a fresh-install acceptance test.

All following native evidence uses local mock fixtures; it does not prove real backend/email/SMS/payment/provider behavior:

- Recovery neutral request/validation/expiry, valid mock link/reset/checklist/keyboard/success requiring login, then relogin/account. Mock accepts nonempty passwords, so real credential validation/revocation is unverified.
- Guest challenge/OTP→OM-8921 detail→tracking; explicit SKU→cart, clear-dialog cancel preserves line, guest Huế/Thuận Hòa checkout→COD85,000→unpaid payment→same order detail.
- Profile save acknowledgement, loyalty→promotions, address list/default/edit/save. Vietnamese calendar selection returns without reopening keyboard.
- B2B MOQ20/date2026-10-27→YC-LOCAL-2 receipt; released BG-LOCAL-001 consent→accept→LOCAL-B2B-001 PENDING_PAYMENT4,180,000, matching quote.
- Review owned OM-8860/exact Hộp500g→comment→submit→public5/5 aggregate/count1/verified badge; hot-reloaded keyboard fix automatically reveals caret without manual scroll.
- Trace empty validation→labelled local specimen→batch/artisan/origin/disclaimer→find-product preserves query→empty→clear restores4; price filter under200k returns3.
- History pending-payment filter empty→clear restores4; delivered OM-8874→return exact line/max2/reference110k→quality details/example HTTPS evidence/fake bank fixtures→LOCAL-RET REQUESTED, refund0 and no confirmed transfer.
- Support ORDER_HELP + owned OM-8874→local ticket→public reply visible and composer cleared; no real CSKH transmission/upload.
- Logged-in checkout saved-address fills Huế/Thuận Hòa, VietQR order **OM-82047142 /85,000 /PENDING_PAYMENT**. Payment shows countdown and explicit missing QR because mock supplies no QR URL. Last action clicked Check status; refreshed result still needs observation. No actual QR/payment verified.

## Constraints and unresolved work

No fabricated endpoints, line IDs, food/certificate association, QR or payment success. Stable contract lacks checkout voucher field, production return eligibility/line-ID projection, uploads, Support inbox, loyalty history/redemption, notifications, payment reissue and B2B documents. Global advanced catalog filters remain loaded-scope only. Server must enforce atomic quote-version acceptance; ZIP has no request version field. These external limits remain explicit, not silently completed.

Native order cancellation and final phase/matrix acceptance reconciliation remain OPEN. Existing automated cancellation tests passed; native cancellation is NOT RUN. The goal must not be marked complete merely because this checkpoint is pushed.

## Exact next action

1. Read AGENTS, this handoff, MASTER_PLAN and P4. Verify branch, HEAD, clean status and remote checkpoint.
2. Rediscover ADB/scrcpy/attach state. Observe current payment screen after Check status, open OM-82047142 detail, perform local cancel reason/confirmation and verify server-returned state; preserve honest missing-QR state.
3. If app process/fixture was lost, record it and create one replacement local unpaid order; do not repeat completed flows by default.
4. Reconcile P1–P3 acceptance records and final parity matrix against P4 evidence/contract limits. Fix only verified issues; rerun focused/full/build if source changes. Consider installing current built APK for final physical verification and record exact version.
5. Update final handoff, commit/push only Mobile; assess full Definition of Done before completing goal.

## Runtime hints

Flutter3.44.9/Dart3.12.2 executable: `E:/.vscode/Mobile/flutter/bin/flutter.bat`. Process-local GRADLE_OPTS: `-Dorg.gradle.project.kotlin.incremental=false -Dorg.gradle.project.kotlin.compiler.execution.strategy=in-process` for E-project/C-cache split; no persistent build setting change.

ADB `D:/AS_Sdk/platform-tools/adb.exe`; scrcpy `E:/scrcpy-win64-v4.1/scrcpy.exe`; last authorized LG API29 device LMV409N70de1ce3. Last window68230 and Flutter attach26739 may be stale; rediscover before use. No uninstall/wipe. APK: `apps/mobile-app/build/app/outputs/flutter-apk/app-debug.apk`. Do not commit caches, temporary screenshots/logs or build output.
