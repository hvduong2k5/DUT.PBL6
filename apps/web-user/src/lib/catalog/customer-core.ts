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
  in_stock: boolean;
}

export interface CustomerCoreProductList {
  query?: string;
  items: CustomerCoreProductSummary[];
  total: number;
  page?: number;
  page_size?: number;
  total_pages?: number;
}

function categorySlug(name: string) {
  return name
    .normalize("NFD")
    .replace(/[\u0300-\u036f]/gu, "")
    .replace(/đ/giu, "d")
    .toLocaleLowerCase("vi")
    .replace(/[^a-z0-9]+/gu, "-")
    .replace(/^-|-$/gu, "");
}

export function mapCustomerCoreConfig(categories: CustomerCoreCategory[]): DiscoveryConfig {
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

export function mapCustomerCoreProductSummary(item: CustomerCoreProductSummary): ProductSummary {
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
      priceVnd: item.base_price.units,
      weightGrams: 0,
      packageType: "",
      isAvailable: item.in_stock
    }
  };
}
