import { describe, expect, it } from "vitest";
import { parseAdminCatalogPath } from "./validation";

describe("admin catalog route validation", () => {
  it("accepts Product list and detail routes", () => {
    expect(parseAdminCatalogPath(["products"]).kind).toBe("product-list");
    expect(parseAdminCatalogPath(["products", "prod-001"])).toEqual({ kind: "product-detail", productId: "prod-001" });
  });

  it("accepts catalog write subresources", () => {
    expect(parseAdminCatalogPath(["products", "prod-001", "transition"])).toEqual({ kind: "product-subresource", productId: "prod-001", segments: ["transition"] });
    expect(parseAdminCatalogPath(["products", "prod-001", "skus"])).toEqual({ kind: "product-subresource", productId: "prod-001", segments: ["skus"] });
    expect(parseAdminCatalogPath(["products", "prod-001", "skus", "sku-001"])).toEqual({ kind: "product-subresource", productId: "prod-001", segments: ["skus", "sku-001"] });
    expect(parseAdminCatalogPath(["products", "prod-001", "skus", "sku-001", "prices"])).toEqual({ kind: "product-subresource", productId: "prod-001", segments: ["skus", "sku-001", "prices"] });
    expect(parseAdminCatalogPath(["products", "prod-001", "images"])).toEqual({ kind: "product-subresource", productId: "prod-001", segments: ["images"] });
    expect(parseAdminCatalogPath(["products", "prod-001", "images", "img-001"])).toEqual({ kind: "product-subresource", productId: "prod-001", segments: ["images", "img-001"] });
  });

  it("rejects unknown catalog routes", () => {
    expect(parseAdminCatalogPath(["prices"]).kind).toBe("invalid");
    expect(parseAdminCatalogPath(["products", "prod-001", "unknown"]).kind).toBe("invalid");
    expect(parseAdminCatalogPath(["products", "prod-001", "skus", "sku-001", "unknown"]).kind).toBe("invalid");
    expect(parseAdminCatalogPath(["products", "prod-001", "images", "img-001", "extra"]).kind).toBe("invalid");
  });
});
