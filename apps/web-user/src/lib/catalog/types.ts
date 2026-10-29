export type ProductType = "CAKE" | "CANDY" | "TEA" | "GIFT_SET";
export type CatalogSort = "CURATED" | "RELEVANCE" | "PRICE_ASC" | "PRICE_DESC" | "NAME_ASC";
export type ProductAvailability = "AVAILABLE" | "UNAVAILABLE" | "UNKNOWN";

export interface ProductOffer {
  skuId: string;
  label: string;
  priceVnd: number;
  weightGrams: number;
  packageType: string;
  availability: ProductAvailability;
}

export interface ProductSummary {
  id: string;
  slug: string;
  name: string;
  shortDescription: string;
  categorySlug: string;
  categoryName: string;
  productType?: ProductType;
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
  totalPages: number;
}
