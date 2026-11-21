import { afterEach, describe, expect, it, vi } from "vitest";
import { NextRequest } from "next/server";
import { authorizeCustomerCapability } from "./customer-access-server";
import { authorizeStorefrontCapability, applyStorefrontAuthorization, withGuestCookie } from "./storefront-credential";
import { handleCartRequest } from "@/app/api/cart/cart-handler";
import { POST as createGuestSession } from "@/app/api/guest-session/route";

vi.mock("@/lib/mvp-feature-gates", () => ({ MVP_FEATURE_GATES: { guestOrder: true, cod: true } }));
vi.mock("./customer-access-server", async (original) => ({ ...await original<typeof import("./customer-access-server")>(), authorizeCustomerCapability: vi.fn() }));
afterEach(() => { vi.unstubAllGlobals(); vi.unstubAllEnvs(); vi.clearAllMocks(); });

describe("guest storefront credentials", () => {
  it("rejects client security overrides before issuing a session", async () => {
    const fetcher = vi.fn(); vi.stubGlobal("fetch", fetcher);
    const response = await createGuestSession(new NextRequest("http://localhost/api/guest-session", { method: "POST", headers: { Origin: "http://localhost", "Content-Type": "application/json" }, body: JSON.stringify({ role: "CUSTOMER" }) }));
    expect(response.status).toBe(400);
    expect(fetcher).not.toHaveBeenCalled();
  });
  it("does not send anonymous issuance through the identity customer proxy", async () => {
    vi.stubEnv("WEB_AUTH_MODE", "identity-bff");
    vi.stubEnv("GUEST_SESSION_ISSUANCE_ENABLED", "true");
    vi.stubEnv("CUSTOMER_CORE_UPSTREAM_URL", "http://identity/api/v1/customer-api");
    vi.stubEnv("GUEST_ORDER_UPSTREAM_URL", "");
    const fetcher = vi.fn(); vi.stubGlobal("fetch", fetcher);
    await expect(authorizeStorefrontCapability(new NextRequest("http://localhost/api/guest-session"), "CART_MANAGE", true)).rejects.toMatchObject({ code: "GUEST_ROUTE_UNAVAILABLE" });
    expect(fetcher).not.toHaveBeenCalled();
  });
  it("never switches to a valid guest cookie after customer authorization rejects", async () => {
    const fetcher = vi.fn(); vi.stubGlobal("fetch", fetcher);
    vi.mocked(authorizeCustomerCapability).mockRejectedValueOnce(new Error("expired login / step-up required"));
    const request = new NextRequest("http://localhost/api/cart", { headers: { Cookie: "oma_access_token=expired; oma_guest_session=valid-guest" } });
    await expect(authorizeStorefrontCapability(request, "CART_MANAGE", true)).rejects.toThrow("expired login");
    expect(fetcher).not.toHaveBeenCalled();
  });
  it("rejects malformed login cookies without guest fallback", async () => {
    vi.stubEnv("WEB_AUTH_MODE", "identity-bff");
    const request = new NextRequest("http://localhost/api/cart", { headers: { Cookie: "OMAMA_SESSION=invalid; oma_guest_session=valid-guest" } });
    await expect(authorizeStorefrontCapability(request, "CART_MANAGE", true)).rejects.toMatchObject({ status: 401 });
  });
  it("does not issue sessions on a cart read and skips identity for guests", async () => {
    const fetcher = vi.fn(); vi.stubGlobal("fetch", fetcher);
    const response = await handleCartRequest(new NextRequest("http://localhost/api/cart"), []);
    expect(await response.json()).toEqual({ revision: 0, items: [] });
    expect(fetcher).not.toHaveBeenCalled();
    expect(authorizeCustomerCapability).not.toHaveBeenCalled();
  });
  it("issues with no login material and stores an httpOnly storefront cookie", async () => {
    vi.stubEnv("GUEST_SESSION_ISSUANCE_ENABLED", "true");
    vi.stubEnv("GUEST_ORDER_UPSTREAM_URL", "http://gateway/api/v1");
    const fetcher = vi.fn().mockResolvedValue(Response.json({ access_token: "guest.jwt.signature" }, { status: 201 }));
    vi.stubGlobal("fetch", fetcher);
    const request = new NextRequest("https://shop.example/api/guest-session", { method: "POST" });
    const response = await withGuestCookie(request, async () => {
      await authorizeStorefrontCapability(request, "CART_MANAGE", true);
      const headers = new Headers(); applyStorefrontAuthorization(headers, request);
      expect(headers.get("Authorization")).toBe("Bearer guest.jwt.signature");
      expect(headers.has("Cookie")).toBe(false);
      return Response.json({ ready: true });
    });
    expect(fetcher.mock.calls[0][0]).toBe("http://gateway/api/v1/guest-sessions");
    expect(fetcher.mock.calls[0][1].headers).not.toHaveProperty("Authorization");
    const cookie = response.headers.get("set-cookie");
    expect(cookie).toContain("oma_guest_session=guest.jwt.signature");
    for (const attribute of ["HttpOnly", "Secure", "SameSite=lax", "Path=/"]) expect(cookie).toContain(attribute);
    expect(authorizeCustomerCapability).not.toHaveBeenCalled();
  });
  it("does not grant account-only capabilities to a guest", async () => {
    const request = new NextRequest("http://localhost/api/customer/profile", { headers: { Cookie: "oma_guest_session=guest" } });
    await expect(authorizeStorefrontCapability(request, "PROFILE_MANAGE")).rejects.toMatchObject({ status: 401 });
  });
});
