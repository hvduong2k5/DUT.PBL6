# Identity Mobile integration target — 2026-11-04

Current correction slice: backend source is now available at dev a24492b (see [audit](IDENTITY_SOURCE_AUDIT_2026-11-04.md)). Flutter now requires the login response's OMAMA_AUTH_STATE cookie to match the authorization state, keeps it only in the pending login call, and sends it only to callback. Session GET/refresh/logout use OMAMA_SESSION without Bearer, allowing recovery with an expired access token. Profile and gateway still use verified Bearer. GET session public correlation IDs never replace stored cookie credentials. Registration requires ACTIVE/COMPLETED; PENDING shows retry feedback with unchanged idempotency identity, unknown/missing status fails closed. Backend derives the HMAC fingerprint; Flutter no longer sends the brief's email-as-fingerprint. Live activation remains deferred; backend password/Gateway/social issues require user approval before any work outside Mobile.

Authority: user supplied prompt_tinh_nang_login_mobile.txt (SHA2567FBAB4545048818EDDF3D14BFED5404EE96909D83AAA9D95203A234FD620375C). User confirmed no authservice branch exists yet and authorized integration against this file. This is an implementation target with fixture-backed tests, not a verified backend/OpenAPI snapshot. Original ZIP provenance is unchanged. See [M6 task goals/status](../plans/active/M6_IDENTITY_AUTH.md).

## Runtime selection

Default remains M5 local accounts. Existing backend/mockoon without AUTH_MODE=identity keeps ZIP auth. To opt in, provide these Dart defines:

| Define | Meaning |
|---|---|
| DATA_SOURCE=backend | Use remote commerce adapters |
| AUTH_MODE=identity | Use Identity browser login and session adapter |
| API_BASE_URL=https://gateway.example.invalid | Gateway origin for existing commerce APIs; Bearer only |
| IDENTITY_BASE_URL=https://identity.example.invalid/api/v1 | Exact BFF API base (may be gateway /api/v1); Cookie only to this configured transport |
| IDENTITY_AUTHORIZATION_ORIGINS=https://sso.example.invalid | Comma-separated exact approved authorization origins; no wildcard or URL learned from an untrusted response |
| IDENTITY_PROVIDERS=google | Optional enabled providers, google and/or zalo; omit until configured server-side |
| IDENTITY_ALLOW_HTTP=true | Explicit local-development opt-in only; never use for production |

Example hosts above are deliberately non-routable placeholders. Configure real development endpoints before attempting sign-in. For Android emulator use an accessible host (often10.0.2.2); a USB phone needs a reachable LAN endpoint or an intentionally configured adb reverse mapping. API URLs/origins are not secrets. No actual backend, reverse mapping, IdP client, provider or deployment was configured by this work.

Stable flutter_web_auth_2 **5.1.0** is pinned, with four added packages including three transitives. Existing packages are not downgraded/upgraded. Current Android template already opts out of AGP9 built-in Kotlin/new DSL; build validation determines compatibility. Primary package setup: https://pub.dev/packages/flutter_web_auth_2/versions/5.1.0. Android callback is handled exclusively by the plugin CallbackActivity for **omama://auth/callback**; MainActivity keeps singleTop and its separate historical recovery link. This avoids two activities competing for the same callback. iOS URL scheme added; iOS build/native acceptance requires an Apple host.

## Contract used by the adapter

All paths below are appended once to /api/v1. Response envelope is success:true, data:object (data may be null for command acknowledgements). HTTP errors are mapped separately; a malformed envelope never becomes an authenticated state.

| Method/path | Input | Expected success data |
|---|---|---|
| GET /public/auth/login | redirect_uri=omama://auth/callback, format=json, optional enabled provider | authorizationUrl |
| GET /public/auth/callback | code, state, format=json; required OMAMA_AUTH_STATE from login | native accessToken, refreshToken, sessionId or Set-Cookie OMAMA_SESSION |
| GET /public/auth/session | BFF Cookie only | authenticated:boolean; public sessionId is not a credential |
| GET /users/me | BFF Cookie and Bearer | principalId,email,firstName?,lastName?,principalType,provisioningStatus,roles,permissions |
| POST /public/auth/tokens/refresh | BFF Cookie only; no raw refresh-token body or expired Bearer | new accessToken; optional refreshToken/credential sessionId rotation/Set-Cookie |
| POST /public/auth/logout | BFF Cookie only | command acknowledgement |
| POST /public/customer-registrations | firstName,lastName,email,rawPassword,idempotencyKey; matching Idempotency-Key header; fingerprint computed server-side | status ACTIVE or COMPLETED confirms registration; PENDING does not |
| PUT /users/me/password | currentPassword,newPassword; authenticated session | command acknowledgement |

The UI holds a body signature and idempotency key for unchanged retries while the page is mounted. Keys/password bodies are not persisted for cross-process registration retry. Only a completed registration opens browser login next; cancellation retains a continue-login path. Existing15–128-character registration/change-password UI validation remains until the final Keycloak policy is confirmed (DTOs have different minima, see audit). No phone is invented when /users/me has none; account header uses email.

## Security and integration assumptions

Backend is responsible for PKCE verifier, single-use state and code exchange. Mobile requires an approved authorization origin, code response, exact redirect URI, nonempty state and S256 challenge before opening the system browser. It compares exact callback scheme/host/path/state, rejects duplicate critical parameters/error/fragment and stale transactions (>10minutes), exchanges once per pending flow, and does not use embedded password login/client secrets. This does not prove server RFC8252 compliance or browser/BFF cookie correlation. Never log URLs containing code/state, response credentials or password payloads.

Identity session is one versioned secure-storage record, scoped to configured Identity+gateway environment, with a separate Android namespace and resetOnError:false. Save before publishing identity. Logout invalidates in-flight auth, clears local credentials first, then attempts server revocation; a revocation failure is reported. Transient refresh failure retains recovery material;401 clears it. Refresh is single-flight, and failed business mutations are never replayed automatically. Only Bearer reaches the configured gateway /api/v1 paths; the BFF cookie stays in Identity transport. No arbitrary cross-origin credential forwarding.

Saved Bearer is withheld from the gateway until the current persona is verified through restore/login/refresh; a failed/malformed persona restore retains recovery material without attaching credentials as an anonymous actor. Final review reproduced this gap in25c8d7a with a failing regression and fixed the access-token source. Identity's own restoration calls still use the scoped saved record to validate the session/profile.

Purchase requires an explicitly ACTIVE CUSTOMER with empty roles. Missing/unknown fields never default to Customer/ACTIVE; Employee and non-active users see restrictions. Cart add/update and checkout repositories reject forbidden writes even if a caller bypasses widgets. These client checks supplement required server authorization. Customer capabilities do not override persona restrictions.

Out of contract: real recovery endpoint, editable Identity profile, loyalty/address/provider capabilities, session invalidation semantics after password change, exact registration/provisioning status workflow and gateway acceptance across services. Existing ZIP commerce adapters are retained with an Identity Bearer credential source, not declared migrated to an unavailable new backend. These integration questions require backend evidence before end-to-end acceptance. Password recovery is reached through the Identity login page; no ZIP reset endpoint is called for Identity credentials.
