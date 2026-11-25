import { afterEach, describe, expect, it, vi } from "vitest";
import { NextRequest } from "next/server";
import { GET } from "./[...path]/route";

vi.mock("@/lib/auth/customer-auth", () => ({ withCustomerAuth: (_request: NextRequest, next: (token: string) => Promise<Response>) => next("verified-customer-token") }));
vi.mock("@/lib/auth/customer-access-server", () => ({ authorizeCustomerCapability: vi.fn(), customerAccessFailure: vi.fn() }));

afterEach(() => { vi.unstubAllEnvs(); vi.unstubAllGlobals(); });

describe("Promotion customer upstreams", () => {
  it("routes public voucher listing to Kong's root and preserves supported filters", async () => {
    vi.stubEnv("WEB_AUTH_MODE", "oidc");
    vi.stubEnv("CUSTOMER_CORE_UPSTREAM_URL", "https://gateway.test/api/v1/customer-api");
    vi.stubEnv("API_GATEWAY_UPSTREAM_URL", "https://gateway.test/api/v1");
    const fetch = vi.fn().mockResolvedValue(Response.json({ vouchers: [] })); vi.stubGlobal("fetch", fetch);
    const response = await GET(new NextRequest("http://localhost/api/customer-core/promotions/vouchers?q=welcome&page=2&customer_id=forged", { headers: { "X-User-ID": "forged" } }), { params: Promise.resolve({ path: ["promotions", "vouchers"] }) });
    expect(response.status).toBe(200);
    expect(fetch.mock.calls[0][0]).toBe("https://gateway.test/api/v1/promotions/vouchers?q=welcome&page=2");
    expect(new Headers(fetch.mock.calls[0][1].headers).has("X-User-ID")).toBe(false);
  });
  it("forwards the resolved customer bearer to the real loyalty route", async () => {
    vi.stubEnv("WEB_AUTH_MODE", "oidc"); vi.stubEnv("API_GATEWAY_UPSTREAM_URL", "https://gateway.test/api/v1");
    const fetch = vi.fn().mockResolvedValue(Response.json({ available_points: 0 })); vi.stubGlobal("fetch", fetch);
    const response = await GET(new NextRequest("http://localhost/api/customer-core/loyalty/points"), { params: Promise.resolve({ path: ["loyalty", "points"] }) });
    expect(response.status).toBe(200);
    expect(fetch.mock.calls[0][0]).toBe("https://gateway.test/api/v1/loyalty/points");
    expect(new Headers(fetch.mock.calls[0][1].headers).get("Authorization")).toBe("Bearer verified-customer-token");
  });
});
