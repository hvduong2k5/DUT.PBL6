import type { CatalogProductSummary } from "@/lib/api-contract";
import { mapCatalogCategories, mapCatalogProductSummary } from "./mvp";
import type { DiscoveryConfig, ProductSummary } from "./types";

export interface CustomerCoreCategory {
  category_id: string;
  name: string;
  slug: string;
  description: string;
  image_url: string;
  parent_id: string | null;
}

export interface CustomerCoreProductSummary {
  product_id: string;
  name: string;
  slug: string;
  summary: string;
  thumbnail_url: string;
  base_price: { currency_code: string; units: number; nanos: number };
  ocop_star: number | null;
  category_name: string;
  in_stock: boolean | null;
  stock_available?: number | null;
}

export interface CustomerCoreProductList {
  query?: string;
  items: CustomerCoreProductSummary[];
  total: number;
  page?: number;
  page_size?: number;
  total_pages?: number;
}

export function mapCustomerCoreConfig(categories: CustomerCoreCategory[]): DiscoveryConfig {
  return mapCatalogCategories(categories);
}

export function mapCustomerCoreProductSummary(item: CustomerCoreProductSummary): ProductSummary {
  return mapCatalogProductSummary({
    ...item,
    ocop_star: item.ocop_star ?? 0,
    stock_available: item.stock_available ?? (item.in_stock === null ? null : item.in_stock ? 1 : 0)
  } satisfies CatalogProductSummary);
}
