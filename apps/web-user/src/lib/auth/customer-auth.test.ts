import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { createServer, type Server } from "node:http";
import { createHash } from "node:crypto";
import { decodeJwt, exportJWK, generateKeyPair, SignJWT } from "jose";
import { NextRequest } from "next/server";
import { oidcCallback, oidcLogin } from "./oidc";
import { resolveCustomerCredential, withCustomerAuth } from "./customer-auth";

const principal = "11111111-1111-4111-8111-111111111111";
const original = { ...process.env };
let server: Server;
let issuer = "";
let nonce = "";
let challenge = "";
let mutation = "";
let runId = "";
let tokenCalls = 0;
let lastRefreshPresented = "";

const ENV_KEYS = ["WEB_AUTH_MODE", "WEB_PUBLIC_URL", "WEB_OIDC_ISSUER", "WEB_OIDC_CLIENT_ID", "WEB_OIDC_COOKIE_SECRET", "WEB_OIDC_ALLOW_LOCAL_HTTP"];

describe("customer-auth credential boundary", () => {
  beforeEach(async () => {
    nonce = ""; challenge = ""; mutation = ""; tokenCalls = 0; lastRefreshPresented = "";
    runId = Math.random().toString(36).slice(2, 8);
    const { privateKey, publicKey } = await generateKeyPair("RS256");
    const jwk = { ...await exportJWK(publicKey), kid: "test-key", use: "sig", alg: "RS256" };
    const signAccess = (seq: number, shortLived: boolean) => new SignJWT({ user_id: principal, typ: "Bearer", principal_type: "CUSTOMER", seq })
      .setProtectedHeader({ alg: "RS256", kid: "test-key" }).setIssuer(issuer).setAudience("omama-api").setSubject("idp-subject").setIssuedAt().setExpirationTime(shortLived ? "20s" : "5m").sign(privateKey);
    const signId = () => new SignJWT({ nonce })
      .setProtectedHeader({ alg: "RS256", kid: "test-key" }).setIssuer(issuer).setAudience("frontend-web").setSubject("idp-subject").setIssuedAt().setExpirationTime("5m").sign(privateKey);
    server = createServer(async (request, response) => {
      if (request.url === "/echo-stream") {
        response.setHeader("x-seen-authorization", request.headers.authorization ?? "");
        response.setHeader("Content-Type", "text/plain");
        response.write("chunk-1-");
        response.end("chunk-2");
        return;
      }
      response.setHeader("Content-Type", "application/json");
      const send = (body: unknown, status = 200) => { response.statusCode = status; response.end(JSON.stringify(body)); };
      if (request.url === "/.well-known/openid-configuration") return send({ issuer, authorization_endpoint: `${issuer}/auth`, token_endpoint: `${issuer}/token`, jwks_uri: `${issuer}/certs`, end_session_endpoint: `${issuer}/logout` });
      if (request.url === "/certs") return send({ keys: [jwk] });
      if (request.url === "/token") {
        tokenCalls++;
        let body = ""; for await (const chunk of request) body += chunk;
        const form = new URLSearchParams(body);
        const grant = form.get("grant_type");
        if (grant === "refresh_token") {
          lastRefreshPresented = form.get("refresh_token") ?? "";
          if (mutation === "refresh-denied") return send({ error: "invalid_grant" }, 400);
          if (mutation === "provider-error") return send({ error: "server_error" }, 500);
        } else if (createHash("sha256").update(form.get("code_verifier") ?? "").digest("base64url") !== challenge) {
          return send({ error: "invalid_grant" }, 400);
        }
        const access = await signAccess(tokenCalls, mutation === "short-access" && grant === "authorization_code");
        return send({ token_type: "Bearer", expires_in: 300, refresh_expires_in: 3600, access_token: access, refresh_token: `refresh-${runId}-${tokenCalls}`, id_token: await signId() });
      }
      return send({ code: "UNSUPPORTED" }, 501);
    });
    await new Promise<void>(resolve => server.listen(0, "127.0.0.1", resolve));
    const address = server.address();
    if (!address || typeof address === "string") throw new Error("No provider address");
    issuer = `http://127.0.0.1:${address.port}`;
    Object.assign(process.env, { WEB_AUTH_MODE: "oidc", WEB_PUBLIC_URL: "http://localhost:13000", WEB_OIDC_ISSUER: issuer, WEB_OIDC_CLIENT_ID: "frontend-web", WEB_OIDC_COOKIE_SECRET: "test-cookie-secret-more-than-thirty-two-bytes", WEB_OIDC_ALLOW_LOCAL_HTTP: "true" });
  });
  afterEach(async () => {
    server.closeAllConnections();
    await new Promise<void>(resolve => server.close(() => resolve()));
    for (const key of ENV_KEYS) {
      if (original[key] === undefined) delete process.env[key]; else process.env[key] = original[key];
    }
  });

  async function login() {
    const response = await oidcLogin(new NextRequest("http://localhost:13000/api/auth/oidc/login?returnUrl=%2Faccount"));
    const authorize = new URL(response.headers.get("location")!);
    nonce = authorize.searchParams.get("nonce")!;
    challenge = authorize.searchParams.get("code_challenge")!;
    const flowCookie = response.cookies.get("oma_oidc_flow")!;
    const callback = new NextRequest(`http://localhost:13000/api/auth/oidc/callback?code=test-code&state=${authorize.searchParams.get("state")}`, { headers: { cookie: `${flowCookie.name}=${flowCookie.value}` } });
    const result = await oidcCallback(callback);
    const cookies = new Map(result.cookies.getAll().map((cookie) => [cookie.name, cookie.value]));
    return { cookies, header: [...cookies].map(([name, value]) => `${name}=${value}`).join("; ") };
  }

  const requestWith = (cookieHeader: string) => new NextRequest("http://localhost:13000/api/capabilities", { headers: { cookie: cookieHeader } });
  const setCookiesOf = (response: Response) => (response.headers as Headers & { getSetCookie(): string[] }).getSetCookie();
  const cookieValue = (setCookies: string[], name: string) => setCookies.find((entry) => entry.startsWith(`${name}=`))?.slice(name.length + 1).split(";")[0];

  it("uses a still-valid access token as-is and returns the handler response untouched", async () => {
    const { cookies, header } = await login();
    expect(tokenCalls).toBe(1);
    const resolved = await resolveCustomerCredential(requestWith(header));
    expect(resolved.credential).toBe(cookies.get("oma_access_token"));
    expect(tokenCalls).toBe(1);
    const sentinel = new Response("ok");
    expect(resolved.finalize(sentinel)).toBe(sentinel);
  });

  it("stays anonymous when the request carries no session cookies", async () => {
    const resolved = await resolveCustomerCredential(new NextRequest("http://localhost:13000/api/capabilities"));
    expect(resolved.credential).toBeUndefined();
    expect(tokenCalls).toBe(0);
  });

  it("refreshes before the handler runs: upstream receives the fresh token, the body streams intact, and both rotated cookies reach the browser", async () => {
    mutation = "short-access";
    const { header } = await login();
    const response = await withCustomerAuth(requestWith(header), async (credential) => {
      const upstream = await fetch(`${issuer}/echo-stream`, { headers: { Authorization: `Bearer ${credential}` } });
      return new Response(upstream.body, { status: upstream.status, headers: { "Content-Type": "text/plain", "x-seen-authorization": upstream.headers.get("x-seen-authorization") ?? "" } });
    });
    expect(tokenCalls).toBe(2);
    const seen = response.headers.get("x-seen-authorization") ?? "";
    expect(decodeJwt(seen.replace(/^Bearer /u, "")).seq).toBe(2);
    expect(await response.text()).toBe("chunk-1-chunk-2");
    const setCookies = setCookiesOf(response);
    expect(cookieValue(setCookies, "oma_access_token")).toBeTruthy();
    expect(cookieValue(setCookies, "oma_refresh_token")).toBe(`refresh-${runId}-2`);
  });

  it("resolves a follow-up request that carries only the rotated refresh cookie", async () => {
    mutation = "short-access";
    const { cookies, header } = await login();
    const first = await withCustomerAuth(requestWith(header), async () => NextResponseJson());
    const setCookies = setCookiesOf(first);
    const rotatedRefresh = cookieValue(setCookies, "oma_refresh_token")!;
    const session = cookies.get("oma_oidc_session")!;
    const followUp = requestWith(`oma_refresh_token=${rotatedRefresh}; oma_oidc_session=${session}`);
    const resolved = await resolveCustomerCredential(followUp);
    expect(tokenCalls).toBe(3);
    expect(lastRefreshPresented).toBe(rotatedRefresh);
    expect(decodeJwt(resolved.credential!).seq).toBe(3);
  });

  it("answers SESSION_EXPIRED when the provider rejects the refresh token", async () => {
    mutation = "short-access";
    const { header } = await login();
    mutation = "refresh-denied";
    let handlerRan = false;
    const response = await withCustomerAuth(requestWith(header), async () => { handlerRan = true; return new Response("ok"); });
    expect(handlerRan).toBe(false);
    expect(response.status).toBe(401);
    expect((await response.json() as { code: string }).code).toBe("SESSION_EXPIRED");
  });

  it("answers 503 without touching the session when the provider is unavailable", async () => {
    mutation = "short-access";
    const { header } = await login();
    mutation = "provider-error";
    const response = await withCustomerAuth(requestWith(header), async () => new Response("ok"));
    expect(response.status).toBe(503);
    expect((await response.json() as { code: string }).code).toBe("AUTH_PROVIDER_UNAVAILABLE");
    expect(setCookiesOf(response).some((entry) => entry.startsWith("oma_access_token=;") || entry.startsWith("oma_refresh_token=;"))).toBe(false);
  });

  it("attaches rotated cookies to an empty 204 response", async () => {
    mutation = "short-access";
    const { header } = await login();
    const resolved = await resolveCustomerCredential(requestWith(header));
    const finalized = resolved.finalize(new Response(null, { status: 204 }));
    expect(finalized.status).toBe(204);
    expect(cookieValue(setCookiesOf(finalized), "oma_access_token")).toBeTruthy();
    expect(cookieValue(setCookiesOf(finalized), "oma_refresh_token")).toBe(`refresh-${runId}-2`);
  });

  it("passes the identity-BFF session cookie through unchanged", async () => {
    process.env.WEB_AUTH_MODE = "identity-bff";
    const sessionId = "33333333-3333-4333-8333-333333333333";
    const resolved = await resolveCustomerCredential(new NextRequest("http://localhost:13000/api/capabilities", { headers: { cookie: `OMAMA_SESSION=${sessionId}` } }));
    expect(resolved.credential).toBe(sessionId);
    const sentinel = new Response("ok");
    expect(resolved.finalize(sentinel)).toBe(sentinel);
    expect(tokenCalls).toBe(0);
  });
});

function NextResponseJson() {
  return new Response(JSON.stringify({ ok: true }), { status: 200, headers: { "Content-Type": "application/json" } });
}
