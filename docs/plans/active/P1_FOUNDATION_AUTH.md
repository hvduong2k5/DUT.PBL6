# P1 — Shared visual foundation and authentication parity

## Goal / why

Bring shared tokens/CTA into line with Web and close the missing password recovery path, while preserving guest cart and return navigation.

## Dependencies / decisions

P0 route inventory and initial contracts mapped. Read-only Web paths and differences are recorded in `docs/04_testing/CUSTOMER_PARITY_MATRIX.md`. User authorized implementation; no Stitch generation is needed. Keep existing Cubit/repository structure and dependencies.

## In / out

In: theme/colors/buttons/inputs, login/register validation/terms/return target, email recovery request/verify/reset flow, error envelope and focused tests. Out: provider email delivery, backend session revocation implementation, password storage, Web changes, uncontracted registration verification endpoints, coupon checkout policy.

## Contracts / security

- Login POST `{identifier,password}`; register snake_case with required email.
- Recovery request `{email}`→202 neutral accepted; verify `{recoveryToken}`→200 `{verified,resetProof,expiresAt}`; reset `{resetProof,newPassword}`→204. Use Idempotency-Key for each distinct retryable action; reuse on retry of identical payload.
- No automatic authentication after reset. Proof remains memory-only; consume it in local mock; no logging token/proof. Clear local session after successful reset and return to login preserving a safe in-app destination.
- Error `{code,message,requestId,errors,retryAfterSeconds}` retained. Show safe messages and field errors; handle expiry and throttling.
- Web password checks 15–128 code points with Unicode character classes; login only requires nonempty password. Required email differs from Web deliberately to obey ZIP.

## Files / steps

- [x] Theme token alignment and brown primary CTA; accessible pending/disabled state.
- [x] Extend reusable input for autofill/keyboard submission.
- [x] Auth validation/return-location helpers; correct login payload and required registration email.
- [x] Recovery domain/repository methods + REST/mock implementations.
- [x] Recovery Cubit finite states with expiry/retry/rate limit handling; recovery page and route.
- [x] Update AuthPage with recovery, checklist, confirmation, terms and safe return continuity.
- [x] Tests for payload/status/error, neutral recovery/proof lifecycle/expired/throttled retry, navigation and guest cart preservation.
- [x] Analyze/tests/build; recorded Android local recovery/login and keyboard QA. See 2026-10-21 reconciliation below for evidence limits.

## UI / failure states

Request→sent, verifying→reset→success→login; expired/invalid link→new request. Inline validation/field errors, disabled pending, rate-limit cooldown, network retry without losing input. Password visibility, narrow/large text, keyboard scrolling and accessible actions.

## Validation / DoD

Focused auth/error/widget tests plus complete existing regression suite; `flutter analyze`; Android debug build then device interaction. DoD: no dead recovery link, no account enumeration, no automatic reset login, proof not reused, login/register return to original valid route, shared token alignment, recorded evidence. Baseline failures stay separate.

## Acceptance reconciliation — 2026-10-21

Status: VERIFIED LOCAL for foundation/auth implementation. This is an engineering evidence review, not a claim of live provider processing or user visual sign-off.

- Web globals.css primary/surface/ink/muted/gold/error/border checked against AppColors, AppTheme and AppPrimaryButton; serif headings, Be Vietnam Pro body and brown/white52px CTA retained.
- Recovery contract/retry/proof/expiry/cooldown and safe-return assertions exist in recovery_test.dart and rest_auth_repository_test.dart. recovery_navigation_test.dart covers checkout intent, successful reset requiring login and narrow large-text keyboard. guest_cart_auth_test.dart covers cart continuity.
- Recorded Android local recovery request/reset/relogin/keyboard and current local login evidence are in P4. Session/capability and protected401 concurrency/no-write-replay tests are in the full144/144 suite at e86f82b. Analyze/debug APK passed on that source.
- Not verified: real email delivery, backend password/session revocation, live cookie lifecycle. These are integration evidence limits, not missing UI routes. Registration native branches are not individually claimed as observed; automated auth/navigation evidence applies.

No P1 source change needed during this audit. Next action belongs to P4 open checks in CUSTOMER_PARITY_MATRIX; do not restart foundation/recovery work from historical pending notes.
