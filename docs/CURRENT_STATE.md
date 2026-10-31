# Customer Mobile — current handoff

Updated2026-10-31. Branch feature/mobile-ui, M5 entry0c10724f010e7e976493039bb1d536cb5b3967de; M5 source checkpoint follows this handoff update. User approved local account persistence, credential checks, session restoration, two-account isolation and APK. No more approval pause is needed for this scope. No dev/Web/backend edits, merge/PR or production provider activation.

## Plans / authority

Master docs/plans/MASTER_PLAN.md; active docs/plans/active/M5_LOCAL_IDENTITY.md. M1–M4 pinned dev7f22226/ZIP provenance/Web visual references unchanged. M4 source7e00825 plus delivery/QA0c10724 is the187-test baseline. Earlier full local native purchase/cancel QA and its standard APK evidence remain in M4 plan. M5 intentionally replaces permissive auth only in the default local cohort.

## Current implementation

- New LocalAuthRepository default local cohort, subclassing old mock only for shared local identity callback shape; overrides all auth/recovery operations. Explicit legacy cohort remains MockAuthRepository (localMvpCheckout:false), REST unchanged.
- Versioned FlutterSecureStorage vault, separate key from REST tokens. Serialized FIFO transactions preserve each caller's Zone and recover after errors. New accounts get UUIDs, canonical phone/email, unique phone/email, profile retention and7-day saved session. Successful logout persists; write failures reported and don't pretend durable logout.
- Password verifier PBKDF2-HMAC-SHA256/random16-byte salt/256-bit digest/120000 iterations in background isolate using pinned cryptography2.9.0. Only this dependency added; no package upgrades/native plugin. No plaintext passwords or user QA credentials in Git.
- Reserved registered/B2B fixture IDs retained with one explicit test credential; unknown/new phones no longer auto-login, new registration cannot gain B2B. New local recovery fails closed without real verification; legacy recovery demo explicitly tested in its old cohort; REST recovery unchanged.
- Principal-scoped in-memory cart/address, one-time genuine guest-cart adoption; old account cart never transfers to next. Legacy local orders filter by owner; authenticated B cannot inherit guest grants. Existing MVP order/operation/command owner checks retained. CartCubit clears visible state on actor change and guards late loads/mutations by generation/principal; app auth listener reloads it. Profile callback now awaited to persist changes.
- Two seeded fixture passwords in affected default-cohort tests now use canonical fixture credential; legacy assertions retained. Test storage initialized/reset per test; UI form uses injected deterministic hasher while real PBKDF2 has separate checks. No production test-mode bypass.

## Validation / modifications

M5 full regression200/200 PASS; flutter analyze --no-pub PASS (no issues). Includes12 focused identity/domain checks, persistent registration/login form and existing profile/navigation/commerce regressions. Test harness now preserves caller Zones and injects a deterministic verifier only for fake-clock form state; real KDF is separately covered. APK/native M5 PENDING.

Modified areas: app provider/factory, auth store/hasher/repository/Cubit, cart/address repositories and CartCubit, profile callback, legacy order ownership, cryptography pubspec/lock, focused tests/fixture setup, M5/current/master/decisions. Content-identical pub-generated Linux/Windows header noise restored. Separate untracked docs/MOBILE_MVP_DISCUSSION_REFERENCE.md belongs to Issue38; never stage automatically.

## Limits and exact next action

Source checkpoint/push Mobile only, then build/install APK with process-local GRADLE_OPTS workaround. Use ignored native QA harness to create the user-provided synthetic account only if absent, validate wrong-password rejection/relogin/new instance/cold-start restore and second-account boundaries; never commit credentials. Restore standard APK afterwards and remove harness, retaining the now-persistent local account/session. Reuse previous chat credential privately; no credential in docs.

Cart/address/commerce business data remain process-local; only accounts/session persist in this phase. Backend/Identity/BFF/payment acceptance, real email recovery, geography creation, history cursor inputs, typed tracking and Care review remain outside approved M5. Native UI loops bounded; prioritize recorded tests. Update final evidence and verify remote checkpoint before claiming done.
