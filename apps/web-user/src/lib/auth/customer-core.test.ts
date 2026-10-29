import { describe, expect, it } from "vitest";
import { mapCustomerCoreSession, toCustomerCoreLogin, toCustomerCoreRegistration } from "./customer-core";

describe("Customer API auth adapter", () => {
  it("maps the customer identity without exposing tokens", () => {
    const result = mapCustomerCoreSession({ customer_id: "CUST-8820", phone_number: "0905123456", full_name: "Nguyễn Văn An", email: "an@example.com", membership_tier: "GOLD", loyalty_points: 450 });
    expect(result).toBeDefined();
    if (!result) return;
    expect(result.customer).toMatchObject({ id: "CUST-8820", displayName: "Nguyễn Văn An", phoneNumber: "0905123456", membershipTier: "GOLD" });
    expect(result.principal).toMatchObject({ principalId: "CUST-8820", principalType: "CUSTOMER", roles: [{ roleCode: "CUSTOMER" }] });
    expect(result.customerAccess).toBe(true);
    expect(result).not.toHaveProperty("tokens");
  });

  it("keeps an employee out of Customer features until backend returns a customer mapping", () => {
    const withoutMapping = mapCustomerCoreSession({ principal_id: "PRINCIPAL-EMP-01", principal_type: "EMPLOYEE", full_name: "Nhân viên kho", roles: ["CUSTOMER", "WAREHOUSE_STAFF"], permissions: ["INVENTORY_VIEW"] });
    expect(withoutMapping).toMatchObject({ authenticated: true, customer: null, customerAccess: false });

    const withMapping = mapCustomerCoreSession({ principal_id: "PRINCIPAL-EMP-01", principal_type: "EMPLOYEE", customer_id: "CUST-EMP-01", full_name: "Nhân viên kho", roles: ["CUSTOMER", "WAREHOUSE_STAFF"], permissions: ["INVENTORY_VIEW"] });
    expect(withMapping).toMatchObject({ authenticated: true, customerAccess: true, customer: { id: "CUST-EMP-01" } });
  });

  it("writes phone-based login and registration payloads", () => {
    expect(toCustomerCoreLogin({ phoneNumber: "0905123456", password: "secret" })).toEqual({ phone_number: "0905123456", password: "secret" });
    expect(toCustomerCoreRegistration({ fullName: "Nguyễn Văn An", phoneNumber: "0905123456", email: "an@example.com", password: "secret" })).toEqual({ phone_number: "0905123456", password: "secret", full_name: "Nguyễn Văn An", email: "an@example.com" });
  });
});
