import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { NextRequest } from "next/server";
import { handleOrderRequest } from "./order-handler";

// The shared capability/session implementation has its own real OIDC tests.
// This fixture isolates the new read transport's payload and routing boundaries.
vi.mock("@/lib/auth/customer-access-server", async (original) => ({
  ...await original<typeof import("@/lib/auth/customer-access-server")>(),
  authorizeCustomerCapability: vi.fn().mockResolvedValue({}),
}));
const id = "11111111-1111-4111-8111-111111111111";
const line = "22222222-2222-4222-8222-222222222222";
const batch = "33333333-3333-4333-8333-333333333333";
const origin = "https://shop.example.test";
const result = { order_id: id, state: "READY", total: 1, items: [{ order_line_id: line, batch_id: batch, batch_code: "LOT-A", sku_code: "SKU-1", packed_quantity: 5, status: "AVAILABLE", qr_url: origin + "/trace/token" }] };
function request(query = "") { return new NextRequest(`http://localhost/api/orders/${id}/traceability${query}`, { headers: { Cookie: "oma_access_token=trusted-customer-fixture", Authorization: "Bearer forged" } }); }
beforeEach(() => { vi.stubEnv("WEB_AUTH_MODE", "legacy"); vi.stubEnv("CUSTOMER_CORE_UPSTREAM_URL", "http://order.test/api/v1"); vi.stubEnv("WEB_PUBLIC_URL", origin); vi.stubEnv("NEXT_PUBLIC_TRACEABILITY_PUBLIC_ORIGIN", origin); });
afterEach(() => { vi.unstubAllGlobals(); vi.unstubAllEnvs(); });
describe("owned order traceability BFF", () => {
  it("uses the existing trusted credential and preserves sealed rows", async () => {
    const fetcher = vi.fn().mockResolvedValue(Response.json(result)); vi.stubGlobal("fetch", fetcher);
    const response = await handleOrderRequest(request(), [id, "traceability"]);
    expect(response.status).toBe(200); expect(response.headers.get("cache-control")).toBe("no-store");
    expect(await response.json()).toEqual(result);
    expect(fetcher.mock.calls[0][0]).toBe(`http://order.test/api/v1/orders/${id}/traceability`);
    expect(fetcher.mock.calls[0][1].headers.get("authorization")).toBe("Bearer trusted-customer-fixture");
  });
  it("rejects wrong order echoes and untrusted link origins", async () => {
    for (const payload of [{ ...result, order_id: line }, { ...result, items: [{ ...result.items[0], qr_url: "https://attacker.test/trace/token" }] }]) {
      vi.stubGlobal("fetch", vi.fn().mockResolvedValue(Response.json(payload)));
      expect((await handleOrderRequest(request(), [id, "traceability"])).status).toBe(502);
    }
  });
  it("rejects caller-provided batch filters before contacting Order", async () => {
    const fetcher = vi.fn(); vi.stubGlobal("fetch", fetcher);
    expect((await handleOrderRequest(request("?batch_id=" + batch), [id, "traceability"])).status).toBe(400);
    expect(fetcher).not.toHaveBeenCalled();
  });
});
