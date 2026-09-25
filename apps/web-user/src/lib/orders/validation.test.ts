import { describe, expect, it } from "vitest";
import { parseOrderRoute, validateCancelOrder, validateGuestChallenge, validateGuestOtp } from "./validation";

describe("Order route validation", () => {
  it("allows only the list, detail, cancellation and Guest challenge routes", () => {
    expect(parseOrderRoute("GET", [], new URLSearchParams()).operation).toBe("list");
    expect(parseOrderRoute("GET", ["order-mock-001"], new URLSearchParams()).operation).toBe("detail");
    expect(parseOrderRoute("POST", ["order-mock-001", "cancellations"], new URLSearchParams()).operation).toBe("cancel");
    expect(parseOrderRoute("POST", ["guest-access", "challenges"], new URLSearchParams()).operation).toBe("guest-challenge");
    expect(parseOrderRoute("POST", ["guest-access", "challenges", "challenge-001", "verify"], new URLSearchParams()).operation).toBe("guest-verify");
  });

  it("rejects arbitrary queries and invalid pagination", () => {
    expect(parseOrderRoute("GET", [], new URLSearchParams("customerId=other")).error?.field).toBe("customerId");
    expect(parseOrderRoute("GET", [], new URLSearchParams("page=0")).error?.field).toBe("page");
    expect(parseOrderRoute("DELETE", ["order-mock-001"], new URLSearchParams()).error?.field).toBe("path");
  });
});

describe("Order mutation validation", () => {
  it("validates cancellation reason and idempotency", () => {
    expect(validateCancelOrder({ reasonCode: "CHANGED_MIND", idempotencyKey: "order-cancel-550e8400-e29b-41d4-a716-446655440000" }).data?.reasonCode).toBe("CHANGED_MIND");
    expect(validateCancelOrder({ reasonCode: "OTHER", note: "x", idempotencyKey: "bad" }).errors.length).toBeGreaterThan(0);
  });

  it("normalizes Guest lookup and requires a six-digit OTP", () => {
    expect(validateGuestChallenge({ orderNumber: "#oma-260926-001", contact: "0914 288 668" }).data).toEqual({ orderNumber: "OMA-260926-001", contact: "0914288668" });
    expect(validateGuestOtp({ otp: "789214" }).data?.otp).toBe("789214");
    expect(validateGuestOtp({ otp: "123" }).errors[0].field).toBe("otp");
  });
});
