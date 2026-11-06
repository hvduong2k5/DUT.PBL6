import { describe, expect, it } from "vitest";
import { emptyCart, mapCustomerCoreCart, withCartItemQuantity } from "./customer-core";

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

  it("derives totals from quantity and unit price when a mock response has stale totals", () => {
    const cart = mapCustomerCoreCart({
      cart_id: "CART-1",
      items: [{
        item_id: "ITEM-1",
        sku_code: "MX-GION-500G",
        product_name: "Kẹo Mè Xửng",
        variant_name: "Hộp 500g",
        thumbnail_url: "",
        unit_price: { currency_code: "VND", units: 345000, nanos: 0 },
        quantity: 2,
        subtotal: { currency_code: "VND", units: 345000, nanos: 0 },
        in_stock: true
      }],
      total_items: 3,
      subtotal_amount: { currency_code: "VND", units: 345000, nanos: 0 }
    });

    expect(cart).toMatchObject({ itemCount: 2, subtotalVnd: 690000 });
    expect(cart.items[0]).toMatchObject({ quantity: 2, lineSubtotalVnd: 690000 });
  });

  it("recalculates line and cart totals after a local mock quantity update", () => {
    const cart = withCartItemQuantity({
      items: [{
        itemId: "ITEM-1", productId: "", productSlug: "keo", productName: "Kẹo", imageUrl: "", imageAlt: "Kẹo",
        skuId: "SKU-1", skuLabel: "Hộp", weightGrams: 500, packageType: "Hộp", unitPriceVnd: 345000,
        priceChanged: false, quantity: 2, lineSubtotalVnd: 690000, isAvailable: true
      }],
      itemCount: 2,
      subtotalVnd: 690000,
      hasBlockingIssues: false,
      notices: [],
      updatedAt: "2026-10-03T00:00:00.000Z"
    }, "ITEM-1", 3);

    expect(cart).toMatchObject({ itemCount: 3, subtotalVnd: 1035000 });
    expect(cart.items[0]).toMatchObject({ quantity: 3, lineSubtotalVnd: 1035000 });
  });
});
