# Customer Mobile — current handoff

Updated2026-11-04. Branch feature/mobile-ui, entry HEAD63acfd6ef674a93d5163b5c7f3436f3ef4d10613 (M5 delivered). M6 implementation is currently uncommitted and a source checkpoint follows this update. Active plan: docs/plans/active/M6_IDENTITY_AUTH.md; master: docs/plans/MASTER_PLAN.md; target/config: docs/03_api_specs/IDENTITY_MOBILE_TARGET.md.

## Task / authority

User requested Mobile auth UI/integration using prompt_tinh_nang_login_mobile.txt, per-task plan/goals/implementation, and checkpoint+wait if reported usage<13%. Source file read fully and hashed in plan. Remote has no authservice branch; user explicitly answered no branch exists yet, integrate anyway. Supplied file is the target contract; no backend source/provider acceptance claimed. Work/commit/push only feature/mobile-ui; no merges/backend edits/provider configuration. Preserve separate untracked docs/MOBILE_MVP_DISCUSSION_REFERENCE.md (Issue38).

## Completed implementation / active task

AUTH-00 intake and AUTH-01 target matrix COMPLETE. AUTH-02 persona/atomic scoped session/Identity transport IMPLEMENTED (6 tests PASS). AUTH-03 injectable system-browser login/callback, BFF session restore, single-flight refresh, logout, registration/password commands and factory selection IMPLEMENTED (12 tests PASS). AUTH-04 separate Identity email/name registration/browser login, account/password UI and Employee/inactive purchase guards IMPLEMENTED (4 domain/widget tests PASS). AUTH-05 CURRENT: full regression/build/native acceptance pending.

Identity is opt-in for remote backend with AUTH_MODE=identity, explicit IDENTITY_BASE_URL and approved IDENTITY_AUTHORIZATION_ORIGINS. Local M5 and ZIP remote cohort remain. Native browser plugin5.1.0 pinned (three transitives), no other package upgrades. Android plugin CallbackActivity exclusively handles omama://auth/callback, MainActivity keeps historical link/singleTop; iOS scheme source added. Generated platform registrants reflect the plugin; content-identical header noise restored.

Persona missing/unknown type/status never grants Customer purchase; Customer roles must be empty. Gateway bearer is origin/path-scoped, no BFF cookie and no redirects or mutation replay. Dedicated Identity transport uses JSON envelope/cookie only to its configured origin; session is one scoped secure-storage record, independent Android namespace/resetOnError:false. Browser origin/state/S256/redirect/expiry/callback validated, stale auth after logout ignored. Registration acknowledged separately from login; retry key stable for unchanged mounted-page body; no password-body persistence. Existing password policy retained until actual backend policy exists.

## Validation / changes

22 new focused checks PASS (6 foundation,12 repository/transport,4 domain/UI). Final analyze PASS14.6s/no issues; full223-test regression/build still required. Initial checkbox/async fixture harness failures diagnosed and fixed, no assertions/warnings disabled. Android LMV409N70de1ce3 is attached. Do not confuse prior M5 201-test/APK/native results with M6 acceptance.

Modified: mobile auth domain/data/Cubit/pages, factory, shared API credential source, account/product/cart/checkout purchase UI guards, Android/iOS callback setup, pubspec/lock/generated plugin registrants, focused tests; M6/master/decisions/target notes/handoff. No secrets/provider keys/temporary test credentials in committed source. Fixture token strings only in tests.

## Blockers / exact next action

Native goal creation was attempted but rejected because the existing parity goal is unfinished/usageLimited; durable per-task goals are recorded in M6, no new native goal claimed. No account/quota percentage is exposed (remainingTokens:null); never invent usage. User threshold<13% supersedes prior10%; checkpoint and wait if runtime reports that threshold/low-usage warning.

Next: source checkpoint Mobile only, final analyze and full tests; diagnose introduced failures, then debug APK using process-local GRADLE_OPTS Kotlin workaround. Bounded native plugin callback smoke may use synthetic local fixtures and MemoryIdentitySessionStore only, preserving user account vault; real IdP sign-in remains user-controlled and NOT VERIFIED without backend. Restore standard APK/remove temporary harness afterwards. Final evidence/checkpoint/push.

M5 baseline preserved: sourcee1d7d1e, docs63acfd6; accounts/session persistent, business collections process-local; old ESP warning investigation separate. Historical APK evidence is in M5 plan. M6 target still needs real backend evidence for PKCE verifier/cookie correlation, registration fingerprint/workflow, refresh/logout/password invalidation, editable Identity profile, gateway/capability/provider integration. Existing ZIP commerce adapters receive Bearer but are not declared migrated to new unavailable contracts. No external endpoint or deployment was activated.
