import type { CatalogProductDetail } from "@/lib/api-contract";
import { mapCatalogProductDetail } from "./mvp";
import type { ProductDetail } from "./types";

export interface CustomerCoreProductDetail {
  product_id: string;
  name: string;
  slug: string;
  description: string;
  images: string[];
  ocop_star: number | null;
  ocop_certificate_no: string | null;
  story: string | null;
  variants: Array<{
    variant_id: string;
    sku_code: string;
    name: string;
    weight_gram: number;
    price: { currency_code: string; units: number; nanos: number };
    original_price: { currency_code: string; units: number; nanos: number };
    stock_available: number | null;
  }>;
}

export function mapCustomerCoreProductDetail(product: CustomerCoreProductDetail): ProductDetail {
  return mapCatalogProductDetail({
    ...product,
    ocop_star: product.ocop_star ?? 0,
    ocop_certificate_no: product.ocop_certificate_no ?? "",
    story: product.story ?? ""
  } satisfies CatalogProductDetail);
}
