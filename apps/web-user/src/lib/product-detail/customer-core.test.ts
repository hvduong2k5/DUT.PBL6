import { describe, expect, it } from "vitest";
import { mapCustomerCoreProductDetail } from "./customer-core";

describe("customer core product detail mapping", () => {
  it("maps variants to selectable SKUs and omits unavailable food information", () => {
    const detail = mapCustomerCoreProductDetail({
      product_id: "PROD-1",
      name: "Mè xửng giòn",
      slug: "me-xung-gion",
      description: "Đặc sản Huế",
      images: ["https://cdn.omama.vn/product.jpg"],
      ocop_star: 4,
      ocop_certificate_no: "OCOP-001",
      story: "Ba đời giữ nghề",
      variants: [{
        variant_id: "VAR-1",
        sku_code: "MX-500",
        name: "Hộp 500g",
        weight_gram: 500,
        price: { currency_code: "VND", units: 345000, nanos: 0 },
        original_price: { currency_code: "VND", units: 345000, nanos: 0 },
        stock_available: 10
      }]
    });
    expect(detail.skus[0]).toMatchObject({ skuId: "MX-500", weightGrams: 500, isAvailable: true });
    expect(detail.foodInformation).toBeUndefined();
    expect(detail.ocopCertification?.label).toContain("OCOP-001");
  });
});
