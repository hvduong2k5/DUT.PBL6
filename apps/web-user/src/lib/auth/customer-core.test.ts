import { describe, expect, it } from "vitest";
import { mapCustomerCoreSession, toCustomerCoreLogin, toCustomerCoreRegistration } from "./customer-core";

describe("Customer API auth adapter", () => {
  it("maps the customer identity without exposing tokens", () => {
    const result = mapCustomerCoreSession({ customer_id: "CUST-8820", phone_number: "0905123456", full_name: "Nguyễn Văn An", email: "an@example.com", membership_tier: "GOLD", loyalty_points: 450 });
    expect(result.customer).toMatchObject({ id: "CUST-8820", displayName: "Nguyễn Văn An", phoneNumber: "0905123456", membershipTier: "GOLD" });
    expect(result).not.toHaveProperty("tokens");
  });

  it("writes phone-based login and registration payloads", () => {
    expect(toCustomerCoreLogin({ phoneNumber: "0905123456", password: "secret" })).toEqual({ phone_number: "0905123456", password: "secret" });
    expect(toCustomerCoreRegistration({ fullName: "Nguyễn Văn An", phoneNumber: "0905123456", email: "an@example.com", password: "secret" })).toEqual({ phone_number: "0905123456", password: "secret", full_name: "Nguyễn Văn An", email: "an@example.com" });
  });
});
