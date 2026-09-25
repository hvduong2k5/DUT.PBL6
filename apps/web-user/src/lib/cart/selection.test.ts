import { describe, expect, it } from "vitest";
import type { CartItem } from "./types";
import { getInitiallySelectedItemIds, getSelectedCartSummary, reconcileSelectedItemIds } from "./selection";

const items: CartItem[] = [
  {
    itemId: "line-1", productId: "PRD-001", productSlug: "banh-ngu-sac", productName: "Bánh Ngũ Sắc",
    imageUrl: "https://example.com/1.jpg", imageAlt: "Bánh", skuId: "SKU-001", skuLabel: "Hộp 250g",
    weightGrams: 250, packageType: "Hộp", unitPriceVnd: 185000, priceChanged: false, quantity: 2,
    lineSubtotalVnd: 370000, isAvailable: true
  },
  {
    itemId: "line-2", productId: "PRD-002", productSlug: "tra-sen", productName: "Trà Sen",
    imageUrl: "https://example.com/2.jpg", imageAlt: "Trà", skuId: "SKU-002", skuLabel: "Hộp 150g",
    weightGrams: 150, packageType: "Hộp thiếc", unitPriceVnd: 295000, priceChanged: false, quantity: 1,
    lineSubtotalVnd: 295000, isAvailable: true
  },
  {
    itemId: "line-3", productId: "PRD-003", productSlug: "mut-sen", productName: "Mứt Sen",
    imageUrl: "https://example.com/3.jpg", imageAlt: "Mứt", skuId: "SKU-003", skuLabel: "Hũ 200g",
    weightGrams: 200, packageType: "Hũ", unitPriceVnd: 165000, priceChanged: false, quantity: 4,
    lineSubtotalVnd: null, isAvailable: false, unavailableReason: "Tạm hết hàng"
  }
];

describe("cart purchase selection", () => {
  it("selects all available lines initially and excludes unavailable lines", () => {
    expect(getInitiallySelectedItemIds(items)).toEqual(["line-1", "line-2"]);
  });

  it("calculates quantity and subtotal from selected available lines only", () => {
    expect(getSelectedCartSummary(items, ["line-2", "line-3"])).toEqual({
      lineCount: 1,
      itemCount: 1,
      subtotalVnd: 295000
    });
  });

  it("removes deleted or newly unavailable lines while preserving valid choices", () => {
    expect(reconcileSelectedItemIds(items.slice(1), ["line-1", "line-2", "line-3"])).toEqual(["line-2"]);
  });
});
