# Customer Mobile parity handoff

Updated: 2026-10-19. Goal ACTIVE; whole task is not complete.

- Task/scope: Customer Flutter UI/flow parity with read-only Customer Web; supplied ZIP governs API.
- Branch: `feature/mobile-ui`. Source HEAD before first checkpoint: `8e1994a7401a5c4759cfc24149182b5e2148b6be`. This handoff is included in the following auth checkpoint; use `git log -1` for its resulting commit ID.
- Web baseline: `b1ff2918c7297fc93d0cb9a15cc1d0a0ee46b72e`, confirmed against remote; read through Git objects without Web checkout/edit.
- Plans: `docs/plans/MASTER_PLAN.md`; audit matrix `docs/04_testing/CUSTOMER_PARITY_MATRIX.md`; implementation `docs/plans/active/P1_FOUNDATION_AUTH.md`.
- Completed: Customer route inventory; copied Customer contracts byte-for-byte with hashes; theme/primary CTA alignment; login identifier payload; Web password checks/registration email+terms; safe auth return destination; complete recovery request/link verification/reset/relogin UI with REST/mock adapters; Android recovery URI handler; structured Error/cooldown metadata.
- Modified areas: root AGENTS, task docs/contracts; mobile Android manifest; router, theme/shared button/input/error mapping; auth domain/adapters/Cubit/pages; auth/error/viewport tests. No Web/Admin/service changes.
- Validation baseline: analyze PASS; 60/60 tests PASS. Changed slice: analyze PASS; 30/30 focused auth/error/viewport tests PASS; 3/3 recovery navigation/keyboard tests PASS. One initial narrow-viewport test needed lazy-list scrolling in its test driver; corrected and passed. Full post-change suite pending.
- Android: Flutter 3.44.9/Dart 3.12.2; authorized ADB device and existing scrcpy window verified. Debug APK build currently running (exec session 34233); native app QA NOT RUN yet. Do not claim real email/session/backend/payment verification from mock results.
- Decisions: Web approved UI replaces obsolete Stitch prompt-by-prompt workflow for this task; retain architecture/dependencies. ZIP requires registration email and login identifier; Checkout forbids district/extra price fields. Custom native recovery URI `oma://customer/account-recovery?recoveryToken=...` is scoped to recovery; no HTTPS app-link or email-provider configuration is implied.
- Open decision: async question to user about missing voucher field in stable POST /checkout. List/validate can proceed; do not fabricate an applied submitted discount. P2/P3/P4 remain OPEN.
- Exact next action: finish Android build, install debug APK and test recovery/login through scrcpy; record native evidence. Then refine P2 from Web checkout/cart/promotions code, fixing selection/two-level address/strict payload and adding voucher discovery/validation with contract limitation explicit.
- Git: first scoped checkpoint commit/push is next; only feature/mobile-ui is authorized. Verify remote SHA after push. Never merge/PR or push Web.
