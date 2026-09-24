export type ProductType = "CAKE" | "CANDY" | "TEA" | "GIFT_SET";
export type CatalogSort = "CURATED" | "RELEVANCE" | "PRICE_ASC" | "PRICE_DESC" | "NAME_ASC";

export interface ProductOffer {
  skuId: string;
  label: string;
  priceVnd: number;
  weightGrams: number;
  packageType: string;
  isAvailable: boolean;
}

export interface ProductSummary {
  id: string;
  slug: string;
  name: string;
  shortDescription: string;
  categorySlug: string;
  categoryName: string;
  productType: ProductType;
  imageUrl: string;
  ocopStars: number | null;
  badges: string[];
  matchedOffer: ProductOffer;
}

export interface CatalogOption<T extends string = string> {
  value: T;
  label: string;
}

export interface CatalogCategory {
  slug: string;
  name: string;
}

export interface PricePreset {
  label: string;
  minPrice?: number;
  maxPrice?: number;
}

export interface DiscoveryConfig {
  categories: CatalogCategory[];
  productTypes: CatalogOption<ProductType>[];
  weightOptions: number[];
  pricePresets: PricePreset[];
}

export interface CatalogResult {
  items: ProductSummary[];
  filteredCount: number;
  totalCount: number;
  page: number;
  pageSize: number;
}

interface CatalogErrorBody {
  code?: string;
  message?: string;
  requestId?: string;
  errors?: Array<{ field: string; message: string }>;
}

export class CatalogApiError extends Error {
  readonly status: number;
  readonly code: string;
  readonly requestId?: string;
  readonly errors: Array<{ field: string; message: string }>;

  constructor(status: number, body: CatalogErrorBody) {
    super(body.message || "Không thể tải sản phẩm lúc này.");
    this.name = "CatalogApiError";
    this.status = status;
    this.code = body.code || "UNKNOWN_ERROR";
    this.requestId = body.requestId;
    this.errors = body.errors ?? [];
  }
}
