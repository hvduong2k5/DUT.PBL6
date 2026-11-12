# ADR-001: Customer API routes resolve and renew the OIDC credential through one server-side boundary

- Status: Accepted (implemented with this change)
- Scope: `apps/web-user` — customer API routes (`/api/capabilities`, `/api/customer`,
  `/api/customer-core`, `/api/payments`, `/api/returns`, `/api/support`, `/api/b2b`)
- Modes: `WEB_AUTH_MODE=oidc` is the deployed production mode; `identity-bff` remains
  supported; `legacy` remains local-development only.

## Context

The customer handlers read their credential with `getTrustedSessionCookie()`, which only
knows the identity-BFF session cookie (`OMAMA_SESSION`). The OIDC login flow never sets
that cookie — it sets `oma_access_token` (900 s) and `oma_refresh_token` — so in the
deployed OIDC mode every handler saw "no credential" inside a live session:
`/api/capabilities` answered `AUTH_REQUIRED`, and the account pages rendered
"capability denied" even though the backend projection was healthy.

Two further defects compounded this:

1. Handlers used the raw access cookie with no renewal path, so even with the right
   cookie name they would have failed 15 minutes after login. Only `/api/auth/me`
   renewed, via `oidcSessionResponse()`.
2. The session provider mapped every 401 and 403 from the capability check to
   "forbidden", so a session failure was presented to the user as a backend
   authorization denial.

## Decision

### 1. One credential boundary: `src/lib/auth/customer-auth.ts`

- `withCustomerAuth(request, run)` wraps each customer handler. It resolves the
  credential **before** the handler runs, so proxies forward the fresh token, and
  finalizes the response afterwards.
- OIDC mode: a still-valid access cookie (verified, more than 30 s from expiry — the
  same margin `oidcSessionResponse()` uses) is used as-is. Otherwise the session is
  refreshed through the existing `refreshOidc()` (in-flight de-duplication, token
  re-verification, same-subject enforcement all reused, not reimplemented). Refresh
  tokens are never forwarded upstream.
- identity-BFF mode: the `OMAMA_SESSION` cookie passes through unchanged — no
  refresh, no cookie writes.
- Handlers pass the resolved credential to the existing session/capability checks
  and use the existing `credentialHeaders()` to forward it upstream
  (`Authorization: Bearer` in OIDC mode, the session cookie in BFF mode).
- A request with no session material at all resolves to `credential: undefined`;
  the downstream checks keep producing their own `AUTH_REQUIRED`, preserving the
  previous guest behaviour (e.g. guest support context).

### 2. Response-finalisation contract

When a refresh rotated the tokens, the rotated cookies must reach the browser on
**every** handler response, including proxied and streaming ones:

- The handler's response is never mutated in place — a proxied `fetch` response can
  carry immutable headers. Instead the cookies are attached through a fresh
  `NextResponse` envelope around the original body stream.
- The body is never consumed, buffered or re-encoded; status, status text and
  headers are carried over; empty-body statuses (204/304) are handled explicitly.
- Each cookie is emitted as its own `Set-Cookie` header, never comma-joined.
- If finalisation itself fails, the original response is returned untouched and the
  failure is logged without any token or cookie values; the next request then
  surfaces `SESSION_EXPIRED` and asks for a fresh sign-in.

### 3. Typed refresh failures

| Situation | Status | Code |
| --- | --- | --- |
| No session material / no refresh token | 401 | `AUTH_REQUIRED` |
| Identity provider rejects the refresh token | 401 | `SESSION_EXPIRED` |
| Sealed session unusable, subject changed, tokens no longer verify | 401 | `SESSION_INVALID` |
| Identity provider timeout / 5xx / unusable token response | 503 | `AUTH_PROVIDER_UNAVAILABLE` |
| Broken server OIDC configuration | 500 | `AUTH_CONFIGURATION_ERROR` |

A provider outage never clears the session: no cookies are deleted on the 503 path,
so the user can simply retry.

Upstream results are not reclassified either: handlers keep relaying upstream
statuses; in particular an upstream 403 stays a 403 authorization denial (it may be
a legitimate per-user denial), and a capability projection failure keeps its
existing mapping.

### 4. Client contract (`customer-session-provider`)

The provider maps failures through `capabilityFailureAction(status, code)`:

- 403 → `forbidden` (a genuine denial).
- 401 with `AUTH_REQUIRED` / `SESSION_EXPIRED` / `SESSION_INVALID` → session
  failure: the provider re-probes the auth session once; if the session really
  ended, the auth layer flips to guest and the account guard redirects to login.
- Any other 401 → generic error, logged with its code. A bare 401 is **not** proof
  the session ended and must never render as "forbidden" or silently log the user
  out (a proxied upstream 401 would otherwise create login loops).
- A single failed capability request never clears the authenticated state by
  itself.

## Known limitations

- **Refresh de-duplication scope: one Node.js process.** The in-flight refresh map
  in `oidc.ts` is process memory. The web app is deployed as a single container, so
  this is exact today; scaling to multiple replicas would need the de-duplication
  (or the session) rethought, because parallel refreshes with one refresh token
  can race at the provider.
- **Staff surface unchanged.** The employee resolver
  (`features/employee/auth/server.ts`) and the admin analytics route authenticate
  against the identity-BFF session contract (`/public/auth/session` with the
  `OMAMA_SESSION` cookie) and therefore remain BFF-mode only. Giving staff an OIDC
  path is a separate design (the Bearer-based plumbing already exists in
  `identitySession()`), tracked as follow-up work — it is deliberately not folded
  into this fix.
- Whether OIDC or identity-BFF is the long-term production mode is an architecture
  decision for the team to record; this ADR only documents that OIDC is the mode
  the live deployment runs and that both modes stay supported.

## Alternatives considered

- **Next.js middleware** performing the refresh: rejected. The refresh machinery
  (AES-256-GCM sealed cookies, remote JWKS verification) is Node-runtime code that
  the Edge middleware runtime cannot reuse, and a middleware response is not the
  handler's response, so cookie propagation onto streamed proxy responses is
  unproven.
- **Switching the deployment back to identity-BFF mode**: rejected for this fix.
  It would re-open the live-domain proof (return-URL allowlist, cookie domain)
  that OIDC mode has already passed, to avoid fixing the actual defect.
- **Raising the access-token TTL**: rejected. It weakens a deliberate security
  posture and only postpones the failure.
- **Client-side refresh timer**: rejected. The access cookie is `httpOnly`; the
  client cannot rotate it, and timers do not fire reliably for background tabs.
