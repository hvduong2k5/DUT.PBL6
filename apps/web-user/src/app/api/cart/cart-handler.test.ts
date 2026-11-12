import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { createServer, type Server } from "node:http";
import { createHash } from "node:crypto";
import { exportJWK, generateKeyPair, SignJWT } from "jose";
import { NextRequest } from "next/server";
import { oidcCallback, oidcLogin } from "@/lib/auth/oidc";
import { handleCartRequest } from "./cart-handler";

/**
 * Regression test for F-12: in OIDC mode the deployment configures
 * CUSTOMER_CORE_UPSTREAM_URL as the identity customer-api mount
 * (…/api/v1/customer-api). The cart handler must still call the cart API at
 * the gateway root (…/api/v1/cart) with the resolved OIDC credential —
 * before the fix it fetched …/customer-api/cart and identity answered 401.
 */
const principal = "11111111-1111-4111-8111-111111111111";
const customerId = "22222222-2222-4222-8222-222222222222";
const gateway = "http://gateway.test/api/v1";
const original = { ...process.env };
const ENV_KEYS = [
  "WEB_AUTH_MODE", "WEB_PUBLIC_URL", "WEB_OIDC_ISSUER", "WEB_OIDC_CLIENT_ID",
  "WEB_OIDC_COOKIE_SECRET", "WEB_OIDC_ALLOW_LOCAL_HTTP", "CUSTOMER_CORE_UPSTREAM_URL",
  "CUSTOMER_EXTENSIONS_UPSTREAM_URL", "API_GATEWAY_UPSTREAM_URL", "IDENTITY_UPSTREAM_URL"
];

let server: Server;
let issuer = "";
let nonce = "";
let challenge = "";
let seenCartUrl = "";
let seenCartAuthorization = "";

describe("cart handler upstream wiring (oidc mode)", () => {
  beforeEach(async () => {
    seenCartUrl = ""; seenCartAuthorization = "";
    const { privateKey, publicKey } = await generateKeyPair("RS256");
    const jwk = { ...await exportJWK(publicKey), kid: "test-key", use: "sig", alg: "RS256" };
    const signAccess = () => new SignJWT({ user_id: principal, typ: "Bearer", principal_type: "CUSTOMER" })
      .setProtectedHeader({ alg: "RS256", kid: "test-key" }).setIssuer(issuer).setAudience("omama-api").setSubject("idp-subject").setIssuedAt().setExpirationTime("5m").sign(privateKey);
    const signId = () => new SignJWT({ nonce })
      .setProtectedHeader({ alg: "RS256", kid: "test-key" }).setIssuer(issuer).setAudience("frontend-web").setSubject("idp-subject").setIssuedAt().setExpirationTime("5m").sign(privateKey);
    server = createServer(async (request, response) => {
      response.setHeader("Content-Type", "application/json");
      const send = (body: unknown, status = 200) => { response.statusCode = status; response.end(JSON.stringify(body)); };
      if (request.url === "/.well-known/openid-configuration") return send({ issuer, authorization_endpoint: `${issuer}/auth`, token_endpoint: `${issuer}/token`, jwks_uri: `${issuer}/certs`, end_session_endpoint: `${issuer}/logout` });
      if (request.url === "/certs") return send({ keys: [jwk] });
      if (request.url === "/token") {
        let body = ""; for await (const chunk of request) body += chunk;
        const form = new URLSearchParams(body);
        if (form.get("grant_type") !== "refresh_token" && createHash("sha256").update(form.get("code_verifier") ?? "").digest("base64url") !== challenge) {
          return send({ error: "invalid_grant" }, 400);
        }
        return send({ token_type: "Bearer", expires_in: 300, refresh_expires_in: 3600, access_token: await signAccess(), refresh_token: "refresh-1", id_token: await signId() });
      }
      return send({ code: "UNSUPPORTED" }, 501);
    });
    await new Promise<void>(resolve => server.listen(0, "127.0.0.1", resolve));
    const address = server.address();
    if (!address || typeof address === "string") throw new Error("No provider address");
    issuer = `http://127.0.0.1:${address.port}`;
    Object.assign(process.env, {
      WEB_AUTH_MODE: "oidc",
      WEB_PUBLIC_URL: "http://localhost:13000",
      WEB_OIDC_ISSUER: issuer,
      WEB_OIDC_CLIENT_ID: "frontend-web",
      WEB_OIDC_COOKIE_SECRET: "test-cookie-secret-more-than-thirty-two-bytes",
      WEB_OIDC_ALLOW_LOCAL_HTTP: "true",
      CUSTOMER_CORE_UPSTREAM_URL: `${gateway}/customer-api`,
      CUSTOMER_EXTENSIONS_UPSTREAM_URL: `${gateway}/extensions`
    });
    delete process.env.API_GATEWAY_UPSTREAM_URL;
    delete process.env.IDENTITY_UPSTREAM_URL;

    const realFetch = globalThis.fetch;
    vi.stubGlobal("fetch", async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      if (url.startsWith(issuer)) return realFetch(input, init);
      if (url === `${gateway}/customer-api/users/me`) {
        return Response.json({ success: true, data: { principalId: principal, idpSubject: "idp-subject", principalType: "CUSTOMER", provisioningStatus: "ACTIVE", email: "customer@test.local", roles: [{ roleCode: "CUSTOMER" }], permissions: [] } });
      }
      if (url === `${gateway}/customer-api/profile`) {
        return Response.json({ user_id: principal, id: customerId, status: "ACTIVE", full_name: "Test Customer", phone_number: "0900000000" });
      }
      if (url === `${gateway}/extensions/customer/capabilities`) {
        return Response.json({ actor: "REGISTERED", capabilities: ["CART_MANAGE", "CHECKOUT_CREATE", "ORDER_HISTORY_VIEW"], version: 1 });
      }
      if (url === `${gateway}/cart`) {
        seenCartUrl = url;
        seenCartAuthorization = new Headers(init?.headers).get("authorization") ?? "";
        return Response.json({ revision: 0, items: [] });
      }
      return new Response(JSON.stringify({ code: "UNEXPECTED_URL", url }), { status: 500, headers: { "Content-Type": "application/json" } });
    });
  });

  afterEach(async () => {
    vi.unstubAllGlobals();
    server.closeAllConnections();
    await new Promise<void>(resolve => server.close(() => resolve()));
    for (const key of ENV_KEYS) {
      if (original[key] === undefined) delete process.env[key]; else process.env[key] = original[key];
    }
  });

  async function login() {
    const response = await oidcLogin(new NextRequest("http://localhost:13000/api/auth/oidc/login?returnUrl=%2Fcart"));
    const authorize = new URL(response.headers.get("location")!);
    nonce = authorize.searchParams.get("nonce")!;
    challenge = authorize.searchParams.get("code_challenge")!;
    const flowCookie = response.cookies.get("oma_oidc_flow")!;
    const callback = new NextRequest(`http://localhost:13000/api/auth/oidc/callback?code=test-code&state=${authorize.searchParams.get("state")}`, { headers: { cookie: `${flowCookie.name}=${flowCookie.value}` } });
    const result = await oidcCallback(callback);
    const cookies = new Map(result.cookies.getAll().map((cookie) => [cookie.name, cookie.value]));
    return { cookies, header: [...cookies].map(([name, value]) => `${name}=${value}`).join("; ") };
  }

  it("fetches the cart at the gateway root with the OIDC access token, not under the customer-api mount", async () => {
    const { cookies, header } = await login();
    const request = new NextRequest("http://localhost:13000/api/cart", { headers: { cookie: header } });
    const response = await handleCartRequest(request, []);
    expect(response.status).toBe(200);
    await expect(response.json()).resolves.toEqual({ revision: 0, items: [] });
    expect(seenCartUrl).toBe(`${gateway}/cart`);
    expect(seenCartAuthorization).toBe(`Bearer ${cookies.get("oma_access_token")}`);
  });
});
