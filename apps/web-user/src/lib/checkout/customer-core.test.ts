import { describe, expect, it } from "vitest";
import { buildCustomerCoreCheckoutRequest, mapCustomerCoreCheckoutConfirmation, mapCustomerCoreCheckoutPreferences } from "./customer-core";

describe("customer core checkout mapping", () => {
  it("maps the current checkout form and lines to POST /checkout", () => {
    const request = buildCustomerCoreCheckoutRequest([{
      itemId: "ITEM-1", productSlug: "product", productName: "Product", imageUrl: "", imageAlt: "Product",
      skuId: "MX-500", skuLabel: "Hộp 500g", weightGrams: 500, packageType: "Hộp", unitPriceVnd: 345000,
      quantity: 2, lineSubtotalVnd: 690000, priceChanged: false
    }], { fullName: "Nguyễn Văn An", phone: "0905123456" }, {
      provinceCode: "HUE", provinceName: "Thành phố Huế", wardCode: "VY-DA", wardName: "Phường Vỹ Dạ", addressLine: "123 Nguyễn Sinh Cung"
    }, "BANK_TRANSFER", "OMAMA_FREESHIP_30K");
    expect(request).toMatchObject({ channel: "D2C_WEB", payment_method: "VIETQR", voucher_code: "OMAMA_FREESHIP_30K" });
    expect(request.items[0]).toMatchObject({ sku_code: "MX-500", quantity: 2, price: { units: 345000 } });
  });

  it("maps VietQR checkout output to the existing confirmation view", () => {
    const confirmation = mapCustomerCoreCheckoutConfirmation({
      order_id: "ORD-1", status: "PENDING_PAYMENT",
      subtotal_amount: { currency_code: "VND", units: 345000, nanos: 0 },
      discount_amount: { currency_code: "VND", units: 0, nanos: 0 },
      shipping_fee: { currency_code: "VND", units: 30000, nanos: 0 },
      final_amount: { currency_code: "VND", units: 375000, nanos: 0 },
      vietqr_url: "https://img.vietqr.io/example.png", payment_expires_at: "2026-10-15T08:45:00.000Z"
    }, "BANK_TRANSFER");
    expect(confirmation).toMatchObject({ orderId: "ORD-1", totalVnd: 375000, nextStep: "PAYMENT_REQUIRED" });
  });

  it("maps registered profile and address data to checkout-only defaults", () => {
    const preferences = mapCustomerCoreCheckoutPreferences({
      customer_id: "CUST-1", phone_number: "0905123456", full_name: "Nguyễn Văn An", email: "an@example.com"
    }, [{
      id: "ADDR-1", recipient_name: "Nguyễn Văn An", phone_number: "0905123456", street_address: "123 Lê Duẩn",
      ward: "Phường Thuận Hòa", district: "Thành phố Huế", province: "Tỉnh Thừa Thiên Huế", is_default: true
    }]);
    expect(preferences.defaultRecipient).toEqual({ fullName: "Nguyễn Văn An", phone: "0905123456", email: "an@example.com" });
    expect(preferences.savedAddresses[0]).toMatchObject({
      addressId: "ADDR-1", isDefault: true, address: { provinceCode: "HUE", wardCode: "THUAN-HOA", wardName: "Phường Thuận Hòa" }
    });
  });
});
