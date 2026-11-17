import sample from "@/data/goihue-sample.json";
import { parseCatalogQuery } from "@/lib/catalog/query";
import { parseProductDetailRequest } from "@/lib/product-detail/validation";

export type StorefrontDataMode = "mock" | "real" | "auto";
export function storefrontDataMode(): StorefrontDataMode {
  const mode = process.env.STOREFRONT_DATA_MODE;
  return mode === "mock" || mode === "auto" ? mode : "real";
}

const products = sample.products.filter((p) => p.active);
const categories = sample.categories
  .filter((c) => c.active)
  .sort((a, b) => a.sortOrder - b.sortOrder);
const imageByCategory: Record<string, string> = {
  "me-xung-keo-hue": "/images/goi-hue/photo-sp-me-xung-dia.jpg",
  "tra-cung-dinh": "/images/goi-hue/photo-sp-tra-sen-lon.jpg",
  "qua-bieu": "/images/goi-hue/photo-sp-hop-qua-mo.jpg",
};
const normalize = (text: string) =>
  text
    .normalize("NFD")
    .replace(/[\u0300-\u036f]/gu, "")
    .replace(/đ/giu, "d")
    .toLowerCase();
const money = (units: number) => ({ currency_code: "VND", units, nanos: 0 });
const weight = (name: string) => Number(name.match(/(\d+)\s*g\b/iu)?.[1] ?? 0);
const photo = (p: (typeof products)[number]) =>
  imageByCategory[p.category] ?? p.image;
const sku = (id: string) => "DEMO-" + id;

export function sampleCatalog(
  path: string[],
  params: URLSearchParams,
): Response {
  const headers = {
    "X-Storefront-Data-Source": "mock",
    "Cache-Control": "no-store",
  };
  const json = (data: unknown, status = 200) =>
    Response.json(data, { status, headers });
  if (path.join("/") === "discovery-config")
    return json({
      categories: categories.map((c) => ({ slug: c.code, name: c.name })),
      productTypes: [],
      weightOptions: [
        ...new Set(products.map((p) => weight(p.name)).filter(Boolean)),
      ],
      pricePresets: [
        { label: "Dưới 200.000đ", maxPrice: 200000 },
        { label: "200.000đ – 500.000đ", minPrice: 200000, maxPrice: 500000 },
      ],
    });
  if (path.join("/") === "products") {
    const validation = parseCatalogQuery(params);
    if (!validation.query)
      return json(
        {
          code: "INVALID_QUERY",
          message: "Bộ lọc không hợp lệ.",
          details: { errors: validation.errors },
        },
        400,
      );
    const q = validation.query;
    const types: Record<string, string> = {
      "me-xung-keo-hue": "CANDY",
      "banh-cung-dinh": "CAKE",
      "tra-cung-dinh": "TEA",
      "qua-bieu": "GIFT_SET",
    };
    let filtered = products.filter(
      (p) =>
        (!q.q ||
          normalize(p.name + " " + p.description).includes(normalize(q.q))) &&
        (!q.category ||
          p.category === q.category ||
          categories.find((c) => c.id === q.category)?.code === p.category) &&
        (!q.productType || types[p.category] === q.productType) &&
        (q.minPrice === undefined || p.price >= q.minPrice) &&
        (q.maxPrice === undefined || p.price <= q.maxPrice) &&
        !q.ocopStars &&
        (!q.inStock || p.stock > 0) &&
        (!q.weights.length || q.weights.includes(weight(p.name))),
    );
    if (q.sort === "PRICE_ASC")
      filtered = filtered.sort((a, b) => a.price - b.price);
    if (q.sort === "PRICE_DESC")
      filtered = filtered.sort((a, b) => b.price - a.price);
    if (q.sort === "NAME_ASC")
      filtered = filtered.sort((a, b) => a.name.localeCompare(b.name, "vi"));
    const total = filtered.length;
    const items = filtered
      .slice((q.page - 1) * q.pageSize, q.page * q.pageSize)
      .map((p) => ({
        product_id: p.id,
        slug: p.slug,
        name: p.name,
        summary: p.description,
        thumbnail_url: photo(p),
        base_price: money(p.price),
        ocop_star: 0,
        category_name:
          categories.find((c) => c.code === p.category)?.name ?? p.category,
        in_stock: p.stock > 0,
        stock_available: p.stock,
      }));
    return json(
      q.q
        ? { query: q.q, items, total }
        : {
            items,
            total,
            page: q.page,
            page_size: q.pageSize,
            total_pages: Math.ceil(total / q.pageSize),
          },
    );
  }
  const validation = parseProductDetailRequest(path, params);
  if (!validation.slug)
    return json(
      {
        code: "INVALID_PRODUCT_SLUG",
        message: "Đường dẫn sản phẩm không hợp lệ.",
      },
      400,
    );
  const p = products.find((p) => p.slug === validation.slug);
  if (!p)
    return json(
      { code: "NOT_FOUND", message: "Không tìm thấy sản phẩm mẫu." },
      404,
    );
  return json({
    product_id: p.id,
    slug: p.slug,
    name: p.name,
    description: p.description,
    images: [photo(p)],
    ocop_star: 0,
    ocop_certificate_no: "",
    story: p.description,
    variants: [
      {
        variant_id: sku(p.id),
        sku_code: sku(p.id),
        name: weight(p.name) ? weight(p.name) + "g" : "Hộp quà mẫu",
        weight_gram: weight(p.name),
        price: money(p.price),
        original_price: money(p.price),
        stock_available: p.stock,
      },
    ],
  });
}

export async function chooseCatalogSource(
  mode: StorefrontDataMode,
  live: () => Promise<Response>,
  fixture: () => Response,
  isCatalogEmpty: () => Promise<boolean>,
): Promise<Response> {
  if (mode === "mock") return fixture();
  if (mode === "real") return live();
  let response: Response;
  try {
    response = await live();
  } catch {
    return fixture();
  }
  if (response.status >= 500) return fixture();
  // Respect authentication, validation and valid empty search results.
  if (response.ok || response.status === 404) {
    const payload = await response
      .clone()
      .json()
      .catch(() => null);
    const empty =
      response.status === 404 ||
      payload?.total === 0 ||
      (Array.isArray(payload?.categories) && !payload.categories.length);
    if (empty && (await isCatalogEmpty())) return fixture();
  }
  return response;
}
