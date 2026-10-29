import { describe, expect, it } from "vitest";
import { parsePaymentRoute } from "./validation";

const ORDER_ID = "11111111-1111-4111-8111-111111111111";

describe("parsePaymentRoute", () => {
  it("allows only known order payment routes", () => {
    expect(parsePaymentRoute("GET", ["orders", ORDER_ID, "instructions"], new URLSearchParams()).operation).toBe("instructions");
    expect(parsePaymentRoute("GET", ["orders", ORDER_ID, "status"], new URLSearchParams()).operation).toBe("status");
  });

  it("rejects arbitrary paths and queries", () => {
    expect(parsePaymentRoute("GET", ["admin", "payments"], new URLSearchParams()).error?.field).toBe("path");
    expect(parsePaymentRoute("GET", ["orders", ORDER_ID, "status"], new URLSearchParams("redirect=https://example.com")).error?.field).toBe("redirect");
    expect(parsePaymentRoute("POST", ["orders", ORDER_ID, "attempts"], new URLSearchParams()).error?.field).toBe("path");
  });
});
