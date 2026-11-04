# Identity Mobile integration target — 2026-11-04

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
| GET /public/auth/callback | code, state, format=json; transaction cookie when returned by login | accessToken, refreshToken, sessionId or Set-Cookie OMAMA_SESSION |
| GET /public/auth/session | BFF Cookie and Bearer | authenticated:boolean |
| GET /users/me | BFF Cookie and Bearer | principalId,email,firstName?,lastName?,principalType,provisioningStatus,roles,permissions |
| POST /public/auth/tokens/refresh | BFF Cookie and current Bearer; no invented raw refresh-token body | new accessToken; optional refreshToken/sessionId rotation/Set-Cookie |
| POST /public/auth/logout | BFF Cookie and Bearer | command acknowledgement |
| POST /public/customer-registrations | firstName,lastName,email,rawPassword,idempotencyKey,requestFingerprint; matching Idempotency-Key header | command acknowledgement |
| PUT /users/me/password | currentPassword,newPassword; authenticated session | command acknowledgement |

Registration fingerprint uses normalized email exactly as supplied in the brief; the UI holds a body signature and idempotency key for unchanged retries while the page is mounted. Keys/password bodies are not persisted for cross-process registration retry. An acknowledged registration is separate from login; the app opens browser login next and retains a continue-login path if cancelled. Existing15–128-character registration/change-password UI validation remains until backend policy is supplied. No phone is invented when /users/me has none; account header uses email.

## Security and integration assumptions

Backend is responsible for PKCE verifier, single-use state and code exchange. Mobile requires an approved authorization origin, code response, exact redirect URI, nonempty state and S256 challenge before opening the system browser. It compares exact callback scheme/host/path/state, rejects duplicate critical parameters/error/fragment and stale transactions (>10minutes), exchanges once per pending flow, and does not use embedded password login/client secrets. This does not prove server RFC8252 compliance or browser/BFF cookie correlation. Never log URLs containing code/state, response credentials or password payloads.

Identity session is one versioned secure-storage record, scoped to configured Identity+gateway environment, with a separate Android namespace and resetOnError:false. Save before publishing identity. Logout invalidates in-flight auth, clears local credentials first, then attempts server revocation; a revocation failure is reported. Transient refresh failure retains recovery material;401 clears it. Refresh is single-flight, and failed business mutations are never replayed automatically. Only Bearer reaches the configured gateway /api/v1 paths; the BFF cookie stays in Identity transport. No arbitrary cross-origin credential forwarding.

Purchase requires an explicitly ACTIVE CUSTOMER with empty roles. Missing/unknown fields never default to Customer/ACTIVE; Employee and non-active users see restrictions. Cart add/update and checkout repositories reject forbidden writes even if a caller bypasses widgets. These client checks supplement required server authorization. Customer capabilities do not override persona restrictions.

Out of contract: real recovery endpoint, editable Identity profile, loyalty/address/provider capabilities, session invalidation semantics after password change, exact registration/provisioning status workflow and gateway acceptance across services. Existing ZIP commerce adapters are retained with an Identity Bearer credential source, not declared migrated to an unavailable new backend. These integration questions require backend evidence before end-to-end acceptance. Password recovery is reached through the Identity login page; no ZIP reset endpoint is called for Identity credentials.
