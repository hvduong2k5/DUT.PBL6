import { describe, expect, it } from "vitest";
import { parseOrderRoute, validateGuestChallenge, validateGuestOtp, validateMvpCancelRequest } from "./validation";

const ORDER_ID = "11111111-1111-4111-8111-111111111111";

describe("Order route validation", () => {
  it("allows only the public list, detail, tracking, cancel and gated Guest routes", () => {
    expect(parseOrderRoute("GET", [], new URLSearchParams()).operation).toBe("list");
    expect(parseOrderRoute("GET", [ORDER_ID], new URLSearchParams()).operation).toBe("detail");
    expect(parseOrderRoute("GET", [ORDER_ID, "tracking"], new URLSearchParams()).operation).toBe("tracking");
    expect(parseOrderRoute("POST", [ORDER_ID, "cancel"], new URLSearchParams()).operation).toBe("cancel");
    expect(parseOrderRoute("POST", ["guest-access", "challenges"], new URLSearchParams()).operation).toBe("guest-challenge");
    expect(parseOrderRoute("POST", ["guest-access", "challenges", "challenge-001", "verify"], new URLSearchParams()).operation).toBe("guest-verify");
  });

  it("parses opaque cursor pagination and exact Order status filters", () => {
    expect(parseOrderRoute("GET", [], new URLSearchParams("limit=25&status=PAID&cursor=signed%2Bcursor")).listQuery).toEqual({
      limit: 25,
      status: "PAID",
      cursor: "signed+cursor"
    });
    expect(parseOrderRoute("GET", [], new URLSearchParams()).listQuery).toEqual({ limit: 20 });
  });

  it("rejects arbitrary, duplicate or invalid queries and non-UUID Order IDs", () => {
    expect(parseOrderRoute("GET", [], new URLSearchParams("customerId=other")).error?.field).toBe("customerId");
    expect(parseOrderRoute("GET", [], new URLSearchParams("page=1")).error?.field).toBe("page");
    expect(parseOrderRoute("GET", [], new URLSearchParams("limit=0")).error?.field).toBe("limit");
    expect(parseOrderRoute("GET", [], new URLSearchParams("status=PAID&status=SHIPPED")).error?.field).toBe("status");
    expect(parseOrderRoute("GET", [ORDER_ID], new URLSearchParams("cursor=signed")).error?.field).toBe("cursor");
    expect(parseOrderRoute("DELETE", ["order-mock-001"], new URLSearchParams()).error?.field).toBe("path");
  });
});

describe("Order mutation validation", () => {
  it("validates the canonical cancellation body", () => {
    expect(validateMvpCancelRequest({ active: true, reason: "Thay đổi nhu cầu" }).data?.active).toBe(true);
    expect(validateMvpCancelRequest({ active: false, reason: "" }).errors.length).toBeGreaterThan(0);
  });

  it("normalizes Guest lookup and requires a six-digit OTP", () => {
    expect(validateGuestChallenge({ orderNumber: "#oma-260926-001", contact: "0914 288 668" }).data).toEqual({ orderNumber: "OMA-260926-001", contact: "0914288668" });
    expect(validateGuestChallenge({ orderNumber: "ord-20261015-0042", contact: "guest@example.com" }).data).toEqual({ orderNumber: "ORD-20261015-0042", contact: "guest@example.com" });
    expect(validateGuestOtp({ otp: "789214" }).data?.otp).toBe("789214");
    expect(validateGuestOtp({ otp: "123" }).errors[0].field).toBe("otp");
  });
});
