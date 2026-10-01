import { afterEach, describe, expect, it } from "vitest";
import { decodeGuestOrderAccess, encodeGuestOrderAccess } from "./guest-access";

const originalSecret = process.env.ORDER_GUEST_ACCESS_SECRET;

afterEach(() => {
  if (originalSecret === undefined) delete process.env.ORDER_GUEST_ACCESS_SECRET;
  else process.env.ORDER_GUEST_ACCESS_SECRET = originalSecret;
});

describe("guest order access", () => {
  it("round-trips a signed claim bound to one order", () => {
    process.env.ORDER_GUEST_ACCESS_SECRET = "test-guest-order-secret";
    const claim = { orderId: "order-12345678", orderNumber: "OMA-260926-001", accessExpiresAt: Date.now() + 60_000 };
    expect(decodeGuestOrderAccess(encodeGuestOrderAccess(claim))).toEqual(claim);
  });

  it("rejects tampered and expired claims", () => {
    process.env.ORDER_GUEST_ACCESS_SECRET = "test-guest-order-secret";
    const valid = encodeGuestOrderAccess({ orderId: "order-12345678", orderNumber: "OMA-260926-001", accessExpiresAt: Date.now() + 60_000 });
    expect(decodeGuestOrderAccess(`${valid.slice(0, -1)}x`)).toBeUndefined();
    expect(decodeGuestOrderAccess(encodeGuestOrderAccess({ orderId: "order-12345678", orderNumber: "OMA-260926-001", accessExpiresAt: Date.now() - 1 }))).toBeUndefined();
  });
});
