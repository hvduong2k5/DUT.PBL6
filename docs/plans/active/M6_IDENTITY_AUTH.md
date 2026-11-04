> Tích hợp tiếp theo ngày 04/11/2026 trong PR #46: backend Identity #48 + Web #49 + Catalog #19 và Mobile đã nối cùng API MVP. Các checkpoint bên dưới là lịch sử của nhánh Mobile; trạng thái nghiệm thu tích hợp hiện tại và giới hạn thiết bị nằm tại [runbook MVP](../../06_operations/mvp_delivery.md).

# M6 — Mobile Identity authentication

Approved 2026-11-04: use the supplied login requirements, integrate authentication from the user's authservice branch, update Mobile UI, plan/implement task by task, create a goal per task, checkpoint and wait when usage is reported below13%. LARGE / auth+session+native callback change. Work/commit/push only feature/mobile-ui; no merges, backend writes, production deployment or provider configuration.

## Goal and dependencies

Replace the selected backend cohort's ZIP password-login flow with the verified Identity browser/callback/session flow, retaining usable M5 local accounts and existing shopping architecture. Adapt login/register/account and Customer purchase eligibility to actual personas. Completed M5 source e1d7d1e/delivery63acfd6 is the entry baseline;201 tests/build/native evidence belongs to M5 until rerun.

Input: D:/Dyland/Downloads/prompt_tinh_nang_login_mobile.txt, SHA2567FBAB4545048818EDDF3D14BFED5404EE96909D83AAA9D95203A234FD620375C. Read in full. Its requested behavior is accepted through the user's instruction; sample implementations/version numbers/server assertions are reference material to verify against the chosen backend, not permission to deploy/configure providers or downgrade packages.

### Source lookup and user resolution

2026-11-04 git ls-remote --heads origin (https://github.com/hvduong2k5/DUT.PBL6.git) has no authservice branch, including a case-insensitive auth/identity search. Related heads: codex/identity-order-claims988ae330, codex/identity-owner-mvp54c26fa4, codex/order-keycloak-authc0417748, codex/web-identity-gatewayb3735cba. No candidate selected/fetched as authoritative. Requested exact branch/commit/PR from user. Existing origin/dev is7f22226 and has no services/identity-service subtree; do not substitute it for the missing branch. Integration/wiring depends on this answer. Independent domain policy can be prepared from explicit persona requirements.

User answered: “chưa có, tích hợp auth vào thôi”. Source blocker resolved by scope: implement against the supplied file as the integration target, with deterministic HTTP/browser tests; no backend source or live backend acceptance is available/claimed. No candidate branch fetch required. AUTH-01 contract intake is COMPLETE against that target; server PKCE/cookie correlation, registration fingerprint policy and provider readiness remain explicit integration assumptions. Implement configurable AUTH_MODE=identity for the backend cohort; default local M5 remains usable. AUTH-02 now includes scoped atomic Identity session storage/transport; AUTH-03/04 proceed within this authorized target.

## Task goals and ordering

| Task | Goal / reviewable output | Dependencies | State |
|---|---|---|---|
| AUTH-00 | Requirements intake, current architecture audit, roadmap and explicit source gaps | Input file and M5 checkpoint | COMPLETE |
| AUTH-01 | Confirm source authority; route/DTO/status/cookie/state/PKCE/persona contract matrix | User source clarification | COMPLETE against supplied brief; backend unavailable |
| AUTH-02 | Strict immutable persona/purchase policy; isolated session persistence and transport boundaries | AUTH-01 target requirements | COMPLETE;6 focused checks PASS |
| AUTH-03 | Browser login/callback; restoration/single-flight refresh/logout; registration/change password | AUTH-01,02 and supported platform plugin | COMPLETE against target;18 focused checks PASS |
| AUTH-04 | Customer login/register/account UX; Employee/non-active purchase guards across UI and domain entry points | AUTH-03, strict user projection, existing theme | COMPLETE;4 domain/widget checks PASS at393x852 |
| AUTH-05 | Focused contract/state/navigation tests, full regression/analyze/APK, bounded Android native QA and delivery | AUTH-02–04 validated | LOCAL COMPLETE;229 tests/analyze/build/install/native callback PASS; live backend unverified |

Each row's Goal is the durable per-task goal. Native create_goal was attempted for AUTH-01 but rejected because this chat's old parity goal remains unfinished with status usageLimited. Do not mark that broader goal complete to free the slot. Record task outcomes here until the runtime permits new goals; no new native goal has been claimed. Future detail is refined only at task entry.

## AUTH-00 findings / architecture affected

- Existing factory remote path creates ApiClient+RestAuthRepository; legacy ZIP /api/v1/auth/login/register/me/refresh and snake_case DTOs differ from supplied camelCase Identity wrappers. Local default is strict LocalAuthRepository; explicit legacy local cohort remains available. No hybrid remote-auth/local-commerce mode should silently imply backend shopping acceptance.
- Existing Dio5.9, secure-storage10.3.4, uuid4.5, crypto/cryptography and cookie manager already exist. Preserve them; no downgrade to sample versions. Native browser callback plugin is absent. Select supported version after verifying current official package API/platform requirements.
- Existing cookie jar is process-memory only, token store writes tokens separately, bearer interceptor attaches without explicit destination allowlist, refresh has ZIP-specific routes, mutations are not auto-replayed. Identity integration must preserve no-automatic-write-replay and add atomic scoped credential/session storage, cookie persistence/expiry and allowed-origin separation.
- AuthUser has only customer ID/phone/name/email; AuthCubit validates phone/password and assumes register creates a session. New email/browser signup must distinguish registration success from authenticated state and handle browser cancellation after registration. Do not split a Vietnamese full name heuristically: collect firstName/lastName separately if required by backend.
- Existing capability projections GUEST/REGISTERED/B2B are separate from principalType/provisioningStatus. A capabilities fetch must not grant purchase to Employee, suspended, unknown or malformed identities. Do not infer roles from email suffix alone.
- Android MainActivity singleTop has only the legacy oma://customer/account-recovery link. Verify browser-plugin callback activity versus MainActivity routing to avoid dual handlers. iOS custom URL configuration is source-only until built on an appropriate host.

## AUTH-01 contract questions to resolve from code

| Surface in supplied file | Required verification before implementation |
|---|---|
| GET /api/v1/public/auth/login?redirect_uri=omama://auth/callback&format=json | Exact wrapper/status, allowed redirect URI/provider, authorizationUrl origin, state/verifier ownership, transaction lifetime and browser-vs-Dio cookie correlation |
| GET /api/v1/public/auth/callback?code&state&format=json | One-time state binding, required cookie, token/session response fields and expiry, malformed/error/cancel behavior; client checks exact callback scheme/host/path and expected state |
| GET /api/v1/public/auth/session; GET /api/v1/users/me | Authenticated projection vs principal projection, bearer/cookie prerequisites, ACTIVE/SUSPENDED variants, customer phone/profile availability |
| POST /api/v1/public/auth/tokens/refresh | BFF cookie versus refresh-token body, token/cookie rotation, expiry/errors; no recursion or destructive logout for transient network failure |
| POST /api/v1/public/auth/logout | Revoke semantics, CSRF/cookie requirements, local-clear failure recovery and truthful server-logout feedback |
| POST /api/v1/public/customer-registrations | Required body/header fields, rawPassword policy, requestFingerprint algorithm, status/provisioning workflow; preserve idempotency identity after uncertain result |
| PUT /api/v1/users/me/password | Supported personas, current/new password policy, session invalidation/reauthentication and response semantics |
| Google/Zalo, recovery | Actual enabled providers and recovery contract; don't expose functioning-looking options without support |

PKCE is stated as backend-owned in the brief; prove actual ownership/correlation in code before claiming RFC conformance. Never send a native client secret, log authorization URLs/code/state/token/cookies/passwords, blindly forward credentials across origins, or trust client-parsed JWT claims as backend authorization. Guard Employee purchase in domain/repository entry points as well as UI; server authorization remains required.

## AUTH-02 entry slice (completed and subsequently extended to session transport)

Goal: prepare a typed, immutable IdentityPersona and purchase eligibility policy independent of an unconfirmed backend. In scope: principal ID, customer/employee/unknown type, active/suspended/unknown provisioning, immutable roles/permissions and strict input validation against the explicitly supplied shape. Unknown or incomplete projection never defaults to active Customer; Customer with privileged roles is rejected. Optional profile names remain optional; no fabricated phone or local identity conversion. Parser is PREPARED against the supplied reference shape until AUTH-01 confirms it. No existing AuthUser/factory/UI behavior changes in this slice.

Expected files: features/auth/domain/identity_persona.dart; test/features/auth/identity_persona_test.dart. Steps: define enums and immutable value; parse required identifiers/type/status/collections; derive canPurchase only for valid active Customer with empty roles; cover Employee irrespective of roles/permissions, suspension, future/unknown values, missing/wrong fields, mutable input aliases and equality. Then focused tests/analyze, record exact outcomes and checkpoint. Native plugin/storage/transport work begins only after backend DTO verification and its own entry analysis.

Acceptance: no permissive defaults or authority from permissions/email; malformed fields fail visibly; caller mutation cannot elevate policy; tests pass; M5 runtime untouched. No APK/native claim for an unwired domain object.

## UI/UX and future validation requirements

Reuse current cream/brown/gold tokens, serif headers, existing AppTextField/AppPrimaryButton/header and return-location policy. Browser handoff exposes pending/cancel/retry; no duplicate opens. Restore distinguishes signed-out, transient connectivity and invalid session. Registration separates submitted/provisioning/authentication; no fabricated auto-login. Employee banner explains switching to Customer and disables add/cart/place-order; direct routes and domain calls must enforce the same decision. Missing/unknown persona blocks account purchase actions with recovery/retry. Preserve safe back navigation, guest shopping, keyboard, scroll, large type and terms validation. Password-change success must follow actual backend session semantics.

Focused tests cover response contracts, origin-scoped bearer/cookie handling, state mismatch/replay/expired/cancel/duplicate browser flow, single-flight refresh, logout during inflight login/refresh, corrupt/write-failing storage and no mutation retry. Then appropriate UI tests, flutter analyze, full tests and flutter build apk --debug. Native QA only resolves callback/handoff/session behaviors not established by source/tests. No automated control of authentication dialogs; real sign-in/OTP remain user-controlled.

## Checkpoint / usage discipline

User threshold is **usage below13%**, superseding prior10%. Runtime currently exposes no account/quota percentage (remainingTokens:null); do not invent a percentage or equate context with quota. Checkpoint at meaningful milestones and any runtime low-usage/availability warning; if explicit usage below13% appears, commit/push intentional Mobile/task files, verify remote, record exact next action, and wait for the user's command. Do not start the next task after that checkpoint. Preserve unrelated untracked docs/MOBILE_MVP_DISCUSSION_REFERENCE.md. No secrets, env files, generated artifacts or temporary QA inputs committed.

Exact next action (supersedes original source-blocked sequence): checkpoint implemented AUTH-02–04 with their22 passing focused checks, then full regression/analyze and APK. Use [Identity target/configuration notes](../../03_api_specs/IDENTITY_MOBILE_TARGET.md) for route assumptions and opt-in configuration. Backend remains unavailable, so native/live IdP acceptance is not claimed.

Final next action2026-11-04 supersedes the intermediate sequence above: local implementation/validation/delivery is complete; resume only with actual backend endpoint/source scope to validate assumptions and perform user-controlled live sign-in. Native callback-only evidence is recorded below; live IdP/backend acceptance remains unverified.

## AUTH-03 entry and implementation

Goal: implement the supplied route/DTO target behind AuthRepository+BrowserAuthRepository, without copying its unsafe generic interceptors/error logging. Reuse Dio and secure-storage; pin flutter_web_auth_2 5.1.0 and its platform setup, no existing dependency upgrades. Isolate native browser behind an injectable function. Validate expected origin/state/redirect/S256/callback and ten-minute lifetime before exchange; serialize credential commits, reject late callbacks after logout, restore BFF session then strict persona, refresh single-flight with no mutation replay. Persist one scoped record, await rotation before notifying UI. Register and password-change are separate commands; no direct password login through Identity adapter and no invented recovery API. Integration assumptions and primary package reference are in the target notes.

Completed: IdentityApi/IdentityAuthRepository/native browser/scoped session store; factory AUTH_MODE opt-in; origin-restricted gateway bearer (no BFF cookie), mapped failures, transient refresh retention, local-clear-first logout and delayed identity commit. Tests12 PASS: callback contract/parameter rejection, S256+origin validation, cancellation/duplicate launch, logout during handoff, storage failure, persona restore, concurrent refresh/rotation,503 vs401, idempotency separation, password-change/logout and gateway credential/no-replay boundaries. These are deterministic HTTP/browser fixtures, not backend/provider evidence.

## AUTH-04 entry and implementation

Goal: reuse current Modern Heritage tokens/components and safe return navigation for Identity login/register, keeping local/ZIP form intact. Login uses the system browser; registration collects first/last name separately and email (no fabricated phone), terms/confirmation/current password policy, stable retry identity, then browser login. Google/Zalo buttons appear only when enabled in configuration. Pending/cancel/error/retry/registration-complete-without-login states are explicit. Account header uses email if phone absent; authenticated Identity users get a password-change form. Session switch prevents stale browser success. Unknown/inactive/Employee users see a purchase warning; product/cart/checkout controls and repository guards enforce denial. Preserve existing legacy capabilities and contracts.

Completed: IdentityAuthPage/password form/purchase banner, AuthCubit browser methods, selected factory guards and shared credential source.4 domain/widget checks PASS: direct forbidden cart/checkout calls make zero repository calls; browser UI pending/cancel/retry; email/name registration→browser→account; Employee product CTA disabled. Initial test harness issues (checkbox obscured by keyboard and awaiting async transport before a widget pump) diagnosed and corrected without disabling warnings/assertions. No full redesign, provider setup, phone/OTP collection or backend deployment.

## AUTH-05 validation checkpoint and final refinement

Source25c8d7a pushed/remote verified. Full223/223 PASS69s/analyze14.6s PASS; four added lifecycle checks gave16/16 repository tests PASS. UI cases also revalidated at393x852. Source checkpoint173bc4a preserves lifecycle evidence. First Gradle attempt was stopped after532.8s under memory pressure (8GB daemon on8GB machine,~122MB free, heavy memory compression); this is an intentional build interruption, not a compiler error. Process-local -Dorg.gradle.jvmargs=-Xmx2048m -Dorg.gradle.workers.max=2 plus existing Kotlin workaround built the standard debug APK in173.1s. Stable plugin5.1.0 builds with current template opt-outs; Flutter emits a future built-in-Kotlin migration warning, recorded without upgrading SDK/plugin to a prerelease.

Native callback-only probe: ignored temporary entrypoint uses no backend or storage, system browser opens reserved example.invalid fixture URL; SDK-injected synthetic deep link returned to Flutter, M6_NATIVE_CALLBACK_CHANNEL PASS. Wrong callback path has no resolving activity; exact omama://auth/callback resolves plugin CallbackActivity. Probe built104.5s/installed12.5s; standard APK restored and installed-r Success, cold launch5789ms. Probe removed. This is platform-channel/link evidence, not real Identity login/PKCE/provider or backend acceptance. iOS plist XML is valid; Apple native build NOT RUN. Final pre-refinement analyze82.1s PASS.

Final review then added a failing regression proving stored credentials could reach the gateway before persona restore was verified. Small access-token-source fix now withholds saved bearer while currentUser is absent and rechecks actor/epoch after async storage reads; Identity restoration itself retains/uses saved credentials, so recovery is preserved. Final source regression/analyze/APK after this refinement PENDING; native callback configuration/plugin unchanged, no need to repeat that native probe. Build/install latest standard APK and remove any temporary artifacts. Native goal tool still blocked by the old unfinished goal; per-task goals remain durable here. Usage percentage remains unavailable, no sub13% value invented.

### Final AUTH-05 acceptance / handoff

The preceding PENDING statements are historical checkpoint states. Final source3108d3770882cfb4ce2de63ec6bdcf5f03c6bb2b was pushed/remote verified. Full229/229 tests63s PASS; analyze12.2s/no issues PASS; standard debug APK build213.2s PASS with bounded2GB/workers2 process-local options.28 new tests total:6 foundation,18 transport/lifecycle,4 domain/UI. UI tests use393x852. No source/test/lint assertions disabled; no further application source changes after this validation.

Latest artifact build/app/outputs/flutter-apk/app-debug.apk,162023338 bytes, SHA256 E601A38CF2738B10ACA3C115CF10636EA779FE95AE27A797065EEBC343AF84F1. Install-r Success; final app process/resumed MainActivity verified after clearing only the probe's app task stack and restarting. Initial LG am-start outputs temporarily named a restored Chrome/launcher activity; timing is not a launch-performance result. Final PID30218 log window178 lines: zero Flutter/AndroidRuntime crash/Unhandled/RenderFlex matches; two pre-existing ESP migration fallbacks, no probe markers. Ignored native harness removed, no secure-store/user data cleared. Native plugin callback configuration is unchanged from the passing synthetic channel probe; no repeat necessary. iOS native build NOT RUN (plist XML valid).

All per-task implementation goals met in the plan; actual native goal objects could not be created because of the unfinished old goal, and this tool limitation remains explicit. Backend and natural IdP redirect acceptance is NOT VERIFIED. Standard APK intentionally defaults to local mode; configure the documented Identity defines against a reachable environment to exercise actual auth UI/login. No real account, provider, secret, server policy or deployment was created/changed. Final documentation checkpoint/push follows; then user review and actual backend evidence are the next scope.
