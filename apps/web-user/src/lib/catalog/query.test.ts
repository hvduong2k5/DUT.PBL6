import { describe, expect, it } from "vitest";
import { parseCatalogQuery, toMockoonQuery } from "./query";

describe("parseCatalogQuery", () => {
  it("normalizes a valid combined query", () => {
    const result = parseCatalogQuery(new URLSearchParams("q=%20m%C3%A8%20%20x%E1%BB%ADng%20&category=me-xung-keo-hue&productType=CANDY&minPrice=100000&maxPrice=200000&weights=200,250,200&inStock=true&sort=PRICE_ASC&page=2&pageSize=9"));
    expect(result.errors).toEqual([]);
    expect(result.query).toMatchObject({ q: "mè xửng", weights: [200, 250], inStock: true, page: 2 });
  });

  it("rejects invalid ranges and enums", () => {
    const result = parseCatalogQuery(new URLSearchParams("q=a&productType=UNKNOWN&minPrice=300&maxPrice=100&sort=POPULAR&page=0"));
    expect(result.query).toBeUndefined();
    expect(result.errors.map((error) => error.field)).toEqual(expect.arrayContaining(["q", "productType", "price", "sort", "page"]));
  });

  it("uses relevance for keyword searches", () => {
    expect(parseCatalogQuery(new URLSearchParams("q=sen")).query?.sort).toBe("RELEVANCE");
  });
});

describe("toMockoonQuery", () => {
  it("maps the public contract to CRUD filters", () => {
    const validation = parseCatalogQuery(new URLSearchParams("category=banh-cung-dinh&weights=150,250&inStock=true&sort=PRICE_DESC"));
    const mapped = toMockoonQuery(validation.query!);
    expect(mapped.get("categorySlug_eq")).toBe("banh-cung-dinh");
    expect(mapped.get("weightGrams_like")).toBe("^(150|250)$");
    expect(mapped.get("isAvailable_eq")).toBe("true");
    expect(mapped.get("sort")).toBe("priceVnd");
    expect(mapped.get("order")).toBe("desc");
  });
});
