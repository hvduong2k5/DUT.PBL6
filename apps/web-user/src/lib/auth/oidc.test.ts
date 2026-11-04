import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { createServer, type Server } from "node:http";
import { createHash } from "node:crypto";
import { generateKeyPair, exportJWK, SignJWT } from "jose";
import { NextRequest } from "next/server";
import { oidcCallback, oidcLogin, oidcSessionResponse, OidcError } from "./oidc";
import { identitySession } from "./identity-session";

const principal = "11111111-1111-4111-8111-111111111111";
const customer = "22222222-2222-4222-8222-222222222222";
const original = { ...process.env };
let server: Server;
let issuer = "";
let nonce = "";
let challenge = "";
let mutation = "";
let tokenCalls = 0;

describe("OIDC wire security", () => {
  beforeEach(async () => {
    nonce = ""; challenge = ""; mutation = ""; tokenCalls = 0;
    const { privateKey, publicKey } = await generateKeyPair("RS256");
    const jwk = { ...await exportJWK(publicKey), kid: "test-key", use: "sig", alg: "RS256" };
    const sign = (claims: Record<string, unknown>, audience: string) => new SignJWT(claims).setProtectedHeader({ alg: "RS256", kid: "test-key" }).setIssuer(issuer).setAudience(audience).setSubject("idp-subject").setIssuedAt().setExpirationTime(mutation === "expiry" && tokenCalls === 1 ? "20s" : "5m").sign(privateKey);
    server = createServer(async (request, response) => {
      response.setHeader("Content-Type", "application/json");
      const send = (body: unknown, status = 200) => { response.statusCode = status; response.end(JSON.stringify(body)); };
      if (request.url === "/.well-known/openid-configuration") return send({ issuer, authorization_endpoint: `${issuer}/auth`, token_endpoint: `${issuer}/token`, jwks_uri: `${issuer}/certs`, end_session_endpoint: `${issuer}/logout` });
      if (request.url === "/certs") return send({ keys: [jwk] });
      if (request.url === "/token") {
        tokenCalls++;
        let body = ""; for await (const chunk of request) body += chunk;
        const form = new URLSearchParams(body);
        if (form.get("grant_type") !== "refresh_token" && createHash("sha256").update(form.get("code_verifier") ?? "").digest("base64url") !== challenge) return send({ error: "invalid_grant" }, 400);
        const idToken = await sign({ nonce: mutation === "nonce" ? "wrong" : nonce }, mutation === "audience" ? "other-client" : "frontend-web");
        const access = await sign({ user_id: principal, typ: "Bearer", realm_access: { roles: ["CUSTOMER"] } }, "omama-api");
        return send({ token_type: "Bearer", expires_in: 300, access_token: access, refresh_token: "test-refresh-value", id_token: idToken });
      }
      if (request.url === "/users/me") return send({ success: true, data: { principalId: principal, idpSubject: "idp-subject", principalType: "CUSTOMER", provisioningStatus: "ACTIVE", roles: ["CUSTOMER"], permissions: [] } });
      if (request.url === "/profile") return send({ id: customer, user_id: mutation === "mapping" ? customer : principal, full_name: "Customer", status: "ACTIVE" });
      return send({ code: "UNSUPPORTED" }, 501);
    });
    await new Promise<void>(resolve => server.listen(0, "127.0.0.1", resolve));
    const address = server.address();
    if (!address || typeof address === "string") throw new Error("No provider address");
    issuer = `http://127.0.0.1:${address.port}`;
    Object.assign(process.env, { WEB_AUTH_MODE: "oidc", WEB_PUBLIC_URL: "http://localhost:13000", WEB_OIDC_ISSUER: issuer, WEB_OIDC_CLIENT_ID: "frontend-web", WEB_OIDC_COOKIE_SECRET: "test-cookie-secret-more-than-thirty-two-bytes", WEB_OIDC_ALLOW_LOCAL_HTTP: "true", CUSTOMER_CORE_UPSTREAM_URL: issuer });
  });
  afterEach(async () => {
    server.closeAllConnections();
    await new Promise<void>(resolve => server.close(() => resolve()));
    for (const key of ["WEB_AUTH_MODE", "WEB_PUBLIC_URL", "WEB_OIDC_ISSUER", "WEB_OIDC_CLIENT_ID", "WEB_OIDC_COOKIE_SECRET", "WEB_OIDC_ALLOW_LOCAL_HTTP", "CUSTOMER_CORE_UPSTREAM_URL"]) {
      if (original[key] === undefined) delete process.env[key]; else process.env[key] = original[key];
    }
  });

  async function begin(returnUrl = "/cart") {
    const response = await oidcLogin(new NextRequest(`http://localhost:13000/api/auth/oidc/login?returnUrl=${encodeURIComponent(returnUrl)}`));
    const authorize = new URL(response.headers.get("location")!);
    nonce = authorize.searchParams.get("nonce")!;
    challenge = authorize.searchParams.get("code_challenge")!;
    expect(authorize.searchParams.get("code_challenge_method")).toBe("S256");
    const cookie = response.cookies.get("oma_oidc_flow")!;
    return { response, authorize, cookie: `${cookie.name}=${cookie.value}` };
  }

  it("binds state, nonce, PKCE and principal mapping without exposing tokens in session JSON", async () => {
    const flow = await begin();
    const callback = new NextRequest(`http://localhost:13000/api/auth/oidc/callback?code=test-code&state=${flow.authorize.searchParams.get("state")}`, { headers: { cookie: flow.cookie } });
    const result = await oidcCallback(callback);
    expect(result.headers.get("location")).toBe("http://localhost:13000/cart");
    expect(result.cookies.get("oma_access_token")?.httpOnly).toBe(true);
    const cookies = result.cookies.getAll().map(c => `${c.name}=${c.value}`).join("; ");
    const session = await oidcSessionResponse(new NextRequest("http://localhost:13000/api/auth/me", { headers: { cookie: cookies } }), identitySession);
    expect(await session.json()).toMatchObject({ authenticated: true, customer: { id: customer }, principal: { principalId: principal, customerId: customer } });
    expect(session.headers.get("cache-control")).toBe("no-store");
    expect(session.cookies.get("oma_access_token")).toBeUndefined();
  });

  it("rejects wrong state before token exchange", async () => {
    const flow = await begin();
    await expect(oidcCallback(new NextRequest("http://localhost:13000/api/auth/oidc/callback?code=code&state=wrong", { headers: { cookie: flow.cookie } }))).rejects.toMatchObject({ status: 401 });
    expect(tokenCalls).toBe(0);
  });

  it("refreshes a nearly expired session before returning it to the UI", async () => {
    const flow = await begin(); mutation = "expiry";
    const result = await oidcCallback(new NextRequest(`http://localhost:13000/api/auth/oidc/callback?code=test-code&state=${flow.authorize.searchParams.get("state")}`, { headers: { cookie: flow.cookie } }));
    const cookie = result.cookies.getAll().map(c => `${c.name}=${c.value}`).join("; ");
    const response = await oidcSessionResponse(new NextRequest("http://localhost:13000/api/auth/me", { headers: { cookie } }), identitySession);
    expect(tokenCalls).toBe(2);
    expect(response.cookies.get("oma_access_token")?.httpOnly).toBe(true);
    const session = await response.json();
    expect(Date.parse(session.sessionExpiresAt)).toBeGreaterThan(Date.now() + 200_000);
  });

  for (const kind of ["nonce", "audience"]) it(`rejects incorrect ID token ${kind}`, async () => {
    const flow = await begin(); mutation = kind;
    await expect(oidcCallback(new NextRequest(`http://localhost:13000/api/auth/oidc/callback?code=code&state=${flow.authorize.searchParams.get("state")}`, { headers: { cookie: flow.cookie } }))).rejects.toBeInstanceOf(OidcError);
  });

  it("does not accept a Customer bound to another principal", async () => {
    const flow = await begin();
    const response = await oidcCallback(new NextRequest(`http://localhost:13000/api/auth/oidc/callback?code=code&state=${flow.authorize.searchParams.get("state")}`, { headers: { cookie: flow.cookie } }));
    mutation = "mapping";
    await expect(identitySession(response.cookies.get("oma_access_token")!.value)).rejects.toMatchObject({ code: "CUSTOMER_MAPPING_INVALID" });
  });

  it("blocks external return URLs and unauthenticated callbacks", async () => {
    const flow = await begin("//attacker.invalid/");
    const response = await oidcCallback(new NextRequest(`http://localhost:13000/api/auth/oidc/callback?code=code&state=${flow.authorize.searchParams.get("state")}`, { headers: { cookie: flow.cookie } }));
    expect(response.headers.get("location")).toBe("http://localhost:13000/");
    await expect(oidcCallback(new NextRequest("http://localhost:13000/api/auth/oidc/callback?code=code&state=anything"))).rejects.toMatchObject({ status: 401 });
  });
});
