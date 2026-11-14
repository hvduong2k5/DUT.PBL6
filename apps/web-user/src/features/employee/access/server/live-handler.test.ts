import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { NextRequest } from "next/server";
import { handleLiveAdminAccess } from "./live-handler";

const data = { items: [], statusCounts: [{ status: "ALL", label: "Tất cả", count: 0 }], calculatedAt: "2026-11-07T12:00:00Z" };
const request = () => new NextRequest("http://localhost:3000/api/admin/access/roles", { headers: { Cookie: "OMAMA_SESSION=11111111-1111-4111-8111-111111111111; private=no", Authorization: "Bearer forged", "X-Admin-Profile": "SYSTEM_ADMIN" } });
describe("live IAM read consumer", () => {
  beforeEach(() => { vi.stubEnv("WEB_AUTH_MODE", "identity-bff"); vi.stubEnv("WEB_PUBLIC_URL", "http://localhost:3000"); vi.stubEnv("WEB_OIDC_ALLOW_LOCAL_HTTP", "true"); vi.stubEnv("IDENTITY_UPSTREAM_URL", "http://identity.test/api/v1"); });
  afterEach(() => { vi.unstubAllGlobals(); vi.unstubAllEnvs(); });
  it("reads one versioned projection using only the opaque server session", async () => {
    const fetcher = vi.fn().mockResolvedValue(Response.json({ contractVersion: "iam-admin-read.v1", capabilities: { accessReview: false }, data: { ...data, secret: "never-forward" } }));
    vi.stubGlobal("fetch", fetcher);
    const response = await handleLiveAdminAccess(request(), ["roles"]);
    expect(response.status).toBe(200); expect(await response.json()).toEqual(data);
    expect(response.headers.get("cache-control")).toBe("no-store");
    expect(fetcher).toHaveBeenCalledTimes(1);
    const [url, options] = fetcher.mock.calls[0];
    expect(url).toBe("http://identity.test/api/v1/admin/access/roles");
    expect(options.headers.get("Cookie")).toBe("OMAMA_SESSION=11111111-1111-4111-8111-111111111111");
    for (const key of ["Authorization", "X-User-ID", "X-Admin-Profile", "X-Internal-Token"]) expect(options.headers.get(key)).toBeNull();
  });
  it("never turns an unavailable producer or invalid projection into mock success", async () => {
    const fetcher = vi.fn(); vi.stubGlobal("fetch", fetcher);
    for (const status of [401, 403, 429, 503]) {
      fetcher.mockResolvedValueOnce(new Response(null, { status }));
      expect((await handleLiveAdminAccess(request(), ["roles"])).status).toBe(status);
    }
    fetcher.mockResolvedValueOnce(new Response(null, { status: 404 }));
    expect((await handleLiveAdminAccess(request(), ["roles"])).status).toBe(503);
    fetcher.mockResolvedValueOnce(new Response(null, { status: 404, headers: { "X-IAM-Read-Contract": "iam-admin-read.v1" } }));
    expect((await handleLiveAdminAccess(request(), ["roles", "UNKNOWN"])).status).toBe(404);
    for (const payload of [data, { contractVersion: "iam-admin-read.v2", capabilities: { accessReview: false }, data }, { contractVersion: "iam-admin-read.v1", capabilities: { accessReview: false }, data: { ...data, calculatedAt: undefined } }]) {
      fetcher.mockResolvedValueOnce(Response.json(payload));
      expect((await handleLiveAdminAccess(request(), ["roles"])).status).toBe(502);
    }
  });
  it("blocks mutations and path/query injection before making an upstream request", async () => {
    const fetcher = vi.fn(); vi.stubGlobal("fetch", fetcher);
    expect((await handleLiveAdminAccess(request(), ["roles", "../permissions"])).status).toBe(404);
    expect((await handleLiveAdminAccess(new NextRequest("http://localhost:3000/api/admin/access/roles", { method: "POST" }), ["roles"])).status).toBe(405);
    expect((await handleLiveAdminAccess(new NextRequest("http://localhost:3000/api/admin/access/roles"), ["roles"])).status).toBe(401);
    expect(fetcher).not.toHaveBeenCalled();
  });
  it("requires a trusted token instead of falling back to mock data in standalone OIDC mode", async () => {
    vi.stubEnv("WEB_AUTH_MODE", "oidc");
    const fetcher = vi.fn(); vi.stubGlobal("fetch", fetcher);
    // The probe request carries no OIDC session cookie, so composition never starts.
    expect((await handleLiveAdminAccess(request(), ["roles"])).status).toBe(401);
    expect(fetcher).not.toHaveBeenCalled();
  });
});
