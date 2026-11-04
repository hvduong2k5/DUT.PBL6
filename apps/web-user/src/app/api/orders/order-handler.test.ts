import { NextRequest } from "next/server";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const originalGuestOrderGate = process.env.NEXT_PUBLIC_MVP_GUEST_ORDER_ENABLED;

function identityResponse(data: unknown) {
  return Response.json({ success: true, data });
}

beforeEach(() => {
  vi.stubEnv("IDENTITY_UPSTREAM_URL", "http://identity.test/api/v1");
  vi.resetModules(); vi.stubEnv("WEB_AUTH_MODE", "identity-bff"); vi.stubEnv("WEB_PUBLIC_URL", "http://localhost"); vi.stubEnv("WEB_OIDC_ALLOW_LOCAL_HTTP", "true");
  process.env.NEXT_PUBLIC_MVP_GUEST_ORDER_ENABLED = "true";
});

afterEach(() => {
  vi.unstubAllGlobals(); vi.unstubAllEnvs();
  if (originalGuestOrderGate === undefined) delete process.env.NEXT_PUBLIC_MVP_GUEST_ORDER_ENABLED;
  else process.env.NEXT_PUBLIC_MVP_GUEST_ORDER_ENABLED = originalGuestOrderGate;
});

describe("Order principal isolation", () => {
  it("rejects an authenticated Employee from Guest Order endpoints", async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(identityResponse({ authenticated: true, principalId:"10000000-0000-0000-0000-000000000001", expiresAt:"2099-01-01T00:00:00Z" }))
      .mockResolvedValueOnce(identityResponse({
        principalId: "10000000-0000-0000-0000-000000000001", provisioningStatus:"ACTIVE",
        principalType: "EMPLOYEE",
        email: "staff@omama.vn",
        roles: [{ roleCode: "CUSTOMER" }],
        permissions: []
      }));
    vi.stubGlobal("fetch", fetchMock);
    const { handleOrderRequest } = await import("./order-handler");
    const request = new NextRequest("http://localhost/api/orders/guest-access/challenges", {
      method: "POST",
      headers: {
        Cookie: "OMAMA_SESSION=00000000-0000-0000-0000-000000000001",
        Origin: "http://localhost"
      }
    });

    const response = await handleOrderRequest(request, ["guest-access", "challenges"]);

    expect(response.status).toBe(403);
    await expect(response.json()).resolves.toMatchObject({ code: "EMPLOYEE_PURCHASE_NOT_ALLOWED" });
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });
});
