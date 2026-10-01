import { describe, expect, it } from "vitest";
import { emptyCart, mapCustomerCoreCart } from "./customer-core";

describe("customer core cart mapping", () => {
  it("maps the mobile cart into the existing cart view model", () => {
    const cart = mapCustomerCoreCart({
      cart_id: "CART-1",
      items: [{
        item_id: "ITEM-1",
        sku_code: "MX-GION-500G",
        product_name: "Kẹo Mè Xửng Giòn O Mạ Hộp 500g",
        variant_name: "Hộp 500g",
        thumbnail_url: "https://cdn.omama.vn/product.jpg",
        unit_price: { currency_code: "VND", units: 345000, nanos: 0 },
        quantity: 2,
        subtotal: { currency_code: "VND", units: 690000, nanos: 0 },
        in_stock: true
      }],
      total_items: 2,
      subtotal_amount: { currency_code: "VND", units: 690000, nanos: 0 }
    });
    expect(cart).toMatchObject({ itemCount: 2, subtotalVnd: 690000, hasBlockingIssues: false });
    expect(cart.items[0]).toMatchObject({ skuId: "MX-GION-500G", weightGrams: 500, productSlug: "keo-me-xung-gion-o-ma-hop-500g" });
  });

  it("creates an empty cart after clearing", () => {
    expect(emptyCart()).toMatchObject({ items: [], itemCount: 0, subtotalVnd: 0 });
  });
});
