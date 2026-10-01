import type { CatalogSort, ProductType } from "./types";

const SLUG_PATTERN = /^[a-z0-9]+(?:-[a-z0-9]+)*$/u;
const PRODUCT_TYPES = new Set<ProductType>(["CAKE", "CANDY", "TEA", "GIFT_SET"]);
const SORTS = new Set<CatalogSort>(["CURATED", "RELEVANCE", "PRICE_ASC", "PRICE_DESC", "NAME_ASC"]);

export interface CatalogQuery {
  q?: string;
  category?: string;
  productType?: ProductType;
  minPrice?: number;
  maxPrice?: number;
  ocopStars?: number;
  weights: number[];
  inStock: boolean;
  sort: CatalogSort;
  page: number;
  pageSize: number;
}

export interface QueryValidation {
  query?: CatalogQuery;
  errors: Array<{ field: string; message: string }>;
}

function parseInteger(value: string | null, field: string, min: number, max: number, errors: QueryValidation["errors"]): number | undefined {
  if (value === null || value === "") return undefined;
  if (!/^\d+$/u.test(value)) {
    errors.push({ field, message: `${field} phải là số nguyên.` });
    return undefined;
  }
  const parsed = Number(value);
  if (parsed < min || parsed > max) {
    errors.push({ field, message: `${field} phải nằm trong khoảng ${min}–${max}.` });
    return undefined;
  }
  return parsed;
}

export function parseCatalogQuery(params: URLSearchParams): QueryValidation {
  const errors: QueryValidation["errors"] = [];
  const rawKeyword = params.get("q");
  const q = rawKeyword?.trim().replace(/\s+/gu, " ") || undefined;
  if (q && (q.length < 2 || q.length > 100)) {
    errors.push({ field: "q", message: "Từ khóa phải có từ 2 đến 100 ký tự." });
  }

  const category = params.get("category") || undefined;
  if (category && !SLUG_PATTERN.test(category)) errors.push({ field: "category", message: "Danh mục không hợp lệ." });

  const rawType = params.get("productType") || undefined;
  const productType = rawType && PRODUCT_TYPES.has(rawType as ProductType) ? rawType as ProductType : undefined;
  if (rawType && !productType) errors.push({ field: "productType", message: "Loại sản phẩm không hợp lệ." });

  const minPrice = parseInteger(params.get("minPrice"), "minPrice", 0, 100_000_000, errors);
  const maxPrice = parseInteger(params.get("maxPrice"), "maxPrice", 0, 100_000_000, errors);
  if (minPrice !== undefined && maxPrice !== undefined && minPrice > maxPrice) {
    errors.push({ field: "price", message: "Giá tối thiểu không được lớn hơn giá tối đa." });
  }

  const ocopStars = parseInteger(params.get("ocopStars"), "ocopStars", 3, 5, errors);

  const rawWeights = params.get("weights");
  const weights = rawWeights ? [...new Set(rawWeights.split(",").map(Number))] : [];
  if (weights.some((weight) => !Number.isInteger(weight) || weight < 1 || weight > 5_000) || weights.length > 10) {
    errors.push({ field: "weights", message: "Khối lượng lọc không hợp lệ." });
  }

  const rawInStock = params.get("inStock");
  if (rawInStock !== null && rawInStock !== "true" && rawInStock !== "false") {
    errors.push({ field: "inStock", message: "Trạng thái tồn kho không hợp lệ." });
  }
  const inStock = rawInStock === "true";

  const rawSort = params.get("sort");
  const fallbackSort: CatalogSort = q ? "RELEVANCE" : "CURATED";
  const sort = rawSort && SORTS.has(rawSort as CatalogSort) ? rawSort as CatalogSort : fallbackSort;
  if (rawSort && !SORTS.has(rawSort as CatalogSort)) errors.push({ field: "sort", message: "Cách sắp xếp không hợp lệ." });

  const page = parseInteger(params.get("page"), "page", 1, 10_000, errors) ?? 1;
  const pageSize = parseInteger(params.get("pageSize"), "pageSize", 1, 24, errors) ?? 9;

  if (errors.length) return { errors };
  return { errors, query: { q, category, productType, minPrice, maxPrice, ocopStars, weights, inStock, sort, page, pageSize } };
}

export function toCustomerCoreQuery(query: CatalogQuery): URLSearchParams {
  const result = new URLSearchParams({ page: String(query.page), page_size: String(query.pageSize) });
  if (query.q) result.set("q", query.q);
  if (query.category) result.set("category_id", query.category);
  if (query.minPrice !== undefined) result.set("min_price", String(query.minPrice));
  if (query.maxPrice !== undefined) result.set("max_price", String(query.maxPrice));
  if (query.ocopStars !== undefined) result.set("ocop_star", String(query.ocopStars));

  const sorting: Partial<Record<CatalogSort, "PRICE_ASC" | "PRICE_DESC" | "BEST_SELLING">> = {
    CURATED: "BEST_SELLING",
    PRICE_ASC: "PRICE_ASC",
    PRICE_DESC: "PRICE_DESC"
  };
  const sort = sorting[query.sort];
  if (sort && !query.q) result.set("sort_by", sort);
  return result;
}
