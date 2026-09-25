import { afterEach, describe, expect, it, vi } from "vitest";
import { decodePaymentAccess, encodePaymentAccess, paymentAccessCookie, type PaymentAccessClaim } from "./access";

function claim(overrides: Partial<PaymentAccessClaim> = {}): PaymentAccessClaim {
  return {
    orderId: "order-mock-001",
    orderNumber: "OMA-001",
    method: "BANK_TRANSFER",
    amountVnd: 665_000,
    paymentExpiresAt: new Date(Date.now() + 15 * 60_000).toISOString(),
    accessExpiresAt: Date.now() + 30 * 60_000,
    ...overrides
  };
}

describe("payment access", () => {
  afterEach(() => vi.unstubAllEnvs());

  it("round-trips a valid signed claim and creates a server-only cookie", () => {
    vi.stubEnv("PAYMENT_ACCESS_SECRET", "test-payment-secret-that-is-not-public");
    const source = claim();

    expect(decodePaymentAccess(encodePaymentAccess(source))).toEqual(source);
    expect(paymentAccessCookie(source)).toContain("HttpOnly; SameSite=Lax");
    expect(paymentAccessCookie(source)).toContain("Path=/api/payments");
  });

  it("rejects tampered and expired claims", () => {
    vi.stubEnv("PAYMENT_ACCESS_SECRET", "test-payment-secret-that-is-not-public");
    const encoded = encodePaymentAccess(claim());
    const [payload, signature] = encoded.split(".");

    expect(decodePaymentAccess(`${payload}x.${signature}`)).toBeUndefined();
    expect(decodePaymentAccess(encodePaymentAccess(claim({ accessExpiresAt: Date.now() - 1 })))).toBeUndefined();
  });
});
