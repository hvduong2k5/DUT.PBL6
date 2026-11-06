import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { NextRequest } from "next/server";
import { credentialHeaders, getTrustedAccessToken } from "./server-credential";
import { ownerLogin, ownerSession } from "./owner-bff";
import { identitySession } from "./identity-session";

const original = { ...process.env };
const sessionId = "11111111-1111-4111-8111-111111111111";
const principalId = "22222222-2222-4222-8222-222222222222";
const customerId = "33333333-3333-7333-8333-333333333333";
beforeEach(() => Object.assign(process.env, { WEB_AUTH_MODE: "identity-bff", WEB_PUBLIC_URL: "https://shop.example.test", IDENTITY_UPSTREAM_URL: "https://gateway.example.test/api/v1", CUSTOMER_CORE_UPSTREAM_URL: "https://gateway.example.test/api/v1/customer-api" }));
afterEach(() => { vi.unstubAllGlobals(); for (const key of ["WEB_AUTH_MODE", "WEB_PUBLIC_URL", "IDENTITY_UPSTREAM_URL", "CUSTOMER_CORE_UPSTREAM_URL"]) { if (original[key] === undefined) delete process.env[key]; else process.env[key] = original[key]; } });

describe("Identity owner BFF", () => {
  it("accepts only the opaque session cookie, and never forwards browser tokens or identity headers", () => {
    const request = new NextRequest("https://shop.example.test/api/cart", { headers: { Cookie: `OMAMA_SESSION=${sessionId}; oma_access_token=old-jwt`, Authorization: "Bearer forged" } });
    expect(getTrustedAccessToken(request)).toBe(sessionId);
    const headers = credentialHeaders(new Headers({ Authorization: "Bearer forged" }), getTrustedAccessToken(request));
    expect(headers.get("authorization")).toBeNull();
    expect(headers.get("cookie")).toBe(`OMAMA_SESSION=${sessionId}`);
    expect(headers.get("origin")).toBe("https://shop.example.test");
    expect(getTrustedAccessToken(new NextRequest(request.url, { headers: { Cookie: "oma_access_token=old-jwt", Authorization: "Bearer forged" } }))).toBeUndefined();
  });

  it("starts login at Identity and constrains the return URL to this Web origin", () => {
    const response = ownerLogin(new NextRequest("https://shop.example.test/api/auth/oidc/login?returnUrl=https://evil.test"));
    const target = new URL(response.headers.get("location")!);
    expect(target.pathname).toBe("/api/v1/public/auth/login");
    expect(target.searchParams.get("redirect_uri")).toBe("https://shop.example.test/");
  });

  it("fails closed for an unknown server session", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(Response.json({ success: true, data: { authenticated: false } })));
    await expect(ownerSession(sessionId)).rejects.toMatchObject({ status: 401 });
  });

  it("maps the authenticated Identity principal to a pending Profile Customer without token cookies", async () => {
    const fetcher = vi.fn()
      .mockResolvedValueOnce(Response.json({ success: true, data: { authenticated: true, principalId, expiresAt: "2026-11-02T00:00:00Z" } }))
      .mockResolvedValueOnce(Response.json({ success: true, data: { principalId, principalType: "CUSTOMER", provisioningStatus: "ACTIVE", roles: [{ roleCode: "CUSTOMER" }], permissions: [], email: "synthetic@example.test" } }))
      .mockResolvedValueOnce(Response.json({ id: customerId, user_id: principalId, status: "PENDING_VERIFICATION", full_name: "Synthetic Customer" }));
    vi.stubGlobal("fetch", fetcher);
    const session = await identitySession(sessionId);
    expect(session.principal.principalId).toBe(principalId);
    expect(session.customer?.id).toBe(customerId);
    expect(session.customer?.status).toBe("PENDING_VERIFICATION");
    expect(JSON.stringify(session)).not.toContain(sessionId);
    for (const [, options] of fetcher.mock.calls) {
      expect(options.headers.get("cookie")).toBe(`OMAMA_SESSION=${sessionId}`);
      expect(options.headers.get("authorization")).toBeNull();
    }
  });
});
