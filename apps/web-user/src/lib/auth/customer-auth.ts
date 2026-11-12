import { NextRequest, NextResponse } from "next/server";
import { applyTokens, OidcError, refreshOidc, verifiedAccess } from "./oidc";
import { usesIdentityBff, usesOidc } from "./oidc-config";
import { getTrustedAccessToken, getTrustedSessionCookie } from "./server-credential";

/**
 * Credential boundary for the customer API routes (capabilities, customer,
 * customer-core, payments, returns, support, b2b).
 *
 * Why this module exists: the handlers used to read the credential through
 * getTrustedSessionCookie(), which only knows the identity-BFF session
 * cookie (OMAMA_SESSION). In OIDC mode (WEB_AUTH_MODE=oidc, the deployed
 * mode) the login flow never sets that cookie — it sets oma_access_token /
 * oma_refresh_token — so every handler saw "no credential" inside a live
 * session and the account surface rendered capability-denied. The handlers
 * also used the raw 900-second access token with no renewal path, unlike
 * /api/auth/me, which resolves its session through oidcSessionResponse().
 *
 * This module owns the whole lifecycle instead: resolveCustomerCredential()
 * returns the credential the handler must use (and forward upstream) plus a
 * finalize() step that attaches rotated cookies to the handler's response
 * when a refresh happened. See docs/02_architecture/adr/web-user/
 * adr_001_oidc_customer_auth.md for the contract.
 */

const ACCESS_COOKIE = "oma_access_token";
const REFRESH_COOKIE = "oma_refresh_token";
// Mirrors the safety margin oidcSessionResponse() applies before it decides
// an access token is still usable.
const REFRESH_MARGIN_SECONDS = 30;

export type CustomerAuthErrorCode =
  | "AUTH_REQUIRED"
  | "SESSION_EXPIRED"
  | "SESSION_INVALID"
  | "AUTH_PROVIDER_UNAVAILABLE"
  | "AUTH_CONFIGURATION_ERROR";

export class CustomerAuthError extends Error {
  constructor(readonly status: number, readonly code: CustomerAuthErrorCode, message: string) {
    super(message);
    this.name = "CustomerAuthError";
  }
}

export interface ResolvedCustomerAuth {
  /**
   * The credential handlers pass to the session/capability checks and
   * forward upstream. Undefined when the request carries no session
   * material at all; the downstream checks turn that into their own
   * AUTH_REQUIRED, exactly as before.
   */
  credential: string | undefined;
  /** Attach rotated session cookies to the handler response. Identity when nothing rotated. */
  finalize: (response: Response) => Response;
}

const passThroughFinalize = (response: Response): Response => response;

const CONFIGURATION_ERROR_MARKERS = [
  "Invalid WEB_AUTH_MODE",
  "OIDC is not enabled",
  "Missing OIDC configuration",
  "OIDC requires",
  "Invalid WEB_PUBLIC_URL",
  "WEB_PUBLIC_URL must be an origin"
];

/**
 * Maps an OIDC-layer failure onto the public error contract:
 * - the identity provider rejecting the refresh token means the session is
 *   over (SESSION_EXPIRED);
 * - a session that cannot be unsealed, or tokens that no longer verify for
 *   the same subject, are SESSION_INVALID;
 * - a provider outage is AUTH_PROVIDER_UNAVAILABLE and never clears the
 *   session — the user keeps their cookies and can retry;
 * - broken server configuration is AUTH_CONFIGURATION_ERROR.
 */
export function classifyCustomerAuthFailure(cause: unknown): CustomerAuthError {
  if (cause instanceof CustomerAuthError) return cause;
  if (cause instanceof OidcError) {
    if (cause.status === 503 || cause.code === "OIDC_PROVIDER_UNAVAILABLE" || cause.code === "OIDC_TOKEN_RESPONSE_INVALID") {
      return new CustomerAuthError(503, "AUTH_PROVIDER_UNAVAILABLE", "Máy chủ xác thực đang tạm gián đoạn. Phiên đăng nhập vẫn được giữ nguyên, vui lòng thử lại sau ít phút.");
    }
    if (cause.code === "AUTH_REQUIRED") {
      return new CustomerAuthError(401, "AUTH_REQUIRED", "Vui lòng đăng nhập để sử dụng chức năng này.");
    }
    if (cause.code === "OIDC_SESSION_INVALID") {
      return new CustomerAuthError(401, "SESSION_EXPIRED", "Phiên đăng nhập đã hết hạn. Vui lòng đăng nhập lại.");
    }
    return new CustomerAuthError(401, "SESSION_INVALID", "Phiên đăng nhập không còn hợp lệ. Vui lòng đăng nhập lại.");
  }
  if (cause instanceof Error && CONFIGURATION_ERROR_MARKERS.some((marker) => cause.message.includes(marker))) {
    return new CustomerAuthError(500, "AUTH_CONFIGURATION_ERROR", "Cấu hình xác thực trên máy chủ chưa hoàn chỉnh. Vui lòng liên hệ quản trị viên.");
  }
  return new CustomerAuthError(503, "AUTH_PROVIDER_UNAVAILABLE", "Chưa thể xác minh phiên đăng nhập lúc này. Vui lòng thử lại sau ít phút.");
}

type RefreshedSession = Awaited<ReturnType<typeof refreshOidc>>;

function finalizeRefreshedResponse(original: Response, refreshed: RefreshedSession): Response {
  try {
    // The original response may be a proxied fetch response whose headers
    // are immutable, so rotated cookies are attached through a fresh
    // envelope around the original body stream — the body is never consumed
    // or buffered here, and applyTokens emits one Set-Cookie header per
    // cookie instead of a combined value.
    const envelope = new NextResponse(original.body, {
      status: original.status,
      statusText: original.statusText,
      headers: new Headers(original.headers)
    });
    applyTokens(envelope, refreshed.tokens, refreshed.session);
    return envelope;
  } catch {
    // Never log token or cookie values. Degrading to the original response
    // keeps the current request working; the next one will surface
    // SESSION_EXPIRED and ask for a fresh sign-in.
    console.error("customer-auth: unable to attach refreshed session cookies to the response");
    return original;
  }
}

async function resolveOidcCustomerAuth(request: NextRequest): Promise<ResolvedCustomerAuth> {
  const access = request.cookies.get(ACCESS_COOKIE)?.value;
  if (access) {
    try {
      const verified = await verifiedAccess(access);
      const remainingSeconds = (verified.exp ?? 0) - Math.floor(Date.now() / 1000);
      if (remainingSeconds > REFRESH_MARGIN_SECONDS) return { credential: access, finalize: passThroughFinalize };
    } catch {
      // An unverifiable or nearly expired access token falls through to the
      // refresh path instead of failing the request outright.
    }
  }
  if (!access && !request.cookies.get(REFRESH_COOKIE)?.value) {
    // No session material at all: stay anonymous and let the downstream
    // checks produce their usual AUTH_REQUIRED.
    return { credential: undefined, finalize: passThroughFinalize };
  }
  let refreshed: RefreshedSession;
  try {
    refreshed = await refreshOidc(request);
  } catch (cause) {
    throw classifyCustomerAuthFailure(cause);
  }
  return {
    credential: refreshed.tokens.access_token,
    finalize: (response) => finalizeRefreshedResponse(response, refreshed)
  };
}

export async function resolveCustomerCredential(request: NextRequest): Promise<ResolvedCustomerAuth> {
  if (usesOidc()) return resolveOidcCustomerAuth(request);
  if (usesIdentityBff()) return { credential: getTrustedSessionCookie(request), finalize: passThroughFinalize };
  return { credential: getTrustedAccessToken(request), finalize: passThroughFinalize };
}

/**
 * Route adapter: resolves (and when needed refreshes) the customer
 * credential before the handler runs, so proxies forward the fresh token,
 * then finalizes the handler's response with any rotated cookies. Refresh
 * tokens are never forwarded upstream. Resolution failures are answered
 * directly with the typed error contract above.
 */
export async function withCustomerAuth(
  request: NextRequest,
  run: (credential: string | undefined) => Promise<Response>
): Promise<Response> {
  let resolved: ResolvedCustomerAuth;
  try {
    resolved = await resolveCustomerCredential(request);
  } catch (cause) {
    const failure = classifyCustomerAuthFailure(cause);
    return NextResponse.json({ code: failure.code, message: failure.message }, { status: failure.status });
  }
  const response = await run(resolved.credential);
  return resolved.finalize(response);
}
