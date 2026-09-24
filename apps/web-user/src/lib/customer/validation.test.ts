import { describe, expect, it } from "vitest";
import { isValidPhone, normalizePhone, validateAddress, validateProfile } from "./validation";

describe("customer profile validation", () => {
  it("normalizes display separators without inventing a country code", () => {
    expect(normalizePhone("(+84) 912-345.892")).toBe("+84912345892");
  });

  it("accepts Vietnamese local and international phone input", () => {
    expect(isValidPhone("0912 345 892")).toBe(true);
    expect(isValidPhone("+84 912 345 892")).toBe(true);
    expect(isValidPhone("abc123")).toBe(false);
  });

  it("rejects a future birthday", () => {
    expect(validateProfile({ fullName: "Tôn Thất Hoàng", phone: null, dateOfBirth: "2999-01-01", gender: null }))
      .toHaveProperty("dateOfBirth");
  });

  it("requires delivery identity and a detailed address", () => {
    const errors = validateAddress({
      label: "",
      type: "HOME",
      recipientName: "",
      recipientPhone: "123",
      province: { code: "", name: "" },
      district: null,
      ward: { code: "", name: "" },
      addressLine: "1",
      deliveryNote: null
    });
    expect(Object.keys(errors)).toEqual(expect.arrayContaining(["label", "recipientName", "recipientPhone", "province", "ward", "addressLine"]));
  });
});
