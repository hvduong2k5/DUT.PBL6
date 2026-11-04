import { describe, expect, it } from "vitest";
import { parseAdminCatalogPath } from "./validation";

describe("admin catalog route validation", () => {
  it("accepts only Product list and detail routes", () => {
    expect(parseAdminCatalogPath(["products"]).kind).toBe("product-list");
    expect(parseAdminCatalogPath(["products", "prod-001"])).toEqual({ kind: "product-detail", productId: "prod-001" });
    expect(parseAdminCatalogPath(["products", "prod-001", "skus"]).kind).toBe("invalid");
    expect(parseAdminCatalogPath(["prices"]).kind).toBe("invalid");
  });
});
