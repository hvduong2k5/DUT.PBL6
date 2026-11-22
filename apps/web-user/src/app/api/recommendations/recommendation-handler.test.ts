import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { NextRequest } from "next/server";
import { recommendationProxy } from "./recommendation-handler";

const origin = "http://127.0.0.1:13046";
const session = "f47ac10b-58cc-4372-a567-0e02b2c3d477";
const csrf = "a".repeat(43);

beforeEach(() => {
  vi.stubEnv("NEXT_PUBLIC_RECOMMENDATIONS_ENABLED", "true");
  vi.stubEnv("ANALYTICS_RECOMMENDATION_UPSTREAM_URL", "http://analytics.test:8011");
  vi.stubEnv("WEB_AUTH_MODE", "identity-bff");
  vi.stubEnv("WEB_PUBLIC_URL", origin);
  vi.stubEnv("WEB_OIDC_ALLOW_LOCAL_HTTP", "true");
});
afterEach(() => { vi.unstubAllGlobals(); vi.unstubAllEnvs(); });

function request(path = "", init?: ConstructorParameters<typeof NextRequest>[1]) {
  return new NextRequest(origin + "/api/recommendations" + path, init);
}

describe("Recommendation BFF", () => {
  it("routes the query to MS-11 and forwards the verified credential form", async () => {
    const fetcher = vi.fn().mockResolvedValue(Response.json({ items: [] }));
    vi.stubGlobal("fetch", fetcher);
    const response = await recommendationProxy(request("?placement=home", { headers: {
      Cookie: `OMAMA_SESSION=${session}`, Authorization: "Bearer forged", "X-User-Id": "forged"
    } }), []);
    expect(response.status).toBe(200);
    const [url, options] = fetcher.mock.calls[0];
    expect(url.href).toBe("http://analytics.test:8011/api/v1/products/recommended?placement=home");
    expect(options.headers.get("Cookie")).toBe(`OMAMA_SESSION=${session}`);
    expect(options.headers.has("Authorization")).toBe(false);
    expect(options.headers.has("X-User-Id")).toBe(false);
    expect(response.headers.get("Cache-Control")).toBe("private, no-store");
    expect(options.redirect).toBe("manual");
  });

  it("issues a CSRF token only after successful authenticated consent read", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(Response.json({ enabled: false }, { headers: { ETag: '"0"' } })));
    const response = await recommendationProxy(request("/consent"), ["consent"]);
    expect(response.cookies.get("oma_rec_csrf")?.value).toMatch(/^[A-Za-z0-9_-]{43}$/u);
    expect(response.headers.get("etag")).toBe('"0"');
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(Response.json({}, { status: 401 })));
    expect((await recommendationProxy(request("/consent"), ["consent"])).cookies.has("oma_rec_csrf")).toBe(false);
  });

  it.each([undefined, "http://other.test"])("rejects mutation with Origin %s", async (value) => {
    const fetcher = vi.fn();
    vi.stubGlobal("fetch", fetcher);
    const headers: Record<string, string> = { Cookie: `oma_rec_csrf=${csrf}`, "X-CSRF-Token": csrf, "Content-Type": "application/json" };
    if (value) headers.Origin = value;
    const response = await recommendationProxy(request("/consent", { method: "PUT", headers, body: "{}" }), ["consent"]);
    expect(response.status).toBe(403);
    expect(fetcher).not.toHaveBeenCalled();
  });

  it("forwards consent preconditions and limits mutation body size", async () => {
    const fetcher = vi.fn().mockResolvedValue(Response.json({ enabled: true }));
    vi.stubGlobal("fetch", fetcher);
    const headers = { Origin: origin, Cookie: `OMAMA_SESSION=${session}; oma_rec_csrf=${csrf}`,
      "X-CSRF-Token": csrf, "Content-Type": "application/json", "If-Match": '"1"', "Idempotency-Key": "key" };
    const response = await recommendationProxy(request("/consent", { method: "PUT", headers, body: '{"enabled":true}' }), ["consent"]);
    expect(response.status).toBe(200);
    expect(fetcher.mock.calls[0][1].headers.get("If-Match")).toBe('"1"');
    const tooLarge = await recommendationProxy(request("/events", { method: "POST", headers, body: "x".repeat(4097) }), ["events"]);
    expect(tooLarge.status).toBe(400);
    expect(fetcher).toHaveBeenCalledTimes(1);
  });

  it("handles upstream timeouts and blocks redirects", async () => {
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new DOMException("timeout", "AbortError")));
    expect((await recommendationProxy(request("?placement=home"), [])).status).toBe(503);
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(null, { status: 307, headers: { Location: "http://other.test" } })));
    expect((await recommendationProxy(request("?placement=home"), [])).status).toBe(502);
  });

  it("keeps method and feature allowlists", async () => {
    expect((await recommendationProxy(request("/events"), ["events"])).status).toBe(404);
    vi.stubEnv("NEXT_PUBLIC_RECOMMENDATIONS_ENABLED", "false");
    expect((await recommendationProxy(request(), [])).status).toBe(503);
  });
});
