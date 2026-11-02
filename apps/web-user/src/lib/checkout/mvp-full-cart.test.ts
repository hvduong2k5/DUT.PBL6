import { describe, expect, it } from "vitest";
import type { Cart } from "@/lib/cart/types";
import { mapMvpCheckoutPreparation } from "./mvp";

const cart: Cart = {
  revision: 7,
  itemCount: 3,
  subtotalVnd: 665000,
  hasBlockingIssues: false,
  notices: [],
  items: [
    {
      itemId: "line-1", productId: "product-1", productSlug: "banh-ngu-sac", productName: "Bánh Ngũ Sắc",
      imageUrl: "https://example.com/1.jpg", imageAlt: "Bánh", skuId: "SKU-001", skuLabel: "Hộp 250g",
      weightGrams: 250, packageType: "Hộp", unitPriceVnd: 185000, priceChanged: false, quantity: 2,
      lineSubtotalVnd: 370000, isAvailable: true
    },
    {
      itemId: "line-2", productId: "product-2", productSlug: "tra-sen", productName: "Trà Sen",
      imageUrl: "https://example.com/2.jpg", imageAlt: "Trà", skuId: "SKU-002", skuLabel: "Hộp 150g",
      weightGrams: 150, packageType: "Hộp", unitPriceVnd: 295000, priceChanged: false, quantity: 1,
      lineSubtotalVnd: 295000, isAvailable: true
    }
  ]
};

describe("MVP full-cart checkout preparation", () => {
  it("keeps every server cart line at the current revision", () => {
    const result = mapMvpCheckoutPreparation(cart, [], undefined, []);
    expect(result.cartRevision).toBe(7);
    expect(result.items.map((item) => item.itemId)).toEqual(["line-1", "line-2"]);
    expect(result.itemCount).toBe(3);
    expect(result.subtotalVnd).toBe(665000);
  });
});
