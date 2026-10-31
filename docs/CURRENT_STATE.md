# Customer Mobile — current handoff

Updated 2026-10-31. Branch `feature/mobile-ui`; M5 source HEAD `e1d7d1ecbd8a0b416c9548bef113185ee2cb06ca` pushed and remote verified (preceding source `ff6c344`, entry `0c10724`). Final delivery documentation follows that source. M5 approved scope is locally complete. No dev/Web/backend edits, merge/PR or provider activation.

## Plans and completed scope

[Master](plans/MASTER_PLAN.md), [M5 evidence and decisions](plans/active/M5_LOCAL_IDENTITY.md). M1–M4 pinned dev7f22226/ZIP provenance/Web visual references remain unchanged. Their native and automated results retain their original cohort limits.

- Default local cohort has persistent UUID accounts, actual password verification, unique canonical phone/email, retained name/profile, seven-day saved session and durable successful logout. Explicit legacy demo and REST adapters stay separate.
- Account vault uses FlutterSecureStorage, dedicated Android crypto/storage namespace `oma_local_identity_v1`, and `resetOnError:false`. Corrupt/failed storage fails closed; no silent erase. PBKDF2-HMAC-SHA256, per-account random salt, 120000 iterations in a background isolate; only cryptography2.9.0 added, no package upgrades. No user-supplied QA credentials committed.
- Cart/address and legacy orders are principal-scoped; genuine guest-cart adoption never transfers A's cart to B. Cart UI resets and rejects late responses after identity change. Existing MVP order/operation/command ownership retained. Profile persistence callback is awaited. New persisted accounts cannot use public mock-token recovery; legacy demo and REST recovery remain separate.

## Validation and delivery

- Final source: full **201/201 tests PASS** (89s), **flutter analyze PASS** (21.8s, no issues), **debug APK PASS** (62.9s). Real KDF/storage tests and deterministic form-state tests are separate; no assertions/lints disabled.
- Android10 LG LM-V409N: ignored native API harness passed wrong-password rejection, relogin, durable logout/new-instance restore, distinct A/B cart/address/legacy+MVP order/operation boundaries. Cold-process restore PASS (4167ms). This is native storage/API evidence; registration form has automated widget coverage.
- Standard APK restored to both output locations, SHA-256 verified, install-r Success, cold launch5658ms. Scrcpy Home→Account showed the same QA profile after reinstall; session retained. Credential-bearing harness removed, no QA entrypoint markers in standard app.
- Artifact: `apps/mobile-app/build/app/outputs/flutter-apk/app-debug.apk`, 198276012 bytes, source e1d7d1e. SHA-256 `67BE67940E292DFC1E739A428BE52305B8E3CB47C513B0D0E081A7FAE39ED51C`. Generated APK/logs/screenshots are not committed.
- Captured standard-app log window:188 lines, zero Flutter/AndroidRuntime crash/Unhandled/RenderFlex matches. Installed plugin still reports two legacy EncryptedSharedPreferences decryption fallbacks. Plugin source probes old global ESP before custom cipher initialization; isolated accounts restore successfully with destructive reset disabled. Do not claim warning-free storage, clear old prefs or upgrade the plugin without a separate investigation.

## Modified areas and boundaries

App providers/factory; auth store/hasher/repository/Cubit; cart/address repositories and CartCubit; profile callback; legacy order ownership; dependency lock; tests/fixture setup; M5/master/decisions/parity matrix/handoff. Source committed in ff6c344 and e1d7d1e. Content-identical generated-header noise restored. Separate untracked `docs/MOBILE_MVP_DISCUSSION_REFERENCE.md` belongs to Issue38 and remains untouched/uncommitted. No temporary QA artifacts staged.

## Remaining limits and exact next action

No required M5 implementation remains. User can log out/relogin using the synthetic credentials shared privately; they are not a backend account. **Only accounts/session persist**: cart/address/commerce business data remain process-local and reset with process restart. Real email recovery, Identity/BFF/payment provider acceptance, canonical geography create/edit, history cursor inputs, typed tracking and Care review remain external gates. Local checks do not establish production security/provider acceptance.

Next session: read handoff/master/M5, verify branch/HEAD/status and final documentation checkpoint on origin. Present M5 results for user review, then obtain the next implementation scope (recommended: Identity/BFF integration readiness against the pinned public contract). A separate small investigation may trace old-ESP warnings without clearing data. Do not resume broad UI/device loops or activate providers automatically. Build commands/SDK/workaround remain in AGENTS.md/M5 history.
