# M5 — Persistent local accounts and identity isolation

Approved2026-10-31 (“ok làm đi”): local account persistence, credential checks, restart/session restoration, two-account isolation and APK. LARGE auth/storage change. Branch feature/mobile-ui, entry0c10724; dev/Web/backend read-only; no merges/PRs/provider activation. M1–M4/187 tests is the baseline. Discussion-reference MD stays untracked.

## Goal and entry decisions

Current mock has only currentUser, arbitrary password acceptance and shared CUST-MOCK. Cart/address collections are shared. MVP operation/order/command ownership is already principal-based. New default local runtime uses strict LocalAuthRepository; explicit localMvpCheckout:false retains old MockAuthRepository demo/test cohort. Reserve existing registered/B2B fixture identities with one test-only credential; new registrations always get UUIDs, never B2B authority from a phone. User QA credentials are never seeded in Git.

Keep AuthRepository/remote transport/UI forms unchanged. Store one versioned local registry/verifiers/active session JSON in FlutterSecureStorage, separately namespaced from REST tokens; serialize read-modify-write across instances. Corrupt/failed storage must not erase accounts or downgrade authentication. Publish identity only after a successful commit. Preserve name/email on login/restore/profile update; finite session expiry; successful logout durable, write failure reported honestly. Normalize +84/0 phone; reject duplicate phone/email and invalid credentials.

Verifier: cryptography2.9.0 PBKDF2-HMAC-SHA256, random16-byte salt,256-bit output,120000 iterations in a background isolate and constant-work comparison. Existing crypto has no password KDF; add only this package, no existing package upgrades/native plugin. Primary API: https://pub.dev/documentation/cryptography/2.9.0/cryptography/Pbkdf2-class.html and https://pub.dev/packages/cryptography/versions/2.9.0. This is offline QA authentication, not production Identity. No plaintext passwords, bearer tokens or credentials in logs/checkpoints.

Recovery request remains neutral/no real mail. New persistent accounts must not accept legacy public mock-token as password-reset authority; fail closed without an out-of-band verifier. Explicit legacy cohort keeps its recovery demo/tests. Real REST recovery unchanged. This is necessary when replacing permissive auth, not a new email service.

## Scope, files and state boundaries

New auth/data local_account_store.dart, local_password_hasher.dart and local_auth_repository.dart. Factory/app wiring, awaitable profile callback and mapped validation errors. Optional principal-scoped cart/address buckets plus legacy mock-order ownership. Guest cart may move/merge into the logging-in owner, never A→B. CartCubit resets visible state on identity changes and rejects stale generation/principal responses. Existing MVP owner checks stay intact. Commerce business persistence across process restart is out of scope.

Expected source areas: auth data/Cubit, app factory/providers, cart repository/Cubit, address repository/profile callback, legacy order mock filtering; pubspec/lock only cryptography addition. Reuse existing UI/theme/validation/storage/uuid. No branding, remote endpoint/schema change, dependency upgrades, production auth/payment/QR/refund, geography/Care work or unrelated refactor.

## Implementation / acceptance

- [x] Strict vault schema/serialized transaction and password KDF; corruption/write failure fail closed.
- [x] Unique registration, duplicates, unknown/wrong password denial, profile persistence, finite saved session and durable logout; no main-thread KDF loop.
- [x] Scoped cart/address/legacy orders; guest continuity; Cart state reset/stale guards; existing MVP isolation retained.
- [x] Focused tests: store reconstruction/no plaintext, duplicate/concurrent registration, wrong password, failed/corrupt storage, expiry/logout, phone normalization and two-account cart/address/order/operation/command/late-response isolation.
- [x] Full regression/analyze and APK. Native ignored harness registers synthetic QA account only if absent, checks wrong password/relogin/restart/identity separation. Standard APK restored afterwards, credential-bearing harness removed; no Windows auth-dialog automation.
- [x] Update current/master/decisions/evidence, Mobile-only source commits/push and verified remote; final delivery documentation checkpoint follows.

Definition of Done: local registration has a stable separate ID, credentials genuinely checked, session survives restart and successful logout survives restart; A/B mutable data and late responses do not leak; regression/new checks pass; APK installed; backend limits explicit.

## Risks / current checkpoint

KDF/storage latency must be checked natively. Deterministic test hasher allowed for state/concurrency tests only; cover real PBKDF2 separately. Isolate injected/secure storage per test, no production test-mode bypass. Restore guest cart only into intended owner. Preserve explicit B2B/legacy fixtures and their role boundary. New storage must not silently erase corrupt accounts. These choices amend the permissive M4 local auth baseline because the user now requires actual persistent local accounts.

LOCAL COMPLETE2026-10-31; final source e1d7d1e/evidence below supersede intermediate checkpoint statuses. Profile form uses a deterministic injected verifier for fake-clock UI state; real PBKDF2 including fixture credentials is tested separately. No assertions or lints disabled. No required M5 work remains; provider integration is a separate scope.

### Native refinement checkpoint (historical pre-delivery state)

Source checkpoint ff6c344 pushed/remote verified. Its debug APK built255.7s. Ignored native harness passed real KDF/wrong-password/relogin/durable logout/new-instance restore, A/B cart/address/legacy+MVP order/operation isolation, and second cold-process restore (Android10 LG, COLD4190ms). Initial account preparation2995ms/2666ms includes KDF and storage; no per-frame/performance claim. Standard APK not yet restored at this point.

Native logs exposed an old default EncryptedSharedPreferences decryption fallback. Installed plugin10.3.4 source confirms destructive resetOnError:true default. Before final delivery, new local vault now uses Android storageNamespace oma_local_identity_v1 and resetOnError:false. Old namespace/data is untouched, onboarding/REST configuration unchanged; no shipped-user migration assumed.13 focused checks PASS including plugin options; final201/201 regression89s and analyze21.8s PASS; new standard APK/native in isolated namespace PENDING. Recreate only synthetic QA identities if absent there; never overwrite conflicting existing accounts. Next: finalize checks, source checkpoint, rebuild and rerun bounded native probe/standard APK restore.

### Final delivery / acceptance

Source e1d7d1ecbd8a0b416c9548bef113185ee2cb06ca pushed and remote SHA verified.201/201 tests89s and analyze21.8s PASS. Standard debug APK build62.9s PASS using process-local GRADLE_OPTS (`kotlin.incremental=false`, compiler execution strategy in-process). Flutter3.44.9/Dart3.12.2, no Android/dependency upgrade. Artifact build/app/outputs/flutter-apk/app-debug.apk,198276012 bytes, SHA256 67BE67940E292DFC1E739A428BE52305B8E3CB47C513B0D0E081A7FAE39ED51C.

Final isolated-namespace native probe: temporary entrypoint built88.4s/installed10.8s; A/B account preparation3027ms/2690ms includes storage and one or more real KDF operations, not a frame-performance benchmark. PASS wrong-password denial, stable UUID/profile on relogin, new repository restore, durable logout, A/B cart/address/legacy+MVP order and operation isolation with A's data retained on return. Second cold process4167ms and restore PASS. Native QA account credentials only in ignored temporary entrypoint, not Git. Registration tested through the real repository API on Android; form flow has automated widget coverage.

Exact standard backup restored to both flutter-apk and apk/debug outputs; SHA256 checked and APK checksum refreshed. Install-r Success, cold standard launch5658ms; scrcpy Home→Account visibly retained the same QA identity/name after reinstall. Temporary entrypoint removed. Captured188 standard PID log lines: zero Flutter/AndroidRuntime crash/Unhandled/RenderFlex matches, zero QA entrypoint markers. Two old-ESP decryption fallback diagnostics remain: plugin10.3.4 probes legacy global ESP before custom cipher initialization even with a namespace; local source inspected. Do not claim zero plugin warnings or erase old data. Final local accounts restore successfully; resetOnError:false protects their error path. A future plugin/legacy-storage investigation is optional and separate from this completed slice.

M5 Definition of Done met for local cohort. Only accounts/session persist; addresses/cart/orders/operations remain process-local. Real email recovery and backend/provider acceptance remain NOT VERIFIED/NOT ACTIVATED. Next: user review and selection of the next integration scope; no automatic provider activation, broad device rerun or unrelated refactor.
