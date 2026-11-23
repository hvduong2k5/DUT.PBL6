import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { NextRequest } from "next/server";
let authorizeCustomerCapability: typeof import("./customer-access-server")["authorizeCustomerCapability"];
let authorizeStorefrontCapability: typeof import("./storefront-credential")["authorizeStorefrontCapability"];
let applyStorefrontAuthorization: typeof import("./storefront-credential")["applyStorefrontAuthorization"];
let withGuestCookie: typeof import("./storefront-credential")["withGuestCookie"];
let handleCartRequest: typeof import("@/app/api/cart/cart-handler")["handleCartRequest"];
let createGuestSession: typeof import("@/app/api/guest-session/route")["POST"];

beforeEach(async () => {
  vi.resetModules();
  vi.doUnmock("./customer-access-server");
  vi.doMock("@/lib/mvp-feature-gates", () => ({ MVP_FEATURE_GATES: { guestOrder: true, cod: true } }));
  const access = await import("./customer-access-server");
  authorizeCustomerCapability = vi.spyOn(access, "authorizeCustomerCapability");
  ({ authorizeStorefrontCapability, applyStorefrontAuthorization, withGuestCookie } = await import("./storefront-credential"));
  ({ handleCartRequest } = await import("@/app/api/cart/cart-handler"));
  ({ POST: createGuestSession } = await import("@/app/api/guest-session/route"));
});
afterEach(() => { vi.unstubAllGlobals(); vi.unstubAllEnvs(); vi.restoreAllMocks(); vi.doUnmock("@/lib/mvp-feature-gates"); });

describe("guest storefront credentials", () => {
  it("keeps a disabled guest mutation from issuing or forwarding anything", async () => {
    const fetcher = vi.fn(); vi.stubGlobal("fetch", fetcher);
    const request = new NextRequest("http://localhost/api/cart/items", { method: "POST", headers: { Origin: "http://localhost" }, body: JSON.stringify({ sku_code: "MX-GION-500G", quantity: 2, revision: 0 }) });
    const response = await handleCartRequest(request, ["items"]);
    expect(response.status).toBe(403);
    expect(fetcher).not.toHaveBeenCalled();
  });
  it("forwards a guest cart mutation using only its guest credential", async () => {
    vi.stubEnv("GUEST_CART_MUTATION_ENABLED", "true");
    vi.stubEnv("GUEST_ORDER_UPSTREAM_URL", "http://gateway/api/v1");
    const fetcher = vi.fn().mockResolvedValue(Response.json({ revision: 1, items: [{ sku_code: "MX-GION-500G", quantity: 2 }] })); vi.stubGlobal("fetch", fetcher);
    const request = new NextRequest("http://localhost/api/cart/items", { method: "POST", headers: { Origin: "http://localhost", Cookie: "oma_guest_session=guest-token" }, body: JSON.stringify({ sku_code: "MX-GION-500G", quantity: 2, revision: 0 }) });
    const response = await handleCartRequest(request, ["items"]);
    expect(response.status).toBe(200);
    expect(fetcher.mock.calls[0][0]).toBe("http://gateway/api/v1/cart/items");
    expect(fetcher.mock.calls[0][1].headers.get("Authorization")).toBe("Bearer guest-token");
    expect(fetcher.mock.calls[0][1].headers.has("Cookie")).toBe(false);
    expect(authorizeCustomerCapability).not.toHaveBeenCalled();
  });
  it("keeps VietQR disabled independently of guest COD", async () => {
    vi.stubEnv("GUEST_COD_CHECKOUT_ENABLED", "true");
    const fetcher = vi.fn(); vi.stubGlobal("fetch", fetcher);
    const { handleCheckoutRequest } = await import("@/app/api/checkout/checkout-handler");
    const request = new NextRequest("http://localhost/api/checkout/quote", { method: "POST", headers: { Origin: "http://localhost", Cookie: "oma_guest_session=guest-token" }, body: JSON.stringify({ cart_revision: 1, payment_method: "VIETQR", address: { recipient_name: "Guest Customer", phone: "0901234567", street: "1 Test Street", province_code: "75", ward_code: "HUE-01" } }) });
    const response = await handleCheckoutRequest(request, ["quote"]);
    expect(response.status).toBe(403);
    expect(fetcher).not.toHaveBeenCalled();
  });
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
