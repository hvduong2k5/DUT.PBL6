import { describe, expect, it } from "vitest";
import type { OrderQuote } from "@/lib/api-contract";
import { mapMvpCheckoutQuote, buildMvpQuoteRequest } from "./mvp";

const quote = {
  quote_id: "quote-1", cart_revision: 1, payment_method: "VIETQR", subtotal: 100000,
  shipping_fee: 20000, merchandise_discount: 10000, shipping_discount: 0, final_amount: 110000,
  voucher_code: "WELCOME", expires_at: "2026-11-14T10:00:00Z", shipping_expires_at: "2026-11-14T10:01:00Z",
  items: [{ sku_code: "SKU", quantity: 1, product_name: "Product", unit_price: 100000 }]
} as OrderQuote;

describe("server-owned checkout discounts", () => {
  it("displays the voucher and its authoritative discount", () => {
    expect(mapMvpCheckoutQuote(quote)).toMatchObject({ voucherCode: "WELCOME", merchandiseDiscountVnd: 10000, shippingDiscountVnd: 0, totalVnd: 110000 });
  });
  it("rejects incoherent or excessive discounts", () => {
    expect(() => mapMvpCheckoutQuote({ ...quote, final_amount: 120000 })).toThrow();
    expect(() => mapMvpCheckoutQuote({ ...quote, shipping_discount: 30000 })).toThrow();
  });
  it("keeps undiscounted existing quotes compatible", () => {
    expect(mapMvpCheckoutQuote({ ...quote, merchandise_discount: undefined, shipping_discount: undefined, final_amount: 120000 }).totalVnd).toBe(120000);
  });
  it("submits only the voucher code with the quote input", () => {
    const input = buildMvpQuoteRequest({ cartRevision: 1, paymentMethod: "VIETQR", selectedAddressId: "address", recipient: { fullName: "Test", phone: "0900000000", email: "" }, address: { provinceCode: "75", provinceName: "Hue", wardCode: "01", wardName: "Ward", addressLine: "Street" }, voucherCode: "WELCOME" });
    expect(input).toEqual({ cart_revision: 1, payment_method: "VIETQR", shipping_address_id: "address", voucher_code: "WELCOME" });
  });
});
