import { afterEach, describe, expect, it, vi } from "vitest";
import {
  chooseCatalogSource,
  sampleCatalog,
  storefrontDataMode,
} from "./storefront-sample";

afterEach(() => vi.unstubAllEnvs());
const fixture = () => Response.json({ items: ["sample"] });
describe("storefront source switching", () => {
  it("defaults to real and supports explicit mock/auto", () => {
    vi.stubEnv("STOREFRONT_DATA_MODE", "");
    expect(storefrontDataMode()).toBe("real");
    vi.stubEnv("STOREFRONT_DATA_MODE", "auto");
    expect(storefrontDataMode()).toBe("auto");
  });
  it("mock never calls upstream and real never substitutes failures", async () => {
    const live = vi.fn(async () => Response.json({}, { status: 503 }));
    expect(
      (await chooseCatalogSource("mock", live, fixture, async () => false))
        .status,
    ).toBe(200);
    expect(live).not.toHaveBeenCalled();
    expect(
      (await chooseCatalogSource("real", live, fixture, async () => true))
        .status,
    ).toBe(503);
  });
  it("auto handles outage and empty catalogue but preserves real filtered zero and permission errors", async () => {
    const unavailable = async () => Response.json({}, { status: 503 });
    expect(
      await (
        await chooseCatalogSource(
          "auto",
          unavailable,
          fixture,
          async () => false,
        )
      ).json(),
    ).toEqual({ items: ["sample"] });
    for (const status of [400, 401, 403]) {
      expect(
        (
          await chooseCatalogSource(
            "auto",
            async () => Response.json({}, { status }),
            fixture,
            async () => true,
          )
        ).status,
      ).toBe(status);
    }
    const empty = async () => Response.json({ items: [], total: 0 });
    expect(
      await (
        await chooseCatalogSource("auto", empty, fixture, async () => true)
      ).json(),
    ).toEqual({ items: ["sample"] });
    expect(
      await (
        await chooseCatalogSource("auto", empty, fixture, async () => false)
      ).json(),
    ).toEqual({ items: [], total: 0 });
    expect(
      (
        await chooseCatalogSource(
          "auto",
          async () => Response.json({}, { status: 404 }),
          fixture,
          async () => false,
        )
      ).status,
    ).toBe(404);
  });
  it("preserves SQL prices, filters Vietnamese queries and keeps list/detail identifiers aligned", async () => {
    const response = sampleCatalog(
      ["products"],
      new URLSearchParams("q=me+xung&sort=PRICE_ASC&pageSize=1"),
    );
    const list = await response.json();
    expect(response.headers.get("X-Storefront-Data-Source")).toBe("mock");
    expect(list.total).toBe(3);
    expect(list.items[0].base_price.units).toBe(74998);
    const detail = await sampleCatalog(
      ["products", list.items[0].slug],
      new URLSearchParams(),
    ).json();
    expect(detail.product_id).toBe(list.items[0].product_id);
    expect(detail.variants[0].sku_code).toMatch(/^DEMO-/);
    expect(detail.variants[0].weight_gram).toBe(250);
    expect(detail.ocop_star).toBe(0);
  });
  it("keeps no-match, unknown slugs and invalid filters truthful in mock mode", async () => {
    expect(
      (
        await sampleCatalog(
          ["products"],
          new URLSearchParams("q=khong-co-san-pham"),
        ).json()
      ).total,
    ).toBe(0);
    expect(
      sampleCatalog(["products", "khong-co"], new URLSearchParams()).status,
    ).toBe(404);
    expect(
      sampleCatalog(["products"], new URLSearchParams("minPrice=nope")).status,
    ).toBe(400);
    expect(
      sampleCatalog(["products"], new URLSearchParams("ocopStars=4")).status,
    ).toBe(200);
    expect(
      (
        await sampleCatalog(
          ["products"],
          new URLSearchParams("ocopStars=4"),
        ).json()
      ).total,
    ).toBe(0);
  });
});
