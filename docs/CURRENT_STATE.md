# Customer Mobile parity handoff

Updated 2026-10-19. Goal ACTIVE. Whole Customer parity NOT COMPLETE.

## Git / scope / authority

Branch `feature/mobile-ui`. HEAD before current milestone `1d91c15390f020887edd2b8ca3b8fdd0324f3e29`, previously pushed/verified. Prior auth checkpoint `82fd358`. This file belongs to the following payment/account milestone; resolve its resulting SHA using git log. Only Mobile commits/pushes allowed; no merge/PR. Web `origin/feature/ui-web` at `b1ff2918c7297fc93d0cb9a15cc1d0a0ee46b72e` stays read-only, inspected with git show. No Web edits/checkout performed.

## Completed implementation

- P0: route inventory, parity matrix, source instructions, exact ZIP Customer snapshots with SHA-256 and -text attributes. Plans in docs/plans; decisions in docs/DECISIONS.md.
- P1: shared palette/primary CTA; login identifier, registration required email/terms/Web Unicode password rules; safe auth return URLs; recovery request/neutral sent/token proof/reset/relogin/expiry/429 and errors. Android recovery URI. Tests prove reset does not log in and proof cannot be reused.
- P2: explicit product SKU selection, max quantity 20 verified from Web validation, no fake list SKU; add success only after repository success; real Web heritage imagery reused in mock mode with missing-image fallback. Available cart selection and selected subtotal; selected-only checkout/consumption preserves other lines. Strict ZIP request/header, two-level locations, retry/idempotency, voucher list/condition validation, actual confirmation/payment UI using Checkout QR/expiry and Order read. Invalid order ID no longer fabricates success; mock checkout records created orders into its paired mock Orders repository. Paid state is never toggled locally.
- P3 account slice: profile name/email update, read-only phone, conflict reload; live account display update; loyalty server points/tier/rate; profile/loyalty/address login gates with return destination. Account address form/model/payload has no district, canonical two-level locations, recoverable save errors, delete confirmation, pending/default errors, long-name/row layout handling. Address create uses retry key. Native server-cookie handling is at ApiClient only, memory-only and domain/path/expiry scoped; logout/reset clears it. Removed fabricated native X-Session-Id. Added only dio_cookie_manager 3.5.0, cookie_jar 4.0.9 and transitive universal_io 2.3.1; no existing package upgrade.

## Accepted constraints / remaining work

- User explicitly chose unchanged ZIP. Voucher check works; actual voucher application at POST /checkout is waiting for contract extension. Never send voucher_code or subtract preview savings.
- ZIP has no shipping quote endpoint, no durable QR recovery endpoint and no advanced catalog filter contract matching Web's legacy query fields. UI must state tentative merchandise total and avoid inventing final fees/QR. Discovery advanced filtering/pagination remains to reconcile; product reviews not implemented yet.
- P3 still OPEN: customer capabilities/verified B2B guards; Guest order challenge/OTP/exact-order access; current-session restore; order cancellation/tracking/reorder; support create/detail/public reply; after-sales create/detail with draft operations separated; trace entry/detail; B2B request/released quote/version acceptance. Read each Web source and corresponding ZIP schema before editing. No placeholder hotline or uncontracted upload endpoints.
- P4 full cross-flow/native QA still OPEN. P1 successful reset/relogin and P2 complete order creation have automated coverage but not full native coverage. Do not mark goal or roadmap DONE at checkpoint.

## Validation and native state

- Baseline analyze PASS / 60 tests. Latest full suite 90/90 PASS and analyze PASS before final address mutation refinements. After those refinements: analyze PASS, focused account/cart/network suite 14/14 PASS. Cookie scope/clear and profile/login-return/save tests pass. No known failing automated check.
- P1 APK build PASS (321.5s), installed by update. P2 incremental build PASS (57.6s), installed and Flutter attached. Native scrcpy observed recovery validation/sent/mock-link/reset checklist/keyboard scroll/expired proof; Home inherited images; list→SKU detail→successful add; populated cart selection/subtotal; checkout two-level address and sticky CTA without visible overflow. No real backend/email/payment/provider claim.
- A pre-existing device secure-storage decryption warning appeared during initial launch; fallback allowed local UI. No device data was wiped. Remote token-storage behavior on that device remains NOT RUN.
- Active Flutter attach PTY session `35097` supports `r` reload / `R` restart. It is running source from before account/cookie slice; hot restart is required before native account QA. Initial non-PTY Flutter driver was intentionally stopped; app preserved, attach works. scrcpy via node_repl @oai/sky: window id 4130356, app process E:/scrcpy-win64-v4.1/scrcpy.exe. Refresh state before every action. Current phone shows Checkout from one selected SKU. Use authorized ADB device discovered live, never hard-code device ID in tracked files.
- Generated Linux/macOS/Windows registrant files show line-ending-only tool noise; do not stage them. android/.kotlin ignored. No screenshots/logs/build/cache in commits.

## Exact next action

Finish scoped commit/push and verify remote SHA, then continue P3 ownership/capability/Guest lookup before service flows. Update active P3 plan and matrix as each slice lands. Native account QA: restart attached app, sign in with local mock, open profile/loyalty/addresses and test error/keyboard/back. Run focused checks per slice and full suite/build at meaningful milestones. Preserve all original scope and accepted ZIP limitations.
