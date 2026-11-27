import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { NextRequest } from "next/server";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { handleProcurement } from "./handler";
import { procurementQuery, procurementRoute } from "../routes";

const session = "11111111-1111-4111-8111-111111111111";
const id = "22222222-2222-4222-8222-222222222222";
const failure = (status: number) => Response.json({ code: "FORBIDDEN", message: "Denied", requestId: "test", errors: [] }, { status });
function request(path = "suppliers", method = "GET", headers: Record<string, string> = {}, body?: string) {
  return new NextRequest(`http://localhost:3000/api/admin/procurement/${path}`, { method, headers: { Cookie: `OMAMA_SESSION=${session}; unrelated=secret`, Authorization: "Bearer forged", "X-User-Role": "ADMIN", Origin: "http://localhost:3000", "Content-Type": "application/json", "Idempotency-Key": "procurement-test-command-01", ...headers }, body });
}
describe("live Procurement BFF", () => {
  beforeEach(() => {
    vi.stubEnv("ADMIN_PROCUREMENT_ENABLED", "true"); vi.stubEnv("WEB_AUTH_MODE", "identity-bff");
    vi.stubEnv("WEB_PUBLIC_URL", "http://localhost:3000"); vi.stubEnv("WEB_OIDC_ALLOW_LOCAL_HTTP", "true");
    vi.stubEnv("IDENTITY_UPSTREAM_URL", "https://identity.test/api/v1");
  });
  afterEach(() => { vi.unstubAllEnvs(); vi.unstubAllGlobals(); });
  it("allows all and only approved contract operations, not plans", () => {
    const contract = readFileSync(resolve(process.cwd(), "../../docs/03_api_specs/admin-stable.openapi.yaml"), "utf8");
    let count = 0;
    for (const match of contract.matchAll(/^  (\/[^\r\n]+):\r?\n([\s\S]*?)(?=^  \/|^components:)/gmu)) {
      const [, path, item] = match;
      if (!path.startsWith("/api/v1/admin/procurement/")) continue;
      const concrete = path.replace("/api/v1/admin/procurement/", "").replace(/\{[^}]+\}/gu, id);
      for (const method of ["GET", "POST", "PUT", "PATCH", "DELETE"]) {
        const present = new RegExp(`^    ${method.toLowerCase()}:`, "mu").test(item);
        expect(procurementRoute(method, concrete), `${method} ${path}`).toBe(present);
        if (present) count++;
      }
    }
    expect(count).toBe(37);
    for (const path of ["supply-plans", `supply-plans/${id}`, "../internal", `suppliers/${id}/contracts/extra`, "suppliers/%2e%2e", "suppliers/00000000-0000-0000-0000-000000000000"]) for (const method of ["GET", "POST", "PUT", "PATCH"]) expect(procurementRoute(method, path)).toBe(false);
  });
  it("forwards only the opaque session, not browser bearer, roles or unrelated cookies", async () => {
    const fetcher = vi.fn().mockResolvedValue(Response.json({ items: [], total: 0 })); vi.stubGlobal("fetch", fetcher);
    expect((await handleProcurement(request("suppliers?limit=20"), ["suppliers"])).status).toBe(200);
    const [url, init] = fetcher.mock.calls[0];
    expect(url).toBe("https://identity.test/api/v1/admin/procurement-api/suppliers?limit=20");
    expect(init.headers.get("Cookie")).toBe(`OMAMA_SESSION=${session}`);
    for (const name of ["Authorization", "X-User-Role", "X-Internal-Token"]) expect(init.headers.get(name)).toBeNull();
    expect(init.redirect).toBe("manual"); expect(init.cache).toBe("no-store");
  });
  it("uses the trusted JWT cookie only in OIDC mode", async () => {
    vi.stubEnv("WEB_AUTH_MODE", "oidc"); vi.stubEnv("PROCUREMENT_ADMIN_UPSTREAM_URL", "https://gateway.test/api/v1");
    const fetcher = vi.fn().mockResolvedValue(Response.json({ items: [] })); vi.stubGlobal("fetch", fetcher);
    expect((await handleProcurement(request(), ["suppliers"])).status).toBe(401);
    expect((await handleProcurement(request("suppliers", "GET", { Cookie: "oma_access_token=server-token" }), ["suppliers"])).status).toBe(200);
    expect(fetcher.mock.calls[0][1].headers.get("Authorization")).toBe("Bearer server-token");
    expect(fetcher.mock.calls[0][1].headers.get("Cookie")).toBeNull();
  });
  it("is disabled by default, rejects legacy authentication and missing credentials", async () => {
    const fetcher = vi.fn(); vi.stubGlobal("fetch", fetcher);
    vi.stubEnv("ADMIN_PROCUREMENT_ENABLED", "false"); expect((await handleProcurement(request(), ["suppliers"])).status).toBe(404);
    vi.stubEnv("ADMIN_PROCUREMENT_ENABLED", "true"); expect((await handleProcurement(request("suppliers", "GET", { Cookie: "" }), ["suppliers"])).status).toBe(401);
    vi.stubEnv("WEB_AUTH_MODE", "legacy"); expect((await handleProcurement(request(), ["suppliers"])).status).toBe(503);
    expect(fetcher).not.toHaveBeenCalled();
  });
  it("rejects query injection, duplicates, invalid pagination and mutation queries", async () => {
    vi.stubGlobal("fetch", vi.fn());
    for (const query of ["target=http://evil", "limit=1&limit=2", "limit=101", "offset=-1", "offset=99999999999999999"]) expect((await handleProcurement(request(`suppliers?${query}`), ["suppliers"])).status).toBe(400);
    expect(procurementQuery(`suppliers/${id}`, new URLSearchParams("limit=1"))).toBeUndefined();
    expect((await handleProcurement(request("suppliers?limit=1", "POST", {}, "{}"), ["suppliers"])).status).toBe(400);
  });
  it("enforces same-origin mutations and a bounded JSON object with idempotency", async () => {
    const fetcher = vi.fn().mockResolvedValue(Response.json({ id }, { status: 201 })); vi.stubGlobal("fetch", fetcher);
    expect((await handleProcurement(request("suppliers", "POST", { Origin: "https://evil.test" }, "{}"), ["suppliers"])).status).toBe(403);
    expect((await handleProcurement(request("suppliers", "POST", { "Idempotency-Key": "" }, "{}"), ["suppliers"])).status).toBe(400);
    expect((await handleProcurement(request("suppliers", "POST", { "Content-Type": "text/plain" }, "{}"), ["suppliers"])).status).toBe(415);
    for (const body of ["[]", "null", "bad"]) expect((await handleProcurement(request("suppliers", "POST", {}, body), ["suppliers"])).status).toBe(400);
    expect((await handleProcurement(request("suppliers", "POST", {}, JSON.stringify({ notes: "x".repeat(65536) })), ["suppliers"])).status).toBe(413);
    const result = await handleProcurement(request("suppliers", "POST", {}, '{"name":"Test"}'), ["suppliers"]);
    expect(result.status).toBe(201); expect(fetcher).toHaveBeenCalledTimes(1);
    expect(fetcher.mock.calls[0][1].headers.get("Idempotency-Key")).toBe("procurement-test-command-01");
    expect(fetcher.mock.calls[0][1].body).toBe('{"name":"Test"}');
  });
  it("preserves backend denials and conflicts without retrying or replacing them with empty success", async () => {
    const fetcher = vi.fn(); vi.stubGlobal("fetch", fetcher);
    for (const status of [400, 401, 403, 404, 409, 429, 503]) {
      fetcher.mockResolvedValueOnce(failure(status));
      const result = await handleProcurement(request(), ["suppliers"]);
      expect(result.status).toBe(status); expect(result.headers.get("cache-control")).toBe("no-store");
    }
    expect(fetcher).toHaveBeenCalledTimes(7);
  });
  it("rejects redirects, non-JSON, oversized and malformed upstream bodies", async () => {
    const fetcher = vi.fn(); vi.stubGlobal("fetch", fetcher);
    for (const response of [new Response(null, { status: 302 }), new Response("html"), Response.json({ data: "x".repeat(1 << 20) }), new Response("broken", { headers: { "Content-Type": "application/json" } })]) {
      fetcher.mockResolvedValueOnce(response); expect((await handleProcurement(request(), ["suppliers"])).status).toBe(502);
    }
    fetcher.mockRejectedValueOnce(new Error("offline")); expect((await handleProcurement(request(), ["suppliers"])).status).toBe(503);
  });
  it("keeps security-filter 401/403 responses even without a JSON envelope", async () => {
    const fetcher = vi.fn(); vi.stubGlobal("fetch", fetcher);
    for (const status of [401, 403]) {
      fetcher.mockResolvedValueOnce(new Response(null, { status }));
      const result = await handleProcurement(request(), ["suppliers"]);
      expect(result.status).toBe(status);
      expect((await result.json()).requestId).toBeTruthy();
    }
  });
  it("requires explicit private HTTP opt-in and never follows an unconfigured upstream", async () => {
    const fetcher = vi.fn(); vi.stubGlobal("fetch", fetcher);
    for (const url of ["", "http://identity.test/api/v1", "https://user:pass@identity.test/api/v1", "https://identity.test/api/v1?target=x", "https://identity.test/api/v1/../other"]) {
      vi.stubEnv("IDENTITY_UPSTREAM_URL", url); expect((await handleProcurement(request(), ["suppliers"])).status).toBe(503);
    }
    expect(fetcher).not.toHaveBeenCalled();
  });
});
