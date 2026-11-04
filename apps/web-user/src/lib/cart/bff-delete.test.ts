import { afterEach, expect, it, vi } from "vitest";
import { NextRequest } from "next/server";
import { handleCartRequest } from "@/app/api/cart/cart-handler";

vi.mock("@/lib/auth/customer-access-server", () => ({
  authorizeCustomerCapability: vi.fn().mockResolvedValue({}),
  customerAccessFailure: vi.fn()
}));
afterEach(() => vi.unstubAllGlobals());

it("reads the new cart after a successful 204 DELETE even when a proxy keeps the JSON content type", async () => {
  const fetcher = vi.fn()
    .mockResolvedValueOnce(new Response(null, { status: 204, headers: { "Content-Type": "application/json" } }))
    .mockResolvedValueOnce(Response.json({ revision: 2, items: [] }));
  vi.stubGlobal("fetch", fetcher);
  const request = new NextRequest("http://localhost/api/cart/items/PRIVATE-A", {
    method: "DELETE", headers: { "If-Match": '"1"', Origin: "http://localhost", Cookie: "oma_access_token=fixture" }
  });
  const response = await handleCartRequest(request, ["items", "PRIVATE-A"]);
  expect(response.status).toBe(200);
  expect(await response.json()).toMatchObject({ revision: 2, items: [] });
  expect(fetcher).toHaveBeenCalledTimes(2);
});
