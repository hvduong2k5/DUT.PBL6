# Customer Mobile parity handoff

Updated 2026-10-20. Goal ACTIVE. Whole Customer parity NOT COMPLETE.

## Git and boundaries

Branch feature/mobile-ui; HEAD before trace milestone 86dcef2db67bfdd5dd17976b12160a6851287b4e, pushed/verified. Resolve resulting checkpoint using git log. Prior checkpoints: 457eea6 support, eb57d76 Guest, ec8545a payment/account, 1d91c15 purchase, 82fd358 auth. Commit/push Mobile only; no merges/PRs. Read-only Web reference b1ff2918c7297fc93d0cb9a15cc1d0a0ee46b72e using git show origin/feature/ui-web:<path>; no Web checkout/edits performed.

## Implemented

- P0: route inventory/matrix, two-level plans, exact ZIP Customer snapshots/hashes under docs/03_api_specs/frontend-contracts. Decisions in docs/DECISIONS.md. Ancestor historical Stitch review does not govern this approved direct parity task.
- P1: Web palette/type/CTA, login identifier, registration required email/terms/Unicode password rules, auth return destinations; recovery neutral request/link/proof/reset/relogin/expiry/429. Android recovery URI. Reset never auto-logs in.
- P2: explicit SKU, quantity bound 20 from Web; actual images/fallback; available cart selection/selected subtotal; selected-only strict ZIP checkout and safe cart cleanup; two-level locations and saved addresses; voucher discovery/validation; actual order confirmation and payment/QR expiry/status. Mock Checkout creates Orders; no fabricated successful lookup or paid toggle.
- P3 account/session: editable name/email, immutable phone; loyalty actual overview; address CRUD/default/delete/errors; login-return and server capability gates. Native memory-only server cookie jar, no fabricated X-Session-Id, clear on logout/reset. Auth/me restore, 401 refresh, offline retention/local logout cleanup, stale-response guards. Mock has no implicit B2B privileges.
- P3 Guest: neutral challenge/contact/OTP/resend/cooldown/expiry, exact-order navigation; REST only supplied ZIP fields/cookies. Mock denies history/other orders/direct ID/expired/consumed grants and grants only exact newly created checkout. P3 support: context/create/detail/public reply, owned-order selection, retry idempotency, no fake upload/hotline/CSAT; local tickets scope/idempotency.
- P3 trace: manual code and deep link/detail/error/retry, production/expiry/artisan/origin/certificates; mock clearly labelled specimen. Find-product preserves q in catalog route/controller. Shared HTTPS external-media button rejects executable/local/credential schemes and handles launch failure. Added Flutter-published url_launcher 6.3.3 + seven platform packages only; no existing package upgrades. Lock Flutter lower bound becomes 3.44.0, within installed 3.44.9. Official docs checked. Actual generated URL plugin registrant additions are intentional now.

## Open work and contract limits

- User confirmed unchanged ZIP: voucher application at POST checkout waits for contract extension; never submit voucher_code or reduce final total. No shipping quote/durable payment QR recovery endpoint.
- P2 OPEN: discovery pagination/filter/sort reconciliation, legacy REST category_id/page_size fields versus ZIP q/status/page/pageSize; product reviews/eligibility; cart clear confirmation. Never present current-page filters as complete server results.
- P3 OPEN: cancellation/tracking/reorder; after-sales request/detail; verified B2B context/request/released quote/version acceptance. After-sales stable POST requires line_id, absent from stable Order response; draft eligibility lines are unspecified objects. Do not fabricate line IDs or silently use draft endpoints. Web return/request and detail source read; stable/draft return schema read. Stable detail can be implemented independently.
- Support ZIP forbids Web guest contact object and has no stable upload/download API; do not collect unsendable contact, pretend metadata uploaded, or invent file download URLs. Ticket list, loyalty history/redemption and notifications planned/draft. Missing native destinations must not be interpreted as complete.
- P4 full native/cross-flow QA remains OPEN; real backend/email/SMS/payment/credential lifecycle NOT VERIFIED.

## Validation and native state

- Baseline analyze / 60 tests PASS. Last recorded full suite 97/97 PASS (support milestone). Session/capability focused 4/4 PASS; trace/catalog/session focused 11/11 PASS and analyze PASS. A newer full run was started in session 86657 but its final output became unavailable; rerun, do not claim it passed. Trace Android build NOT RUN.
- Prior P1 APK build PASS (321.5s), P2 incremental build PASS (57.6s), installed without wipe. Native observed recovery validation/sent/local link/reset checklist/keyboard/expired proof, Home images, list→SKU→add→cart selection→checkout two-level form. Native support form/options and title/content keyboard scrolling observed; no overflow seen. Support submission/reply, account/Guest/trace full native QA still pending.
- Device secure-storage old-cipher warning pre-existed; fallback allowed mock app. No uninstall/data wipe. Use adb devices to discover current authorized device; no hard-coded ID in source/docs.
- Previous attach PTY 35097 (r/R) and scrcpy window 4130356, process E:/scrcpy-win64-v4.1/scrcpy.exe; verify live before reuse. App still has support source before capability/trace; URL native plugin requires rebuild/install, not reload. Last phone view: support title entered dummy text, content field just focused with keyboard. Node REPL @oai/sky variables scrcpyWindow/scrcpyState; refresh before every action. UI operations local mock only.
- Never commit .kotlin/build/cache/screenshots/logs. Generated registrant headers still only line-ending noise; URL launcher cc/cmake/Swift additions should be reviewed/staged explicitly.

## Exact next action

Inspect/stage trace milestone and push Mobile; verify SHA. Run full tests and Android build (independent), then install/update and reattach to authorized device without wiping. Finish native support/Guest/account/trace checks when practical. Continue after-sales stable detail and explicit request contract gap, B2B and remaining orders; read narrow Web/schema before edits. Keep matrix/plans/status current; goal remains active until original acceptance criteria are resolved.
