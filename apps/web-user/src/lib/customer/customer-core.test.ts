import { describe, expect, it } from "vitest";
import { mapCustomerCoreAddress, mapCustomerCoreProfile, toCustomerCoreAddressInput, toCustomerCoreProfileUpdate } from "./customer-core";

describe("customer profile mapping", () => {
  it("maps the customer profile and leaves unsupported demographic fields empty", () => {
    const profile = mapCustomerCoreProfile({ customer_id: "CUST-1", phone_number: "0905123456", full_name: "Nguyễn Văn An", email: "an@example.com" });
    expect(profile).toMatchObject({ id: "CUST-1", fullName: "Nguyễn Văn An", email: "an@example.com", emailVerified: false, phone: "0905123456", dateOfBirth: null, gender: null });
    expect(toCustomerCoreProfileUpdate({ fullName: "Nguyễn Văn An", email: "an@example.com" })).toEqual({ full_name: "Nguyễn Văn An", email: "an@example.com" });
  });

  it("maps an address and builds the API write payload", () => {
    const address = mapCustomerCoreAddress({ id: "ADDR-1", recipient_name: "An", phone_number: "0905", street_address: "123 Lê Duẩn", ward: "Phường Thuận Hòa", district: "Thành phố Huế", province: "Thừa Thiên Huế", is_default: true });
    expect(address).toMatchObject({ id: "ADDR-1", recipientName: "An", isDefault: true, province: { code: "HUE" }, ward: { name: "Phường Thuận Hòa" } });
    expect(toCustomerCoreAddressInput({ ...address, isDefault: false })).toMatchObject({ recipient_name: "An", street_address: "123 Lê Duẩn", district: "", is_default: false });
  });
});
