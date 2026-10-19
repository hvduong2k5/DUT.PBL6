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
- [ ] Analyze/tests/build; Android/scrcpy flow and keyboard QA.

## UI / failure states

Request→sent, verifying→reset→success→login; expired/invalid link→new request. Inline validation/field errors, disabled pending, rate-limit cooldown, network retry without losing input. Password visibility, narrow/large text, keyboard scrolling and accessible actions.

## Validation / DoD

Focused auth/error/widget tests plus complete existing regression suite; `flutter analyze`; Android debug build then device interaction. DoD: no dead recovery link, no account enumeration, no automatic reset login, proof not reused, login/register return to original valid route, shared token alignment, recorded evidence. Baseline failures stay separate.

## Current / next / handoff

Status: READY. Baseline checks running. No implementation yet. Next: implement shared error/validation and auth recovery interfaces, then pages. Pending external decision: voucher parameter absent from Checkout contract (does not block P1).

Checkpoint evidence: analyze PASS; focused auth/error/viewport 30/30 PASS; navigation/keyboard 3/3 PASS. Debug build running; native validation pending.
