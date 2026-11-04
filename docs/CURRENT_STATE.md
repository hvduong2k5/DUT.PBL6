# Customer Mobile — current handoff

## Latest read-only API audit — 2026-11-04

User requested checking every Git branch and the supplied Drive folder against the Mobile login prompt, after deferring live auth activation. [Source audit and exact next slice](03_api_specs/IDENTITY_SOURCE_AUDIT_2026-11-04.md): all25 remote branches inventoried after fetch; dev now a24492b63c364f93ad19d880798137e554898c3f and contains all8 prompt endpoints. The historical missing-source statement below is superseded. Mobile source remains unchanged at HEAD0c0dd05 during audit. No merge/checkout/backend/provider changes or runtime tests.

Open integration gaps: missing OMAMA_AUTH_STATE transaction cookie in Mobile; expired Bearer attached to cookie-based session/refresh/logout; registration PENDING treated as success. Backend blockers: Gateway missing PUT password route, password-grant verification conflicts with realm template; social/JIT/config/live acceptance unverified. Drive Customer contracts match the historical Mobile snapshots, not the new Identity routes. Next recommended implementation: small Mobile cookie/session lifecycle correction with existing tests, then registration status handling; actual auth activation remains deferred. Audit documents are the only intentional new changes; preserve unrelated MOBILE_MVP_DISCUSSION_REFERENCE.md.

Updated2026-11-04. Branch feature/mobile-ui. Final application source3108d3770882cfb4ce2de63ec6bdcf5f03c6bb2b pushed/remote verified; preceding implementation25c8d7a and lifecycle checkpoint173bc4a. Final documentation checkpoint follows this source. Active M6 implementation is complete and locally verified; real Identity backend acceptance is unverified.

## Plans and authority

[Master](plans/MASTER_PLAN.md), [M6 task goals/evidence](plans/active/M6_IDENTITY_AUTH.md), [target and configuration](03_api_specs/IDENTITY_MOBILE_TARGET.md). User requested auth UI/integration from prompt_tinh_nang_login_mobile.txt, then confirmed no authservice branch exists yet and authorized integration against the file. Its hash/provenance and remaining backend assumptions are recorded. No candidate branch was substituted. Only feature/mobile-ui writes/pushes; no merge, backend/provider configuration or deployment.

## Completed task goals

AUTH-00/01 intake and target matrix complete. AUTH-02 typed immutable personas/scoped atomic session/transport implemented (6 checks). AUTH-03 browser callback/session restore/single-flight refresh/logout/registration/password commands implemented (18 checks). AUTH-04 Identity login/register/account/password UI and Employee/inactive purchase UI+repository guards implemented (4 checks, phone393x852). AUTH-05 full regression/analyze/build/native callback/standard APK delivery complete locally. Actual backend/IdP end-to-end acceptance is separate and NOT VERIFIED.

Identity is opt-in for backend mode: AUTH_MODE=identity, explicit IDENTITY_BASE_URL and approved IDENTITY_AUTHORIZATION_ORIGINS; providers enabled only via configuration. Default M5 local and ZIP backend cohorts remain usable. Existing ZIP commerce adapters receive the new Bearer source in Identity mode; no unsupported server contract migration is claimed. Account phone is not fabricated; header falls back to email. Recovery is delegated to Identity login, registration success is distinct from authentication.

Security boundaries: unknown/missing persona never defaults to active Customer; Customer roles empty; Employee/non-active purchases rejected in UI and cart/checkout repositories. Credentials live in a separate scoped secure-storage record with Android namespace/resetOnError:false. BFF cookie only to Identity transport; Bearer only to configured gateway paths, no redirects or automatic mutation replay. Do not release stored Bearer until current persona is verified, and recheck actor/epoch after async credential reads. Logout prevents late callback/refresh/credential-read resurrection. Registration keeps an unchanged retry key while mounted, with no persisted password body.

## Final verification and artifact

- Application source3108d37: full **229/229 tests PASS63s**, **flutter analyze PASS12.2s** (no issues), **debug APK PASS213.2s**.28 new checks (6 foundation+18 transport/lifecycle+4 domain/UI). Final credential-source fix was first reproduced by a failing regression, then verified; no assertions/lints disabled.
- Android10 LM-V409N, LMV409N70de1ce3: synthetic native system-browser/plugin callback channel PASS; wrong callback path resolves no activity, exact omama://auth/callback resolves plugin CallbackActivity. This uses synthetic SDK-injected callback and no credentials/backend/storage; it does not prove a live IdP redirect flow, PKCE verifier or natural browser-tab cleanup.
- Latest standard APK install-r Success; final stopped-process startup returns to resumed com.example.mobile_app/.MainActivity. LG am-start timing briefly reported restored Chrome/launcher activity, so no cold-start performance metric is claimed. Cleared only the app's task stack to remove the synthetic probe tab; no app data or browser process was cleared. Final PID30218,178 log lines: zero Flutter/AndroidRuntime crash/Unhandled/RenderFlex matches, zero probe markers; two historical ESP migration/decryption warnings remain outside scope.
- Artifact: apps/mobile-app/build/app/outputs/flutter-apk/app-debug.apk,162023338 bytes, source3108d37. SHA256 **E601A38CF2738B10ACA3C115CF10636EA779FE95AE27A797065EEBC343AF84F1**. APK default is local mode; actual Identity UI/runtime requires the documented defines and reachable backend/IdP. No endpoints/providers were configured or activated externally.
- iOS scheme source/plist XML valid; iOS native build NOT RUN. Temporary .dart_tool/m6_native_callback_smoke.dart removed. No APK/log/screenshot/env/credential artifacts committed.

## Build/runtime decisions and changes

Stable flutter_web_auth_2 5.1.0 pinned with three transitives; no existing package upgrades. Android CallbackActivity owns omama://auth/callback exclusively, MainActivity retains singleTop/legacy recovery; iOS URL scheme added. Generated registrants track plugin dependencies; no desktop auth support claimed. The current Android template opts out of built-in Kotlin/new DSL; build succeeds but Flutter warns about future KGP migration. Review before a future SDK/plugin upgrade, no prerelease upgrade applied here.

First8GB Gradle attempt was deliberately interrupted after532.8s under memory pressure; not a compiler defect. Successful builds used process-local GRADLE_OPTS additions: -Dorg.gradle.project.kotlin.incremental=false -Dorg.gradle.project.kotlin.compiler.execution.strategy=in-process -Dorg.gradle.jvmargs=-Xmx2048m -Dorg.gradle.workers.max=2. Run expensive checks sequentially on this8GB host. Only task-owned build/analyze processes were stopped and idle Gradle memory released; project Gradle properties/SDK unchanged.

Modified areas: auth domain/adapters/storage/Cubit/pages; factory/shared API credential source; account/product/cart/checkout guards; Android/iOS callback config; dependency lock/generated registrants; focused tests; AGENTS/master/M6/decisions/parity matrix/target notes/handoff. Separate untracked docs/MOBILE_MVP_DISCUSSION_REFERENCE.md (Issue38) remains untouched/uncommitted. M5 local vault and its historical evidence remain intact; business collections are still process-local.

## Remaining limits / exact next action

No M6 implementation work remains against the supplied target. Need actual Identity/gateway/IdP endpoint configuration or backend source/contracts to validate cookie/PKCE ownership, refresh/logout/password invalidation, registration fingerprint/provisioning, enabled providers and service/capability/profile compatibility. Existing editable profile/commerce contracts remain ZIP, not silently mapped to missing Identity endpoints. Real email recovery/provider processing and backend login remain unverified. Real sign-in/OTP is user-controlled; don't collect real credentials or create production accounts automatically.

Next session: read AGENTS/current/master/M6, verify current HEAD/status and final remote checkpoint. When user supplies actual backend scope/endpoints, validate the recorded target assumptions first, then configure the opt-in cohort and perform a bounded user-controlled sign-in acceptance. No broad UI reread/device loops, merges or provider activation inferred.

Goal-tool limitation: create_goal was attempted but rejected because the old parity goal is unfinished/usageLimited. Per-task goals/outcomes are durable in M6; no native task goals were fabricated, and the broader old goal was not falsely completed. Runtime exposes no usage percentage (remainingTokens:null). User rule remains: if usage is reported below13% or a low-usage warning appears, checkpoint/push verified progress and wait for command. No numeric threshold crossing was observed or invented in this run.
