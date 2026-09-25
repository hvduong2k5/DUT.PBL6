import { describe, expect, it } from "vitest";
import { parseCartRoute, validateAddCartItemInput, validateUpdateCartItemInput } from "./validation";

describe("parseCartRoute", () => {
  it("allows only the cart capability paths", () => {
    expect(parseCartRoute("GET", [], new URLSearchParams())).toMatchObject({ operation: "get-cart" });
    expect(parseCartRoute("POST", ["items"], new URLSearchParams())).toMatchObject({ operation: "add-item" });
    expect(parseCartRoute("PATCH", ["items", "line-123"], new URLSearchParams())).toMatchObject({ operation: "update-item", itemId: "line-123" });
    expect(parseCartRoute("DELETE", ["items", "line-123"], new URLSearchParams())).toMatchObject({ operation: "remove-item" });
  });

  it("rejects arbitrary paths, queries and invalid identifiers", () => {
    expect(parseCartRoute("GET", ["items"], new URLSearchParams()).error?.field).toBe("path");
    expect(parseCartRoute("PATCH", ["items", "bad/id"], new URLSearchParams()).error?.field).toBe("itemId");
    expect(parseCartRoute("GET", [], new URLSearchParams("upstream=https://example.com")).error?.field).toBe("upstream");
  });
});

describe("cart body validation", () => {
  it("accepts exact SKU and a positive bounded quantity", () => {
    expect(validateAddCartItemInput({ skuId: "SKU-001-250-BOX", quantity: 2 }).data).toEqual({ skuId: "SKU-001-250-BOX", quantity: 2 });
    expect(validateUpdateCartItemInput({ quantity: 20 }).data).toEqual({ quantity: 20 });
  });

  it("rejects extra fields, zero, fractions and out-of-range quantity", () => {
    expect(validateAddCartItemInput({ skuId: "SKU-001", quantity: 1, upstream: "x" }).errors[0].field).toBe("body");
    expect(validateUpdateCartItemInput({ quantity: 0 }).errors[0].field).toBe("quantity");
    expect(validateUpdateCartItemInput({ quantity: 1.5 }).errors[0].field).toBe("quantity");
    expect(validateUpdateCartItemInput({ quantity: 21 }).errors[0].field).toBe("quantity");
  });
});
