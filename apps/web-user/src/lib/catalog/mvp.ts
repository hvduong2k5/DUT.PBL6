import { parseVndMoney } from "@/lib/api-contract";
import type {
  CatalogCategoriesResponse,
  CatalogCategory as ContractCategory,
  CatalogProductSummary as ContractProductSummary,
  CatalogProductsResponse,
  CatalogSearchResponse
} from "@/lib/api-contract";
import type { CatalogResult, DiscoveryConfig, ProductAvailability, ProductSummary } from "./types";

function categorySlug(name: string): string {
  return name
    .normalize("NFD")
    .replace(/[\u0300-\u036f]/gu, "")
    .replace(/đ/giu, "d")
    .toLocaleLowerCase("vi")
    .replace(/[^a-z0-9]+/gu, "-")
    .replace(/^-|-$/gu, "");
}

function availability(inStock: boolean | null, stockAvailable: number | null): ProductAvailability {
  if (inStock === null || stockAvailable === null) return "UNKNOWN";
  return inStock && stockAvailable > 0 ? "AVAILABLE" : "UNAVAILABLE";
}

export function mapCatalogCategories(categories: ContractCategory[]): DiscoveryConfig {
  return {
    categories: categories.map((category) => ({ slug: category.slug, name: category.name })),
    productTypes: [],
    weightOptions: [],
    pricePresets: [
      { label: "Dưới 200.000đ", maxPrice: 200_000 },
      { label: "200.000đ – 500.000đ", minPrice: 200_000, maxPrice: 500_000 },
      { label: "Trên 500.000đ", minPrice: 500_000 }
    ]
  };
}

export function mapCatalogCategoriesResponse(response: CatalogCategoriesResponse): DiscoveryConfig {
  return mapCatalogCategories(response.categories);
}

export function mapCatalogProductSummary(item: ContractProductSummary): ProductSummary {
  return {
    id: item.product_id,
    slug: item.slug,
    name: item.name,
    shortDescription: item.summary,
    categorySlug: categorySlug(item.category_name),
    categoryName: item.category_name,
    imageUrl: item.thumbnail_url,
    ocopStars: item.ocop_star,
    badges: item.ocop_star ? [`OCOP ${item.ocop_star} sao`] : [],
    matchedOffer: {
      skuId: item.product_id,
      label: "Giá từ",
      priceVnd: parseVndMoney(item.base_price),
      weightGrams: 0,
      packageType: "",
      availability: availability(item.in_stock, item.stock_available)
    }
  };
}

export function mapCatalogProducts(response: CatalogProductsResponse): CatalogResult {
  return {
    items: response.items.map(mapCatalogProductSummary),
    filteredCount: response.total,
    totalCount: response.total,
    page: response.page,
    pageSize: response.page_size,
    totalPages: response.total_pages
  };
}

export function mapCatalogSearch(response: CatalogSearchResponse, page: number, pageSize: number): CatalogResult {
  return {
    items: response.items.map(mapCatalogProductSummary),
    filteredCount: response.total,
    totalCount: response.total,
    page,
    pageSize,
    totalPages: response.total === 0 ? 0 : Math.ceil(response.total / pageSize)
  };
}
