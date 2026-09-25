import { describe, expect, it } from "vitest";
import { parseProductDetailRequest, toProductDetailUpstreamPath } from "./validation";

describe("parseProductDetailRequest", () => {
  it("accepts one valid product slug and a local scenario", () => {
    expect(parseProductDetailRequest(
      ["products", "banh-ngu-sac-cung-dinh"],
      new URLSearchParams("mockScenario=product-slow")
    )).toEqual({ slug: "banh-ngu-sac-cung-dinh", mockScenario: "product-slow" });
  });

  it("rejects invalid and nested paths", () => {
    expect(parseProductDetailRequest(["products", "Bánh Huế"], new URLSearchParams()).error?.field).toBe("slug");
    expect(parseProductDetailRequest(["products", "valid", "nested"], new URLSearchParams()).error?.field).toBe("path");
  });

  it("rejects arbitrary query parameters", () => {
    const result = parseProductDetailRequest(["products", "valid-slug"], new URLSearchParams("url=https://example.com"));
    expect(result.error).toMatchObject({ field: "url" });
  });
});

describe("toProductDetailUpstreamPath", () => {
  it("maps a validated slug to the fixed upstream namespace", () => {
    expect(toProductDetailUpstreamPath("tra-sen-tinh-tam")).toBe("catalog/products/tra-sen-tinh-tam");
  });
});
