import { describe, expect, it } from "vitest";
import { parsePaymentRoute, validateRetryInput } from "./validation";

describe("parsePaymentRoute", () => {
  it("allows only known order payment routes", () => {
    expect(parsePaymentRoute("GET", ["orders", "order-mock-001"], new URLSearchParams()).operation).toBe("summary");
    expect(parsePaymentRoute("GET", ["orders", "order-mock-001", "status"], new URLSearchParams()).operation).toBe("status");
    expect(parsePaymentRoute("POST", ["orders", "order-mock-001", "attempts"], new URLSearchParams()).operation).toBe("retry");
  });

  it("rejects arbitrary paths and queries", () => {
    expect(parsePaymentRoute("GET", ["admin", "payments"], new URLSearchParams()).error?.field).toBe("path");
    expect(parsePaymentRoute("GET", ["orders", "order-mock-001"], new URLSearchParams("redirect=https://example.com")).error?.field).toBe("redirect");
  });
});

describe("validateRetryInput", () => {
  it("accepts a generated payment retry key", () => {
    expect(validateRetryInput({ idempotencyKey: "payment-retry-550e8400-e29b-41d4-a716-446655440000" }).error).toBeUndefined();
  });

  it("rejects amount and unknown fields from the browser", () => {
    expect(validateRetryInput({ idempotencyKey: "bad", amountVnd: 1 }).error?.field).toBe("idempotencyKey");
  });
});
