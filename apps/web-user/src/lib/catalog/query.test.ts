import { describe, expect, it } from "vitest";
import { parseCatalogQuery, toCustomerCoreQuery } from "./query";

describe("parseCatalogQuery", () => {
  it("normalizes a valid combined query", () => {
    const result = parseCatalogQuery(new URLSearchParams("q=%20m%C3%A8%20%20x%E1%BB%ADng%20&category=me-xung-keo-hue&productType=CANDY&minPrice=100000&maxPrice=200000&ocopStars=4&weights=200,250,200&inStock=true&sort=PRICE_ASC&page=2&pageSize=9"));
    expect(result.errors).toEqual([]);
    expect(result.query).toMatchObject({ q: "mè xửng", ocopStars: 4, weights: [200, 250], inStock: true, page: 2 });
  });

  it("rejects invalid ranges and enums", () => {
    const result = parseCatalogQuery(new URLSearchParams("q=a&productType=UNKNOWN&minPrice=300&maxPrice=100&ocopStars=2&sort=POPULAR&page=0"));
    expect(result.query).toBeUndefined();
    expect(result.errors.map((error) => error.field)).toEqual(expect.arrayContaining(["q", "productType", "price", "ocopStars", "sort", "page"]));
  });

  it("uses relevance for keyword searches", () => {
    expect(parseCatalogQuery(new URLSearchParams("q=sen")).query?.sort).toBe("RELEVANCE");
  });
});

describe("toCustomerCoreQuery", () => {
  it("maps the existing UI query to the customer product API", () => {
    const validation = parseCatalogQuery(new URLSearchParams("category=banh-cung-dinh&ocopStars=5&weights=150,250&inStock=true&sort=PRICE_DESC"));
    const mapped = toCustomerCoreQuery(validation.query!);
    expect(mapped.get("category_id")).toBe("banh-cung-dinh");
    expect(mapped.get("page_size")).toBe("9");
    expect(mapped.get("sort_by")).toBe("PRICE_DESC");
    expect(mapped.get("ocop_star")).toBe("5");
    expect(mapped.has("weights")).toBe(false);
    expect(mapped.has("in_stock")).toBe(false);
  });

  it("uses q and pagination for the search endpoint", () => {
    const validation = parseCatalogQuery(new URLSearchParams("q=m%C3%A8+x%E1%BB%ADng&page=2&pageSize=12"));
    expect(Object.fromEntries(toCustomerCoreQuery(validation.query!))).toMatchObject({ q: "mè xửng", page: "2", page_size: "12" });
  });
});
