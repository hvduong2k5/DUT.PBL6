import { createCipheriv, createDecipheriv, createHash, randomBytes, timingSafeEqual } from "node:crypto";
import { createRemoteJWKSet, jwtVerify, type JWTPayload } from "jose";
import { NextRequest, NextResponse } from "next/server";
import { oidcConfig } from "./oidc-config";
import { getSafeReturnUrl } from "./return-url";

const FLOW_COOKIE = "oma_oidc_flow";
const SESSION_COOKIE = "oma_oidc_session";
const ACCESS_COOKIE = "oma_access_token";
const REFRESH_COOKIE = "oma_refresh_token";
type Flow = { state: string; nonce: string; verifier: string; returnUrl: string; at: number };
type Session = { sub: string; nonce: string; idToken: string };
type Tokens = { access_token: string; refresh_token: string; id_token: string; expires_in: number; refresh_expires_in?: number };
type Metadata = { issuer: string; authorization_endpoint: string; token_endpoint: string; jwks_uri: string; end_session_endpoint: string };
let discoveryCache: { issuer: string; until: number; promise: Promise<Metadata> } | undefined;
const keySets = new Map<string, ReturnType<typeof createRemoteJWKSet>>();
const refreshes = new Map<string, Promise<Tokens>>();

export class OidcError extends Error {
  constructor(readonly status: number, readonly code: string) { super(code); }
}

function seal(value: unknown, purpose: string): string {
  const iv = randomBytes(12);
  const cipher = createCipheriv("aes-256-gcm", createHash("sha256").update(oidcConfig().secret).digest(), iv);
  cipher.setAAD(Buffer.from(purpose));
  const body = Buffer.concat([cipher.update(JSON.stringify(value), "utf8"), cipher.final()]);
  return Buffer.concat([iv, cipher.getAuthTag(), body]).toString("base64url");
}

function unseal<T>(raw: string | undefined, purpose: string): T {
  try {
    if (!raw || raw.length > 6000) throw new Error();
    const bytes = Buffer.from(raw, "base64url");
    const decipher = createDecipheriv("aes-256-gcm", createHash("sha256").update(oidcConfig().secret).digest(), bytes.subarray(0, 12));
    decipher.setAAD(Buffer.from(purpose));
    decipher.setAuthTag(bytes.subarray(12, 28));
    return JSON.parse(Buffer.concat([decipher.update(bytes.subarray(28)), decipher.final()]).toString()) as T;
  } catch { throw new OidcError(401, "OIDC_SESSION_INVALID"); }
}

function equal(a: string, b: string): boolean {
  const left = Buffer.from(a), right = Buffer.from(b);
  return left.length === right.length && timingSafeEqual(left, right);
}

async function boundedJSON(response: Response): Promise<Record<string, unknown>> {
  const reader = response.body?.getReader();
  if (!reader) throw new OidcError(503, "OIDC_PROVIDER_UNAVAILABLE");
  const chunks: Uint8Array[] = [];
  let size = 0;
  try {
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      size += value.byteLength;
      if (size > 1_048_576) throw new OidcError(503, "OIDC_PROVIDER_UNAVAILABLE");
      chunks.push(value);
    }
    return JSON.parse(Buffer.concat(chunks).toString("utf8")) as Record<string, unknown>;
  } finally { await reader.cancel(); }
}

async function metadata(): Promise<Metadata> {
  const config = oidcConfig();
  if (discoveryCache?.issuer === config.issuer && discoveryCache.until > Date.now()) return discoveryCache.promise;
  const promise = (async () => {
    const response = await fetch(`${config.issuer}/.well-known/openid-configuration`, { cache: "no-store", redirect: "error", signal: AbortSignal.timeout(3000) });
    if (!response.ok) throw new OidcError(503, "OIDC_PROVIDER_UNAVAILABLE");
    const value = await boundedJSON(response);
    if (value.issuer !== config.issuer) throw new OidcError(503, "OIDC_DISCOVERY_INVALID");
    for (const field of ["authorization_endpoint", "token_endpoint", "jwks_uri", "end_session_endpoint"]) {
      const endpoint = new URL(String(value[field]));
      if (endpoint.origin !== new URL(config.issuer).origin || endpoint.username || endpoint.password || endpoint.hash) throw new OidcError(503, "OIDC_DISCOVERY_INVALID");
    }
    return value as unknown as Metadata;
  })();
  discoveryCache = { issuer: config.issuer, until: Date.now() + 300_000, promise };
  try { return await promise; } catch (error) { discoveryCache = undefined; throw error; }
}

async function verify(raw: string, audience: string, required: string[]): Promise<JWTPayload> {
  if (typeof raw !== "string" || raw.length > 8192) throw new OidcError(401, "OIDC_TOKEN_INVALID");
  const config = oidcConfig();
  const doc = await metadata();
  let keys = keySets.get(doc.jwks_uri);
  if (!keys) {
    keys = createRemoteJWKSet(new URL(doc.jwks_uri), { timeoutDuration: 3000, cacheMaxAge: 300_000 });
    keySets.set(doc.jwks_uri, keys);
  }
  try {
    const payload = (await jwtVerify(raw, keys, { issuer: config.issuer, audience, algorithms: ["RS256"], requiredClaims: ["sub", "iat", "exp", ...required], clockTolerance: 5 })).payload;
    if (typeof payload.iat !== "number" || payload.iat > Date.now() / 1000 + 5 || !payload.sub || typeof payload.exp !== "number" || payload.exp <= payload.iat) throw new Error("Invalid token lifetime");
    return payload;
  } catch { throw new OidcError(401, "OIDC_TOKEN_INVALID"); }
}

export async function verifiedAccess(raw: string): Promise<JWTPayload> {
  const payload = await verify(raw, oidcConfig().audience, ["user_id", "typ"]);
  if (payload.typ !== "Bearer" || typeof payload.user_id !== "string" || !/^[0-9a-f]{8}-(?:[0-9a-f]{4}-){3}[0-9a-f]{12}$/u.test(payload.user_id) || payload.user_id === "00000000-0000-0000-0000-000000000000") throw new OidcError(401, "OIDC_TOKEN_INVALID");
  return payload;
}

async function exchange(values: Record<string, string>): Promise<Tokens> {
  const doc = await metadata();
  const response = await fetch(doc.token_endpoint, { method: "POST", headers: { "Content-Type": "application/x-www-form-urlencoded" }, body: new URLSearchParams({ client_id: oidcConfig().clientId, ...values }), redirect: "error", cache: "no-store", signal: AbortSignal.timeout(5000) });
  if (!response.ok) throw new OidcError(response.status < 500 ? 401 : 503, response.status < 500 ? "OIDC_SESSION_INVALID" : "OIDC_PROVIDER_UNAVAILABLE");
  const value = await boundedJSON(response);
  if (typeof value.access_token !== "string" || typeof value.refresh_token !== "string" || typeof value.id_token !== "string" || typeof value.expires_in !== "number" || value.expires_in < 1 || value.expires_in > 86400 || String(value.token_type).toLowerCase() !== "bearer") throw new OidcError(503, "OIDC_TOKEN_RESPONSE_INVALID");
  return value as unknown as Tokens;
}

function cookieOptions() { return { httpOnly: true, sameSite: "lax" as const, secure: oidcConfig().secure, path: "/" }; }

function applyTokens(response: NextResponse, tokens: Tokens, session: Session) {
  response.cookies.set(ACCESS_COOKIE, tokens.access_token, { ...cookieOptions(), maxAge: Math.floor(tokens.expires_in) });
  response.cookies.set(REFRESH_COOKIE, tokens.refresh_token, { ...cookieOptions(), maxAge: Math.min(tokens.refresh_expires_in ?? 28800, 28800) });
  response.cookies.set(SESSION_COOKIE, seal(session, SESSION_COOKIE), { ...cookieOptions(), maxAge: 28800 });
  response.headers.set("Cache-Control", "no-store");
}

export function clearOidc(response: NextResponse) {
  for (const name of [ACCESS_COOKIE, REFRESH_COOKIE, SESSION_COOKIE, FLOW_COOKIE]) response.cookies.set(name, "", { ...cookieOptions(), maxAge: 0 });
  response.headers.set("Cache-Control", "no-store");
}

export function trustedOidcOrigin(request: NextRequest): boolean { return request.headers.get("origin") === oidcConfig().origin; }

export async function oidcLogin(request: NextRequest) {
  const config = oidcConfig();
  const doc = await metadata();
  const flow: Flow = { state: randomBytes(32).toString("hex"), nonce: randomBytes(32).toString("hex"), verifier: randomBytes(48).toString("base64url"), returnUrl: getSafeReturnUrl(request.nextUrl.searchParams.get("returnUrl") ?? undefined), at: Date.now() };
  const url = new URL(doc.authorization_endpoint);
  for (const [key, value] of Object.entries({ client_id: config.clientId, redirect_uri: `${config.origin}/api/auth/oidc/callback`, response_type: "code", scope: "openid", state: flow.state, nonce: flow.nonce, code_challenge_method: "S256", code_challenge: createHash("sha256").update(flow.verifier).digest("base64url") })) url.searchParams.set(key, value);
  const response = NextResponse.redirect(url);
  response.cookies.set(FLOW_COOKIE, seal(flow, FLOW_COOKIE), { ...cookieOptions(), maxAge: 300 });
  response.headers.set("Cache-Control", "no-store");
  return response;
}

export async function oidcCallback(request: NextRequest) {
  const config = oidcConfig();
  const flow = unseal<Flow>(request.cookies.get(FLOW_COOKIE)?.value, FLOW_COOKIE);
  const state = request.nextUrl.searchParams.get("state") ?? "";
  const code = request.nextUrl.searchParams.get("code");
  const issuer = request.nextUrl.searchParams.get("iss");
  if (typeof flow.state !== "string" || !equal(state, flow.state) || !Number.isFinite(flow.at) || Date.now() - flow.at > 300_000 || flow.at > Date.now() + 5000 || !code || code.length > 4096 || issuer !== null && issuer !== config.issuer) throw new OidcError(401, "OIDC_CALLBACK_INVALID");
  const tokens = await exchange({ grant_type: "authorization_code", code, code_verifier: flow.verifier, redirect_uri: `${config.origin}/api/auth/oidc/callback` });
  const id = await verify(tokens.id_token, config.clientId, ["nonce"]);
  const access = await verifiedAccess(tokens.access_token);
  if (id.nonce !== flow.nonce || id.sub !== access.sub || id.azp !== undefined && id.azp !== config.clientId || Array.isArray(id.aud) && id.aud.length > 1 && id.azp !== config.clientId) throw new OidcError(401, "OIDC_TOKEN_INVALID");
  const response = NextResponse.redirect(new URL(getSafeReturnUrl(flow.returnUrl), config.origin));
  applyTokens(response, tokens, { sub: id.sub!, nonce: flow.nonce, idToken: tokens.id_token });
  response.cookies.set(FLOW_COOKIE, "", { ...cookieOptions(), maxAge: 0 });
  return response;
}

export async function refreshOidc(request: NextRequest): Promise<{ tokens: Tokens; session: Session }> {
  const refresh = request.cookies.get(REFRESH_COOKIE)?.value;
  const previous = unseal<Session>(request.cookies.get(SESSION_COOKIE)?.value, SESSION_COOKIE);
  if (!refresh) throw new OidcError(401, "AUTH_REQUIRED");
  const key = createHash("sha256").update(refresh).digest("hex");
  let promise = refreshes.get(key);
  if (!promise) {
    promise = exchange({ grant_type: "refresh_token", refresh_token: refresh });
    refreshes.set(key, promise);
    void promise.then(() => { setTimeout(() => refreshes.delete(key), 5000).unref(); }, () => { refreshes.delete(key); });
  }
  const tokens = await promise;
  const id = await verify(tokens.id_token, oidcConfig().clientId, []);
  const access = await verifiedAccess(tokens.access_token);
  if (id.sub !== previous.sub || access.sub !== previous.sub || id.nonce !== undefined && id.nonce !== previous.nonce) throw new OidcError(401, "OIDC_TOKEN_INVALID");
  return { tokens, session: { ...previous, idToken: tokens.id_token } };
}

export async function oidcSessionResponse(request: NextRequest, load: (token: string) => Promise<unknown>, forceRefresh = false) {
  const access = request.cookies.get(ACCESS_COOKIE)?.value;
  if (!forceRefresh && access) {
    try {
      const claims = await verifiedAccess(access);
      if (claims.exp! > Date.now() / 1000 + 30) return NextResponse.json(await load(access), { headers: { "Cache-Control": "no-store" } });
    }
    catch (error) { if (!(error instanceof Error && "status" in error && error.status === 401)) throw error; }
  }
  const { tokens, session } = await refreshOidc(request);
  const payload = await load(tokens.access_token);
  const response = NextResponse.json(forceRefresh ? { refreshed: true } : payload);
  applyTokens(response, tokens, session);
  return response;
}

export async function oidcLogout(request: NextRequest) {
  const session = unseal<Session>(request.cookies.get(SESSION_COOKIE)?.value, SESSION_COOKIE);
  const doc = await metadata();
  const state = randomBytes(32).toString("hex");
  const url = new URL(doc.end_session_endpoint);
  url.searchParams.set("id_token_hint", session.idToken);
  url.searchParams.set("post_logout_redirect_uri", `${oidcConfig().origin}/api/auth/oidc/logged-out`);
  url.searchParams.set("state", state);
  const response = NextResponse.json({ success: true, logoutUrl: url.href });
  clearOidc(response);
  response.cookies.set(FLOW_COOKIE, seal({ state, at: Date.now() }, FLOW_COOKIE), { ...cookieOptions(), maxAge: 300 });
  return response;
}

export function oidcLoggedOut(request: NextRequest) {
  const flow = unseal<{ state: string; at: number }>(request.cookies.get(FLOW_COOKIE)?.value, FLOW_COOKIE);
  if (!equal(request.nextUrl.searchParams.get("state") ?? "", flow.state) || Date.now() - flow.at > 300_000) throw new OidcError(401, "OIDC_CALLBACK_INVALID");
  const response = NextResponse.redirect(new URL("/login", oidcConfig().origin));
  clearOidc(response);
  return response;
}
