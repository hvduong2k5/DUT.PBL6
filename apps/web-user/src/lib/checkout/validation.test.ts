import { describe, expect, it } from "vitest";
import { parseCheckoutRoute, validateConfirmCheckoutInput, validateMvpQuoteRequest, validatePrepareCheckoutInput, validateQuoteShippingInput } from "./validation";

const address = {
  provinceCode: "HUE", provinceName: "Thành phố Huế", wardCode: "PHU-NHUAN",
  wardName: "Phường Phú Nhuận", addressLine: "Tầng 3, 84 Nguyễn Huệ"
};

describe("Checkout route allowlist", () => {
  it("allows only prepare, quote and confirm POST routes", () => {
    expect(parseCheckoutRoute("POST", ["prepare"], new URLSearchParams()).operation).toBe("prepare");
    expect(parseCheckoutRoute("POST", ["shipping-quotes"], new URLSearchParams()).operation).toBe("quote-shipping");
    expect(parseCheckoutRoute("POST", ["confirm"], new URLSearchParams()).operation).toBe("confirm");
  });

  it("rejects arbitrary method, path and query", () => {
    expect(parseCheckoutRoute("GET", ["prepare"], new URLSearchParams()).error?.field).toBe("path");
    expect(parseCheckoutRoute("POST", ["proxy", "https://example.com"], new URLSearchParams()).error?.field).toBe("path");
    expect(parseCheckoutRoute("POST", ["prepare"], new URLSearchParams("upstream=x")).error?.field).toBe("upstream");
  });
});

describe("Checkout request validation", () => {
  it("requires exactly one shipping address representation for an MVP quote", () => {
    const inlineAddress = { recipient_name: "Nguyễn Văn An", phone: "0914288668", street: "84 Nguyễn Huệ", ward_code: "WARD-1", province_code: "PROVINCE-1" };
    expect(validateMvpQuoteRequest({ cart_revision: 3, payment_method: "VIETQR", address: inlineAddress }).errors).toEqual([]);
    expect(validateMvpQuoteRequest({ cart_revision: 3, payment_method: "VIETQR", shipping_address_id: "11111111-1111-4111-8111-111111111111" }).errors).toEqual([]);
    expect(validateMvpQuoteRequest({ cart_revision: 3, payment_method: "VIETQR" }).errors[0].field).toBe("address");
    expect(validateMvpQuoteRequest({ cart_revision: 3, payment_method: "VIETQR", address: inlineAddress, shipping_address_id: "11111111-1111-4111-8111-111111111111" }).errors[0].field).toBe("address");
  });

  it("accepts unique selected cart lines and a complete quote request", () => {
    expect(validatePrepareCheckoutInput({ itemIds: ["line-1", "line-2"] }).data?.itemIds).toHaveLength(2);
    expect(validateQuoteShippingInput({ checkoutSessionId: "checkout-123e4567-e89b-12d3-a456-426614174000", itemIds: ["line-1"], address }).errors).toEqual([]);
  });

  it("rejects duplicate lines, invalid addresses and client-computed money", () => {
    expect(validatePrepareCheckoutInput({ itemIds: ["line-1", "line-1"] }).errors[0].field).toBe("itemIds");
    expect(validateQuoteShippingInput({ checkoutSessionId: "bad", itemIds: ["line-1"], address: { ...address, addressLine: "Huế" } }).errors.length).toBeGreaterThan(0);
    expect(validateConfirmCheckoutInput({ subtotalVnd: 1 }).errors[0].field).toBe("body");
  });

  it("accepts a confirmation without trusting totals from the browser", () => {
    const result = validateConfirmCheckoutInput({
      checkoutSessionId: "checkout-123e4567-e89b-12d3-a456-426614174000",
      itemIds: ["line-1"], recipient: { fullName: "Nguyễn Văn An", phone: "0914288668", email: "an@example.com" },
      address, shippingOptionId: "STANDARD", paymentMethod: "BANK_TRANSFER",
      idempotencyKey: "checkout-request-123456", priceRevalidatedAt: "2026-09-25T10:00:00.000Z",
      priceChangesAcknowledged: false, voucherCode: "OMAMA_FREESHIP_30K"
    });
    expect(result.errors).toEqual([]);
  });
});
