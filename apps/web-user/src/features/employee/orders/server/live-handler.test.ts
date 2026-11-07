import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { NextRequest } from "next/server";
import { handleLiveAdminOrders } from "./live-handler";

const session = "11111111-1111-4111-8111-111111111111";
function request(path = "/api/admin/orders", authenticated = true) {
  return new NextRequest(`http://localhost:3000${path}`, { headers: { ...(authenticated ? { Cookie: `OMAMA_SESSION=${session}; unrelated=private` } : {}), Authorization: "Bearer browser-forged", "X-User-ID": "forged", "X-Admin-Profile": "SYSTEM_ADMIN" } });
}
describe("live Order BFF", () => {
  beforeEach(() => { vi.stubEnv("WEB_AUTH_MODE", "identity-bff"); vi.stubEnv("WEB_PUBLIC_URL", "http://localhost:3000"); vi.stubEnv("WEB_OIDC_ALLOW_LOCAL_HTTP", "true"); vi.stubEnv("IDENTITY_UPSTREAM_URL", "http://identity.test/api/v1"); });
  afterEach(() => { vi.unstubAllGlobals(); vi.unstubAllEnvs(); });
  it("forwards only the server session to the allowlisted projection, without a mutable permission lookup", async () => {
    const fetcher = vi.fn().mockResolvedValue(Response.json({ items: [], calculated_at: "2026-11-07T12:00:00Z", next_cursor: null }));
    vi.stubGlobal("fetch", fetcher);
    const result = await handleLiveAdminOrders(request("/api/admin/orders?view=queue&limit=10"), []);
    expect(result.status).toBe(200);
    expect(result.headers.get("cache-control")).toBe("no-store");
    expect(fetcher).toHaveBeenCalledTimes(1);
    const [url, init] = fetcher.mock.calls[0];
    expect(url).toBe("http://identity.test/api/v1/admin/order-api/orders/queue?limit=10");
    const headers = init.headers as Headers;
    expect(headers.get("Cookie")).toBe(`OMAMA_SESSION=${session}`);
    expect(headers.get("Accept")).toBe("application/vnd.omama.admin-order+json");
    for (const name of ["Authorization", "X-User-ID", "X-Admin-Profile", "X-Internal-Token"]) expect(headers.get(name)).toBeNull();
  });
  it("does not obtain authority from browser headers or expose a mutation route", async () => {
    const fetcher = vi.fn(); vi.stubGlobal("fetch", fetcher);
    expect((await handleLiveAdminOrders(request("/api/admin/orders", false), [])).status).toBe(401);
    expect((await handleLiveAdminOrders(new NextRequest("http://localhost:3000/api/admin/orders", { method: "POST" }), [])).status).toBe(405);
    expect((await handleLiveAdminOrders(request("/api/admin/orders?target=http://attacker.test"), [])).status).toBe(400);
    expect((await handleLiveAdminOrders(request("/api/admin/orders?limit=1&limit=2"), [])).status).toBe(400);
    expect(fetcher).not.toHaveBeenCalled();
  });
  it("does not enter the mock adapter through standalone OIDC mode", async () => {
    vi.stubEnv("WEB_AUTH_MODE", "oidc");
    const fetcher = vi.fn(); vi.stubGlobal("fetch", fetcher);
    expect((await handleLiveAdminOrders(request(), [])).status).toBe(503);
    expect(fetcher).not.toHaveBeenCalled();
  });
  it("preserves denial and outages without falling back to mock data", async () => {
    const fetcher = vi.fn(); vi.stubGlobal("fetch", fetcher);
    for (const status of [401, 403, 404, 503]) {
      fetcher.mockResolvedValueOnce(new Response(null, { status }));
      expect((await handleLiveAdminOrders(request(), [])).status).toBe(status);
    }
    fetcher.mockRejectedValueOnce(new Error("offline"));
    expect((await handleLiveAdminOrders(request(), [])).status).toBe(503);
    fetcher.mockResolvedValueOnce(Response.json({ items: [] }));
    expect((await handleLiveAdminOrders(request(), [])).status).toBe(502);
  });
});
