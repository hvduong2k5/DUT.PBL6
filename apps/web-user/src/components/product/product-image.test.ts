import { describe, expect, it } from "vitest";
import { PRODUCT_IMAGE_FALLBACK, resolveProductImageSource } from "./product-image";

describe("resolveProductImageSource", () => {
  it("uses the local fallback for the placeholder CDN returned by mobile_pbl", () => {
    expect(resolveProductImageSource("https://cdn.omama.vn/products/mx_gion_1.jpg")).toBe(PRODUCT_IMAGE_FALLBACK);
  });

  it("keeps valid local and external image sources", () => {
    expect(resolveProductImageSource("/images/custom.jpg")).toBe("/images/custom.jpg");
    expect(resolveProductImageSource("https://res.cloudinary.com/demo/image/upload/sample.jpg"))
      .toBe("https://res.cloudinary.com/demo/image/upload/sample.jpg");
  });

  it("uses the fallback for missing or malformed sources", () => {
    expect(resolveProductImageSource()).toBe(PRODUCT_IMAGE_FALLBACK);
    expect(resolveProductImageSource("not a url")).toBe(PRODUCT_IMAGE_FALLBACK);
  });
});
