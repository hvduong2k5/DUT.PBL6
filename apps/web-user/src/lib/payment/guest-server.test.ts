import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { NextRequest } from "next/server";

const order = "10000000-0000-4000-8000-000000000053";
let server: typeof import("./guest-server");
let handler: typeof import("@/app/api/payments/payment-handler");
let access: typeof import("@/lib/auth/customer-access-server");
beforeEach(async () => {
  vi.resetModules();
  vi.doUnmock("@/lib/auth/customer-access-server");
  vi.doMock("@/lib/mvp-feature-gates", () => ({ MVP_FEATURE_GATES: { guestOrder: true } }));
  vi.stubEnv("WEB_AUTH_MODE", "legacy");
  vi.stubEnv("GUEST_ORDER_UPSTREAM_URL", "http://gateway/api/v1");
  vi.stubEnv("GUEST_VIETQR_CHECKOUT_ENABLED", "true");
  access = await import("@/lib/auth/customer-access-server");
  vi.spyOn(access, "authorizeCustomerCapability");
  server = await import("./guest-server");
  handler = await import("@/app/api/payments/payment-handler");
});
afterEach(() => { vi.unstubAllGlobals(); vi.unstubAllEnvs(); vi.restoreAllMocks(); vi.doUnmock("@/lib/mvp-feature-gates"); });

it("mints with the guest session and keeps the grant in an order-scoped httpOnly cookie", async () => {
  const fetcher = vi.fn().mockResolvedValue(Response.json({ access_token: "order-grant", token_type: "Bearer", order_id: order, expires_at: new Date(Date.now() + 600000).toISOString() }));
  vi.stubGlobal("fetch", fetcher);
  const request = new NextRequest(`https://shop.example/api/payments/orders/${order}/guest-access`, { method: "POST", headers: { Cookie: "oma_guest_session=guest-session", Origin: "https://shop.example" }, body: "{}" });
  const response = await server.handleGuestPaymentAccess(request, order);
  expect(response.status).toBe(204);
  expect(await response.text()).toBe("");
  expect(fetcher.mock.calls[0][1].headers.Authorization).toBe("Bearer guest-session");
  const cookie = response.headers.get("set-cookie");
  for (const value of [server.guestPaymentCookie(order), "order-grant", "HttpOnly", "Secure", "SameSite=lax", `Path=/api/payments/orders/${order}`]) expect(cookie).toContain(value);
  expect(access.authorizeCustomerCapability).not.toHaveBeenCalled();
});

it("polls with the payment grant only even when the session cookie is present", async () => {
  const fetcher = vi.fn().mockResolvedValue(Response.json({ order_id: order, intent_id: "20000000-0000-4000-8000-000000000053", provider: "SEPAY", method: "VIETQR", order_status: "PENDING_PAYMENT", payment_status: "PENDING", display_state: "WAITING_PAYMENT", intent_status: "ACTIVE", expected_amount: 1000, allocated_amount: 0, refunded_amount: 0, currency: "VND", expires_at: new Date(Date.now() + 600000).toISOString(), server_time: new Date().toISOString(), has_open_reconciliation: false, can_pay: true, version: 1, verification_started_at: null, verification_due_at: null, support_reference: null, support_reason: null, next_poll_after_seconds: 2 }));
  vi.stubGlobal("fetch", fetcher);
  const request = new NextRequest(`http://localhost/api/payments/orders/${order}/status`, { headers: { Cookie: `oma_guest_session=session-token; ${server.guestPaymentCookie(order)}=order-grant` } });
  const response = await handler.handlePaymentRequest(request, ["orders", order, "status"]);
  expect(response.status).toBe(200);
  const headers = fetcher.mock.calls[0][1].headers as Headers;
  expect(headers.get("Authorization")).toBe("Bearer order-grant");
  expect(headers.has("Cookie")).toBe(false);
  expect(access.authorizeCustomerCapability).not.toHaveBeenCalled();
});

it("rejects an invalid customer login without selecting the valid guest payment cookie", async () => {
  vi.mocked(access.authorizeCustomerCapability).mockRejectedValue(new access.CustomerAccessError(401, "AUTH_REQUIRED", "Expired"));
  const fetcher = vi.fn(); vi.stubGlobal("fetch", fetcher);
  const request = new NextRequest(`http://localhost/api/payments/orders/${order}/status`, { headers: { Cookie: `oma_access_token=expired; oma_guest_session=guest; ${server.guestPaymentCookie(order)}=valid-grant` } });
  const response = await handler.handlePaymentRequest(request, ["orders", order, "status"]);
  expect(response.status).toBe(401);
  expect(fetcher).not.toHaveBeenCalled();
});

it("does not use another order's payment cookie", async () => {
  const fetcher = vi.fn(); vi.stubGlobal("fetch", fetcher);
  const request = new NextRequest(`http://localhost/api/payments/orders/${order}/status`, { headers: { Cookie: `${server.guestPaymentCookie("30000000-0000-4000-8000-000000000053")}=other-grant` } });
  const response = await handler.handlePaymentRequest(request, ["orders", order, "status"]);
  expect(response.status).toBe(401);
  expect(fetcher).not.toHaveBeenCalled();
});
