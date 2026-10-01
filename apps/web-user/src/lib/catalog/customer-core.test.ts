import { describe, expect, it } from "vitest";
import { mapCustomerCoreConfig, mapCustomerCoreProductSummary } from "./customer-core";

describe("customer core catalog mapping", () => {
  it("maps category data to the existing discovery config", () => {
    const config = mapCustomerCoreConfig([{ category_id: "CAT-1", name: "Mè Xửng", slug: "me-xung", description: "", image_url: "", parent_id: null }]);
    expect(config.categories).toEqual([{ slug: "me-xung", name: "Mè Xửng" }]);
    expect(config.productTypes).toEqual([]);
  });

  it("maps a mobile product card without exposing inventory quantity", () => {
    const product = mapCustomerCoreProductSummary({
      product_id: "PROD-1",
      name: "Mè xửng giòn",
      slug: "me-xung-gion",
      summary: "Đặc sản Huế",
      thumbnail_url: "https://cdn.omama.vn/product.jpg",
      base_price: { currency_code: "VND", units: 345000, nanos: 0 },
      ocop_star: 4,
      category_name: "Mè Xửng Giòn",
      in_stock: true
    });
    expect(product).toMatchObject({ id: "PROD-1", categorySlug: "me-xung-gion", ocopStars: 4 });
    expect(product.matchedOffer).toMatchObject({ priceVnd: 345000, isAvailable: true });
    expect(product.matchedOffer).not.toHaveProperty("availableQuantity");
  });
});
